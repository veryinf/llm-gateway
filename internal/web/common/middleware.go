package common

import (
	"crypto/md5"
	"encoding/hex"
	"math"
	"net/http"
	"strconv"
	"strings"
	"time"

	"llm-gateway/internal/core"
	"llm-gateway/internal/model"

	"github.com/labstack/echo/v4"
	"github.com/samber/lo"
)

type LeContext struct {
	echo.Context
	AuthToken *TokenInfo
	AuthUser  *model.User
	APIKeyID  uint
}

type LeMiddlewareConfig struct {
	IgnorePaths  []string
	TokenManager *TokenManager
}

func LeMiddleware(config LeMiddlewareConfig) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			cc := &LeContext{Context: c}
			if !lo.Contains(config.IgnorePaths, c.Path()) {
				authorization := c.Request().Header.Get("Authorization")
				// 验证 Authorization token
				if authorization != "" {
					token := authorization[7:]
					if token == "" {
						token = c.QueryParam("token")
					}
					tokenInfo, ok := config.TokenManager.ValidateToken(token)
					if ok {
						var user model.User
						if core.DB.Where("uid = ?", tokenInfo.UID).Find(&user).Error == nil {
							cc.AuthUser = &user
							cc.AuthToken = tokenInfo
						}
					}
				}
				// 验证API 请求用户
				if cc.AuthUser == nil {
					cc.AuthUser = validateApiRequest(c)
				}
				if cc.AuthUser == nil {
					return NewResponse(401, "用户未登录")
				}
			}
			return next(cc)
		}
	}
}

func validateApiRequest(c echo.Context) *model.User {
	apiSignature := c.Request().Header.Get("X-Api-Signature")
	if apiSignature == "" {
		return nil
	}
	apiTimeStr := c.Request().Header.Get("X-Api-Time")
	if apiTimeStr == "" {
		return nil
	}
	apiTime, err := strconv.ParseInt(apiTimeStr, 10, 64)
	if err != nil {
		return nil
	}
	//不在有效时间内
	if math.Abs(float64(time.Now().Unix()-apiTime)) > 60*5 {
		return nil
	}
	accessKey := c.Request().Header.Get("X-Api-Key")
	var user model.User
	if err = core.DB.Where("access_key = ?", accessKey).First(&user).Error; err != nil {
		return nil
	}
	hash := md5.Sum([]byte(apiTimeStr + user.SecretKey))
	if apiSignature != hex.EncodeToString(hash[:]) {
		return nil
	}
	return &user
}

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

			var user model.User
			if err := core.DB.Where("uid = ?", apiKey.UID).First(&user).Error; err != nil {
				return proxyUnauthorized(c)
			}
			if user.Status != "active" {
				return proxyUnauthorized(c)
			}

			cc.AuthUser = &user
			cc.APIKeyID = apiKey.KeyID

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
