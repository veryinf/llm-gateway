package handlers

import (
	"strconv"
	"strings"
	"time"

	"llm-gateway/internal/model"
	"llm-gateway/internal/web/common"

	"github.com/labstack/echo/v4"
)

type RequestLogHandler struct {
	common.BaseHandler
}

func (h *RequestLogHandler) RegisterRoutes(g *echo.Group) {
	g.GET("/request-logs", h.ListRequestLogs)
	g.GET("/request-logs/:trace_id", h.GetRequestLogByTrace)
	g.GET("/request-logs/:trace_id/chunks", h.GetRequestChunks)
}

func (h *RequestLogHandler) ListRequestLogs(c echo.Context) error {
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
	countSQL := "SELECT COUNT(*) FROM request_logs " + whereClause
	if err := h.Store.QueryRow(countSQL, args...).Scan(&total); err != nil {
		return h.Error(-20, err.Error())
	}

	offset := (page - 1) * pageSize
	querySQL := `SELECT
		trace_id, user_id, api_key_id, model_name,
		is_stream, prompt_tokens, completion_tokens, total_tokens,
		COALESCE(request_body, '') as request_body,
		COALESCE(response_body, '') as response_body,
		is_detail,
		status_code, COALESCE(error_message, '') as error_message,
		latency_ms, cost,
		COALESCE(ip_address, '') as ip_address,
		COALESCE(user_agent, '') as user_agent,
		created_at
		FROM request_logs ` + whereClause + ` ORDER BY created_at DESC LIMIT ? OFFSET ?`

	queryArgs := append(args, pageSize, offset)
	rows, err := h.Store.Query(querySQL, queryArgs...)
	if err != nil {
		return h.Error(-20, err.Error())
	}
	defer rows.Close()

	var logs []model.RequestLog
	for rows.Next() {
		var log model.RequestLog
		if err := rows.Scan(
			&log.TraceID, &log.UserID, &log.APIKeyID, &log.ModelName,
			&log.IsStream, &log.PromptTokens, &log.CompletionTokens, &log.TotalTokens,
			&log.RequestBody, &log.ResponseBody, &log.IsDetail,
			&log.StatusCode, &log.ErrorMessage, &log.LatencyMs, &log.Cost,
			&log.IPAddress, &log.UserAgent, &log.CreatedAt,
		); err != nil {
			return h.Error(-20, err.Error())
		}
		logs = append(logs, log)
	}

	if logs == nil {
		logs = []model.RequestLog{}
	}

	return c.JSON(200, common.NewDataSet(logs, total))
}

func (h *RequestLogHandler) GetRequestLogByTrace(c echo.Context) error {
	traceID := c.Param("trace_id")
	if traceID == "" {
		return h.Error(-11, "trace_id is required")
	}

	querySQL := `SELECT
		trace_id, user_id, api_key_id, model_name,
		is_stream, prompt_tokens, completion_tokens, total_tokens,
		COALESCE(request_body, '') as request_body,
		COALESCE(response_body, '') as response_body,
		is_detail,
		status_code, COALESCE(error_message, '') as error_message,
		latency_ms, cost,
		COALESCE(ip_address, '') as ip_address,
		COALESCE(user_agent, '') as user_agent,
		created_at
		FROM request_logs WHERE trace_id = ? ORDER BY created_at ASC`

	rows, err := h.Store.Query(querySQL, traceID)
	if err != nil {
		return h.Error(-20, err.Error())
	}
	defer rows.Close()

	var logs []model.RequestLog
	for rows.Next() {
		var log model.RequestLog
		if err := rows.Scan(
			&log.TraceID, &log.UserID, &log.APIKeyID, &log.ModelName,
			&log.IsStream, &log.PromptTokens, &log.CompletionTokens, &log.TotalTokens,
			&log.RequestBody, &log.ResponseBody, &log.IsDetail,
			&log.StatusCode, &log.ErrorMessage, &log.LatencyMs, &log.Cost,
			&log.IPAddress, &log.UserAgent, &log.CreatedAt,
		); err != nil {
			return h.Error(-20, err.Error())
		}
		logs = append(logs, log)
	}

	if logs == nil {
		logs = []model.RequestLog{}
	}

	return c.JSON(200, common.NewData(logs))
}

func (h *RequestLogHandler) GetRequestChunks(c echo.Context) error {
	traceID := c.Param("trace_id")
	if traceID == "" {
		return h.Error(-11, "trace_id is required")
	}

	querySQL := `SELECT COALESCE(id, 0), trace_id, chunk_index, COALESCE(chunk_data, ''), created_at
		FROM request_chunks WHERE trace_id = ? ORDER BY chunk_index ASC`

	rows, err := h.Store.Query(querySQL, traceID)
	if err != nil {
		return h.Error(-20, err.Error())
	}
	defer rows.Close()

	var chunks []model.RequestChunk
	for rows.Next() {
		var chunk model.RequestChunk
		if err := rows.Scan(&chunk.ID, &chunk.TraceID, &chunk.ChunkIndex, &chunk.ChunkData, &chunk.CreatedAt); err != nil {
			return h.Error(-20, err.Error())
		}
		chunks = append(chunks, chunk)
	}

	if chunks == nil {
		chunks = []model.RequestChunk{}
	}

	return c.JSON(200, common.NewData(chunks))
}

func defaultQueryParam(c echo.Context, key, defaultValue string) string {
	if v := c.QueryParam(key); v != "" {
		return v
	}
	return defaultValue
}
