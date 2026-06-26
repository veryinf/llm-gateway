package handlers

import (
	"net/http"

	"llm-gateway/internal/model"
	"llm-gateway/internal/provider"
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
		return h.HandleStream(c, llmReq, router)
	}
	return h.HandleNonStream(c, llmReq, router)
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
