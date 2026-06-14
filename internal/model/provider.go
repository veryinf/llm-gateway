package model

import "time"

type Provider struct {
	ID               uint      `gorm:"primaryKey" json:"id"`
	Name             string    `gorm:"uniqueIndex;size:64;not null" json:"name"`
	BaseURL          string    `gorm:"size:256;not null" json:"base_url"`
	APIKey           string    `gorm:"size:512;not null" json:"api_key,omitempty"`
	SupportOpenAI    bool      `gorm:"default:false" json:"support_openai"`
	OpenAIBaseURL    string    `gorm:"size:256;default:''" json:"openai_base_url"`
	SupportAnthropic bool      `gorm:"default:false" json:"support_anthropic"`
	AnthropicBaseURL string    `gorm:"size:256;default:''" json:"anthropic_base_url"`
	PreferredAPI     string    `gorm:"size:32;default:'openai'" json:"preferred_api"`
	IsActive         bool      `gorm:"default:true" json:"is_active"`
	CreatedAt        time.Time `json:"created_at"`
	UpdatedAt        time.Time `json:"updated_at"`
}
