package provider

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"strings"
	"time"
)

// AnthropicMessage Anthropic 消息格式
type AnthropicMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// AnthropicRequest Anthropic API 请求
type AnthropicRequest struct {
	Model       string             `json:"model"`
	Messages    []AnthropicMessage `json:"messages"`
	MaxTokens   int                `json:"max_tokens"`
	Stream      bool               `json:"stream,omitempty"`
	Temperature float64            `json:"temperature,omitempty"`
	TopP        float64            `json:"top_p,omitempty"`
	System      string             `json:"system,omitempty"`
}

// AnthropicContentBlock Anthropic 响应内容块
type AnthropicContentBlock struct {
	Type string `json:"type"`
	Text string `json:"text,omitempty"`
}

// AnthropicUsage Anthropic 用量统计
type AnthropicUsage struct {
	InputTokens  int `json:"input_tokens"`
	OutputTokens int `json:"output_tokens"`
}

// AnthropicResponse Anthropic API 响应
type AnthropicResponse struct {
	ID           string                  `json:"id"`
	Type         string                  `json:"type"`
	Role         string                  `json:"role"`
	Model        string                  `json:"model"`
	Content      []AnthropicContentBlock `json:"content"`
	StopReason   string                  `json:"stop_reason,omitempty"`
	Usage        AnthropicUsage          `json:"usage"`
}

// AnthropicStreamEvent Anthropic SSE 事件
type AnthropicStreamEvent struct {
	Type         string                  `json:"type"`
	Index        int                     `json:"index,omitempty"`
	ContentBlock *AnthropicContentBlock  `json:"content_block,omitempty"`
	Delta        *AnthropicDelta         `json:"delta,omitempty"`
	Usage        *AnthropicUsage         `json:"usage,omitempty"`
	Message      *AnthropicStreamMessage `json:"message,omitempty"`
}

type AnthropicDelta struct {
	Type         string `json:"type,omitempty"`
	Text         string `json:"text,omitempty"`
	StopReason   string `json:"stop_reason,omitempty"`
}

type AnthropicStreamMessage struct {
	Usage AnthropicUsage `json:"usage"`
}

// AnthropicProvider Anthropic 原生 API 适配器
type AnthropicProvider struct {
	name       string
	baseURL    string
	apiKey     string
	httpClient *http.Client
}

func NewAnthropicProvider(name, baseURL, apiKey string) *AnthropicProvider {
	baseURL = strings.TrimRight(baseURL, "/")
	return &AnthropicProvider{
		name:    name,
		baseURL: baseURL,
		apiKey:  apiKey,
		httpClient: &http.Client{
			Transport: &http.Transport{
				DialContext: (&net.Dialer{
					Timeout:   10 * time.Second,
					KeepAlive: 30 * time.Second,
				}).DialContext,
				TLSHandshakeTimeout:   10 * time.Second,
				ResponseHeaderTimeout: 30 * time.Second,
				MaxIdleConns:          100,
				MaxIdleConnsPerHost:   20,
				IdleConnTimeout:       90 * time.Second,
			},
		},
	}
}

func (p *AnthropicProvider) ID() string   { return p.name }
func (p *AnthropicProvider) Type() string  { return "anthropic" }

// ChatCompletion 将 OpenAI 格式请求转为 Anthropic 格式，再转回 OpenAI 格式响应
func (p *AnthropicProvider) ChatCompletion(ctx context.Context, req *ChatRequest) (*ChatResponse, error) {
	anthReq := p.convertToAnthropic(req)
	anthResp, err := p.sendAnthropicRequest(ctx, anthReq, false)
	if err != nil {
		return nil, err
	}
	return p.convertToOpenAI(req.Model, anthResp), nil
}

