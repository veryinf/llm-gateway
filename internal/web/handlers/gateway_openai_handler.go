package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"

	"llm-gateway/internal/model"
	"llm-gateway/internal/provider"

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
	var req provider.ChatRequest
	gwCtx, err := h.prepareRequest(c, &req)
	if err != nil {
		return h.errorJSON(c, http.StatusBadRequest, err.Error())
	}

	llm, err := h.resolveProvider(req.Model)
	if err != nil {
		return h.errorJSON(c, http.StatusNotFound, "no provider available")
	}

	if req.Stream {
		return h.handleStream(c, llm, &req, gwCtx)
	}
	return h.handleNonStream(c, llm, &req, gwCtx)
}

func (h *OpenAIGatewayHandler) handleNonStream(c echo.Context, llm provider.LLMProvider,
	req *provider.ChatRequest, gwCtx *gatewayContext) error {

	resp, err := llm.ChatCompletion(c.Request().Context(), req)
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
	return c.JSON(http.StatusOK, resp)
}

func (h *OpenAIGatewayHandler) handleStream(c echo.Context, llm provider.LLMProvider,
	req *provider.ChatRequest, gwCtx *gatewayContext) error {

	flusher, err := h.prepareStream(c)
	if err != nil {
		return h.errorJSON(c, http.StatusInternalServerError, err.Error())
	}

	chunkCh, errCh := llm.ChatCompletionStream(c.Request().Context(), req)
	collector := h.newChunkCollector(gwCtx.TraceID)

	var totalTokens, promptTokens, completionTokens int
	var finalErr string
	statusCode := http.StatusOK

	for chunk := range chunkCh {
		data, _ := json.Marshal(chunk)
		fmt.Fprintf(c.Response().Writer, "data: %s\n\n", data)
		flusher.Flush()
		collector.Add(data)

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
			fmt.Fprintf(c.Response().Writer, "data: %s\n\n", errData)
			flusher.Flush()
		}
	default:
	}

	fmt.Fprint(c.Response().Writer, "data: [DONE]\n\n")
	flusher.Flush()

	h.recordRequest(gwCtx.TraceID, gwCtx.UserID, gwCtx.APIKeyID, llm.ID(), req.Model, true,
		promptTokens, completionTokens, totalTokens,
		statusCode, finalErr, gwCtx.Latency(), c, gwCtx.ReqBytes, nil, collector.Chunks())
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
