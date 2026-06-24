package provider

import (
	"bufio"
	"bytes"
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

	"github.com/tidwall/gjson"
)

// ==================== 请求解析函数 ====================

// ParseLLMRequest 从 *http.Request 解析 LLM 请求
// apiType: APITypeOpenAI 或 APITypeAnthropic
// 使用 TeeReader 复制 body，原始请求不被消耗
func ParseLLMRequest(req *http.Request, apiType LLMAPIType) (*LLMRequest, error) {
	// 使用 TeeReader 读取 body 并同时复制一份
	var bodyBuf bytes.Buffer
	bodyBytes, err := io.ReadAll(io.TeeReader(req.Body, &bodyBuf))
	if err != nil {
		return nil, fmt.Errorf("read request body: %w", err)
	}
	_ = req.Body.Close()

	// 重建原始请求的 body（保留原始流不被消耗）
	req.Body = io.NopCloser(&bodyBuf)
	// 使用 gjson 解析必要字段
	if !gjson.ValidBytes(bodyBytes) {
		return nil, fmt.Errorf("invalid request body")
	}
	result := gjson.ParseBytes(bodyBytes)
	return &LLMRequest{
		Raw:     req,
		APIType: apiType,
		Model:   result.Get("model").String(),
		Stream:  result.Get("stream").Bool(),
		BodyRaw: bodyBytes,
	}, nil
}

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

// ID 返回 Provider 唯一标识
func (a *Adapter) ID() string {
	return a.Provider.Title
}

// Type 返回 Provider 类型（支持的协议）
func (a *Adapter) Type() string {
	if a.SupportOpenAI() && a.SupportAnthropic() {
		return string(APITypeOpenAI) + "," + string(APITypeAnthropic)
	}
	if a.SupportOpenAI() {
		return string(APITypeOpenAI)
	}
	return string(APITypeAnthropic)
}

// GetProviderID 返回 Provider ID
func (a *Adapter) GetProviderID() uint {
	return a.Provider.ProviderID
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

	a.setOpenAIHeaders(req.Raw)

	resp, err := a.httpClient.Do(req.Raw)
	if err != nil {
		return nil, fmt.Errorf("http request: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		_ = resp.Body.Close()
		return nil, a.handleHTTPError(resp, "openai")
	}

	return &LLMResponse{
		StatusCode: resp.StatusCode,
		Body:       resp.Body,
	}, nil
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

		a.setOpenAIHeaders(req.Raw)

		resp, err := a.httpClient.Do(req.Raw)
		if err != nil {
			errCh <- fmt.Errorf("http request: %w", err)
			return
		}
		defer func(Body io.ReadCloser) {
			_ = Body.Close()
		}(resp.Body)

		if resp.StatusCode != http.StatusOK {
			errCh <- a.handleHTTPError(resp, "openai")
			return
		}

		for event := range readSSE(ctx, resp.Body) {
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
		return nil, a.handleHTTPError(resp, "openai")
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

	a.setAnthropicHeaders(req.Raw)

	resp, err := a.httpClient.Do(req.Raw)
	if err != nil {
		return nil, fmt.Errorf("http request: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		resp.Body.Close()
		return nil, a.handleHTTPError(resp, "anthropic")
	}

	return &LLMResponse{
		StatusCode: resp.StatusCode,
		Body:       resp.Body,
	}, nil
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

		a.setAnthropicHeaders(req.Raw)

		resp, err := a.httpClient.Do(req.Raw)
		if err != nil {
			errCh <- fmt.Errorf("http request: %w", err)
			return
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			errCh <- a.handleHTTPError(resp, "anthropic")
			return
		}

		for event := range readSSE(ctx, resp.Body) {
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

func (a *Adapter) handleHTTPError(resp *http.Response, apiType string) error {
	body, _ := io.ReadAll(resp.Body)
	return fmt.Errorf("%s API error (status %d): %s", apiType, resp.StatusCode, string(body))
}

// ==================== SSE 解析 ====================

type sseEvent struct {
	Event string
	Data  string
}

func readSSE(ctx context.Context, body io.Reader) <-chan sseEvent {
	ch := make(chan sseEvent, 100)

	go func() {
		defer close(ch)

		scanner := bufio.NewScanner(body)
		scanner.Buffer(make([]byte, 64*1024), 1024*1024)

		var event sseEvent
		for scanner.Scan() {
			select {
			case <-ctx.Done():
				return
			default:
			}

			line := scanner.Text()

			if line == "" {
				if event.Data != "" {
					ch <- event
					event = sseEvent{}
				}
				continue
			}

			if strings.HasPrefix(line, "event:") {
				event.Event = strings.TrimSpace(strings.TrimPrefix(line, "event:"))
			} else if strings.HasPrefix(line, "data:") {
				data := strings.TrimSpace(strings.TrimPrefix(line, "data:"))
				if event.Data == "" {
					event.Data = data
				} else {
					event.Data += "\n" + data
				}
			}
		}

		if event.Data != "" {
			ch <- event
		}
	}()

	return ch
}
