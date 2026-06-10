package model

import "time"

type RequestLog struct {
	ID               uint      `gorm:"primaryKey" json:"id"`
	TraceID          string    `gorm:"index;size:36" json:"trace_id"`
	UserID           uint      `gorm:"index" json:"user_id"`
	APIKeyID         uint      `gorm:"index" json:"api_key_id"`
	ProviderID       uint      `json:"provider_id"`
	ModelName        string    `gorm:"size:128" json:"model_name"`
	IsStream         bool      `json:"is_stream"`
	PromptTokens     int       `json:"prompt_tokens"`
	CompletionTokens int      `json:"completion_tokens"`
	TotalTokens      int       `json:"total_tokens"`
	StatusCode       int       `json:"status_code"`
	ErrorMessage     string    `gorm:"size:1024" json:"error_message"`
	LatencyMs        int64     `json:"latency_ms"`
	Cost             float64   `json:"cost"`
	CreatedAt        time.Time `gorm:"index" json:"created_at"`

	User *User `gorm:"foreignKey:UserID" json:"user,omitempty"`
}

type StatisticsHourly struct {
	ID               uint    `gorm:"primaryKey" json:"id"`
	UserID           uint    `gorm:"index" json:"user_id"`
	Department       string  `gorm:"size:128;index" json:"department"`
	ProviderID       uint    `json:"provider_id"`
	ModelName        string  `gorm:"size:128" json:"model_name"`
	Hour             int64   `gorm:"index;not null" json:"hour"` // Unix timestamp truncated to hour
	RequestCount     int64   `json:"request_count"`
	SuccessCount     int64   `json:"success_count"`
	ErrorCount       int64   `json:"error_count"`
	PromptTokens     int64   `json:"prompt_tokens"`
	CompletionTokens int64  `json:"completion_tokens"`
	TotalTokens      int64   `json:"total_tokens"`
	TotalCost        float64 `json:"total_cost"`
	AvgLatencyMs     float64 `json:"avg_latency_ms"`
	P50LatencyMs     float64 `json:"p50_latency_ms"`
	P95LatencyMs     float64 `json:"p95_latency_ms"`
	P99LatencyMs     float64 `json:"p99_latency_ms"`

	User *User `gorm:"foreignKey:UserID" json:"user,omitempty"`
}

type PricingRule struct {
	ID                    uint      `gorm:"primaryKey" json:"id"`
	ProviderID            uint      `gorm:"index" json:"provider_id"`
	ModelName             string    `gorm:"size:128;not null" json:"model_name"`
	PromptPricePer1K      float64   `json:"prompt_price_per_1k"`
	CompletionPricePer1K  float64   `json:"completion_price_per_1k"`
	EffectiveDate         time.Time `json:"effective_date"`
	CreatedAt             time.Time `json:"created_at"`

	Provider *Provider `gorm:"foreignKey:ProviderID" json:"provider,omitempty"`
}
