package provider

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"net/http"
	"net/url"
	"strings"
	"time"

	"golang.org/x/time/rate"

	"llm-gateway/internal/model"
)

// ==================== Adapter ====================

// Adapter 统一 Provider 适配器
type Adapter struct {
	Provider     model.Provider
	httpClient   *http.Client
	openaiURL    string
	anthropicURL string
	limiter      *rate.Limiter // 速率限制器（可选，为 nil 时不限流）
}

// NewAdapter 创建适配器
func NewAdapter(p *model.Provider, opts ...AdapterOption) (*Adapter, error) {
	if !p.SupportOpenai && !p.SupportAnthropic {
		return nil, fmt.Errorf("provider %q has no supported API types", p.Title)
	}

	adapter := &Adapter{
		Provider:   *p,
		httpClient: newHTTPClient(),
	}

	// 应用选项
	for _, opt := range opts {
		opt(adapter)
	}

	if p.SupportOpenai {
		adapter.openaiURL = p.OpenaiBaseURL
		if adapter.openaiURL == "" {
			adapter.openaiURL = strings.TrimRight(p.BaseURL, "/") + "/v1"
		}
	}

	if p.SupportAnthropic {
		adapter.anthropicURL = p.AnthropicBaseURL
		if adapter.anthropicURL == "" {
			adapter.anthropicURL = strings.TrimRight(p.BaseURL, "/") + "/anthropic/v1"
		}
	}

	return adapter, nil
}

// AdapterOption 适配器选项
type AdapterOption func(*Adapter)

// WithRateLimit 设置速率限制
// qpm: 每分钟最大请求数
// burst: 瞬时并发上限，传 0 时默认为 max(qpm/10, 1)
func WithRateLimit(qpm, burst int) AdapterOption {
	return func(a *Adapter) {
		if qpm <= 0 {
			return
		}
		r := rate.Every(time.Minute / time.Duration(qpm))
		if burst <= 0 {
			burst = int(math.Max(float64(qpm/10), 1))
		}
		a.limiter = rate.NewLimiter(r, burst)
	}
}

// waitRateLimit 等待速率限制
func (a *Adapter) waitRateLimit(ctx context.Context) error {
	if a.limiter == nil {
		return nil
	}
	return a.limiter.Wait(ctx)
}

// SupportOpenAI 是否支持 OpenAI 协议
func (a *Adapter) SupportOpenAI() bool {
	return a.Provider.SupportOpenai
}

// SupportAnthropic 是否支持 Anthropic 协议
func (a *Adapter) SupportAnthropic() bool {
	return a.Provider.SupportAnthropic
}

func (a *Adapter) ProviderAPIType(inputType model.LLMAPIType) model.LLMAPIType {
	if inputType == model.APITypeOpenAI {
		if a.SupportOpenAI() {
			return model.APITypeOpenAI
		}
		return model.APITypeAnthropic
	} else {
		if a.SupportAnthropic() {
			return model.APITypeAnthropic
		}
		return model.APITypeOpenAI
	}
}

func (a *Adapter) AutoChat(ctx context.Context, req *LLMRequest) (*LLMResponse, error) {
	if a.ProviderAPIType(req.APIType) == model.APITypeOpenAI {
		return a.ChatCompletion(ctx, req)
	}
	return a.Message(ctx, req)
}

// AutoStream 根据请求类型自动选择流式接口
func (a *Adapter) AutoStream(ctx context.Context, req *LLMRequest) (<-chan *LLMResponseChunk, <-chan error) {
	if a.ProviderAPIType(req.APIType) == model.APITypeOpenAI {
		return a.ChatCompletionStream(ctx, req)
	}
	return a.MessageStream(ctx, req)
}

// ==================== OpenAI 接口 ====================

// ChatCompletion OpenAI 非流式对话
func (a *Adapter) ChatCompletion(ctx context.Context, req *LLMRequest) (*LLMResponse, error) {
	if !a.SupportOpenAI() {
		return nil, fmt.Errorf("provider %q does not support OpenAI API", a.Provider.Title)
	}

	if err := a.waitRateLimit(ctx); err != nil {
		return nil, fmt.Errorf("rate limit wait cancelled: %w", err)
	}

	httpReq, err := a.newOpenAIRequest(ctx, "/chat/completions", req.Request.Body)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

	resp, err := a.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("http request: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		_ = resp.Body.Close()
		return nil, fmt.Errorf("%s api error: status=%d body=%s", "openai", resp.StatusCode, string(body))
	}

	return NewLLMResponse(resp, model.APITypeOpenAI)
}

