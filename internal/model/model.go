package model

import "time"

type APIType string

const (
	APITypeOpenAI    APIType = "openai"
	APITypeAnthropic APIType = "anthropic"
)

type Model struct {
	ID                uint      `gorm:"primaryKey" json:"id"`
	ProviderID        uint      `gorm:"index;not null" json:"provider_id"`
	Name              string    `gorm:"size:128;not null" json:"name"`
	APIType           APIType   `gorm:"size:32;default:'openai'" json:"api_type"`
	DisplayName       string    `gorm:"size:128;default:''" json:"display_name"`
	Description       string    `gorm:"size:512;default:''" json:"description"`
	MaxContextTokens  int64     `gorm:"default:0" json:"max_context_tokens"`
	MaxOutputTokens   int64     `gorm:"default:0" json:"max_output_tokens"`
	InputPrice        float64   `gorm:"default:0" json:"input_price"`
	OutputPrice       float64   `gorm:"default:0" json:"output_price"`
	TPM               int       `gorm:"default:0" json:"tpm"`
	QPM               int       `gorm:"default:0" json:"qpm"`
	IsChat            bool      `gorm:"default:true" json:"is_chat"`
	IsCompletion      bool      `gorm:"default:false" json:"is_completion"`
	IsVision          bool      `gorm:"default:false" json:"is_vision"`
	IsEmbedding       bool      `gorm:"default:false" json:"is_embedding"`
	IsActive          bool      `gorm:"default:true" json:"is_active"`
	CreatedAt         time.Time `json:"created_at"`
	UpdatedAt         time.Time `json:"updated_at"`

	Provider *Provider `gorm:"foreignKey:ProviderID" json:"provider,omitempty"`
}
