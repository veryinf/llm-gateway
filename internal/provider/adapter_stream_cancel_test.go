package provider

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"llm-gateway/internal/model"
)

// TestStreamClientCancel_UpstreamClosed 验证：客户端 ctx 取消后，上游连接同步断开
func TestStreamClientCancel_UpstreamClosed(t *testing.T) {
	var connClosed atomic.Bool

	// --- mock 上游 SSE 服务端 ---
	// 每 100ms 发一个 chunk，同时检测连接是否被客户端断开
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		w.WriteHeader(http.StatusOK)

		flusher, ok := w.(http.Flusher)
		if !ok {
			t.Error("no flusher")
			return
		}

		// 写第一个 chunk 确认连接建立
		fmt.Fprintf(w, "data: %s\n\n", `{"choices":[{"delta":{"role":"assistant"}}]}`)
		flusher.Flush()

		// 循环发 chunk，同时检测 r.Context() 是否被取消（即上游连接是否被客户端关）
		ticker := time.NewTicker(100 * time.Millisecond)
		defer ticker.Stop()

		cnt := 0
		for {
			select {
			case <-r.Context().Done():
				// 上游感知到连接断开
				connClosed.Store(true)
				return
			case <-ticker.C:
				cnt++
				fmt.Fprintf(w, "data: %s\n\n", fmt.Sprintf(`{"choices":[{"delta":{"content":"chunk%d"}}]}`, cnt))
				flusher.Flush()
			}
		}
	}))
	defer upstream.Close()

	// --- 构造 Adapter ---
	provider := &model.Provider{
		Title:         "test",
		BaseURL:       upstream.URL,
		APIKey:        "test-key",
		SupportOpenai: true,
		IsActive:      true,
	}
	adapter, err := NewAdapter(provider)
	if err != nil {
		t.Fatalf("NewAdapter: %v", err)
	}

	// --- 构造 LLMRequest ---
	body := `{"model":"test-model","stream":true,"messages":[{"role":"user","content":"hi"}]}`
	req, err := http.NewRequest(http.MethodPost, upstream.URL+"/chat/completions", bytes.NewReader([]byte(body)))
	if err != nil {
		t.Fatalf("NewRequest: %v", err)
	}
	llmReq := &LLMRequest{
		APIType: model.APITypeOpenAI,
		Request: req,
		Model:   "test-model",
		Stream:  true,
	}

	// --- 用可取消的 ctx 发起流式请求 ---
	ctx, cancel := context.WithCancel(context.Background())
	chunkCh, errCh := adapter.ChatCompletionStream(ctx, llmReq)

	// 消费几个 chunk 确认流正常
	received := 0
	for range chunkCh {
		received++
		if received >= 3 {
			break
		}
	}

	// --- 客户端取消 ---
	cancel()

	// --- 等待上游感知到断开 ---
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		if connClosed.Load() {
			break
		}
		time.Sleep(20 * time.Millisecond)
	}

	if !connClosed.Load() {
		t.Fatal("客户端取消后，上游连接未断开 — 上游仍在浪费资源生成数据")
	}

	// --- 验证 goroutine 正常退出：chunkCh 应该被关闭 ---
	drainDeadline := time.Now().Add(2 * time.Second)
	for !closed(chunkCh) {
		if time.Now().After(drainDeadline) {
			t.Fatal("客户端取消后，chunkCh 未关闭 — goroutine 可能泄漏")
		}
		time.Sleep(20 * time.Millisecond)
	}

	// errCh 也应该被关闭（读到零值 nil）
	if err := <-errCh; err != nil {
		t.Logf("errCh 返回错误（非必须）: %v", err)
	}

	t.Logf("客户端取消后，上游在 %d 个 chunk 后断开 ✓", received)
}

