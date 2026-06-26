package handlers

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"llm-gateway/internal/model"
	"llm-gateway/internal/provider"
	"llm-gateway/internal/service"
	"llm-gateway/internal/web/common"

	"github.com/labstack/echo/v4"
)

type OpenAIGatewayHandler struct {
	common.GatewayBase
}

func (h *OpenAIGatewayHandler) RegisterRoutes(g *echo.Group) {
	g.POST("/chat/completions", h.HandleChatCompletion)
	g.GET("/models", h.HandleListModels)
}

func (h *OpenAIGatewayHandler) HandleChatCompletion(c echo.Context) error {
	llmReq, err := provider.NewLLMRequest(c.Request(), model.APITypeOpenAI)
	if err != nil {
		return h.ErrorJSON(c, http.StatusBadRequest, err.Error())
	}
	llmReq, err = llmReq.ToOpenAI()
	if err != nil {
		return h.ErrorJSON(c, http.StatusBadRequest, err.Error())
	}

	router, err := h.RouterService.ResolveProvider(llmReq.Model)
	if err != nil {
		return h.ErrorJSON(c, http.StatusNotFound, "no provider available")
	}

	if llmReq.Stream {
		return h.handleStream(c, llmReq, router)
	}
	return h.handleNonStream(c, llmReq, router)
}

func (h *OpenAIGatewayHandler) handleNonStream(c echo.Context, llmReq *provider.LLMRequest, router *service.RouterResult) error {
	ctx := h.Context(c)

	resp, err := router.Adapter.AutoChat(c.Request().Context(), llmReq)
	log := h.BuildLog(ctx, router, llmReq, resp)
	if err != nil {
		log.ErrorMessage = err.Error()
		h.RequestLogService.RecordRequest(log)
		return h.ErrorJSON(c, http.StatusBadGateway, err.Error())
	}

	c.Response().Header().Set("Content-Type", "application/json")
	c.Response().WriteHeader(http.StatusOK)

	_, copyErr := io.Copy(c.Response().Writer, resp.Response.Body)
	_ = resp.Response.Body.Close()

	if copyErr != nil {
		log.ErrorMessage = copyErr.Error()
		h.RequestLogService.RecordRequest(log)
		return h.ErrorJSON(c, http.StatusBadGateway, copyErr.Error())
	}
	h.RequestLogService.RecordRequest(log)
	return nil
}

func (h *OpenAIGatewayHandler) handleStream(c echo.Context, llmReq *provider.LLMRequest, router *service.RouterResult) error {
	flusher, err := h.PrepareStream(c)
	if err != nil {
		return h.ErrorJSON(c, http.StatusInternalServerError, err.Error())
	}

	ctx := h.Context(c)
	collector := h.NewChunkCollector(ctx.TraceID)

	chunkCh, errCh := router.Adapter.ChatCompletionStream(c.Request().Context(), llmReq)

	for data := range chunkCh {
		fmt.Fprintf(c.Response().Writer, "data: %s\n\n", data)
		flusher.Flush()
		collector.Add(data)
	}

	if err := <-errCh; err != nil {
		errData, _ := json.Marshal(map[string]string{"error": err.Error()})
		fmt.Fprintf(c.Response().Writer, "data: %s\n\n", errData)
		flusher.Flush()
	}

	fmt.Fprint(c.Response().Writer, "data: [DONE]\n\n")
	flusher.Flush()

	h.RecordRequest(ctx, router, llmReq, nil)
	h.RecordChunks(collector.Chunks())
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
