package service

import (
	"fmt"
	"strings"

	"llm-gateway/internal/database"
)

// StatsQueryService 通用统计查询服务
type StatsQueryService struct {
	store *database.Store
}

func NewStatsQueryService(store *database.Store) *StatsQueryService {
	return &StatsQueryService{store: store}
}

// QueryRequest 通用查询请求
type QueryRequest struct {
	Dimensions []string          `json:"dimensions"`
	Measures   []string          `json:"measures"`
	Filters    []QueryFilter     `json:"filters"`
	Sort       []QuerySort       `json:"sort"`
	Page       int               `json:"page"`
	Size       int               `json:"size"`
}

type QueryFilter struct {
	Field string      `json:"field"`
	Op    string      `json:"op"`
	Value interface{} `json:"value"`
}

type QuerySort struct {
	Field string `json:"field"`
	Dir   string `json:"dir"`
}

// QueryResponse 通用查询响应
type QueryResponse struct {
	Rows  []map[string]interface{} `json:"rows"`
	Total int64                    `json:"total"`
}

// 白名单定义

var dimensionWhitelist = map[string]string{
	"hour":           "hour",
	"user_id":        "user_id",
	"user_model":     "user_model",
	"provider_model": "provider_model",
}

var measureWhitelist = map[string]string{
	"request_count":     "SUM(request_count)",
	"success_count":     "SUM(success_count)",
	"error_count":       "SUM(error_count)",
	"prompt_tokens":     "SUM(prompt_tokens)",
	"completion_tokens": "SUM(completion_tokens)",
	"reasoning_tokens":  "SUM(reasoning_tokens)",
	"total_tokens":      "SUM(total_tokens)",
	"avg_latency_ms":    "CASE WHEN SUM(request_count) > 0 THEN CAST(SUM(total_duration) AS DOUBLE) / SUM(request_count) ELSE 0 END",
	"unique_users":      "COUNT(DISTINCT user_id)",
}

var filterFieldWhitelist = map[string]string{
	"user_id":        "user_id",
	"user_model":     "user_model",
	"provider_model": "provider_model",
	"hour":           "hour",
}

var operatorWhitelist = map[string]string{
	"eq":      "=",
	"ne":      "!=",
	"gt":      ">",
	"gte":     ">=",
	"lt":      "<",
	"lte":     "<=",
	"in":      "IN",
	"between": "BETWEEN",
	"like":    "LIKE",
}

// sort 字段可以是 dimension 或 measure 中的任意一个
func sortFieldWhitelist() map[string]string {
	m := make(map[string]string)
	for k, v := range dimensionWhitelist {
		m[k] = v
	}
	for k, v := range measureWhitelist {
		m[k] = v
	}
	return m
}

