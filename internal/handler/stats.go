package handler

import (
	"time"

	"llm-gateway/internal/model"
	"llm-gateway/pkg/apierror"
	"llm-gateway/pkg/response"

	"github.com/labstack/echo/v4"
	"gorm.io/gorm"
)

type StatsHandler struct {
	db *gorm.DB
}

func NewStatsHandler(db *gorm.DB) *StatsHandler {
	return &StatsHandler{db: db}
}

// ======================== Token Stats ========================

type tokenStatsItem struct {
	UserID           uint    `json:"user_id"`
	Username         string  `json:"username"`
	Department       string  `json:"department"`
	ModelName        string  `json:"model_name"`
	PromptTokens     int64   `json:"prompt_tokens"`
	CompletionTokens int64   `json:"completion_tokens"`
	TotalTokens      int64   `json:"total_tokens"`
}

func (h *StatsHandler) TokenStats(c echo.Context) error {
	start, end, err := parseTimeRange(c)
	if err != nil {
		return response.Error(c, apierror.BadRequest(err.Error()))
	}

	var results []tokenStatsItem
	query := h.db.Model(&model.RequestLog{}).
		Select("request_logs.user_id, request_logs.model_name, SUM(request_logs.prompt_tokens) as prompt_tokens, SUM(request_logs.completion_tokens) as completion_tokens, SUM(request_logs.total_tokens) as total_tokens").
		Joins("LEFT JOIN users ON users.id = request_logs.user_id").
		Where("request_logs.created_at BETWEEN ? AND ?", start, end).
		Where("request_logs.status_code = 200").
		Group("request_logs.user_id, request_logs.model_name").
		Order("total_tokens DESC")

	if dept := c.QueryParam("department"); dept != "" {
		query = query.Where("users.department = ?", dept)
	}

	if err := query.Find(&results).Error; err != nil {
		return response.Error(c, apierror.InternalError(err.Error()))
	}

	return response.Success(c, results)
}

// ======================== Request Stats ========================

type requestCountItem struct {
	Date         string `json:"date"`
	RequestCount int64  `json:"request_count"`
	SuccessCount int64  `json:"success_count"`
	ErrorCount   int64  `json:"error_count"`
	AvgLatencyMs int64  `json:"avg_latency_ms"`
}

func (h *StatsHandler) RequestStats(c echo.Context) error {
	start, end, err := parseTimeRange(c)
	if err != nil {
		return response.Error(c, apierror.BadRequest(err.Error()))
	}

	var results []requestCountItem
	query := h.db.Model(&model.RequestLog{}).
		Select("DATE(created_at) as date, COUNT(*) as request_count, "+
			"SUM(CASE WHEN status_code >= 200 AND status_code < 300 THEN 1 ELSE 0 END) as success_count, "+
			"SUM(CASE WHEN status_code >= 400 THEN 1 ELSE 0 END) as error_count, "+
			"AVG(latency_ms) as avg_latency_ms").
		Where("created_at BETWEEN ? AND ?", start, end)

	query = applyRequestFilters(query, c)
	query = query.Group("DATE(created_at)").Order("date ASC")

	if err := query.Find(&results).Error; err != nil {
		return response.Error(c, apierror.InternalError(err.Error()))
	}

	return response.Success(c, results)
}

// ======================== Cost Stats ========================

type costStatsItem struct {
	Date      string  `json:"date"`
	ModelName string  `json:"model_name"`
	TotalCost float64 `json:"total_cost"`
}

func (h *StatsHandler) CostStats(c echo.Context) error {
	start, end, err := parseTimeRange(c)
	if err != nil {
		return response.Error(c, apierror.BadRequest(err.Error()))
	}

	var results []costStatsItem
	query := h.db.Model(&model.RequestLog{}).
		Select("DATE(created_at) as date, model_name, SUM(cost) as total_cost").
		Where("created_at BETWEEN ? AND ?", start, end).
		Where("status_code = 200").
		Group("DATE(created_at), model_name").
		Order("date ASC")

	if err := query.Find(&results).Error; err != nil {
		return response.Error(c, apierror.InternalError(err.Error()))
	}

	return response.Success(c, results)
}

// ======================== Behavior Stats ========================

type behaviorItem struct {
	UserID     uint   `json:"user_id"`
	Username   string `json:"username"`
	Department string `json:"department"`
	ModelName  string `json:"model_name"`
	Count      int64  `json:"count"`
}

