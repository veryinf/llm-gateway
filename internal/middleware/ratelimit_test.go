package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/labstack/echo/v4"
)

func TestRateLimitMiddleware(t *testing.T) {
	e := echo.New()

	limiter := RateLimitMiddleware(5)

	for i := 0; i <= 5; i++ {
		req := httptest.NewRequest(http.MethodGet, "/test", nil)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)
		c.Set(CtxKeyAPIKeyID, uint(1))

		handler := limiter(func(c echo.Context) error {
			return c.NoContent(http.StatusOK)
		})

		err := handler(c)
		if err != nil {
			t.Fatalf("request %d: unexpected error: %v", i+1, err)
		}

		if i < 5 {
			if rec.Code != http.StatusOK {
				t.Errorf("request %d: expected 200, got %d", i+1, rec.Code)
			}
		} else {
			if rec.Code != http.StatusTooManyRequests {
				t.Errorf("request %d: expected 429, got %d", i+1, rec.Code)
			}
		}
	}
}

func TestRateLimitMiddleware_DifferentKeys(t *testing.T) {
	e := echo.New()

	limiter := RateLimitMiddleware(3)

	for i := 0; i < 4; i++ {
		req := httptest.NewRequest(http.MethodGet, "/test", nil)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)
		c.Set(CtxKeyAPIKeyID, uint(1))

		handler := limiter(func(c echo.Context) error {
			return c.NoContent(http.StatusOK)
		})

		_ = handler(c)
		if i >= 3 && rec.Code != http.StatusTooManyRequests {
			t.Errorf("key 1 request %d: expected 429, got %d", i+1, rec.Code)
		}
	}

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.Set(CtxKeyAPIKeyID, uint(2))

	handler := limiter(func(c echo.Context) error {
		return c.NoContent(http.StatusOK)
	})

	_ = handler(c)
	if rec.Code != http.StatusOK {
		t.Errorf("key 2: expected 200, got %d", rec.Code)
	}
}

func TestRateLimitMiddleware_Cleanup(t *testing.T) {
	e := echo.New()

	limiter := RateLimitMiddleware(1)

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.Set(CtxKeyAPIKeyID, uint(1))

	handler := limiter(func(c echo.Context) error {
		return c.NoContent(http.StatusOK)
	})

	_ = handler(c)

	time.Sleep(10 * time.Millisecond)
}
