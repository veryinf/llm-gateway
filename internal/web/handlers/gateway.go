package handlers

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sync/atomic"
	"time"

	"llm-gateway/internal/model"
	"llm-gateway/internal/provider"
	"llm-gateway/internal/router"
	"llm-gateway/internal/web/common"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
)

type GatewayHandler struct {
	common.BaseHandler
	ModelRouter *router.ModelRouter
	StatsSvc    StatsRecorder
	ChunkSvc    ChunkRecorder
	LogDetail   *atomic.Bool
}

type StatsRecorder interface {
	Record(log *model.RequestLog)
}

type ChunkRecorder interface {
	Record(chunk *model.RequestChunk)
}

func (h *GatewayHandler) RegisterRoutes(g *echo.Group) {
	g.POST("/chat/completions", h.HandleChatCompletion)
	g.POST("/messages", h.HandleMessages)
	g.GET("/models", h.HandleListModels)
}

func (h *GatewayHandler) HandleChatCompletion(c echo.Context) error {
	traceID := uuid.New().String()
	c.Set("trace_id", traceID)

	rawBody, _ := io.ReadAll(c.Request().Body)
	reqBytes := rawBody
	c.Request().Body = io.NopCloser(bytes.NewReader(reqBytes))

	var req provider.ChatRequest
	if err := json.NewDecoder(bytes.NewReader(reqBytes)).Decode(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]interface{}{
			"error": fmt.Sprintf("invalid request: %v", err),
		})
	}

	llm, err := h.ModelRouter.Resolve(req.Model)
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
		uid = cc.AuthUser.ID
	}
	kid = cc.APIKeyID

	startTime := time.Now()

	if !req.Stream {
		return h.handleNonStream(c, llm, &req, traceID, uid, kid, startTime, reqBytes)
	}
	return h.handleStream(c, llm, &req, traceID, uid, kid, startTime, reqBytes)
}

func (h *GatewayHandler) handleNonStream(c echo.Context, llm provider.LLMProvider, req *provider.ChatRequest,
	traceID string, userID, apiKeyID uint, startTime time.Time, reqBytes []byte) error {

	resp, err := llm.ChatCompletion(c.Request().Context(), req)
	latencyMs := time.Since(startTime).Milliseconds()

	if err != nil {
		h.recordRequest(traceID, userID, apiKeyID, llm.ID(), req.Model, false,
			0, 0, 0, http.StatusBadGateway, err.Error(), latencyMs, c, reqBytes, nil, nil)
		return c.JSON(http.StatusBadGateway, map[string]interface{}{
			"error": err.Error(),
		})
	}

	respBytes, _ := json.Marshal(resp)
	h.recordRequest(traceID, userID, apiKeyID, llm.ID(), req.Model, false,
		resp.Usage.PromptTokens, resp.Usage.CompletionTokens, resp.Usage.TotalTokens,
		http.StatusOK, "", latencyMs, c, reqBytes, respBytes, nil)
	return c.JSON(http.StatusOK, resp)
}

func (h *GatewayHandler) handleStream(c echo.Context, llm provider.LLMProvider, req *provider.ChatRequest,
	traceID string, userID, apiKeyID uint, startTime time.Time, reqBytes []byte) error {

	c.Response().Header().Set("Content-Type", "text/event-stream")
	c.Response().Header().Set("Cache-Control", "no-cache")
	c.Response().Header().Set("Connection", "keep-alive")

	chunkCh, errCh := llm.ChatCompletionStream(c.Request().Context(), req)

	flusher, ok := c.Response().Writer.(http.Flusher)
	if !ok {
		return c.JSON(http.StatusInternalServerError, map[string]interface{}{
			"error": "streaming not supported",
		})
	}

	logDetail := h.LogDetail.Load()
	var chunkIndex int
	var chunks []*model.RequestChunk
	var totalTokens, promptTokens, completionTokens int
	var finalErr string
	statusCode := http.StatusOK

	for chunk := range chunkCh {
		data, err := json.Marshal(chunk)
		if err != nil {
			continue
		}
		line := "data: " + string(data) + "\n\n"
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

		if chunk.Usage != nil {
			totalTokens = chunk.Usage.TotalTokens
			promptTokens = chunk.Usage.PromptTokens
			completionTokens = chunk.Usage.CompletionTokens
		}
	}

	select {
	case err := <-errCh:
		if err != nil {
			finalErr = err.Error()
			statusCode = http.StatusBadGateway
			errData, _ := json.Marshal(map[string]string{"error": err.Error()})
			fmt.Fprintf(c.Response().Writer, "data: %s\n\n", string(errData))
			flusher.Flush()
		}
	default:
	}

	fmt.Fprint(c.Response().Writer, "data: [DONE]\n\n")
	flusher.Flush()

	latencyMs := time.Since(startTime).Milliseconds()
	h.recordRequest(traceID, userID, apiKeyID, llm.ID(), req.Model, true,
		promptTokens, completionTokens, totalTokens,
		statusCode, finalErr, latencyMs, c, reqBytes, nil, chunks)

	return nil
}

func (h *GatewayHandler) recordRequest(traceID string, userID, apiKeyID uint,
	providerName, modelName string, isStream bool,
	promptTokens, completionTokens, totalTokens int,
	statusCode int, errMsg string, latencyMs int64, c echo.Context,
	reqBytes, respBytes []byte, chunks []*model.RequestChunk) {

	if h.StatsSvc == nil {
		return
	}

	logDetail := h.LogDetail.Load()
	var requestBody, responseBody string
	var isDetail bool
	if logDetail {
		isDetail = true
		requestBody = truncateStr(string(reqBytes), 65536)
		if respBytes != nil {
			responseBody = truncateStr(string(respBytes), 65536)
		}
	}

	h.StatsSvc.Record(&model.RequestLog{
		TraceID:          traceID,
		UserID:           userID,
		APIKeyID:         apiKeyID,
		ModelName:        modelName,
		IsStream:         isStream,
		PromptTokens:     promptTokens,
		CompletionTokens: completionTokens,
		TotalTokens:      totalTokens,
		RequestBody:      requestBody,
		ResponseBody:     responseBody,
		IsDetail:         isDetail,
		StatusCode:       statusCode,
		ErrorMessage:     truncateStr(errMsg, 4096),
		LatencyMs:        latencyMs,
		Cost:             0,
		IPAddress:        c.RealIP(),
		UserAgent:        truncateStr(c.Request().Header.Get("User-Agent"), 512),
		CreatedAt:        time.Now(),
	})

	if logDetail && len(chunks) > 0 && h.ChunkSvc != nil {
		for _, chunk := range chunks {
			h.ChunkSvc.Record(chunk)
		}
	}
}

func (h *GatewayHandler) HandleListModels(c echo.Context) error {
	models := h.ModelRouter.GetAllModels()
	modelList := make([]provider.ModelInfo, len(models))
	for i, m := range models {
		modelList[i] = provider.ModelInfo{
			ID:     m.ID,
			Object: "model",
		}
	}

	return c.JSON(http.StatusOK, map[string]interface{}{
		"object": "list",
		"data":   modelList,
	})
}

func (h *GatewayHandler) HandleMessages(c echo.Context) error {
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

	llm, err := h.ModelRouter.Resolve(req.Model)
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
		uid = cc.AuthUser.ID
	}
	kid = cc.APIKeyID

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
			ID:   resp.ID,
			Type: "message",
			Role: "assistant",
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

func truncateStr(s string, maxLen int) string {
	if len(s) > maxLen {
		return s[:maxLen]
	}
	return s
}
