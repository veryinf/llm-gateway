package provider

import (
	"encoding/json"
	"io"
	"net/http"
)

type LLMAPIType string

// APIType 常量定义
const (
	APITypeOpenAI    LLMAPIType = "openai"
	APITypeAnthropic LLMAPIType = "anthropic"
)

// LLMRequest LLM 请求封装
type LLMRequest struct {
	Raw     *http.Request   // 原始 HTTP 请求（用于转发）
	APIType LLMAPIType      // API 类型：openai / anthropic
	Model   string          // 模型名称（用于路由判断）
	Stream  bool            // 是否流式（用于路由判断）
	BodyRaw json.RawMessage // 原始请求体字节（用于日志记录）
}

// LLMResponse LLM 响应封装
type LLMResponse struct {
	StatusCode int
	Body       io.ReadCloser // 原始响应体，调用方负责关闭和解析
}

// ModelInfo 模型信息
type ModelInfo struct {
	ID      string `json:"id"`
	Object  string `json:"object"`
	Created int64  `json:"created"`
	OwnedBy string `json:"ownedBy"`
}

// ==================== LLMRequest 转换方法 ====================

// ToAnthropic 将 OpenAI 请求转为 Anthropic 格式
func (r *LLMRequest) ToAnthropic() (*LLMRequest, error) {
	converter := NewConverter()
	newRaw, err := converter.RequestOpenAIToAnthropic(r.Raw)
	if err != nil {
		return nil, err
	}
	return &LLMRequest{
		Raw:     newRaw,
		APIType: APITypeAnthropic,
		Model:   r.Model,
		Stream:  r.Stream,
		BodyRaw: r.BodyRaw,
	}, nil
}

// ToOpenAI 将 Anthropic 请求转为 OpenAI 格式
func (r *LLMRequest) ToOpenAI() (*LLMRequest, error) {
	converter := NewConverter()
	newRaw, err := converter.RequestAnthropicToOpenAI(r.Raw)
	if err != nil {
		return nil, err
	}
	return &LLMRequest{
		Raw:     newRaw,
		APIType: APITypeOpenAI,
		Model:   r.Model,
		Stream:  r.Stream,
		BodyRaw: r.BodyRaw,
	}, nil
}

// ==================== LLMResponse 转换方法 ====================

// ToAnthropic 将 OpenAI 响应转为 Anthropic 格式
func (r *LLMResponse) ToAnthropic() (*LLMResponse, error) {
	converter := NewConverter()
	newBody, err := converter.ResponseOpenAIToAnthropic(r.Body)
	if err != nil {
		return nil, err
	}
	return &LLMResponse{
		StatusCode: r.StatusCode,
		Body:       newBody,
	}, nil
}

// ToOpenAI 将 Anthropic 响应转为 OpenAI 格式
func (r *LLMResponse) ToOpenAI() (*LLMResponse, error) {
	converter := NewConverter()
	newBody, err := converter.ResponseAnthropicToOpenAI(r.Body)
	if err != nil {
		return nil, err
	}
	return &LLMResponse{
		StatusCode: r.StatusCode,
		Body:       newBody,
	}, nil
}
