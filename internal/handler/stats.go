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
	"gorm.io/gorm"
)

type StatsHandler struct {
	db     *sql.DB
	sqlite *gorm.DB
}

func NewStatsHandler(db *sql.DB, sqlite *gorm.DB) *StatsHandler {
	return &StatsHandler{db: db, sqlite: sqlite}
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

	query := `SELECT user_id, model_name,
		SUM(prompt_tokens) as prompt_tokens,
		SUM(completion_tokens) as completion_tokens,
		SUM(total_tokens) as total_tokens
		FROM request_logs
		WHERE created_at BETWEEN ? AND ? AND status_code = 200`

	args := []interface{}{start, end}

	if dept := c.QueryParam("department"); dept != "" {
		var userIDs []uint
		h.sqlite.Model(&model.User{}).Where("department = ?", dept).Pluck("id", &userIDs)
		if len(userIDs) == 0 {
			return response.Success(c, []tokenStatsItem{})
		}
		placeholders := make([]string, len(userIDs))
		for i, uid := range userIDs {
			placeholders[i] = "?"
			args = append(args, uid)
		}
		query += " AND user_id IN (" + strings.Join(placeholders, ",") + ")"
	}

	query += " GROUP BY user_id, model_name ORDER BY total_tokens DESC"

	rows, err := h.db.Query(query, args...)
	if err != nil {
		return response.Error(c, apierror.InternalError(err.Error()))
	}
	defer rows.Close()

	var results []tokenStatsItem
	userIDs := make(map[uint]bool)
	for rows.Next() {
		var item tokenStatsItem
		if err := rows.Scan(&item.UserID, &item.ModelName, &item.PromptTokens, &item.CompletionTokens, &item.TotalTokens); err != nil {
			return response.Error(c, apierror.InternalError(err.Error()))
		}
		results = append(results, item)
		userIDs[item.UserID] = true
	}

	if len(results) == 0 {
		return response.Success(c, []tokenStatsItem{})
	}

	ids := make([]uint, 0, len(userIDs))
	for id := range userIDs {
		ids = append(ids, id)
	}
	var users []model.User
	h.sqlite.Where("id IN ?", ids).Find(&users)
	userMap := make(map[uint]*model.User, len(users))
	for i := range users {
		userMap[users[i].ID] = &users[i]
	}
	for i := range results {
		if u, ok := userMap[results[i].UserID]; ok {
			results[i].Username = u.Username
			results[i].Department = u.Department
		}
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

	query := `SELECT CAST(DATE(created_at) AS VARCHAR) as date,
		COUNT(*) as request_count,
		SUM(CASE WHEN status_code >= 200 AND status_code < 300 THEN 1 ELSE 0 END) as success_count,
		SUM(CASE WHEN status_code >= 400 THEN 1 ELSE 0 END) as error_count,
		CAST(AVG(latency_ms) AS BIGINT) as avg_latency_ms
		FROM request_logs
		WHERE created_at BETWEEN ? AND ?`

	args := []interface{}{start, end}
	query, args = applyRequestFilters(query, args, c)

	query += " GROUP BY DATE(created_at) ORDER BY date ASC"

	rows, err := h.db.Query(query, args...)
	if err != nil {
		return response.Error(c, apierror.InternalError(err.Error()))
	}
	defer rows.Close()

	var results []requestCountItem
	for rows.Next() {
		var item requestCountItem
		if err := rows.Scan(&item.Date, &item.RequestCount, &item.SuccessCount, &item.ErrorCount, &item.AvgLatencyMs); err != nil {
			return response.Error(c, apierror.InternalError(err.Error()))
		}
		results = append(results, item)
	}

	if results == nil {
		results = []requestCountItem{}
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

	query := `SELECT CAST(DATE(created_at) AS VARCHAR) as date,
		model_name, SUM(cost) as total_cost
		FROM request_logs
		WHERE created_at BETWEEN ? AND ? AND status_code = 200
		GROUP BY DATE(created_at), model_name
		ORDER BY date ASC`

	rows, err := h.db.Query(query, start, end)
	if err != nil {
		return response.Error(c, apierror.InternalError(err.Error()))
	}
	defer rows.Close()

	var results []costStatsItem
	for rows.Next() {
		var item costStatsItem
		if err := rows.Scan(&item.Date, &item.ModelName, &item.TotalCost); err != nil {
			return response.Error(c, apierror.InternalError(err.Error()))
		}
		results = append(results, item)
	}

	if results == nil {
		results = []costStatsItem{}
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

	query := `SELECT user_id, model_name, COUNT(*) as count
		FROM request_logs
		WHERE created_at BETWEEN ? AND ?`

	args := []interface{}{start, end}
	query, args = applyRequestFilters(query, args, c)

	query += " GROUP BY user_id, model_name ORDER BY count DESC LIMIT 100"

	rows, err := h.db.Query(query, args...)
	if err != nil {
		return response.Error(c, apierror.InternalError(err.Error()))
	}
	defer rows.Close()

	var results []behaviorItem
	userIDs := make(map[uint]bool)
	for rows.Next() {
		var item behaviorItem
		if err := rows.Scan(&item.UserID, &item.ModelName, &item.Count); err != nil {
			return response.Error(c, apierror.InternalError(err.Error()))
		}
		results = append(results, item)
		userIDs[item.UserID] = true
	}

	if len(results) == 0 {
		return response.Success(c, []behaviorItem{})
	}

	ids := make([]uint, 0, len(userIDs))
	for id := range userIDs {
		ids = append(ids, id)
	}
	var users []model.User
	h.sqlite.Where("id IN ?", ids).Find(&users)
	userMap := make(map[uint]*model.User, len(users))
	for i := range users {
		userMap[users[i].ID] = &users[i]
	}
	for i := range results {
		if u, ok := userMap[results[i].UserID]; ok {
			results[i].Username = u.Username
			results[i].Department = u.Department
		}
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
	err = h.db.QueryRow(`SELECT
		COUNT(*) as total_requests,
		COALESCE(SUM(total_tokens), 0) as total_tokens,
		COALESCE(SUM(cost), 0) as total_cost,
		COALESCE(AVG(latency_ms), 0) as avg_latency,
		SUM(CASE WHEN status_code >= 200 AND status_code < 300 THEN 1 ELSE 0 END) as success_count,
		COUNT(DISTINCT user_id) as active_users
		FROM request_logs
		WHERE created_at BETWEEN ? AND ?`, start, end).Scan(
		&agg.TotalRequests, &agg.TotalTokens, &agg.TotalCost,
		&agg.AvgLatency, &agg.SuccessCount, &agg.ActiveUsers,
	)
	if err != nil {
		return response.Error(c, apierror.InternalError(err.Error()))
	}

	overview.TotalRequests = agg.TotalRequests
	overview.TotalTokens = agg.TotalTokens
	overview.TotalCost = agg.TotalCost
	overview.AvgLatencyMs = int64(agg.AvgLatency)
	overview.ActiveUsers = agg.ActiveUsers

	if agg.TotalRequests > 0 {
		overview.SuccessRate = float64(agg.SuccessCount) / float64(agg.TotalRequests) * 100
	}

	topRows, err := h.db.Query(`SELECT model_name, COUNT(*) as count
		FROM request_logs
		WHERE created_at BETWEEN ? AND ?
		GROUP BY model_name
		ORDER BY count DESC
		LIMIT 10`, start, end)
	if err != nil {
		return response.Error(c, apierror.InternalError(err.Error()))
	}
	defer topRows.Close()

	for topRows.Next() {
		var item topModelItem
		if err := topRows.Scan(&item.ModelName, &item.Count); err != nil {
			return response.Error(c, apierror.InternalError(err.Error()))
		}
		overview.TopModels = append(overview.TopModels, item)
	}

	if overview.TopModels == nil {
		overview.TopModels = []topModelItem{}
	}

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

func applyRequestFilters(query string, args []interface{}, c echo.Context) (string, []interface{}) {
	if uid := c.QueryParam("user_id"); uid != "" {
		if _, err := strconv.ParseUint(uid, 10, 64); err == nil {
			query += " AND user_id = ?"
			args = append(args, uid)
		}
	}
	if modelName := c.QueryParam("model"); modelName != "" {
		query += " AND model_name = ?"
		args = append(args, modelName)
	}
	return query, args
}
