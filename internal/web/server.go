package web

import (
	"context"
	"io/fs"
	"strings"

	llmgateway "llm-gateway"
	"llm-gateway/internal/core"
	"llm-gateway/internal/database"
	"llm-gateway/internal/service"
	"llm-gateway/internal/web/common"
	"llm-gateway/internal/web/handlers"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"

	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"gorm.io/gorm"
)

var (
	apiBizPrefix = "/api"
	// 静态文件中间件跳过的路径前缀
	staticSkipPrefixes = []string{"/api/", "/v1/", "/anthropic/"}
)

func InitHttpServer(db *gorm.DB, store *database.Store, cfg *core.Config) (*echo.Echo, *database.Store) {
	tokenManager := common.NewTokenManager()

	e := echo.New()
	e.Debug = cfg.IsDevelopment()
	e.HideBanner = true
	e.HTTPErrorHandler = common.LeErrorHandler

	// 静态文件中间件放在认证之前，前端页面不需要登录
	// 优先检测外置资源目录 dataDir/static，不存在则用嵌入资源
	var staticFS fs.FS
	staticPath := filepath.Join(cfg.DataDir, "static")
	if _, err := os.Stat(staticPath); err == nil {
		staticFS = os.DirFS(staticPath)
		slog.Info("serving frontend from external static", "path", staticPath)
	} else {
		subFS, _ := fs.Sub(llmgateway.StaticFS, "static")
		staticFS = subFS
	}
	e.Use(middleware.StaticWithConfig(middleware.StaticConfig{
		Skipper: func(c echo.Context) bool {
			uri := c.Request().RequestURI
			for _, prefix := range staticSkipPrefixes {
				if strings.HasPrefix(uri, prefix) {
					return true
				}
			}
			return false
		},
		HTML5:      true,
		Filesystem: http.FS(staticFS),
	}))
	// 使用全局 LogService 的 JSON Handler
	accessLogger := slog.New(service.DefaultLogService.JSONHandler())
	e.Use(middleware.RequestLoggerWithConfig(middleware.RequestLoggerConfig{
		LogStatus:   true,
		LogURI:      true,
		LogMethod:   true,
		HandleError: false,
		LogValuesFunc: func(c echo.Context, v middleware.RequestLoggerValues) error {
			accessLogger.LogAttrs(context.Background(), slog.LevelInfo, "REQUEST",
				slog.String("method", v.Method),
				slog.String("uri", v.URI),
				slog.Int("status", v.Status),
			)
			return nil
		},
	}))

	// 生产环境
	if !cfg.IsDevelopment() {
		e.Use(middleware.Recover())
	}

	base := common.BaseHandler{DB: db, Store: store, TokenManager: tokenManager, Config: cfg}

	// 公共接口
	bizApi := e.Group(apiBizPrefix)
	bizApi.Use(common.LeMiddleware(common.LeMiddlewareConfig{
		IgnorePaths: []string{
			apiBizPrefix + "/auth/login",
			apiBizPrefix + "/health",
			apiBizPrefix + "/health/ready",
		},
		TokenManager: tokenManager,
	}))
	(&handlers.AuthHandler{BaseHandler: base}).RegisterRoutes(bizApi)
	(&handlers.ProfileHandler{BaseHandler: base}).RegisterRoutes(bizApi)
	(&handlers.ConfigHandler{BaseHandler: base}).RegisterRoutes(bizApi)
	(&handlers.UserHandler{BaseHandler: base}).RegisterRoutes(bizApi)
	(&handlers.UserKeyHandler{BaseHandler: base}).RegisterRoutes(bizApi)
	(&handlers.UserModelHandler{BaseHandler: base}).RegisterRoutes(bizApi)
	(&handlers.UserModelRouterHandler{BaseHandler: base}).RegisterRoutes(bizApi)
	(&handlers.ProviderHandler{BaseHandler: base}).RegisterRoutes(bizApi)
	(&handlers.ProviderModelHandler{BaseHandler: base}).RegisterRoutes(bizApi)
	(&handlers.RequestLogHandler{BaseHandler: base}).RegisterRoutes(bizApi)
	// Health check
	(&handlers.HealthHandler{BaseHandler: base}).RegisterRoutes(bizApi)

	// LLM Gateway API — uses ProxyMiddleware (sk- API Key)
	gatewayBase := common.GatewayBase{
		BaseHandler:   base,
		RouterService: service.NewRouterService(db),
	}
	v1 := e.Group("/v1")
	v1.Use(common.ProxyMiddleware())
	(&handlers.OpenAIGatewayHandler{GatewayBase: gatewayBase}).RegisterRoutes(v1)
	anthropic := e.Group("/anthropic/v1")
	anthropic.Use(common.ProxyMiddleware())
	(&handlers.AnthropicGatewayHandler{GatewayBase: gatewayBase}).RegisterRoutes(anthropic)
	return e, store
}