// ChatCompletionStream OpenAI 流式对话
func (a *Adapter) ChatCompletionStream(ctx context.Context, req *LLMRequest) (<-chan *LLMResponseChunk, <-chan error) {
	chunkCh := make(chan *LLMResponseChunk, 100)
	errCh := make(chan error, 1)

	go func() {
		defer close(chunkCh)
		defer close(errCh)

		if !a.SupportOpenAI() {
			errCh <- fmt.Errorf("provider %q does not support OpenAI API", a.Provider.Title)
			return
		}

		if err := a.waitRateLimit(ctx); err != nil {
			errCh <- fmt.Errorf("rate limit wait cancelled: %w", err)
			return
		}

		httpReq, err := a.newOpenAIRequest(ctx, "/chat/completions", req.Request.Body)
		if err != nil {
			errCh <- fmt.Errorf("create request: %w", err)
			return
		}

		resp, err := a.httpClient.Do(httpReq)
		if err != nil {
			errCh <- fmt.Errorf("http request: %w", err)
			return
		}
		defer func(Body io.ReadCloser) {
			_ = Body.Close()
		}(resp.Body)

		if resp.StatusCode != http.StatusOK {
			body, _ := io.ReadAll(resp.Body)
			_ = resp.Body.Close()
			errCh <- fmt.Errorf("%s api error: status=%d body=%s", "openai", resp.StatusCode, string(body))
			return
		}

		for event := range ReadSSE(ctx, resp.Body) {
			if event.Data == "[DONE]" {
				break
			}
			chunkCh <- NewLLMResponseChunk([]byte(event.Data), model.APITypeOpenAI)
		}
	}()

	return chunkCh, errCh
}

// ListModels 获取 OpenAI 模型列表
func (a *Adapter) ListModels(ctx context.Context) ([]ModelInfo, error) {
	if !a.SupportOpenAI() {
		return nil, fmt.Errorf("provider %q does not support OpenAI API", a.Provider.Title)
	}

	httpReq, err := a.newOpenAIRequest(ctx, "/models", nil)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}
	httpReq.Method = http.MethodGet

	resp, err := a.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("http request: %w", err)
	}
	defer func(Body io.ReadCloser) {
		_ = Body.Close()
	}(resp.Body)

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("%s api error: status=%d body=%s", "openai", resp.StatusCode, string(body))
	}

	var result struct {
		Data []ModelInfo `json:"data"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}

	return result.Data, nil
}

// ==================== Anthropic 接口 ====================

// Message Anthropic 非流式对话
func (a *Adapter) Message(ctx context.Context, req *LLMRequest) (*LLMResponse, error) {
	if !a.SupportAnthropic() {
		return nil, fmt.Errorf("provider %q does not support Anthropic API", a.Provider.Title)
	}

	if err := a.waitRateLimit(ctx); err != nil {
		return nil, fmt.Errorf("rate limit wait cancelled: %w", err)
	}

	httpReq, err := a.newAnthropicRequest(ctx, "/messages", req.Request.Body)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

	resp, err := a.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("http request: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		_ = resp.Body.Close()
		return nil, fmt.Errorf("%s api error: status=%d body=%s", "anthropic", resp.StatusCode, string(body))
	}

	return NewLLMResponse(resp, model.APITypeAnthropic)
}

// MessageStream Anthropic 流式对话
func (a *Adapter) MessageStream(ctx context.Context, req *LLMRequest) (<-chan *LLMResponseChunk, <-chan error) {
	chunkCh := make(chan *LLMResponseChunk, 100)
	errCh := make(chan error, 1)

	go func() {
		defer close(chunkCh)
		defer close(errCh)

		if !a.SupportAnthropic() {
			errCh <- fmt.Errorf("provider %q does not support Anthropic API", a.Provider.Title)
			return
		}

		if err := a.waitRateLimit(ctx); err != nil {
			errCh <- fmt.Errorf("rate limit wait cancelled: %w", err)
			return
		}

		httpReq, err := a.newAnthropicRequest(ctx, "/messages", req.Request.Body)
		if err != nil {
			errCh <- fmt.Errorf("create request: %w", err)
			return
		}

		resp, err := a.httpClient.Do(httpReq)
		if err != nil {
			errCh <- fmt.Errorf("http request: %w", err)
			return
		}
		defer func(Body io.ReadCloser) {
			_ = Body.Close()
		}(resp.Body)

		if resp.StatusCode != http.StatusOK {
			body, _ := io.ReadAll(resp.Body)
			_ = resp.Body.Close()
			errCh <- fmt.Errorf("%s api error: status=%d body=%s", "anthropic", resp.StatusCode, string(body))
			return
		}

		for event := range ReadSSE(ctx, resp.Body) {
			chunkCh <- NewLLMResponseChunk([]byte(event.Data), model.APITypeAnthropic)
		}
	}()

	return chunkCh, errCh
}

// ==================== 内部辅助方法 ====================

// newOpenAIRequest 创建 OpenAI 请求（新请求，复用原始 body）
func (a *Adapter) newOpenAIRequest(ctx context.Context, endpoint string, body io.Reader) (*http.Request, error) {
	fullURL, err := url.JoinPath(a.openaiURL, endpoint)
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, fullURL, body)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+a.Provider.APIKey)
	return req, nil
}

// newAnthropicRequest 创建 Anthropic 请求（新请求，复用原始 body）
func (a *Adapter) newAnthropicRequest(ctx context.Context, endpoint string, body io.Reader) (*http.Request, error) {
	fullURL, err := url.JoinPath(a.anthropicURL, endpoint)
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, fullURL, body)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("x-api-key", a.Provider.APIKey)
	req.Header.Set("Authorization", "Bearer "+a.Provider.APIKey)
	req.Header.Set("anthropic-version", "2023-06-01")
	return req, nil
}
