package model

type Provider struct {
	ProviderID       uint   `json:"providerId" gorm:"primaryKey;autoIncrement"`
	Title            string `json:"title"`
	BaseURL          string `json:"baseUrl"`
	APIKey           string `json:"apiKey,omitempty"`
	SupportOpenAI    bool   `json:"supportOpenai"`
	OpenAIBaseURL    string `json:"openaiBaseUrl,omitempty"`
	SupportAnthropic bool   `json:"supportAnthropic"`
	AnthropicBaseURL string `json:"anthropicBaseUrl,omitempty"`
	PreferredAPI     string `json:"preferredApi"`
	IsActive         bool   `json:"isActive"`
}

type APIType string

const (
	APITypeOpenAI    APIType = "openai"
	APITypeAnthropic APIType = "anthropic"
)

type ProviderModel struct {
	ModelID          uint    `json:"modelId" gorm:"primaryKey"`
	ProviderID       uint    `json:"providerId" gorm:"index"`
	Name             string  `json:"name"`
	APIType          APIType `json:"apiType"`
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
