package handlers

import (
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
	g.POST("/request-logs/search", h.SearchRequestLogs)
	g.POST("/request-logs/fetch", h.FetchRequestLog)
	g.POST("/request-logs/detail", h.FetchRequestDetail)
	g.POST("/request-logs/chunks", h.FetchRequestChunks)
}

func (h *RequestLogHandler) SearchRequestLogs(c echo.Context) error {
	input := &common.SearchParams{}
	if err := c.Bind(input); err != nil {
		return err
	}

	if input.Pagination.Size <= 0 || input.Pagination.Size > 100 {
		input.Pagination.Size = 20
	}
	if input.Pagination.Index < 1 {
		input.Pagination.Index = 1
	}
	input.Pagination.Offset = (input.Pagination.Index - 1) * input.Pagination.Size

	where := []string{}
	args := []interface{}{}

	// 关键词搜索
	if input.Kw != "" {
		where = append(where, "(trace_id LIKE ? OR model_name LIKE ? OR ip_address LIKE ?)")
		kw := "%" + input.Kw + "%"
		args = append(args, kw, kw, kw)
	}

	// 过滤条件
	for _, filter := range input.Filters {
		switch filter.Field {
		case "user_id":
			where = append(where, "user_id = ?")
			args = append(args, filter.Value)
		case "model_name":
			where = append(where, "model_name = ?")
			args = append(args, filter.Value)
		case "status_code":
			where = append(where, "status_code = ?")
			args = append(args, filter.Value)
		case "is_stream":
			where = append(where, "is_stream = ?")
			args = append(args, filter.Value)
		case "start":
			if t, err := parseTime(filter.Value); err == nil {
				where = append(where, "created_at >= ?")
				args = append(args, t)
			}
		case "end":
			if t, err := parseTime(filter.Value); err == nil {
				where = append(where, "created_at <= ?")
				args = append(args, t)
			}
		}
	}

	whereClause := ""
	if len(where) > 0 {
		whereClause = "WHERE " + strings.Join(where, " AND ")
	}

	var total int64
	countSQL := "SELECT COUNT(*) FROM request_logs " + whereClause
	if err := h.Store.DB().Get(&total, countSQL, args...); err != nil {
		return h.Error(-20, err.Error())
	}

	querySQL := "SELECT * FROM request_logs " + whereClause + " ORDER BY created_at DESC LIMIT ? OFFSET ?"
	queryArgs := append(args, input.Pagination.Size, input.Pagination.Offset)

	var logs []model.RequestLog
	if err := h.Store.DB().Select(&logs, querySQL, queryArgs...); err != nil {
		return h.Error(-20, err.Error())
	}

	if logs == nil {
		logs = []model.RequestLog{}
	}

	return common.NewDataSet(logs, total)
}

func (h *RequestLogHandler) FetchRequestLog(c echo.Context) error {
	input := &struct {
		TraceID string `json:"traceId"`
	}{}
	if err := c.Bind(input); err != nil {
		return err
	}
	if input.TraceID == "" {
		return h.Error(-11, "traceId is required")
	}

	var log model.RequestLog
	if err := h.Store.DB().Get(&log, "SELECT * FROM request_logs WHERE trace_id = ?", input.TraceID); err != nil {
		return h.Error(-24, "request log not found")
	}

	return common.NewData(log)
}

func (h *RequestLogHandler) FetchRequestDetail(c echo.Context) error {
	input := &struct {
		TraceID string `json:"traceId"`
	}{}
	if err := c.Bind(input); err != nil {
		return err
	}
	if input.TraceID == "" {
		return h.Error(-11, "traceId is required")
	}

	var detail model.RequestDetail
	if err := h.Store.DB().Get(&detail, "SELECT * FROM request_details WHERE trace_id = ?", input.TraceID); err != nil {
		return h.Error(-24, "request detail not found")
	}

	return common.NewData(detail)
}

func (h *RequestLogHandler) FetchRequestChunks(c echo.Context) error {
	input := &struct {
		TraceID string `json:"traceId"`
	}{}
	if err := c.Bind(input); err != nil {
		return err
	}
	if input.TraceID == "" {
		return h.Error(-11, "traceId is required")
	}

	var chunks []model.RequestChunk
	if err := h.Store.DB().Select(&chunks, "SELECT * FROM request_chunks WHERE trace_id = ? ORDER BY index ASC", input.TraceID); err != nil {
		return h.Error(-20, err.Error())
	}

	if chunks == nil {
		chunks = []model.RequestChunk{}
	}

	return common.NewData(chunks)
}

func parseTime(v interface{}) (time.Time, error) {
	var s string
	switch val := v.(type) {
	case string:
		s = val
	default:
		s = ""
	}

	if s == "" {
		return time.Time{}, nil
	}

	t, err := time.Parse(time.RFC3339, s)
	if err != nil {
		t, err = time.Parse("2006-01-02", s)
	}
	return t, err
}
