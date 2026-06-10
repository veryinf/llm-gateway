package provider

import (
	"context"
	"fmt"
	"math"
	"time"

	"golang.org/x/time/rate"
)

// RateLimitedProvider 包装 LLMProvider，实现基于令牌桶的速率限制。
// 超限时通过 limiter.Wait(ctx) 排队等待，不返回错误。
type RateLimitedProvider struct {
	inner   LLMProvider
	limiter *rate.Limiter
}

// NewRateLimitedProvider 创建速率限制包装器。
// qpm: 每分钟最大请求数
// burst: 瞬时并发上限，传 0 时默认为 max(qpm/10, 1)
func NewRateLimitedProvider(inner LLMProvider, qpm, burst int) *RateLimitedProvider {
	r := rate.Every(time.Minute / time.Duration(qpm))
	if burst <= 0 {
		burst = int(math.Max(float64(qpm/10), 1))
	}
	return &RateLimitedProvider{
		inner:   inner,
		limiter: rate.NewLimiter(r, burst),
	}
}

func (p *RateLimitedProvider) ID() string {
	return p.inner.ID()
}

func (p *RateLimitedProvider) Type() string {
	return p.inner.Type()
}

func (p *RateLimitedProvider) ChatCompletion(ctx context.Context, req *ChatRequest) (*ChatResponse, error) {
	if err := p.limiter.Wait(ctx); err != nil {
		return nil, fmt.Errorf("rate limit wait cancelled: %w", err)
	}
	return p.inner.ChatCompletion(ctx, req)
}

func (p *RateLimitedProvider) ChatCompletionStream(ctx context.Context, req *ChatRequest) (<-chan *ChatStreamChunk, <-chan error) {
	if err := p.limiter.Wait(ctx); err != nil {
		errCh := make(chan error, 1)
		errCh <- fmt.Errorf("rate limit wait cancelled: %w", err)
		close(errCh)
		return nil, errCh
	}
	return p.inner.ChatCompletionStream(ctx, req)
}

func (p *RateLimitedProvider) ListModels(ctx context.Context) ([]ModelInfo, error) {
	return p.inner.ListModels(ctx)
}