func (h *StatsHandler) BehaviorStats(c echo.Context) error {
	start, end, err := parseTimeRange(c)
	if err != nil {
		return response.Error(c, apierror.BadRequest(err.Error()))
	}

	var results []behaviorItem
	query := h.db.Model(&model.RequestLog{}).
		Joins("LEFT JOIN users ON users.id = request_logs.user_id").
		Select("user_id, users.username, users.department, model_name, COUNT(*) as count").
		Where("request_logs.created_at BETWEEN ? AND ?", start, end)

	query = applyRequestFilters(query, c)
	query = query.Group("user_id, model_name").
		Order("count DESC").
		Limit(100)

	if err := query.Find(&results).Error; err != nil {
		return response.Error(c, apierror.InternalError(err.Error()))
	}

	return response.Success(c, results)
}

// ======================== Dashboard Overview ========================

type dashboardOverview struct {
	TotalRequests int64          `json:"total_requests"`
	TotalTokens   int64          `json:"total_tokens"`
	TotalCost     float64        `json:"total_cost"`
	AvgLatencyMs  int64          `json:"avg_latency_ms"`
	SuccessRate   float64        `json:"success_rate"`
	ActiveUsers   int64          `json:"active_users"`
	TopModels     []topModelItem `json:"top_models"`
}

type topModelItem struct {
	ModelName string `json:"model_name"`
	Count     int64  `json:"count"`
}

func (h *StatsHandler) DashboardOverview(c echo.Context) error {
	start, end, err := parseTimeRange(c)
	if err != nil {
		return response.Error(c, apierror.BadRequest(err.Error()))
	}

	overview := dashboardOverview{}

	type aggResult struct {
		TotalRequests int64
		TotalTokens   int64
		TotalCost     float64
		AvgLatency    float64
		SuccessCount  int64
		ActiveUsers   int64
	}

	var agg aggResult
	h.db.Model(&model.RequestLog{}).
		Select("COUNT(*) as total_requests, COALESCE(SUM(total_tokens), 0) as total_tokens, "+
			"COALESCE(SUM(cost), 0) as total_cost, COALESCE(AVG(latency_ms), 0) as avg_latency, "+
			"SUM(CASE WHEN status_code >= 200 AND status_code < 300 THEN 1 ELSE 0 END) as success_count, "+
			"COUNT(DISTINCT user_id) as active_users").
		Where("created_at BETWEEN ? AND ?", start, end).
		Scan(&agg)

	overview.TotalRequests = agg.TotalRequests
	overview.TotalTokens = agg.TotalTokens
	overview.TotalCost = agg.TotalCost
	overview.AvgLatencyMs = int64(agg.AvgLatency)
	overview.ActiveUsers = agg.ActiveUsers

	if agg.TotalRequests > 0 {
		overview.SuccessRate = float64(agg.SuccessCount) / float64(agg.TotalRequests) * 100
	}

	h.db.Model(&model.RequestLog{}).
		Select("model_name, COUNT(*) as count").
		Where("created_at BETWEEN ? AND ?", start, end).
		Group("model_name").
		Order("count DESC").
		Limit(10).
		Find(&overview.TopModels)

	return response.Success(c, overview)
}

// ======================== Helpers ========================

func parseTimeRange(c echo.Context) (time.Time, time.Time, error) {
	startStr := c.QueryParam("start")
	endStr := c.QueryParam("end")

	now := time.Now()

	var start, end time.Time
	var err error

	if startStr == "" {
		start = now.AddDate(0, 0, -7)
	} else {
		start, err = time.Parse("2006-01-02", startStr)
		if err != nil {
			return start, end, err
		}
	}

	if endStr == "" {
		end = now
	} else {
		end, err = time.Parse("2006-01-02", endStr)
		if err != nil {
			return start, end, err
		}
		end = end.Add(24 * time.Hour)
	}

	return start, end, nil
}

func applyRequestFilters(query *gorm.DB, c echo.Context) *gorm.DB {
	if uid := c.QueryParam("user_id"); uid != "" {
		query = query.Where("user_id = ?", uid)
	}
	if model := c.QueryParam("model"); model != "" {
		query = query.Where("model_name = ?", model)
	}
	return query
}
