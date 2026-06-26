package provider

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"net/http"
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

func (a *Adapter) AutoChat(ctx context.Context, req *LLMRequest) (*LLMResponse, error) {
	if req.APIType == model.APITypeOpenAI {
		if a.SupportOpenAI() {
			return a.ChatCompletion(ctx, req)
		}
		resp, err := a.Message(ctx, req)
		if err != nil {
			return nil, err
		}
		return resp.ToOpenAI()
	} else {
		if a.SupportAnthropic() {
			return a.Message(ctx, req)
		}
		resp, err := a.ChatCompletion(ctx, req)
		if err != nil {
			return nil, err
		}
		return resp.ToAnthropic()
	}
}

// ==================== OpenAI 接口 ====================

// ChatCompletion OpenAI 非流式对话（直接透传原始请求）
func (a *Adapter) ChatCompletion(ctx context.Context, req *LLMRequest) (*LLMResponse, error) {
	if !a.SupportOpenAI() {
		return nil, fmt.Errorf("provider %q does not support OpenAI API", a.Provider.Title)
	}

	if err := a.waitRateLimit(ctx); err != nil {
		return nil, fmt.Errorf("rate limit wait cancelled: %w", err)
	}

	a.setOpenAIHeaders(req.Request)

	resp, err := a.httpClient.Do(req.Request)
	if err != nil {
		return nil, fmt.Errorf("http request: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		_ = resp.Body.Close()
		return nil, handleHTTPError(resp, "openai")
	}

	return NewLLMResponse(resp, model.APITypeOpenAI)
}

// ChatCompletionStream OpenAI 流式对话（直接透传原始请求）
func (a *Adapter) ChatCompletionStream(ctx context.Context, req *LLMRequest) (<-chan []byte, <-chan error) {
	chunkCh := make(chan []byte, 100)
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

		a.setOpenAIHeaders(req.Request)

		resp, err := a.httpClient.Do(req.Request)
		if err != nil {
			errCh <- fmt.Errorf("http request: %w", err)
			return
		}
		defer func(Body io.ReadCloser) {
			_ = Body.Close()
		}(resp.Body)

		if resp.StatusCode != http.StatusOK {
			errCh <- handleHTTPError(resp, "openai")
			return
		}

		for event := range ReadSSE(ctx, resp.Body) {
			if event.Data == "[DONE]" {
				break
			}
			chunkCh <- []byte(event.Data)
		}
	}()

	return chunkCh, errCh
}

// ListModels 获取 OpenAI 模型列表
func (a *Adapter) ListModels(ctx context.Context) ([]ModelInfo, error) {
	if !a.SupportOpenAI() {
		return nil, fmt.Errorf("provider %q does not support OpenAI API", a.Provider.Title)
	}

	url := a.openaiURL + "/models"
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}
	a.setOpenAIHeaders(httpReq)

	resp, err := a.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("http request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, handleHTTPError(resp, "openai")
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

// Message Anthropic 非流式对话（直接透传原始请求）
func (a *Adapter) Message(ctx context.Context, req *LLMRequest) (*LLMResponse, error) {
	if !a.SupportAnthropic() {
		return nil, fmt.Errorf("provider %q does not support Anthropic API", a.Provider.Title)
	}

	if err := a.waitRateLimit(ctx); err != nil {
		return nil, fmt.Errorf("rate limit wait cancelled: %w", err)
	}

	a.setAnthropicHeaders(req.Request)

	resp, err := a.httpClient.Do(req.Request)
	if err != nil {
		return nil, fmt.Errorf("http request: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		resp.Body.Close()
		return nil, handleHTTPError(resp, "anthropic")
	}

	return NewLLMResponse(resp, model.APITypeAnthropic)
}

// MessageStream Anthropic 流式对话（直接透传原始请求）
func (a *Adapter) MessageStream(ctx context.Context, req *LLMRequest) (<-chan []byte, <-chan error) {
	chunkCh := make(chan []byte, 100)
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

		a.setAnthropicHeaders(req.Request)

		resp, err := a.httpClient.Do(req.Request)
		if err != nil {
			errCh <- fmt.Errorf("http request: %w", err)
			return
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			errCh <- handleHTTPError(resp, "anthropic")
			return
		}

		for event := range ReadSSE(ctx, resp.Body) {
			chunkCh <- []byte(event.Data)
		}
	}()

	return chunkCh, errCh
}

// ==================== 内部辅助方法 ====================

func (a *Adapter) setOpenAIHeaders(req *http.Request) {
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+a.Provider.APIKey)
}

func (a *Adapter) setAnthropicHeaders(req *http.Request) {
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("x-api-key", a.Provider.APIKey)
	req.Header.Set("anthropic-version", "2023-06-01")
}
