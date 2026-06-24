package handlers

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"llm-gateway/internal/model"
	"llm-gateway/internal/provider"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
)

type OpenAIGatewayHandler struct {
	GatewayBase
}

func (h *OpenAIGatewayHandler) RegisterRoutes(g *echo.Group) {
	g.POST("/chat/completions", h.HandleChatCompletion)
	g.GET("/models", h.HandleListModels)
}

func (h *OpenAIGatewayHandler) HandleChatCompletion(c echo.Context) error {
	// 解析 LLM 请求
	llmReq, err := provider.ParseLLMRequest(c.Request(), provider.APITypeOpenAI)
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

	if llmReq.Stream {
		return h.handleStream(c, adapter, llmReq, providerModel, passthroughLevel)
	}
	return h.handleNonStream(c, adapter, llmReq, providerModel, passthroughLevel)
}

func (h *OpenAIGatewayHandler) handleNonStream(c echo.Context, adapter *provider.Adapter,
	llmReq *provider.LLMRequest, providerModel, passthroughLevel string) error {

	startTime := time.Now()
	traceID := c.Get("trace_id").(string)
	uid, kid := extractUserInfo(c)

	// 直接透传请求
	resp, err := adapter.ChatCompletion(c.Request().Context(), llmReq)
	duration := time.Since(startTime).Milliseconds()

	if err != nil {
		h.recordRequest(traceID, uid, kid,
			llmReq.Model, providerModel, string(provider.APITypeOpenAI), adapter.Type(), passthroughLevel, false,
			0, 0, 0, 0, http.StatusBadGateway, err.Error(), duration, c, llmReq.BodyRaw, nil, nil)
		return h.errorJSON(c, http.StatusBadGateway, err.Error())
	}

	// 读取响应用于日志
	respBody, _ := io.ReadAll(resp.Body)
	resp.Body.Close()

	h.recordRequest(traceID, uid, kid,
		llmReq.Model, providerModel, string(provider.APITypeOpenAI), adapter.Type(), passthroughLevel, false,
		0, 0, 0, 0, // token 信息需要从响应中解析
		http.StatusOK, "", duration, c, llmReq.BodyRaw, respBody, nil)

	// 返回原始响应
	return c.Blob(http.StatusOK, "application/json", respBody)
}

func (h *OpenAIGatewayHandler) handleStream(c echo.Context, adapter *provider.Adapter,
	llmReq *provider.LLMRequest, providerModel, passthroughLevel string) error {

	flusher, err := h.prepareStream(c)
	if err != nil {
		return h.errorJSON(c, http.StatusInternalServerError, err.Error())
	}

	startTime := time.Now()
	traceID := c.Get("trace_id").(string)
	uid, kid := extractUserInfo(c)
	collector := h.newChunkCollector(traceID)

	// 直接透传流式请求
	chunkCh, errCh := adapter.ChatCompletionStream(c.Request().Context(), llmReq)

	var finalErr string
	statusCode := http.StatusOK

	for data := range chunkCh {
		fmt.Fprintf(c.Response().Writer, "data: %s\n\n", data)
		flusher.Flush()
		collector.Add(data)
	}

	select {
	case err := <-errCh:
		if err != nil {
			finalErr = err.Error()
			statusCode = http.StatusBadGateway
			errData, _ := json.Marshal(map[string]string{"error": err.Error()})
			fmt.Fprintf(c.Response().Writer, "data: %s\n\n", errData)
			flusher.Flush()
		}
	default:
	}

	fmt.Fprint(c.Response().Writer, "data: [DONE]\n\n")
	flusher.Flush()

	h.recordRequest(traceID, uid, kid,
		llmReq.Model, providerModel, string(provider.APITypeOpenAI), adapter.Type(), passthroughLevel, true,
		0, 0, 0, 0, // token 信息需要从 chunk 中解析
		statusCode, finalErr, time.Since(startTime).Milliseconds(), c, llmReq.BodyRaw, nil, collector.Chunks())
	return nil
}

func (h *OpenAIGatewayHandler) HandleListModels(c echo.Context) error {
	var userModels []model.UserModel
	if err := h.DB.Where("is_active = ?", true).Find(&userModels).Error; err != nil {
		return c.JSON(http.StatusOK, map[string]interface{}{
			"object": "list",
			"data":   []provider.ModelInfo{},
		})
	}

	var modelList []provider.ModelInfo
	for _, um := range userModels {
		var count int64
		h.DB.Model(&model.UserModelRouter{}).Where("user_model_id = ?", um.UserModelID).Count(&count)
		if count > 0 {
			modelList = append(modelList, provider.ModelInfo{
				ID:     um.Name,
				Object: "model",
			})
		}
	}

	return c.JSON(http.StatusOK, map[string]interface{}{
		"object": "list",
		"data":   modelList,
	})
}
