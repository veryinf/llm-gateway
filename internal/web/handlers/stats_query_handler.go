package handlers

import (
	"llm-gateway/internal/service"
	"llm-gateway/internal/web/common"

	"github.com/labstack/echo/v4"
)

type StatsQueryHandler struct {
	common.BaseHandler
	Service *service.StatsQueryService
}

func (h *StatsQueryHandler) RegisterRoutes(g *echo.Group) {
	g.POST("/stats/query", h.Query)
}

func (h *StatsQueryHandler) Query(c echo.Context) error {
	var req service.QueryRequest
	if err := c.Bind(&req); err != nil {
		return h.Error(-10, "invalid request body")
	}

	resp, err := h.Service.Query(req)
	if err != nil {
		return h.Error(-20, err.Error())
	}

	return common.NewDataSet(resp.Rows, resp.Total)
}