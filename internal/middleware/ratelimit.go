package middleware

import (
	"sync"
	"time"

	"llm-gateway/pkg/apierror"
	"llm-gateway/pkg/response"

	"github.com/labstack/echo/v4"
)

type tokenBucket struct {
	tokens   int
	lastTime time.Time
}

type rateLimiter struct {
	mu         sync.Mutex
	buckets    map[uint]*tokenBucket
	defaultQPM int
}

var limiter *rateLimiter
var once sync.Once

func getLimiter(defaultQPM int) *rateLimiter {
	once.Do(func() {
		limiter = &rateLimiter{
			buckets:    make(map[uint]*tokenBucket),
			defaultQPM: defaultQPM,
		}
		go limiter.cleanup()
	})
	return limiter
}

func (rl *rateLimiter) allow(apiKeyID uint, customQPM int) bool {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	qpm := rl.defaultQPM
	if customQPM > 0 {
		qpm = customQPM
	}
	if qpm <= 0 {
		return true
	}

	bucket, exists := rl.buckets[apiKeyID]
	now := time.Now()

	if !exists || now.Sub(bucket.lastTime) >= time.Minute {
		rl.buckets[apiKeyID] = &tokenBucket{
			tokens:   qpm - 1,
			lastTime: now,
		}
		return true
	}

	if bucket.tokens > 0 {
		bucket.tokens--
		return true
	}

	return false
}

func (rl *rateLimiter) cleanup() {
	ticker := time.NewTicker(2 * time.Minute)
	defer ticker.Stop()
	for range ticker.C {
		rl.mu.Lock()
		now := time.Now()
		for id, bucket := range rl.buckets {
			if now.Sub(bucket.lastTime) > 2*time.Minute {
				delete(rl.buckets, id)
			}
		}
		rl.mu.Unlock()
	}
}

func RateLimitMiddleware(defaultQPM int) echo.MiddlewareFunc {
	rl := getLimiter(defaultQPM)

	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			apiKeyIDVal := c.Get(CtxKeyAPIKeyID)
			if apiKeyIDVal == nil {
				return next(c)
			}

			apiKeyID, ok := apiKeyIDVal.(uint)
			if !ok {
				return next(c)
			}

			customQPM := 0
			if apiKeyVal := c.Get(CtxKeyAPIKey); apiKeyVal != nil {
				if apiKey, ok := apiKeyVal.(interface{ GetRateLimitQPM() int }); ok {
					customQPM = apiKey.GetRateLimitQPM()
				}
			}

			if !rl.allow(apiKeyID, customQPM) {
				return response.Error(c, apierror.RateLimited("rate limit exceeded"))
			}

			return next(c)
		}
	}
}
