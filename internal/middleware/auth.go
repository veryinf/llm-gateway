package middleware

import (
	"crypto/md5"
	"crypto/sha256"
	"encoding/hex"
	"math"
	"strconv"
	"strings"
	"time"

	"llm-gateway/internal/cache"
	"llm-gateway/internal/model"
	"llm-gateway/pkg/apierror"
	"llm-gateway/pkg/response"

	"github.com/golang-jwt/jwt/v5"
	"github.com/labstack/echo/v4"
)

// ======================== LLM Gateway API Key Auth ========================

func AuthMiddleware(c *cache.GatewayCache) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(ctx echo.Context) error {
			authHeader := ctx.Request().Header.Get("Authorization")
			if authHeader == "" {
				return response.Error(ctx, apierror.Unauthorized("missing authorization header"))
			}

			parts := strings.SplitN(authHeader, " ", 2)
			if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") {
				return response.Error(ctx, apierror.Unauthorized("invalid authorization header format"))
			}

			apiKey := parts[1]
			if apiKey == "" {
				return response.Error(ctx, apierror.Unauthorized("empty api key"))
			}

			hash := sha256.Sum256([]byte(apiKey))
			keyHash := hex.EncodeToString(hash[:])

			keyRecord := c.GetAPIKey(keyHash)
			if keyRecord == nil {
				return response.Error(ctx, apierror.Unauthorized("invalid api key"))
			}

			if !keyRecord.IsActive {
				return response.Error(ctx, apierror.Forbidden("api key is disabled"))
			}

			if keyRecord.ExpiresAt != nil && keyRecord.ExpiresAt.Before(time.Now()) {
				return response.Error(ctx, apierror.Forbidden("api key has expired"))
			}

			user := c.GetUser(keyRecord.UserID)
			if user != nil && !user.IsActive {
				return response.Error(ctx, apierror.Forbidden("user account is disabled"))
			}

			ctx.Set(CtxKeyAPIKeyID, keyRecord.ID)
			ctx.Set(CtxKeyAPIKey, keyRecord)
			ctx.Set(CtxKeyUserID, keyRecord.UserID)

			return next(ctx)
		}
	}
}

// ======================== Admin JWT + AKSK Auth ========================

type AdminClaims struct {
	UserID uint   `json:"user_id"`
	Role   string `json:"role"`
	jwt.RegisteredClaims
}

// AdminAuthMiddleware 支持两种鉴权方式:
// 1. JWT: Authorization: Bearer <token>
// 2. AKSK: X-Api-Key + X-Api-Time + X-Api-Signature
func AdminAuthMiddleware(jwtSecret string, gwCache *cache.GatewayCache) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			// 尝试 JWT 鉴权
			if userID, ok := tryJWT(c, jwtSecret); ok {
				c.Set(CtxKeyUserID, userID)
				c.Set(CtxKeyIsAdmin, true)
				c.Set(CtxKeyAuthMethod, "jwt")
				return next(c)
			}

			// 尝试 AKSK 鉴权
			if user := validateAKSK(c, gwCache); user != nil {
				c.Set(CtxKeyUserID, user.ID)
				c.Set(CtxKeyIsAdmin, user.Role == model.RoleAdmin)
				c.Set(CtxKeyAuthMethod, "aksk")
				return next(c)
			}

			return response.Error(c, apierror.Unauthorized("authentication required"))
		}
	}
}

func tryJWT(c echo.Context, jwtSecret string) (uint, bool) {
	authHeader := c.Request().Header.Get("Authorization")
	if authHeader == "" {
		return 0, false
	}

	parts := strings.SplitN(authHeader, " ", 2)
	if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") {
		return 0, false
	}

	token, err := jwt.ParseWithClaims(parts[1], &AdminClaims{}, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, apierror.Unauthorized("unexpected signing method")
		}
		return []byte(jwtSecret), nil
	})

	if err != nil || !token.Valid {
		return 0, false
	}

	claims, ok := token.Claims.(*AdminClaims)
	if !ok {
		return 0, false
	}

	return claims.UserID, true
}

// validateAKSK 验证 API 签名鉴权 (参考 link-engine-general)
// Headers: X-Api-Key, X-Api-Time (unix seconds), X-Api-Signature (MD5)
func validateAKSK(c echo.Context, gwCache *cache.GatewayCache) *model.User {
	accessKey := c.Request().Header.Get("X-Api-Key")
	if accessKey == "" {
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

	// 时间窗口校验: 5 分钟
	if math.Abs(float64(time.Now().Unix()-apiTime)) > 60*5 {
		return nil
	}

	user := gwCache.GetUserByAccessKey(accessKey)
	if user == nil {
		return nil
	}

	apiSignature := c.Request().Header.Get("X-Api-Signature")
	if apiSignature == "" {
		return nil
	}

	// 验证签名: MD5(apiTimeStr + user.SecretKey)
	hash := md5.Sum([]byte(apiTimeStr + user.SecretKey))
	if apiSignature != hex.EncodeToString(hash[:]) {
		return nil
	}

	return user
}