// TestStreamClientCancel_GoroutineExit 验证：客户端取消后，goroutine 不泄漏
func TestStreamClientCancel_GoroutineExit(t *testing.T) {
	blockCh := make(chan struct{})

	// mock 上游：发一个 chunk 后阻塞，直到连接断开
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		w.WriteHeader(http.StatusOK)
		fmt.Fprintf(w, "data: %s\n\n", `{"choices":[{"delta":{"role":"assistant"}}]}`)
		w.(http.Flusher).Flush()

		// 阻塞直到客户端断开
		<-r.Context().Done()
		close(blockCh)
	}))
	defer upstream.Close()

	provider := &model.Provider{
		Title:         "test",
		BaseURL:       upstream.URL,
		APIKey:        "test-key",
		SupportOpenai: true,
		IsActive:      true,
	}
	adapter, err := NewAdapter(provider)
	if err != nil {
		t.Fatalf("NewAdapter: %v", err)
	}

	body := `{"model":"test-model","stream":true,"messages":[{"role":"user","content":"hi"}]}`
	req, err := http.NewRequest(http.MethodPost, upstream.URL+"/chat/completions", bytes.NewReader([]byte(body)))
	if err != nil {
		t.Fatalf("NewRequest: %v", err)
	}
	llmReq := &LLMRequest{
		APIType: model.APITypeOpenAI,
		Request: req,
		Model:   "test-model",
		Stream:  true,
	}

	ctx, cancel := context.WithCancel(context.Background())
	_, errCh := adapter.ChatCompletionStream(ctx, llmReq)

	// 等一小段时间确保 goroutine 已进入阻塞
	time.Sleep(100 * time.Millisecond)

	// 客户端取消
	cancel()

	// 等待上游感知到断开（blockCh 被关闭）
	select {
	case <-blockCh:
		// 上游已退出阻塞
	case <-time.After(2 * time.Second):
		t.Fatal("客户端取消后，上游 goroutine 未退出 — 可能泄漏")
	}

	// errCh 应该被关闭
	select {
	case <-errCh:
		// OK
	case <-time.After(2 * time.Second):
		t.Fatal("errCh 未关闭 — goroutine 可能泄漏")
	}
}

// TestStreamNormalComplete 对照：正常完成时上游不被强制断开
func TestStreamNormalComplete(t *testing.T) {
	var totalChunks atomic.Int32

	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		w.WriteHeader(http.StatusOK)

		for i := 0; i < 5; i++ {
			fmt.Fprintf(w, "data: %s\n\n", fmt.Sprintf(`{"choices":[{"delta":{"content":"%d"}}]}`, i))
			w.(http.Flusher).Flush()
			time.Sleep(50 * time.Millisecond)
		}
		fmt.Fprint(w, "data: [DONE]\n\n")
		w.(http.Flusher).Flush()
		totalChunks.Add(1)
	}))
	defer upstream.Close()

	provider := &model.Provider{
		Title:         "test",
		BaseURL:       upstream.URL,
		APIKey:        "test-key",
		SupportOpenai: true,
		IsActive:      true,
	}
	adapter, err := NewAdapter(provider)
	if err != nil {
		t.Fatalf("NewAdapter: %v", err)
	}

	body := `{"model":"test-model","stream":true,"messages":[{"role":"user","content":"hi"}]}`
	req, err := http.NewRequest(http.MethodPost, upstream.URL+"/chat/completions", bytes.NewReader([]byte(body)))
	if err != nil {
		t.Fatalf("NewRequest: %v", err)
	}
	llmReq := &LLMRequest{
		APIType: model.APITypeOpenAI,
		Request: req,
		Model:   "test-model",
		Stream:  true,
	}

	ctx := context.Background()
	chunkCh, errCh := adapter.ChatCompletionStream(ctx, llmReq)

	count := 0
	for range chunkCh {
		count++
	}

	if err := <-errCh; err != nil {
		t.Fatalf("正常完成不应返回错误: %v", err)
	}

	// 5 个内容 chunk + 1 个 [DONE] = 6
	if count != 6 {
		t.Fatalf("期望收到 6 个 chunk，实际 %d", count)
	}
}

