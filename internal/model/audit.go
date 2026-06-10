package model

import "time"

type AuditLog struct {
	ID              uint      `gorm:"primaryKey" json:"id"`
	TraceID         string    `gorm:"index;size:36;not null" json:"trace_id"`
	UserID          uint      `gorm:"index" json:"user_id"`
	APIKeyID        uint      `gorm:"index" json:"api_key_id"`
	ProviderID      uint      `json:"provider_id"`
	ModelName       string    `gorm:"size:128" json:"model_name"`
	RequestSummary  string    `gorm:"type:text" json:"request_summary"`
	ResponseSummary string    `gorm:"type:text" json:"response_summary"`
	PromptTokens    int       `json:"prompt_tokens"`
	CompletionTokens int     `json:"completion_tokens"`
	StatusCode      int       `json:"status_code"`
	ErrorMessage    string    `gorm:"size:1024" json:"error_message"`
	LatencyMs       int64     `json:"latency_ms"`
	Cost            float64   `json:"cost"`
	IPAddress       string    `gorm:"size:64" json:"ip_address"`
	UserAgent       string    `gorm:"size:512" json:"user_agent"`
	CreatedAt       time.Time `gorm:"index" json:"created_at"`
}
