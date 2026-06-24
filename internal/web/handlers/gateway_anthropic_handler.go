package handlers

import (
	"fmt"
	"io"
	"net/http"
	"time"

	"llm-gateway/internal/provider"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
)

type AnthropicGatewayHandler struct {
	GatewayBase
}

func (h *AnthropicGatewayHandler) RegisterRoutes(g *echo.Group) {
	g.POST("/messages", h.HandleMessages)
}

func (h *AnthropicGatewayHandler) HandleMessages(c echo.Context) error {
	// 解析 LLM 请求
	llmReq, err := provider.ParseLLMRequest(c.Request(), provider.APITypeAnthropic)
	if err != nil {
		return h.errorJSON(c, http.StatusBadRequest, err.Error())
	}

	// 设置 traceID
	traceID := c.Get("trace_id").(string)
	if traceID == "" {
		traceID = uuid.New().String()
		c.Set("trace_id", traceID)
	}

	adapter, providerModel, passthroughLevel, err := h.resolveProvider(llmReq.Model)
	if err != nil {
		return h.errorJSON(c, http.StatusNotFound, "no provider available")
	}

	// 根据 Provider 支持的协议决定处理方式
	if adapter.SupportAnthropic() {
		// 直接透传 Anthropic 请求
		if llmReq.Stream {
			return h.handleStream(c, adapter, llmReq, providerModel, passthroughLevel, false)
		}
		return h.handleNonStream(c, adapter, llmReq, providerModel, passthroughLevel, false)
	} else if adapter.SupportOpenAI() {
		// 需要转换格式
		if llmReq.Stream {
			return h.handleStream(c, adapter, llmReq, providerModel, passthroughLevel, true)
		}
		return h.handleNonStream(c, adapter, llmReq, providerModel, passthroughLevel, true)
	}

	return h.errorJSON(c, http.StatusNotFound, "no provider available")
}

func (h *AnthropicGatewayHandler) handleNonStream(c echo.Context, adapter *provider.Adapter,
	llmReq *provider.LLMRequest, providerModel, passthroughLevel string, needConvert bool) error {

	startTime := time.Now()
	traceID := c.Get("trace_id").(string)
	uid, kid := extractUserInfo(c)

	var resp *provider.LLMResponse
	var err error

	if needConvert {
		// 需要转换：Anthropic → OpenAI → Provider → OpenAI → Anthropic
		openaiReq, _ := llmReq.ToOpenAI()
		resp, err = adapter.ChatCompletion(c.Request().Context(), openaiReq)
		if err == nil {
			resp, err = resp.ToAnthropic()
		}
	} else {
		// 直接透传
		resp, err = adapter.Message(c.Request().Context(), llmReq)
	}

	duration := time.Since(startTime).Milliseconds()

	if err != nil {
		h.recordRequest(traceID, uid, kid,
			llmReq.Model, providerModel, string(provider.APITypeAnthropic), adapter.Type(), passthroughLevel, false,
			0, 0, 0, 0, http.StatusBadGateway, err.Error(), duration, c, llmReq.BodyRaw, nil, nil)
		return h.errorJSON(c, http.StatusBadGateway, err.Error())
	}

	// 读取响应用于日志
	respBody, _ := io.ReadAll(resp.Body)
	resp.Body.Close()

	h.recordRequest(traceID, uid, kid,
		llmReq.Model, providerModel, string(provider.APITypeAnthropic), adapter.Type(), passthroughLevel, false,
		0, 0, 0, 0,
		http.StatusOK, "", duration, c, llmReq.BodyRaw, respBody, nil)

	return c.Blob(http.StatusOK, "application/json", respBody)
}

func (h *AnthropicGatewayHandler) handleStream(c echo.Context, adapter *provider.Adapter,
	llmReq *provider.LLMRequest, providerModel, passthroughLevel string, needConvert bool) error {

	flusher, err := h.prepareStream(c)
	if err != nil {
		return h.errorJSON(c, http.StatusInternalServerError, err.Error())
	}

	startTime := time.Now()
	traceID := c.Get("trace_id").(string)
	uid, kid := extractUserInfo(c)
	collector := h.newChunkCollector(traceID)

	var chunkCh <-chan []byte
	var errCh <-chan error

	if needConvert {
		// 需要转换格式
		openaiReq, _ := llmReq.ToOpenAI()
		chunkCh, errCh = adapter.ChatCompletionStream(c.Request().Context(), openaiReq)
	} else {
		// 直接透传
		chunkCh, errCh = adapter.MessageStream(c.Request().Context(), llmReq)
	}

	var finalErr string
	statusCode := http.StatusOK

	for data := range chunkCh {
		if needConvert {
			// OpenAI 格式 chunk，需要转换为 Anthropic 格式
			// 这里简化处理，直接写入
			fmt.Fprintf(c.Response().Writer, "event: content_block_delta\ndata: %s\n\n", data)
		} else {
			// Anthropic 格式，直接写入
			fmt.Fprintf(c.Response().Writer, "event: content_block_delta\ndata: %s\n\n", data)
		}
		flusher.Flush()
		collector.Add(data)
	}

	select {
	case err := <-errCh:
		if err != nil {
			finalErr = err.Error()
			statusCode = http.StatusBadGateway
		}
	default:
	}

	// 写入结束事件
	fmt.Fprint(c.Response().Writer, "event: message_stop\ndata: {}\n\n")
	flusher.Flush()

	h.recordRequest(traceID, uid, kid,
		llmReq.Model, providerModel, string(provider.APITypeAnthropic), adapter.Type(), passthroughLevel, true,
		0, 0, 0, 0,
		statusCode, finalErr, time.Since(startTime).Milliseconds(), c, llmReq.BodyRaw, nil, collector.Chunks())
	return nil
}
