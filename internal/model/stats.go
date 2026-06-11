package model

import "time"

type RequestLog struct {
	TraceID          string    `json:"trace_id"`
	UserID           uint      `json:"user_id"`
	APIKeyID         uint      `json:"api_key_id"`
	ProviderID       uint      `json:"provider_id"`
	ModelName        string    `json:"model_name"`
	IsStream         bool      `json:"is_stream"`
	PromptTokens     int       `json:"prompt_tokens"`
	CompletionTokens int       `json:"completion_tokens"`
	TotalTokens      int       `json:"total_tokens"`
	RequestBody      string    `json:"request_body"`
	ResponseBody     string    `json:"response_body"`
	IsDetail         bool      `json:"is_detail"`
	StatusCode       int       `json:"status_code"`
	ErrorMessage     string    `json:"error_message"`
	LatencyMs        int64     `json:"latency_ms"`
	Cost             float64   `json:"cost"`
	IPAddress        string    `json:"ip_address"`
	UserAgent        string    `json:"user_agent"`
	CreatedAt        time.Time `json:"created_at"`
}
