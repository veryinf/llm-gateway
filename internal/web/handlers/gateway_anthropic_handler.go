package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"

	"llm-gateway/internal/provider"

	"github.com/labstack/echo/v4"
)

type AnthropicGatewayHandler struct {
	GatewayBase
}

func (h *AnthropicGatewayHandler) RegisterRoutes(g *echo.Group) {
	g.POST("/messages", h.HandleMessages)
}

func (h *AnthropicGatewayHandler) HandleMessages(c echo.Context) error {
	var req provider.AnthropicRequest
	gwCtx, err := h.prepareRequest(c, &req)
	if err != nil {
		return h.errorJSON(c, http.StatusBadRequest, err.Error())
	}

	llm, err := h.resolveProvider(req.Model)
	if err != nil {
		return h.errorJSON(c, http.StatusNotFound, "no provider available")
	}

	chatReq := h.buildChatRequest(&req)

	if req.Stream {
		return h.handleStream(c, llm, chatReq, &req, gwCtx)
	}
	return h.handleNonStream(c, llm, chatReq, &req, gwCtx)
}

// buildChatRequest 将 AnthropicRequest 转换为统一的 ChatRequest
func (h *AnthropicGatewayHandler) buildChatRequest(req *provider.AnthropicRequest) *provider.ChatRequest {
	chatReq := &provider.ChatRequest{
		Model:       req.Model,
		Stream:      req.Stream,
		MaxTokens:   req.MaxTokens,
		Temperature: req.Temperature,
		TopP:        req.TopP,
	}

	for _, msg := range req.Messages {
		chatReq.Messages = append(chatReq.Messages, provider.ChatMessage{
			Role:    msg.Role,
			Content: provider.FlexContent(msg.Content),
		})
	}
	return chatReq
}

func (h *AnthropicGatewayHandler) handleNonStream(c echo.Context, llm provider.LLMProvider,
	chatReq *provider.ChatRequest, req *provider.AnthropicRequest, gwCtx *gatewayContext) error {

	resp, err := llm.ChatCompletion(c.Request().Context(), chatReq)
	latencyMs := gwCtx.Latency()

	if err != nil {
		h.recordRequest(gwCtx.TraceID, gwCtx.UserID, gwCtx.APIKeyID, llm.ID(), req.Model, false,
			0, 0, 0, http.StatusBadGateway, err.Error(), latencyMs, c, gwCtx.ReqBytes, nil, nil)
		return h.errorJSON(c, http.StatusBadGateway, err.Error())
	}

	respBytes, _ := json.Marshal(resp)
	h.recordRequest(gwCtx.TraceID, gwCtx.UserID, gwCtx.APIKeyID, llm.ID(), req.Model, false,
		resp.Usage.PromptTokens, resp.Usage.CompletionTokens, resp.Usage.TotalTokens,
		http.StatusOK, "", latencyMs, c, gwCtx.ReqBytes, respBytes, nil)

	return c.JSON(http.StatusOK, provider.AnthropicResponse{
		ID:    resp.ID,
		Type:  "message",
		Role:  "assistant",
		Model: resp.Model,
		Content: []provider.AnthropicContentBlock{
			{Type: "text", Text: string(resp.Choices[0].Message.Content)},
		},
		Usage: provider.AnthropicUsage{
			InputTokens:  resp.Usage.PromptTokens,
			OutputTokens: resp.Usage.CompletionTokens,
		},
	})
}

func (h *AnthropicGatewayHandler) handleStream(c echo.Context, llm provider.LLMProvider,
	chatReq *provider.ChatRequest, req *provider.AnthropicRequest, gwCtx *gatewayContext) error {

	flusher, err := h.prepareStream(c)
	if err != nil {
		return h.errorJSON(c, http.StatusInternalServerError, err.Error())
	}

	chunkCh, errCh := llm.ChatCompletionStream(c.Request().Context(), chatReq)
	collector := h.newChunkCollector(gwCtx.TraceID)

	var promptTokens, completionTokens int
	var finalErr string
	statusCode := http.StatusOK

streamLoop:
	for {
		select {
		case chunk, ok := <-chunkCh:
			if !ok {
				break streamLoop
			}
			for _, choice := range chunk.Choices {
				if choice.Delta.Content != "" {
					data := h.writeSSEEvent(c, flusher, "content_block_delta", provider.AnthropicStreamEvent{
						Type:  "content_block_delta",
						Index: 0,
						Delta: &provider.AnthropicDelta{Type: "text_delta", Text: choice.Delta.Content},
					})
					collector.Add(data)
				}
			}
			if chunk.Usage != nil {
				promptTokens = chunk.Usage.PromptTokens
				completionTokens = chunk.Usage.CompletionTokens
			}
		case err := <-errCh:
			if err != nil {
				finalErr = err.Error()
				statusCode = http.StatusBadGateway
			}
			break streamLoop
		case <-c.Request().Context().Done():
			finalErr = "client disconnected"
			statusCode = 499
			break streamLoop
		}
	}

	h.writeSSEEvent(c, flusher, "message_delta", provider.AnthropicStreamEvent{
		Type:  "message_delta",
		Delta: &provider.AnthropicDelta{StopReason: "end_turn"},
		Usage: &provider.AnthropicUsage{InputTokens: promptTokens, OutputTokens: completionTokens},
	})
	fmt.Fprint(c.Response().Writer, "event: message_stop\ndata: {}\n\n")
	flusher.Flush()

	if promptTokens == 0 && completionTokens == 0 {
		promptTokens = 1
	}
	totalTokens := promptTokens + completionTokens
	h.recordRequest(gwCtx.TraceID, gwCtx.UserID, gwCtx.APIKeyID, llm.ID(), req.Model, true,
		promptTokens, completionTokens, totalTokens,
		statusCode, finalErr, gwCtx.Latency(), c, gwCtx.ReqBytes, nil, collector.Chunks())
	return nil
}

// writeSSEEvent 写入一条 SSE 事件并返回序列化后的数据
func (h *AnthropicGatewayHandler) writeSSEEvent(c echo.Context, flusher http.Flusher, event string, v interface{}) []byte {
	data, _ := json.Marshal(v)
	fmt.Fprintf(c.Response().Writer, "event: %s\ndata: %s\n\n", event, data)
	flusher.Flush()
	return data
}