// ChatCompletionStream 流式转发 (OpenAI→Anthropic→OpenAI SSE)
func (p *AnthropicProvider) ChatCompletionStream(ctx context.Context, req *ChatRequest) (<-chan *ChatStreamChunk, <-chan error) {
	chunkCh := make(chan *ChatStreamChunk, 100)
	errCh := make(chan error, 1)

	go func() {
		defer close(chunkCh)
		defer close(errCh)

		anthReq := p.convertToAnthropic(req)
		anthReq.Stream = true

		body, err := json.Marshal(anthReq)
		if err != nil {
			errCh <- fmt.Errorf("marshal request: %w", err)
			return
		}

		url := p.baseURL + "/v1/messages"
		httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
		if err != nil {
			errCh <- fmt.Errorf("create request: %w", err)
			return
		}
		p.setHeaders(httpReq)
		httpReq.Header.Set("Accept", "text/event-stream")

		resp, err := p.httpClient.Do(httpReq)
		if err != nil {
			errCh <- fmt.Errorf("http request: %w", err)
			return
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			respBody, _ := io.ReadAll(resp.Body)
			errCh <- fmt.Errorf("anthropic api error: status=%d body=%s", resp.StatusCode, string(respBody))
			return
		}

		scanner := bufio.NewScanner(resp.Body)
		scanner.Buffer(make([]byte, 64*1024), 1024*1024)
		chunkIndex := 0
		var contentBuilder strings.Builder

		for scanner.Scan() {
			select {
			case <-ctx.Done():
				errCh <- ctx.Err()
				return
			default:
			}

			line := strings.TrimSpace(scanner.Text())
			if line == "" || !strings.HasPrefix(line, "data:") {
				continue
			}

			data := strings.TrimSpace(strings.TrimPrefix(line, "data:"))

			var event AnthropicStreamEvent
			if err := json.Unmarshal([]byte(data), &event); err != nil {
				continue
			}

			switch event.Type {
			case "content_block_delta":
				if event.Delta != nil && event.Delta.Text != "" {
					contentBuilder.WriteString(event.Delta.Text)
			chunkCh <- &ChatStreamChunk{
				ID:      "anthropic-" + fmt.Sprintf("%d", chunkIndex),
				Object:  "chat.completion.chunk",
				Created: time.Now().Unix(),
				Model:   req.Model,
				Choices: []struct {
					Index        int             `json:"index"`
					Delta        struct {
						Role      string          `json:"role,omitempty"`
						Content   string          `json:"content,omitempty"`
						ToolCalls json.RawMessage `json:"tool_calls,omitempty"`
					} `json:"delta"`
					FinishReason string `json:"finish_reason,omitempty"`
				}{
					{
						Index: 0,
						Delta: struct {
							Role      string          `json:"role,omitempty"`
							Content   string          `json:"content,omitempty"`
							ToolCalls json.RawMessage `json:"tool_calls,omitempty"`
						}{Content: event.Delta.Text},
					},
				},
			}
					chunkIndex++
				}

			case "message_delta":
				finishReason := ""
				if event.Delta != nil {
					finishReason = event.Delta.StopReason
				}
				chunkCh <- &ChatStreamChunk{
					ID:      "anthropic-" + fmt.Sprintf("%d", chunkIndex),
					Object:  "chat.completion.chunk",
					Created: time.Now().Unix(),
					Model:   req.Model,
					Choices: []struct {
						Index        int             `json:"index"`
						Delta        struct {
							Role      string          `json:"role,omitempty"`
							Content   string          `json:"content,omitempty"`
							ToolCalls json.RawMessage `json:"tool_calls,omitempty"`
						} `json:"delta"`
						FinishReason string `json:"finish_reason,omitempty"`
					}{
						{Index: 0, FinishReason: finishReason},
					},
					Usage: &ChatUsage{
						PromptTokens:     event.Usage.InputTokens,
						CompletionTokens: event.Usage.OutputTokens,
						TotalTokens:      event.Usage.InputTokens + event.Usage.OutputTokens,
					},
				}
			}
		}

		if err := scanner.Err(); err != nil {
			errCh <- fmt.Errorf("scanner error: %w", err)
		}
	}()

	return chunkCh, errCh
}

// ListModels 获取模型列表 (尝试 Anthropic 格式)
func (p *AnthropicProvider) ListModels(ctx context.Context) ([]ModelInfo, error) {
	url := p.baseURL + "/v1/models"
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}
	p.setHeaders(httpReq)

	resp, err := p.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("http request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("anthropic api error: status=%d body=%s", resp.StatusCode, string(respBody))
	}

	var result struct {
		Data []ModelInfo `json:"data"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}

	return result.Data, nil
}

func (p *AnthropicProvider) setHeaders(req *http.Request) {
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("x-api-key", p.apiKey)
	req.Header.Set("anthropic-version", "2023-06-01")
}

// convertToAnthropic 将 OpenAI 格式转为 Anthropic 格式
func (p *AnthropicProvider) convertToAnthropic(req *ChatRequest) *AnthropicRequest {
	anth := &AnthropicRequest{
		Model:       req.Model,
		MaxTokens:   req.MaxTokens,
		Temperature: req.Temperature,
		TopP:        req.TopP,
	}

	for _, msg := range req.Messages {
		if msg.Role == "system" {
			anth.System = string(msg.Content)
		} else {
			anth.Messages = append(anth.Messages, AnthropicMessage{
				Role:    msg.Role,
				Content: string(msg.Content),
			})
		}
	}

	return anth
}

// convertToOpenAI 将 Anthropic 响应转为 OpenAI 格式
func (p *AnthropicProvider) convertToOpenAI(modelName string, anth *AnthropicResponse) *ChatResponse {
	var content string
	if len(anth.Content) > 0 {
		content = anth.Content[0].Text
	}

	return &ChatResponse{
		ID:      anth.ID,
		Object:  "chat.completion",
		Created: time.Now().Unix(),
		Model:   modelName,
		Choices: []ChatChoice{
			{
				Index: 0,
				Message: ChatMessage{
					Role:    "assistant",
					Content: FlexContent(content),
				},
			},
		},
		Usage: ChatUsage{
			PromptTokens:     anth.Usage.InputTokens,
			CompletionTokens: anth.Usage.OutputTokens,
			TotalTokens:      anth.Usage.InputTokens + anth.Usage.OutputTokens,
		},
	}
}

// sendAnthropicRequest 发送 Anthropic API 请求
func (p *AnthropicProvider) sendAnthropicRequest(ctx context.Context, req *AnthropicRequest, stream bool) (*AnthropicResponse, error) {
	body, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("marshal request: %w", err)
	}

	url := p.baseURL + "/v1/messages"
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}
	p.setHeaders(httpReq)

	resp, err := p.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("http request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("anthropic api error: status=%d body=%s", resp.StatusCode, string(respBody))
	}

	var anthResp AnthropicResponse
	if err := json.NewDecoder(resp.Body).Decode(&anthResp); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}

	return &anthResp, nil
}
