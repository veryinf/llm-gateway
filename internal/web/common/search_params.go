package common

// Pagination 分页参数
type Pagination struct {
	Index  int `json:"index"`
	Size   int `json:"size"`
	Offset int `json:"-"`
}

// FilterItem 过滤条件
type FilterItem struct {
	Field string `json:"field"`
	Value any    `json:"value"`
}

// SearchParams 搜索参数
type SearchParams struct {
	Pagination
	Kw      string       `json:"kw"`
	Filters []FilterItem `json:"filters"`
}
