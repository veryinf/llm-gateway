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

type OpenAIGatewayHandler struct {
	GatewayBase
}

func (h *OpenAIGatewayHandler) RegisterRoutes(g *echo.Group) {
	g.POST("/chat/completions", h.HandleChatCompletion)
	g.GET("/models", h.HandleListModels)
}

func (h *OpenAIGatewayHandler) HandleChatCompletion(c echo.Context) error {
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

	if !req.Stream {
		return h.handleNonStream(c, llm, &req, traceID, uid, kid, startTime, reqBytes)
	}
	return h.handleStream(c, llm, &req, traceID, uid, kid, startTime, reqBytes)
}

func (h *OpenAIGatewayHandler) handleNonStream(c echo.Context, llm provider.LLMProvider, req *provider.ChatRequest,
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

func (h *OpenAIGatewayHandler) handleStream(c echo.Context, llm provider.LLMProvider, req *provider.ChatRequest,
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

func (h *OpenAIGatewayHandler) HandleListModels(c echo.Context) error {
	models := h.getAllModels()
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
