package handlers

import (
	"fmt"
	"io"
	"net/http"

	"llm-gateway/internal/model"
	"llm-gateway/internal/provider"
	"llm-gateway/internal/service"
	"llm-gateway/internal/web/common"

	"github.com/labstack/echo/v4"
)

type AnthropicGatewayHandler struct {
	common.GatewayBase
}

func (h *AnthropicGatewayHandler) RegisterRoutes(g *echo.Group) {
	g.POST("/messages", h.HandleMessages)
}

func (h *AnthropicGatewayHandler) HandleMessages(c echo.Context) error {
	llmReq, err := provider.NewLLMRequest(c.Request(), model.APITypeAnthropic)
	if err != nil {
		return h.ErrorJSON(c, http.StatusBadRequest, err.Error())
	}

	router, err := h.RouterService.ResolveProvider(llmReq.Model)
	if err != nil {
		return h.ErrorJSON(c, http.StatusNotFound, "no provider available")
	}

	if router.Adapter.SupportAnthropic() {
		if llmReq.Stream {
			return h.handleStream(c, llmReq, router, false)
		}
		return h.handleNonStream(c, llmReq, router, false)
	} else if router.Adapter.SupportOpenAI() {
		if llmReq.Stream {
			return h.handleStream(c, llmReq, router, true)
		}
		return h.handleNonStream(c, llmReq, router, true)
	}

	return h.ErrorJSON(c, http.StatusNotFound, "no provider available")
}

func (h *AnthropicGatewayHandler) handleNonStream(c echo.Context, llmReq *provider.LLMRequest, router *service.RouterResult, needConvert bool) error {
	ctx := h.Context(c)

	var resp *provider.LLMResponse
	var err error

	if needConvert {
		openaiReq, _ := llmReq.ToOpenAI()
		resp, err = router.Adapter.ChatCompletion(c.Request().Context(), openaiReq)
		if err == nil {
			resp, err = resp.ToAnthropic()
		}
	} else {
		resp, err = router.Adapter.Message(c.Request().Context(), llmReq)
	}

	if err != nil {
		h.RecordRequest(ctx, router, llmReq, nil)
		return h.ErrorJSON(c, http.StatusBadGateway, err.Error())
	}

	c.Response().Header().Set("Content-Type", "application/json")
	c.Response().WriteHeader(http.StatusOK)

	_, copyErr := io.Copy(c.Response().Writer, resp.Response.Body)
	_ = resp.Response.Body.Close()

	if copyErr != nil {
		h.RecordRequest(ctx, router, llmReq, resp)
		return h.ErrorJSON(c, http.StatusBadGateway, copyErr.Error())
	}
	h.RecordRequest(ctx, router, llmReq, resp)
	return nil
}

func (h *AnthropicGatewayHandler) handleStream(c echo.Context, llmReq *provider.LLMRequest, router *service.RouterResult, needConvert bool) error {
	flusher, err := h.PrepareStream(c)
	if err != nil {
		return h.ErrorJSON(c, http.StatusInternalServerError, err.Error())
	}

	ctx := h.Context(c)
	collector := h.NewChunkCollector(ctx.TraceID)

	var chunkCh <-chan []byte
	var errCh <-chan error

	if needConvert {
		openaiReq, _ := llmReq.ToOpenAI()
		chunkCh, errCh = router.Adapter.ChatCompletionStream(c.Request().Context(), openaiReq)
	} else {
		chunkCh, errCh = router.Adapter.MessageStream(c.Request().Context(), llmReq)
	}

	for data := range chunkCh {
		fmt.Fprintf(c.Response().Writer, "event: content_block_delta\ndata: %s\n\n", data)
		flusher.Flush()
		collector.Add(data)
	}

	<-errCh

	fmt.Fprint(c.Response().Writer, "event: message_stop\ndata: {}\n\n")
	flusher.Flush()

	h.RecordRequest(ctx, router, llmReq, nil)
	h.RecordChunks(collector.Chunks())
	return nil
}