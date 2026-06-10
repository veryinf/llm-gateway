package provider

// OpenAICompatProvider 通用 OpenAI 兼容 Provider
// 适用于 DeepSeek、通义千问、Kimi、Ollama 等兼容 OpenAI 协议的服务
type OpenAICompatProvider struct {
	*OpenAIProvider
}

// NewOpenAICompatibleProvider 创建 OpenAI 兼容 Provider
func NewOpenAICompatibleProvider(name, baseURL, apiKey string) *OpenAICompatProvider {
	return &OpenAICompatProvider{
		OpenAIProvider: NewOpenAIProvider(name, baseURL, apiKey),
	}
}

// Type 返回 provider 类型，覆盖嵌入的 OpenAIProvider.Type()
func (p *OpenAICompatProvider) Type() string {
	return "openai-compatible"
}
