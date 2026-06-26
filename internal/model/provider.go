package model

// LLMAPIType LLM API 类型
type LLMAPIType string

const (
	APITypeOpenAI    LLMAPIType = "openai"
	APITypeAnthropic LLMAPIType = "anthropic"
)

// PassthroughLevel 透传级别
type PassthroughLevel string

const (
	PassthroughLevelNone     PassthroughLevel = "none"
	PassthroughLevelUser     PassthroughLevel = "user"
	PassthroughLevelProvider PassthroughLevel = "provider"
)

type Provider struct {
	ProviderID       uint   `json:"providerId" gorm:"primaryKey;autoIncrement"`
	Title            string `json:"title"`
	BaseURL          string `json:"baseUrl"`
	APIKey           string `json:"apiKey,omitempty"`
	SupportOpenai    bool   `json:"supportOpenai"`
	OpenaiBaseURL    string `json:"openaiBaseUrl,omitempty"`
	SupportAnthropic bool   `json:"supportAnthropic"`
	AnthropicBaseURL string `json:"anthropicBaseUrl,omitempty"`
	IsActive         bool   `json:"isActive"`
	IsDefault        bool   `json:"isDefault"`
}

type ProviderModel struct {
	ModelID          uint    `json:"modelId" gorm:"primaryKey"`
	ProviderID       uint    `json:"providerId" gorm:"index"`
	Name             string  `json:"name"`
	DisplayName      string  `json:"displayName,omitempty"`
	Description      string  `json:"description,omitempty"`
	MaxContextTokens int64   `json:"maxContextTokens"`
	MaxOutputTokens  int64   `json:"maxOutputTokens"`
	InputPrice       float64 `json:"inputPrice"`
	OutputPrice      float64 `json:"outputPrice"`
	TPM              int     `json:"tpm"`
	QPM              int     `json:"qpm"`
	IsActive         bool    `json:"isActive"`
}
