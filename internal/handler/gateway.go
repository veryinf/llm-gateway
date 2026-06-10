package handler

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"llm-gateway/internal/middleware"
	"llm-gateway/internal/model"
	"llm-gateway/internal/provider"
	"llm-gateway/internal/router"
	"llm-gateway/internal/service"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
)

type GatewayHandler struct {
	router   *router.ModelRouter
	statsSvc *service.StatsService
	auditSvc *service.AuditService
}

func NewGatewayHandler(
	router *router.ModelRouter,
	statsSvc *service.StatsService,
	auditSvc *service.AuditService,
) *GatewayHandler {
	return &GatewayHandler{
		router:   router,
		statsSvc: statsSvc,
		auditSvc: auditSvc,
	}
}

func (h *GatewayHandler) HandleChatCompletion(c echo.Context) error {
	traceID := uuid.New().String()
	c.Set("trace_id", traceID)

	rawBody, _ := io.ReadAll(c.Request().Body)
	c.Request().Body = io.NopCloser(bytes.NewReader(rawBody))
	reqSummary := truncateStr(string(rawBody), 1024)

	var req provider.ChatRequest
	if err := json.NewDecoder(bytes.NewReader(rawBody)).Decode(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]interface{}{
			"error": fmt.Sprintf("invalid request: %v", err),
		})
	}

	llm, err := h.router.Resolve(req.Model)
	if err != nil {
		return c.JSON(http.StatusNotFound, map[string]interface{}{
			"error": "no provider available",
		})
	}

	userID := c.Get(middleware.CtxKeyUserID)
	apiKeyID := c.Get(middleware.CtxKeyAPIKeyID)

	var uid uint
	if v, ok := userID.(uint); ok {
		uid = v
	}
	var kid uint
	if v, ok := apiKeyID.(uint); ok {
		kid = v
	}

	startTime := time.Now()

	if !req.Stream {
		return h.handleNonStream(c, llm, &req, traceID, uid, kid, startTime, reqSummary)
	}
	return h.handleStream(c, llm, &req, traceID, uid, kid, startTime, reqSummary)
}

func (h *GatewayHandler) handleNonStream(c echo.Context, llm provider.LLMProvider, req *provider.ChatRequest,
	traceID string, userID, apiKeyID uint, startTime time.Time, reqSummary string) error {

	resp, err := llm.ChatCompletion(c.Request().Context(), req)
	latencyMs := time.Since(startTime).Milliseconds()

	if err != nil {
		h.recordAuditAndStats(traceID, userID, apiKeyID, llm.ID(), req.Model, req.Stream,
			0, 0, 0, http.StatusBadGateway, err.Error(), latencyMs, c, reqSummary, "")
		return c.JSON(http.StatusBadGateway, map[string]interface{}{
			"error": err.Error(),
		})
	}

	respSummary := summarizeResponse(resp)
	h.recordAuditAndStats(traceID, userID, apiKeyID, llm.ID(), req.Model, req.Stream,
		resp.Usage.PromptTokens, resp.Usage.CompletionTokens, resp.Usage.TotalTokens,
		http.StatusOK, "", latencyMs, c, reqSummary, respSummary)
	return c.JSON(http.StatusOK, resp)
}

