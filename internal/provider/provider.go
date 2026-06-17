package provider

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
)

// FlexContent 灵活内容类型，兼容 string 和 array 两种格式
type FlexContent string

func (c *FlexContent) UnmarshalJSON(data []byte) error {
	// Try string first
	var s string
	if err := json.Unmarshal(data, &s); err == nil {
		*c = FlexContent(s)
		return nil
	}

	// Try array of content blocks, extract text
	var blocks []struct {
		Type string `json:"type"`
		Text string `json:"text"`
	}
	if err := json.Unmarshal(data, &blocks); err == nil {
		var texts []string
		for _, b := range blocks {
			if b.Type == "text" && b.Text != "" {
				texts = append(texts, b.Text)
			}
		}
		*c = FlexContent(strings.Join(texts, ""))
		return nil
	}

	return json.Unmarshal(data, (*string)(c))
}

func (c FlexContent) MarshalJSON() ([]byte, error) {
	return json.Marshal(string(c))
}

// ChatMessage 聊天消息
type ChatMessage struct {
	Role             string          `json:"role"`
	Content          FlexContent     `json:"content"`
	ReasoningContent string          `json:"reasoning_content,omitempty"`
	ToolCallID       string          `json:"tool_call_id,omitempty"`
	ToolCalls        json.RawMessage `json:"tool_calls,omitempty"`
	Name             string          `json:"name,omitempty"`
}

// ChatRequest 聊天请求
type ChatRequest struct {
	Model       string          `json:"model"`
	Messages    []ChatMessage   `json:"messages"`
	Stream      bool            `json:"stream,omitempty"`
	MaxTokens   int             `json:"max_tokens,omitempty"`
	Temperature float64         `json:"temperature,omitempty"`
	TopP        float64         `json:"top_p,omitempty"`
	Tools       json.RawMessage `json:"tools,omitempty"`
	ToolChoice  json.RawMessage `json:"tool_choice,omitempty"`
}

// ChatChoice 聊天响应选项
type ChatChoice struct {
	Index   int         `json:"index"`
	Message ChatMessage `json:"message"`
}

// ChatUsage Token 用量统计
type ChatUsage struct {
	PromptTokens     int `json:"prompt_tokens"`
	CompletionTokens int `json:"completion_tokens"`
	TotalTokens      int `json:"total_tokens"`
}

// ChatResponse 聊天响应
type ChatResponse struct {
	ID      string       `json:"id"`
	Object  string       `json:"object"`
	Created int64        `json:"created"`
	Model   string       `json:"model"`
	Choices []ChatChoice `json:"choices"`
	Usage   ChatUsage    `json:"usage"`
}

// ChatStreamChunk 流式响应的单个 chunk
type ChatStreamChunk struct {
	ID      string `json:"id"`
	Object  string `json:"object"`
	Created int64  `json:"created"`
	Model   string `json:"model"`
	Choices []struct {
		Index int `json:"index"`
		Delta struct {
			Role      string          `json:"role,omitempty"`
			Content   string          `json:"content,omitempty"`
			ToolCalls json.RawMessage `json:"tool_calls,omitempty"`
		} `json:"delta"`
		FinishReason string `json:"finish_reason,omitempty"`
	} `json:"choices"`
	Usage *ChatUsage `json:"usage,omitempty"`
}

// ModelInfo 模型信息
type ModelInfo struct {
	ID      string `json:"id"`
	Object  string `json:"object"`
	Created int64  `json:"created"`
	OwnedBy string `json:"owned_by"`
}

// LLMProvider LLM Provider 统一接口
type LLMProvider interface {
	// ID 返回 provider 唯一标识（name）
	ID() string
	// Type 返回 provider 类型（openai / openai-compatible 等）
	Type() string
	// ChatCompletion 非流式聊天补全
	ChatCompletion(ctx context.Context, req *ChatRequest) (*ChatResponse, error)
	// ChatCompletionStream 流式聊天补全，返回 chunk channel 和 error channel
	ChatCompletionStream(ctx context.Context, req *ChatRequest) (<-chan *ChatStreamChunk, <-chan error)
	// ListModels 获取可用模型列表
	ListModels(ctx context.Context) ([]ModelInfo, error)
}

// BaseProvider 所有 Provider 共享的基础设施
type BaseProvider struct {
	Name       string
	BaseURL    string
	APIKey     string
	HTTPClient *http.Client
}

// NewBaseProvider 创建 BaseProvider，baseURL 末尾斜杠会被去除
func NewBaseProvider(name, baseURL, apiKey string) BaseProvider {
	return BaseProvider{
		Name:       name,
		BaseURL:    strings.TrimRight(baseURL, "/"),
		APIKey:     apiKey,
		HTTPClient: newHTTPClient(),
	}
}

func (b *BaseProvider) ID() string   { return b.Name }
func (b *BaseProvider) Type() string { return "base" }

// ListModels 获取可用模型列表 — GET {baseURL}/v1/models
func (b *BaseProvider) ListModels(ctx context.Context) ([]ModelInfo, error) {
	url := b.BaseURL + "/v1/models"

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}
	b.SetHeaders(req)

	resp, err := b.HTTPClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("http request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, handleHTTPError(resp, b.Type())
	}

	var result struct {
		Data []ModelInfo `json:"data"`
	}
	if err := decodeJSON(resp, &result); err != nil {
		return nil, err
	}
	return result.Data, nil
}

// SetHeaders 设置通用请求头（子类型可覆盖）
func (b *BaseProvider) SetHeaders(req *http.Request) {
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+b.APIKey)
}
