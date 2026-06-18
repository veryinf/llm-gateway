package handlers

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"llm-gateway/internal/model"
	"llm-gateway/internal/provider"
	"llm-gateway/internal/web/common"

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
	traceID := uuid.New().String()
	c.Set("trace_id", traceID)

	reqBytes, _ := io.ReadAll(c.Request().Body)
	c.Request().Body = io.NopCloser(bytes.NewReader(reqBytes))

	var req provider.AnthropicRequest
	if err := json.NewDecoder(bytes.NewReader(reqBytes)).Decode(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]interface{}{
			"error": fmt.Sprintf("invalid request: %v", err),
		})
	}

	llm, err := h.resolveProvider(req.Model)
	if err != nil {
		return c.JSON(http.StatusNotFound, map[string]interface{}{
			"error": "no provider available",
		})
	}

	cc, ok := c.(*common.LeContext)
	if !ok {
		return c.JSON(http.StatusInternalServerError, map[string]interface{}{
			"error": "invalid context type",
		})
	}
	var uid, kid uint
	if cc.AuthUser != nil {
		uid = cc.AuthUser.UID
	}
	if cc.UserKey != nil {
		kid = cc.UserKey.KeyID
	}

	startTime := time.Now()

	systemMsg := ""
	for _, msg := range req.Messages {
		if msg.Role == "system" {
			systemMsg = msg.Content
		}
	}
	chatReq := &provider.ChatRequest{
		Model:       req.Model,
		Stream:      req.Stream,
		MaxTokens:   req.MaxTokens,
		Temperature: req.Temperature,
		TopP:        req.TopP,
		Messages:    []provider.ChatMessage{},
	}
	if systemMsg != "" {
		chatReq.Messages = append(chatReq.Messages, provider.ChatMessage{
			Role: "system", Content: provider.FlexContent(systemMsg),
		})
	}
	for _, msg := range req.Messages {
		if msg.Role != "system" {
			chatReq.Messages = append(chatReq.Messages, provider.ChatMessage{
				Role: msg.Role, Content: provider.FlexContent(msg.Content),
			})
		}
	}

	if !req.Stream {
		return h.handleNonStream(c, llm, chatReq, req, traceID, uid, kid, startTime, reqBytes)
	}
	return h.handleStream(c, llm, chatReq, req, traceID, uid, kid, startTime, reqBytes)
}

func (h *AnthropicGatewayHandler) handleNonStream(c echo.Context, llm provider.LLMProvider,
	chatReq *provider.ChatRequest, req provider.AnthropicRequest,
	traceID string, uid, kid uint, startTime time.Time, reqBytes []byte) error {

	resp, err := llm.ChatCompletion(c.Request().Context(), chatReq)
	latencyMs := time.Since(startTime).Milliseconds()

	if err != nil {
		h.recordRequest(traceID, uid, kid, llm.ID(), req.Model, false,
			0, 0, 0, http.StatusBadGateway, err.Error(), latencyMs, c, reqBytes, nil, nil)
		return c.JSON(http.StatusBadGateway, map[string]interface{}{
			"error": err.Error(),
		})
	}

	respBytes, _ := json.Marshal(resp)

	anthResp := provider.AnthropicResponse{
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
	}

	h.recordRequest(traceID, uid, kid, llm.ID(), req.Model, false,
		resp.Usage.PromptTokens, resp.Usage.CompletionTokens, resp.Usage.TotalTokens,
		http.StatusOK, "", latencyMs, c, reqBytes, respBytes, nil)
	return c.JSON(http.StatusOK, anthResp)
}

func (h *AnthropicGatewayHandler) handleStream(c echo.Context, llm provider.LLMProvider,
	chatReq *provider.ChatRequest, req provider.AnthropicRequest,
	traceID string, uid, kid uint, startTime time.Time, reqBytes []byte) error {

	c.Response().Header().Set("Content-Type", "text/event-stream")
	c.Response().Header().Set("Cache-Control", "no-cache")
	c.Response().Header().Set("Connection", "keep-alive")

	chunkCh, errCh := llm.ChatCompletionStream(c.Request().Context(), chatReq)
	flusher, ok := c.Response().Writer.(http.Flusher)
	if !ok {
		return c.JSON(http.StatusInternalServerError, map[string]interface{}{
			"error": "streaming not supported",
		})
	}

	logDetail := h.LogDetail.Load()
	var chunkIndex int
	var chunks []*model.RequestChunk
	var promptTokens, completionTokens int
	var finalErr string
	statusCode := http.StatusOK

	for {
		select {
		case chunk, ok := <-chunkCh:
			if !ok {
				goto streamEnd
			}

			for _, choice := range chunk.Choices {
				if choice.Delta.Content != "" {
					event := provider.AnthropicStreamEvent{
						Type:  "content_block_delta",
						Index: 0,
						Delta: &provider.AnthropicDelta{
							Type: "text_delta",
							Text: choice.Delta.Content,
						},
					}
					data, _ := json.Marshal(event)
					line := "event: content_block_delta\ndata: " + string(data) + "\n\n"
					fmt.Fprint(c.Response().Writer, line)
					flusher.Flush()

					if logDetail {
						chunks = append(chunks, &model.RequestChunk{
							TraceID:    traceID,
							ChunkIndex: chunkIndex,
							ChunkData:  string(data),
							CreatedAt:  time.Now(),
						})
						chunkIndex++
					}
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
			goto streamEnd

		case <-c.Request().Context().Done():
			finalErr = "client disconnected"
			statusCode = 499
			goto streamEnd
		}
	}

streamEnd:
	delta := provider.AnthropicStreamEvent{
		Type: "message_delta",
		Delta: &provider.AnthropicDelta{
			StopReason: "end_turn",
		},
		Usage: &provider.AnthropicUsage{
			InputTokens:  promptTokens,
			OutputTokens: completionTokens,
		},
	}
	data, _ := json.Marshal(delta)
	fmt.Fprintf(c.Response().Writer, "event: message_delta\ndata: %s\n\n", string(data))
	fmt.Fprint(c.Response().Writer, "event: message_stop\ndata: {}\n\n")
	flusher.Flush()

	if promptTokens == 0 && completionTokens == 0 {
		promptTokens = 1
	}
	latencyMs := time.Since(startTime).Milliseconds()
	h.recordRequest(traceID, uid, kid, llm.ID(), req.Model, true,
		promptTokens, completionTokens, promptTokens+completionTokens,
		statusCode, finalErr, latencyMs, c, reqBytes, nil, chunks)

	return nil
}
