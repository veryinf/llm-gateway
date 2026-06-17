package provider

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
)

// OpenAIProvider OpenAI 协议适配器（同时兼容 OpenAI Compatible 服务）
type OpenAIProvider struct {
	BaseProvider
	apiType string // "openai" 或 "openai-compatible"
}

// NewOpenAIProvider 创建 OpenAI Provider
func NewOpenAIProvider(name, baseURL, apiKey string) *OpenAIProvider {
	return &OpenAIProvider{
		BaseProvider: NewBaseProvider(name, baseURL, apiKey),
		apiType:      "openai",
	}
}

// NewOpenAICompatibleProvider 创建 OpenAI 兼容 Provider（DeepSeek、通义、Kimi、Ollama 等）
func NewOpenAICompatibleProvider(name, baseURL, apiKey string) *OpenAIProvider {
	return &OpenAIProvider{
		BaseProvider: NewBaseProvider(name, baseURL, apiKey),
		apiType:      "openai-compatible",
	}
}

func (p *OpenAIProvider) Type() string { return p.apiType }

// ChatCompletion 非流式聊天补全
func (p *OpenAIProvider) ChatCompletion(ctx context.Context, req *ChatRequest) (*ChatResponse, error) {
	url := p.BaseURL + "/chat/completions"

	body, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("marshal request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}
	p.SetHeaders(httpReq)

	resp, err := p.HTTPClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("http request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, handleHTTPError(resp, p.Type())
	}

	var chatResp ChatResponse
	if err := decodeJSON(resp, &chatResp); err != nil {
		return nil, err
	}
	return &chatResp, nil
}

// ChatCompletionStream 流式聊天补全
func (p *OpenAIProvider) ChatCompletionStream(ctx context.Context, req *ChatRequest) (<-chan *ChatStreamChunk, <-chan error) {
	chunkCh := make(chan *ChatStreamChunk, 100)
	errCh := make(chan error, 1)

	go func() {
		defer close(chunkCh)
		defer close(errCh)

		reqCopy := *req
		reqCopy.Stream = true
		url := p.BaseURL + "/chat/completions"

		body, err := json.Marshal(reqCopy)
		if err != nil {
			errCh <- fmt.Errorf("marshal request: %w", err)
			return
		}

		httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
		if err != nil {
			errCh <- fmt.Errorf("create request: %w", err)
			return
		}
		p.SetHeaders(httpReq)
		httpReq.Header.Set("Accept", "text/event-stream")

		resp, err := p.HTTPClient.Do(httpReq)
		if err != nil {
			errCh <- fmt.Errorf("http request: %w", err)
			return
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			errCh <- handleHTTPError(resp, p.Type())
			return
		}

		for event := range ReadSSE(ctx, resp.Body) {
			var chunk ChatStreamChunk
			if err := json.Unmarshal([]byte(event.Data), &chunk); err != nil {
				continue
			}
			chunkCh <- &chunk
		}
	}()

	return chunkCh, errCh
}

// ListModels 获取可用模型列表（继承自 BaseProvider）
