package common

import "strings"

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
	Pagination Pagination   `json:"pagination"`
	Kw         string       `json:"kw"`
	Filters    []FilterItem `json:"filters"`
}

// EscapedKw 返回转义后的 LIKE 模式（首尾带 %），用于安全的关键词搜索。
// SQL 使用时需配合 ESCAPE '\' 子句，例如：
//   WHERE title LIKE ? ESCAPE '\'
func (s *SearchParams) EscapedKw() string {
	if s.Kw == "" {
		return ""
	}
	escaped := strings.ReplaceAll(s.Kw, `\`, `\\`)
	escaped = strings.ReplaceAll(escaped, `%`, `\%`)
	escaped = strings.ReplaceAll(escaped, `_`, `\_`)
	return "%" + escaped + "%"
}
