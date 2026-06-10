package middleware

import (
	"crypto/sha256"
	"encoding/hex"
	"strings"
	"time"

	"llm-gateway/internal/cache"
	"llm-gateway/pkg/apierror"
	"llm-gateway/pkg/response"

	"github.com/golang-jwt/jwt/v5"
	"github.com/labstack/echo/v4"
)

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

type AdminClaims struct {
	UserID uint   `json:"user_id"`
	Role   string `json:"role"`
	jwt.RegisteredClaims
}

func AdminAuthMiddleware(jwtSecret string) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			authHeader := c.Request().Header.Get("Authorization")
			if authHeader == "" {
				return response.Error(c, apierror.Unauthorized("missing authorization header"))
			}

			parts := strings.SplitN(authHeader, " ", 2)
			if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") {
				return response.Error(c, apierror.Unauthorized("invalid authorization header format"))
			}

			tokenString := parts[1]

			token, err := jwt.ParseWithClaims(tokenString, &AdminClaims{}, func(token *jwt.Token) (interface{}, error) {
				if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
					return nil, apierror.Unauthorized("unexpected signing method")
				}
				return []byte(jwtSecret), nil
			})

			if err != nil || !token.Valid {
				return response.Error(c, apierror.Unauthorized("invalid or expired token"))
			}

			claims, ok := token.Claims.(*AdminClaims)
			if !ok {
				return response.Error(c, apierror.Unauthorized("invalid token claims"))
			}

			c.Set(CtxKeyUserID, claims.UserID)
			c.Set(CtxKeyIsAdmin, true)

			return next(c)
		}
	}
}
