package provider

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"strings"
	"time"
)

// defaultTransport 返回默认的 HTTP Transport 配置
func defaultTransport() *http.Transport {
	return &http.Transport{
		DialContext: (&net.Dialer{
			Timeout:   10 * time.Second,
			KeepAlive: 30 * time.Second,
		}).DialContext,
		TLSHandshakeTimeout:   10 * time.Second,
		ResponseHeaderTimeout: 30 * time.Second,
		MaxIdleConns:          100,
		MaxIdleConnsPerHost:   20,
		IdleConnTimeout:       90 * time.Second,
	}
}

// newHTTPClient 创建带有默认 Transport 的 HTTP 客户端
func newHTTPClient() *http.Client {
	return &http.Client{Transport: defaultTransport()}
}

// SSEEvent 表示一个 Server-Sent Event
type SSEEvent struct {
	Data string
}

// ReadSSE 从 io.Reader 中读取 SSE 事件流，通过 channel 返回
func ReadSSE(ctx context.Context, r io.Reader) <-chan SSEEvent {
	ch := make(chan SSEEvent, 100)
	go func() {
		defer close(ch)
		scanner := bufio.NewScanner(r)
		scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)

		for scanner.Scan() {
			select {
			case <-ctx.Done():
				return
			default:
			}
			line := strings.TrimSpace(scanner.Text())
			if line == "" || !strings.HasPrefix(line, "data:") {
				continue
			}
			data := strings.TrimSpace(strings.TrimPrefix(line, "data:"))
			if data == "[DONE]" {
				return
			}
			ch <- SSEEvent{Data: data}
		}
	}()
	return ch
}

// handleHTTPError 处理非 200 状态码的响应，返回格式化错误
func handleHTTPError(resp *http.Response, prefix string) error {
	body, _ := io.ReadAll(resp.Body)
	return fmt.Errorf("%s api error: status=%d body=%s", prefix, resp.StatusCode, string(body))
}

// decodeJSON 解码 JSON 响应体到目标结构
func decodeJSON(resp *http.Response, v any) error {
	if err := json.NewDecoder(resp.Body).Decode(v); err != nil {
		return fmt.Errorf("decode response: %w", err)
	}
	return nil
}
