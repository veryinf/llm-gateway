package provider

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// OpenAIProvider OpenAI Provider 实现
type OpenAIProvider struct {
	name       string
	baseURL    string
	apiKey     string
	httpClient *http.Client
}

// NewOpenAIProvider 创建 OpenAI Provider
func NewOpenAIProvider(name, baseURL, apiKey string) *OpenAIProvider {
	baseURL = strings.TrimRight(baseURL, "/")
	return &OpenAIProvider{
		name:    name,
		baseURL: baseURL,
		apiKey:  apiKey,
		httpClient: &http.Client{
			Timeout: 60 * time.Second,
		},
	}
}

// ID 返回 provider 名称
func (p *OpenAIProvider) ID() string {
	return p.name
}

// Type 返回 provider 类型
func (p *OpenAIProvider) Type() string {
	return "openai"
}

// ChatCompletion 非流式聊天补全
func (p *OpenAIProvider) ChatCompletion(ctx context.Context, req *ChatRequest) (*ChatResponse, error) {
	url := p.baseURL + "/chat/completions"

	body, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("marshal request: %w", err)
	}

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
		return nil, fmt.Errorf("openai api error: status=%d body=%s", resp.StatusCode, string(respBody))
	}

	var chatResp ChatResponse
	if err := json.NewDecoder(resp.Body).Decode(&chatResp); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
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
		url := p.baseURL + "/chat/completions"

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
			errCh <- fmt.Errorf("openai api error: status=%d body=%s", resp.StatusCode, string(respBody))
			return
		}

		scanner := bufio.NewScanner(resp.Body)
		scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)

		for scanner.Scan() {
			select {
			case <-ctx.Done():
				errCh <- ctx.Err()
				return
			default:
			}

			line := scanner.Text()
			line = strings.TrimSpace(line)

			if line == "" {
				continue
			}

			if !strings.HasPrefix(line, "data:") {
				continue
			}

			data := strings.TrimSpace(strings.TrimPrefix(line, "data:"))

			if data == "[DONE]" {
				return
			}

			var chunk ChatStreamChunk
			if err := json.Unmarshal([]byte(data), &chunk); err != nil {
				// 跳过解析失败的行，继续处理后续数据
				continue
			}

			chunkCh <- &chunk
		}

		if err := scanner.Err(); err != nil {
			errCh <- fmt.Errorf("scanner error: %w", err)
		}
	}()

	return chunkCh, errCh
}

// ListModels 获取可用模型列表
func (p *OpenAIProvider) ListModels(ctx context.Context) ([]ModelInfo, error) {
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
		return nil, fmt.Errorf("openai api error: status=%d body=%s", resp.StatusCode, string(respBody))
	}

	var result struct {
		Data []ModelInfo `json:"data"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}

	return result.Data, nil
}

// setHeaders 设置通用请求头
func (p *OpenAIProvider) setHeaders(req *http.Request) {
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+p.apiKey)
}