func (h *GatewayHandler) handleStream(c echo.Context, llm provider.LLMProvider, req *provider.ChatRequest,
	traceID string, userID, apiKeyID uint, startTime time.Time, reqSummary string) error {

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

	var totalTokens, promptTokens, completionTokens int
	var finalErr string
	statusCode := http.StatusOK

	for chunk := range chunkCh {
		data, err := json.Marshal(chunk)
		if err != nil {
			continue
		}
		fmt.Fprintf(c.Response().Writer, "data: %s\n\n", string(data))
		flusher.Flush()

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
	h.recordAuditAndStats(traceID, userID, apiKeyID, llm.ID(), req.Model, req.Stream,
		promptTokens, completionTokens, totalTokens,
		statusCode, finalErr, latencyMs, c, reqSummary, "")

	return nil
}

func (h *GatewayHandler) recordAuditAndStats(traceID string, userID, apiKeyID uint,
	providerName, modelName string, isStream bool,
	promptTokens, completionTokens, totalTokens int,
	statusCode int, errMsg string, latencyMs int64, c echo.Context,
	reqSummary string, respSummary string) {

	h.statsSvc.Record(&model.RequestLog{
		TraceID:          traceID,
		UserID:           userID,
		APIKeyID:         apiKeyID,
		ModelName:        modelName,
		IsStream:         isStream,
		PromptTokens:     promptTokens,
		CompletionTokens: completionTokens,
		TotalTokens:      totalTokens,
		StatusCode:       statusCode,
		ErrorMessage:     truncateStr(errMsg, 4096),
		LatencyMs:        latencyMs,
		Cost:             0,
	})

	h.auditSvc.Record(&model.AuditLog{
		TraceID:          traceID,
		UserID:           userID,
		APIKeyID:         apiKeyID,
		ModelName:        modelName,
		RequestSummary:   truncateJSON(reqSummary, 4096),
		ResponseSummary:  truncateJSON(respSummary, 4096),
		PromptTokens:     promptTokens,
		CompletionTokens: completionTokens,
		StatusCode:       statusCode,
		ErrorMessage:     truncateStr(errMsg, 4096),
		LatencyMs:        latencyMs,
		Cost:             0,
		IPAddress:        c.RealIP(),
		UserAgent:        truncateStr(c.Request().Header.Get("User-Agent"), 512),
	})
}

func (h *GatewayHandler) HandleListModels(c echo.Context) error {
	models := h.router.GetAllModels()
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

// HandleMessages Anthropic 原生 Messages API (POST /v1/messages)
func (h *GatewayHandler) HandleMessages(c echo.Context) error {
	traceID := uuid.New().String()
	c.Set("trace_id", traceID)

	var req provider.AnthropicRequest
	if err := json.NewDecoder(c.Request().Body).Decode(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]interface{}{
			"error": fmt.Sprintf("invalid request: %v", err),
		})
	}

	llm, err := h.router.Resolve(req.Model)
	if err != nil {
		return c.JSON(http.StatusNotFound, map[string]interface{}{
			"error": "no provider available",
		})
	}

	userID := c.Get(middleware.CtxKeyUserID)
	apiKeyID := c.Get(middleware.CtxKeyAPIKeyID)

	var uid uint
	if v, ok := userID.(uint); ok {
		uid = v
	}
	var kid uint
	if v, ok := apiKeyID.(uint); ok {
		kid = v
	}

	startTime := time.Now()

	// Convert Anthropic request to internal ChatRequest
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
			h.recordAuditAndStats(traceID, uid, kid, llm.ID(), req.Model, false,
				0, 0, 0, http.StatusBadGateway, err.Error(), latencyMs, c, "", "")
			return c.JSON(http.StatusBadGateway, map[string]interface{}{
				"error": err.Error(),
			})
		}

		var content string
		if len(resp.Choices) > 0 {
			content = string(resp.Choices[0].Message.Content)
		}
		anthResp := provider.AnthropicResponse{
			ID:    resp.ID,
			Type:  "message",
			Role:  "assistant",
			Model: resp.Model,
			Content: []provider.AnthropicContentBlock{
				{Type: "text", Text: content},
			},
			Usage: provider.AnthropicUsage{
				InputTokens:  resp.Usage.PromptTokens,
				OutputTokens: resp.Usage.CompletionTokens,
			},
		}

		h.recordAuditAndStats(traceID, uid, kid, llm.ID(), req.Model, false,
			resp.Usage.PromptTokens, resp.Usage.CompletionTokens, resp.Usage.TotalTokens,
			http.StatusOK, "", latencyMs, c, "", content[:min(len(content), 200)])
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

	var promptTokens, completionTokens int
	var contentBuilder strings.Builder
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
					contentBuilder.WriteString(choice.Delta.Content)
					event := provider.AnthropicStreamEvent{
						Type:  "content_block_delta",
						Index: 0,
						Delta: &provider.AnthropicDelta{
							Type: "text_delta",
							Text: choice.Delta.Content,
						},
					}
					data, _ := json.Marshal(event)
					fmt.Fprintf(c.Response().Writer, "event: content_block_delta\ndata: %s\n\n", string(data))
					flusher.Flush()
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
	h.recordAuditAndStats(traceID, uid, kid, llm.ID(), req.Model, true,
		promptTokens, completionTokens, promptTokens+completionTokens,
		statusCode, finalErr, latencyMs, c, "", "")

	return nil
}

func truncateStr(s string, maxLen int) string {
	if len(s) > maxLen {
		return s[:maxLen]
	}
	return s
}

// truncateJSON 截断 JSON 字符串，确保结果是有效的 JSON（截断点可能在对象/数组中）
func truncateJSON(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	truncated := s[:maxLen]
	if json.Valid([]byte(truncated)) {
		return truncated
	}
	for i := len(truncated) - 1; i >= 0; i-- {
		if truncated[i] == '}' || truncated[i] == ']' || truncated[i] == '"' {
			candidate := truncated[:i+1]
			if json.Valid([]byte(candidate)) {
				return candidate
			}
		}
	}
	return truncated
}

func summarizeResponse(resp *provider.ChatResponse) string {
	if resp == nil || len(resp.Choices) == 0 {
		return ""
	}
	return truncateStr(string(resp.Choices[0].Message.Content), 500)
}

// suppress unused import warnings
var _ = io.Copy
var _ = strings.TrimSpace
var _ = fmt.Sprint
