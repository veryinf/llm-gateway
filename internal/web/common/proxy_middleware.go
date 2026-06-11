package common

import (
	"net/http"
	"strings"
	"time"

	"llm-gateway/internal/core"
	"llm-gateway/internal/model"

	"github.com/labstack/echo/v4"
)

// ProxyMiddleware validates sk- API Key via Authorization: Bearer header.
// Designed for /v1 and /anthropic proxy routes.
func ProxyMiddleware() echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			cc := &LeContext{Context: c}

			authorization := c.Request().Header.Get("Authorization")
			if authorization == "" {
				return proxyUnauthorized(c)
			}

			if !strings.HasPrefix(authorization, "Bearer ") {
				return proxyUnauthorized(c)
			}

			rawKey := strings.TrimPrefix(authorization, "Bearer ")
			if rawKey == "" {
				return proxyUnauthorized(c)
			}

			var apiKey model.APIKey
			if err := core.DB.Where("`key` = ? AND is_active = ?", rawKey, true).First(&apiKey).Error; err != nil {
				return proxyUnauthorized(c)
			}

			if apiKey.ExpiresAt != nil && apiKey.ExpiresAt.Before(time.Now()) {
				return proxyUnauthorized(c)
			}

			var user model.User
			if err := core.DB.Where("id = ?", apiKey.UserID).First(&user).Error; err != nil {
				return proxyUnauthorized(c)
			}
			if !user.IsActive {
				return proxyUnauthorized(c)
			}

			cc.AuthUser = &user
			cc.APIKeyID = apiKey.ID

			return next(cc)
		}
	}
}

func proxyUnauthorized(c echo.Context) error {
	return c.JSON(http.StatusUnauthorized, map[string]interface{}{
		"error": map[string]interface{}{
			"message": "Invalid API key",
			"type":    "invalid_request_error",
			"code":    "invalid_api_key",
		},
	})
}
