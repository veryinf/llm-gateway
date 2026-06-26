package provider

import (
	"bufio"
	"context"
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
	Event string
	Data  string
}

// ReadSSE 从 io.Reader 中读取 SSE 事件流，通过 channel 返回
// 支持 event 字段和多行 data 拼接
func ReadSSE(ctx context.Context, r io.Reader) <-chan SSEEvent {
	ch := make(chan SSEEvent, 100)
	go func() {
		defer close(ch)
		scanner := bufio.NewScanner(r)
		scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)

		var event SSEEvent
		for scanner.Scan() {
			select {
			case <-ctx.Done():
				return
			default:
			}

			line := scanner.Text()

			// 空行表示事件结束
			if line == "" {
				if event.Data != "" {
					if event.Data == "[DONE]" {
						return
					}
					ch <- event
					event = SSEEvent{}
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

		// 处理最后一个事件（没有以空行结束的情况）
		if event.Data != "" && event.Data != "[DONE]" {
			ch <- event
		}
	}()
	return ch
}

// handleHTTPError 处理非 200 状态码的响应，返回格式化错误
func handleHTTPError(resp *http.Response, prefix string) error {
	body, _ := io.ReadAll(resp.Body)
	return fmt.Errorf("%s api error: status=%d body=%s", prefix, resp.StatusCode, string(body))
}
