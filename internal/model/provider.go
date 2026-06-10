package model

import "time"

type ProviderType string

const (
	ProviderTypeOpenAI            ProviderType = "openai"
	ProviderTypeAzure             ProviderType = "azure"
	ProviderTypeAnthropic         ProviderType = "anthropic"
	ProviderTypeOpenAICompatible  ProviderType = "openai-compatible"
	ProviderTypeOllama            ProviderType = "ollama"
)

type Provider struct {
	ID              uint         `gorm:"primaryKey" json:"id"`
	Name            string       `gorm:"uniqueIndex;size:64;not null" json:"name"`
	Type            ProviderType `gorm:"size:32;not null" json:"type"`
	BaseURL         string       `gorm:"size:256;not null" json:"base_url"`
	APIKey          string       `gorm:"size:512;not null" json:"api_key,omitempty"` // API Key
	IsActive        bool         `gorm:"default:true" json:"is_active"`
	Priority        int          `gorm:"default:0" json:"priority"`
	RateLimitQPM    int          `gorm:"default:0" json:"rate_limit_qpm"`    // 每分钟最大请求数，0=不限制
	RateLimitBurst  int          `gorm:"default:0" json:"rate_limit_burst"`  // 瞬时并发上限，0=默认(max(QPM/10,1))
	CreatedAt       time.Time    `json:"created_at"`
	UpdatedAt       time.Time    `json:"updated_at"`
}