// --- helpers ---

func closed[T any](ch <-chan T) bool {
	select {
	case _, ok := <-ch:
		return !ok
	default:
		return false
	}
}

// 以下两个测试验证 ReadSSE 本身的 ctx 取消行为（更底层）

func TestReadSSE_CancelCtx(t *testing.T) {
	// 用 TCP 连接模拟真实场景：关闭 server 端 = transport 关闭 body
	// io.Pipe 是同步的，不响应 ctx 取消，不能用于此测试
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	defer ln.Close()

	// server 端：接受连接后发一行 data 然后保持连接
	go func() {
		conn, err := ln.Accept()
		if err != nil {
			return
		}
		defer conn.Close()
		fmt.Fprint(conn, "data: hello\n")
		// 保持连接打开，等待客户端断开
		buf := make([]byte, 1024)
		for {
			_, err := conn.Read(buf)
			if err != nil {
				return
			}
		}
	}()

	conn, err := net.Dial("tcp", ln.Addr().String())
	if err != nil {
		t.Fatalf("dial: %v", err)
	}
	defer conn.Close()

	ctx, cancel := context.WithCancel(context.Background())
	ch := ReadSSE(ctx, conn)

	// 等一下让 scanner 进入阻塞读
	time.Sleep(100 * time.Millisecond)

	// 取消 ctx — 但注意：ReadSSE 本身不会关闭 conn
	// 它只在 scanner.Scan() 返回后检查 ctx.Done()
	// 所以我们需要模拟 transport 的行为：关闭 conn 让 Scan() 返回
	cancel()
	conn.Close() // 模拟 transport 在 ctx 取消时关闭 body

	// ReadSSE 应该退出并关闭 channel
	deadline := time.Now().Add(2 * time.Second)
	for !closed(ch) {
		if time.Now().After(deadline) {
			t.Fatal("conn 关闭后 ReadSSE 未退出")
		}
		time.Sleep(20 * time.Millisecond)
	}
}

func TestReadSSE_NormalEnd(t *testing.T) {
	// 模拟完整 SSE 流
	pr, pw := io.Pipe()
	defer pr.Close()

	ctx := context.Background()
	ch := ReadSSE(ctx, pr)

	go func() {
		defer pw.Close()
		fmt.Fprint(pw, `event: message
data: {"choices":[{"delta":{"content":"hello"}}]}

event: message
data: {"choices":[{"delta":{"content":" world"}}]}

data: [DONE]
`)
	}()

	var events []string
	for ev := range ch {
		events = append(events, ev.Data)
	}

	// [DONE] 被 ReadSSE 过滤掉，所以只有 2 个事件
	if len(events) != 2 {
		t.Fatalf("期望 2 个事件，实际 %d: %v", len(events), events)
	}
	if !strings.Contains(events[0], "hello") {
		t.Fatalf("第一个事件应含 hello: %s", events[0])
	}
	if !strings.Contains(events[1], "world") {
		t.Fatalf("第二个事件应含 world: %s", events[1])
	}
}

// 确保 bufio.Scanner 的 buffer 设置与生产一致
func TestReadSSE_LongLine(t *testing.T) {
	pr, pw := io.Pipe()
	defer pr.Close()

	ctx := context.Background()
	ch := ReadSSE(ctx, pr)

	// 构造一行超过默认 bufio 64KB 的 data
	longData := strings.Repeat("x", 200*1024) // 200KB
	go func() {
		defer pw.Close()
		fmt.Fprintf(pw, "data: %s\n\n", longData)
	}()

	var got []string
	for ev := range ch {
		got = append(got, ev.Data)
	}

	if len(got) != 1 {
		t.Fatalf("期望 1 个事件，实际 %d", len(got))
	}
	if len(got[0]) != 200*1024 {
		t.Fatalf("期望 200KB 数据，实际 %d", len(got[0]))
	}
}
