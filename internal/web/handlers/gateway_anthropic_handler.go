package handlers

import (
	"net/http"

	"llm-gateway/internal/model"
	"llm-gateway/internal/provider"
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
	llmReq, err = llmReq.ToAnthropic()
	if err != nil {
		return h.ErrorJSON(c, http.StatusBadRequest, err.Error())
	}
	router, err := h.RouterService.ResolveProvider(llmReq.Model)
	if err != nil {
		return h.ErrorJSON(c, http.StatusNotFound, err.Error())
	}

	if llmReq.Stream {
		return h.HandleStream(c, llmReq, router)
	}
	return h.HandleNonStream(c, llmReq, router)
}
