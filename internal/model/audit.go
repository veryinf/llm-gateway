package model

import "time"

type AuditLog struct {
	ID               uint      `json:"id"`
	TraceID          string    `json:"trace_id"`
	UserID           uint      `json:"user_id"`
	APIKeyID         uint      `json:"api_key_id"`
	ProviderID       uint      `json:"provider_id"`
	ModelName        string    `json:"model_name"`
	RequestSummary   string    `json:"request_summary"`
	ResponseSummary  string    `json:"response_summary"`
	PromptTokens     int       `json:"prompt_tokens"`
	CompletionTokens int       `json:"completion_tokens"`
	StatusCode       int       `json:"status_code"`
	ErrorMessage     string    `json:"error_message"`
	LatencyMs        int64     `json:"latency_ms"`
	Cost             float64   `json:"cost"`
	IPAddress        string    `json:"ip_address"`
	UserAgent        string    `json:"user_agent"`
	CreatedAt        time.Time `json:"created_at"`
}
