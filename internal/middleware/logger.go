package middleware

import (
	"time"

	"github.com/labstack/echo/v4"
	"go.uber.org/zap"
)

func Logger(logger *zap.Logger) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			start := time.Now()
			path := c.Request().URL.Path
			method := c.Request().Method

			err := next(c)

			latency := time.Since(start)
			statusCode := c.Response().Status
			clientIP := c.RealIP()

			fields := []zap.Field{
				zap.Int("status", statusCode),
				zap.String("method", method),
				zap.String("path", path),
				zap.Duration("latency", latency),
				zap.String("ip", clientIP),
			}

			if err != nil {
				logger.Error("request error", append(fields, zap.String("error", err.Error()))...)
				return err
			}

			if statusCode >= 500 {
				logger.Error("request completed", fields...)
			} else if statusCode >= 400 {
				logger.Warn("request completed", fields...)
			} else {
				logger.Info("request completed", fields...)
			}

			return nil
		}
	}
}
