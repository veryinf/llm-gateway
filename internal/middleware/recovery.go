package middleware

import (
	"net/http"
	"runtime/debug"

	"github.com/labstack/echo/v4"
	"go.uber.org/zap"
)

func Recovery(logger *zap.Logger) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			defer func() {
				if r := recover(); r != nil {
					logger.Error("panic recovered",
						zap.Any("panic", r),
						zap.String("stack", string(debug.Stack())),
					)
					_ = c.JSON(http.StatusInternalServerError, map[string]interface{}{
						"code": 50000,
						"msg":  "internal server error",
					})
				}
			}()
			return next(c)
		}
	}
}