// Query 执行通用查询
func (s *StatsQueryService) Query(req QueryRequest) (*QueryResponse, error) {
	// 1. 验证 dimensions
	var dimCols []string
	for _, d := range req.Dimensions {
		col, ok := dimensionWhitelist[d]
		if !ok {
			return nil, fmt.Errorf("invalid dimension: %s", d)
		}
		dimCols = append(dimCols, col+" AS "+d)
	}

	// 2. 验证 measures
	var measCols []string
	for _, m := range req.Measures {
		col, ok := measureWhitelist[m]
		if !ok {
			return nil, fmt.Errorf("invalid measure: %s", m)
		}
		measCols = append(measCols, col+" AS "+m)
	}

	if len(measCols) == 0 {
		return nil, fmt.Errorf("at least one measure is required")
	}

	// 3. 构建 SELECT
	selectPart := strings.Join(append(dimCols, measCols...), ", ")

	// 4. 构建 WHERE
	var whereParts []string
	var args []interface{}
	for _, f := range req.Filters {
		field, ok := filterFieldWhitelist[f.Field]
		if !ok {
			return nil, fmt.Errorf("invalid filter field: %s", f.Field)
		}
		op, ok := operatorWhitelist[f.Op]
		if !ok {
			return nil, fmt.Errorf("invalid operator: %s", f.Op)
		}

		switch f.Op {
		case "in":
			vals, ok := f.Value.([]interface{})
			if !ok || len(vals) == 0 {
				return nil, fmt.Errorf("in operator requires non-empty array")
			}
			placeholders := strings.TrimSuffix(strings.Repeat("?,", len(vals)), ",")
			whereParts = append(whereParts, fmt.Sprintf("%s IN (%s)", field, placeholders))
			args = append(args, vals...)
		case "between":
			vals, ok := f.Value.([]interface{})
			if !ok || len(vals) != 2 {
				return nil, fmt.Errorf("between operator requires exactly 2 values")
			}
			whereParts = append(whereParts, fmt.Sprintf("%s BETWEEN ? AND ?", field))
			args = append(args, vals[0], vals[1])
		case "like":
			whereParts = append(whereParts, fmt.Sprintf("%s %s ?", field, op))
			args = append(args, "%"+fmt.Sprintf("%v", f.Value)+"%")
		default:
			whereParts = append(whereParts, fmt.Sprintf("%s %s ?", field, op))
			args = append(args, f.Value)
		}
	}

	// 5. 构建 ORDER BY
	sortWl := sortFieldWhitelist()
	var orderParts []string
	for _, st := range req.Sort {
		col, ok := sortWl[st.Field]
		if !ok {
			return nil, fmt.Errorf("invalid sort field: %s", st.Field)
		}
		dir := "DESC"
		if st.Dir == "asc" {
			dir = "ASC"
		}
		orderParts = append(orderParts, fmt.Sprintf("%s %s", col, dir))
	}

	// 6. 分页
	page, size := normalizePaging(req.Page, req.Size)

	// 7. 先查 total
	countSQL := "SELECT COUNT(*) FROM (SELECT 1 FROM stats_hourly"
	if len(whereParts) > 0 {
		countSQL += " WHERE " + strings.Join(whereParts, " AND ")
	}
	if len(dimCols) > 0 {
		countSQL += " GROUP BY " + strings.Join(func() []string {
			var cols []string
			for _, d := range req.Dimensions {
				cols = append(cols, dimensionWhitelist[d])
			}
			return cols
		}(), ", ")
	}
	countSQL += ") t"

	var total int64
	if err := s.store.DB().Get(&total, countSQL, args...); err != nil {
		return nil, fmt.Errorf("count query failed: %w", err)
	}

	if total == 0 {
		return &QueryResponse{Rows: []map[string]interface{}{}, Total: 0}, nil
	}

	// 8. 查数据
	dataSQL := fmt.Sprintf("SELECT %s FROM stats_hourly", selectPart)
	if len(whereParts) > 0 {
		dataSQL += " WHERE " + strings.Join(whereParts, " AND ")
	}
	if len(dimCols) > 0 {
		dataSQL += " GROUP BY " + strings.Join(func() []string {
			var cols []string
			for _, d := range req.Dimensions {
				cols = append(cols, dimensionWhitelist[d])
			}
			return cols
		}(), ", ")
	}
	if len(orderParts) > 0 {
		dataSQL += " ORDER BY " + strings.Join(orderParts, ", ")
	}
	dataSQL += fmt.Sprintf(" LIMIT %d OFFSET %d", size, (page-1)*size)

	// 使用 sqlx 扫描到 []map[string]interface{}
	rows, err := s.store.DB().Queryx(dataSQL, args...)
	if err != nil {
		return nil, fmt.Errorf("data query failed: %w", err)
	}
	defer rows.Close()

	var result []map[string]interface{}
	for rows.Next() {
		m := make(map[string]interface{})
		if err := rows.MapScan(m); err != nil {
			return nil, fmt.Errorf("scan failed: %w", err)
		}
		result = append(result, m)
	}

	return &QueryResponse{Rows: result, Total: total}, nil
}