package handlers

import (
	"llm-gateway/internal/web/common"
	"net/http"

	"github.com/labstack/echo/v4"
)

type HealthHandler struct {
	common.BaseHandler
}

func (h *HealthHandler) Liveness(c echo.Context) error {
	return c.JSON(http.StatusOK, map[string]any{"status": "ok"})
}

func (h *HealthHandler) Readiness(c echo.Context) error {
	// 检查 SQLite
	sqlDB, err := h.DB.DB()
	if err != nil {
		return c.JSON(http.StatusServiceUnavailable, map[string]any{"status": "not ready"})
	}
	if err := sqlDB.Ping(); err != nil {
		return c.JSON(http.StatusServiceUnavailable, map[string]any{"status": "not ready"})
	}
	// 检查 DuckDB
	if err := h.Store.Ping(); err != nil {
		return c.JSON(http.StatusServiceUnavailable, map[string]any{"status": "not ready"})
	}
	return c.JSON(http.StatusOK, map[string]any{"status": "ready"})
}

func (h *HealthHandler) RegisterRoutes(e *echo.Group) {
	e.GET("/health", h.Liveness)
	e.GET("/health/ready", h.Readiness)
}
