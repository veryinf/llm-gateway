package handler

import (
	"database/sql"
	"strconv"
	"strings"
	"time"

	"llm-gateway/internal/model"
	"llm-gateway/pkg/apierror"
	"llm-gateway/pkg/response"

	"github.com/labstack/echo/v4"
)

type AuditHandler struct {
	db *sql.DB
}

func NewAuditHandler(db *sql.DB) *AuditHandler {
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

	where := []string{}
	args := []interface{}{}

	if userIDStr := c.QueryParam("user_id"); userIDStr != "" {
		if uid, err := strconv.ParseUint(userIDStr, 10, 64); err == nil {
			where = append(where, "user_id = ?")
			args = append(args, uid)
		}
	}
	if modelName := c.QueryParam("model"); modelName != "" {
		where = append(where, "model_name = ?")
		args = append(args, modelName)
	}
	if startStr := c.QueryParam("start"); startStr != "" {
		t, err := time.Parse(time.RFC3339, startStr)
		if err != nil {
			t, _ = time.Parse("2006-01-02", startStr)
		}
		if !t.IsZero() {
			where = append(where, "created_at >= ?")
			args = append(args, t)
		}
	}
	if endStr := c.QueryParam("end"); endStr != "" {
		t, err := time.Parse(time.RFC3339, endStr)
		if err != nil {
			t, _ = time.Parse("2006-01-02", endStr)
			t = t.Add(24 * time.Hour)
		}
		if !t.IsZero() {
			where = append(where, "created_at <= ?")
			args = append(args, t)
		}
	}
	if statusStr := c.QueryParam("status"); statusStr != "" {
		var statusCode int
		switch statusStr {
		case "success":
			statusCode = 200
		case "error":
			statusCode = 500
		}
		if statusCode > 0 {
			where = append(where, "status_code = ?")
			args = append(args, statusCode)
		}
	}

	whereClause := ""
	if len(where) > 0 {
		whereClause = "WHERE " + strings.Join(where, " AND ")
	}

	var total int64
	countSQL := "SELECT COUNT(*) FROM audit_logs " + whereClause
	if err := h.db.QueryRow(countSQL, args...).Scan(&total); err != nil {
		return response.Error(c, apierror.InternalError(err.Error()))
	}

	offset := (page - 1) * pageSize
	querySQL := `SELECT
		ROW_NUMBER() OVER (ORDER BY created_at DESC) as id,
		trace_id, user_id, api_key_id, provider_id, model_name,
		COALESCE(request_summary, '') as request_summary,
		COALESCE(response_summary, '') as response_summary,
		prompt_tokens, completion_tokens, status_code,
		COALESCE(error_message, '') as error_message,
		latency_ms, cost,
		COALESCE(ip_address, '') as ip_address,
		COALESCE(user_agent, '') as user_agent,
		created_at
		FROM audit_logs ` + whereClause + ` ORDER BY created_at DESC LIMIT ? OFFSET ?`

	queryArgs := append(args, pageSize, offset)
	rows, err := h.db.Query(querySQL, queryArgs...)
	if err != nil {
		return response.Error(c, apierror.InternalError(err.Error()))
	}
	defer rows.Close()

	var logs []model.AuditLog
	for rows.Next() {
		var log model.AuditLog
		if err := rows.Scan(
			&log.ID, &log.TraceID, &log.UserID, &log.APIKeyID, &log.ProviderID,
			&log.ModelName, &log.RequestSummary, &log.ResponseSummary,
			&log.PromptTokens, &log.CompletionTokens, &log.StatusCode,
			&log.ErrorMessage, &log.LatencyMs, &log.Cost,
			&log.IPAddress, &log.UserAgent, &log.CreatedAt,
		); err != nil {
			return response.Error(c, apierror.InternalError(err.Error()))
		}
		logs = append(logs, log)
	}

	if logs == nil {
		logs = []model.AuditLog{}
	}

	return response.SuccessPage(c, logs, total, page, pageSize)
}

// GetAuditLogByTrace handles GET /api/audit/logs/:trace_id.
func (h *AuditHandler) GetAuditLogByTrace(c echo.Context) error {
	traceID := c.Param("trace_id")
	if traceID == "" {
		return response.Error(c, apierror.BadRequest("trace_id is required"))
	}

	querySQL := `SELECT
		ROW_NUMBER() OVER (ORDER BY created_at ASC) as id,
		trace_id, user_id, api_key_id, provider_id, model_name,
		COALESCE(request_summary, '') as request_summary,
		COALESCE(response_summary, '') as response_summary,
		prompt_tokens, completion_tokens, status_code,
		COALESCE(error_message, '') as error_message,
		latency_ms, cost,
		COALESCE(ip_address, '') as ip_address,
		COALESCE(user_agent, '') as user_agent,
		created_at
		FROM audit_logs WHERE trace_id = ? ORDER BY created_at ASC`

	rows, err := h.db.Query(querySQL, traceID)
	if err != nil {
		return response.Error(c, apierror.InternalError(err.Error()))
	}
	defer rows.Close()

	var logs []model.AuditLog
	for rows.Next() {
		var log model.AuditLog
		if err := rows.Scan(
			&log.ID, &log.TraceID, &log.UserID, &log.APIKeyID, &log.ProviderID,
			&log.ModelName, &log.RequestSummary, &log.ResponseSummary,
			&log.PromptTokens, &log.CompletionTokens, &log.StatusCode,
			&log.ErrorMessage, &log.LatencyMs, &log.Cost,
			&log.IPAddress, &log.UserAgent, &log.CreatedAt,
		); err != nil {
			return response.Error(c, apierror.InternalError(err.Error()))
		}
		logs = append(logs, log)
	}

	if logs == nil {
		logs = []model.AuditLog{}
	}

	return response.Success(c, logs)
}

func defaultQueryParam(c echo.Context, key, defaultValue string) string {
	if v := c.QueryParam(key); v != "" {
		return v
	}
	return defaultValue
}
