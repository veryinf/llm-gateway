package handler

import (
	"strconv"
	"time"

	"llm-gateway/internal/model"
	"llm-gateway/pkg/apierror"
	"llm-gateway/pkg/response"

	"github.com/labstack/echo/v4"
	"gorm.io/gorm"
)

type AuditHandler struct {
	db *gorm.DB
}

func NewAuditHandler(db *gorm.DB) *AuditHandler {
	return &AuditHandler{db: db}
}

// ListAuditLogs handles GET /api/audit/logs with pagination and optional filters.
func (h *AuditHandler) ListAuditLogs(c echo.Context) error {
	page, _ := strconv.Atoi(defaultQueryParam(c, "page", "1"))
	pageSize, _ := strconv.Atoi(defaultQueryParam(c, "pageSize", "20"))

	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}

	userIDStr := c.QueryParam("user_id")
	modelName := c.QueryParam("model")
	statusStr := c.QueryParam("status")
	startStr := c.QueryParam("start")
	endStr := c.QueryParam("end")

	tx := h.db.Model(&model.AuditLog{})

	if userIDStr != "" {
		if uid, err := strconv.ParseUint(userIDStr, 10, 64); err == nil {
			tx = tx.Where("user_id = ?", uid)
		}
	}
	if modelName != "" {
		tx = tx.Where("model_name = ?", modelName)
	}
	if startStr != "" {
		t, err := time.Parse(time.RFC3339, startStr)
		if err != nil {
			t, _ = time.Parse("2006-01-02", startStr)
		}
		if !t.IsZero() {
			tx = tx.Where("created_at >= ?", t)
		}
	}
	if endStr != "" {
		t, err := time.Parse(time.RFC3339, endStr)
		if err != nil {
			t, _ = time.Parse("2006-01-02", endStr)
			t = t.Add(24 * time.Hour)
		}
		if !t.IsZero() {
			tx = tx.Where("created_at <= ?", t)
		}
	}
	if statusStr != "" {
		var statusCode int
		switch statusStr {
		case "success":
			statusCode = 200
		case "error":
			statusCode = 500
		}
		if statusCode > 0 {
			tx = tx.Where("status_code = ?", statusCode)
		}
	}

	var total int64
	if err := tx.Count(&total).Error; err != nil {
		return response.Error(c, apierror.InternalError(err.Error()))
	}

	var logs []model.AuditLog
	if err := tx.Offset((page - 1) * pageSize).Limit(pageSize).Order("created_at desc").Find(&logs).Error; err != nil {
		return response.Error(c, apierror.InternalError(err.Error()))
	}

	return response.SuccessPage(c, logs, total, page, pageSize)
}

// GetAuditLogByTrace handles GET /api/audit/logs/:trace_id.
func (h *AuditHandler) GetAuditLogByTrace(c echo.Context) error {
	traceID := c.Param("trace_id")
	if traceID == "" {
		return response.Error(c, apierror.BadRequest("trace_id is required"))
	}

	var logs []model.AuditLog
	if err := h.db.Where("trace_id = ?", traceID).Order("created_at ASC").Find(&logs).Error; err != nil {
		return response.Error(c, apierror.InternalError(err.Error()))
	}

	return response.Success(c, logs)
}

func defaultQueryParam(c echo.Context, key, defaultValue string) string {
	if v := c.QueryParam(key); v != "" {
		return v
	}
	return defaultValue
}
