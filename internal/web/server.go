package web

import (
	"database/sql"
	"io/fs"
	"net/http"
	"sync/atomic"

	llmgateway "llm-gateway"
	"llm-gateway/internal/core"
	"llm-gateway/internal/router"
	"llm-gateway/internal/service"
	"llm-gateway/internal/web/common"
	"llm-gateway/internal/web/handlers"

	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"gorm.io/gorm"
)

var (
	adminPrefix = "/api/admin"
	statsPrefix = "/api"
	v1Prefix    = "/v1"
)

func InitHttpServer(db *gorm.DB, sqlDB *sql.DB, registry interface{}, modelRouter interface{}, providerSvc *service.ProviderService, statsSvc *service.StatsService, chunkSvc *service.RequestChunkService, cfg *core.Config) *echo.Echo {
	tokenManager := common.NewTokenManager()

	e := echo.New()
	e.HideBanner = true
	e.HTTPErrorHandler = common.LeErrorHandler

	// 静态文件中间件放在认证之前，前端页面不需要登录
	staticFS, _ := fs.Sub(llmgateway.StaticFS, "static")
	e.Use(middleware.StaticWithConfig(middleware.StaticConfig{
		Skipper: func(c echo.Context) bool {
			uri := c.Request().RequestURI
			return len(uri) >= 5 && uri[:5] == "/api/"
		},
		HTML5:      true,
		Filesystem: http.FS(staticFS),
	}))

	e.Use(middleware.Logger())

	if !cfg.IsDevelopment() {
		e.Use(middleware.Recover())
	}

	e.Use(middleware.CORS())

	base := common.BaseHandler{DB: db, DuckDB: sqlDB, Config: cfg}

	// Admin API (login, profile, user/provider/model/config CRUD) — uses LeMiddleware (JWT token)
	adminApi := e.Group(adminPrefix)
	adminApi.Use(common.LeMiddleware(common.LeMiddlewareConfig{
		IgnorePaths: []string{
			adminPrefix + "/login",
		},
		TokenManager: tokenManager,
	}))
	(&handlers.AuthHandler{BaseHandler: base, TokenManager: tokenManager}).RegisterRoutes(adminApi)
	(&handlers.ProfileHandler{BaseHandler: base}).RegisterRoutes(adminApi)
	(&handlers.AdminHandler{BaseHandler: base, ProviderSvc: providerSvc}).RegisterRoutes(adminApi)

	// LLM Gateway API — uses ProxyMiddleware (sk- API Key)
	logDetail := &atomic.Bool{}
	logDetail.Store(true)
	v1 := e.Group(v1Prefix)
	v1.Use(common.ProxyMiddleware())
	(&handlers.GatewayHandler{BaseHandler: base, ModelRouter: modelRouter.(*router.ModelRouter), StatsSvc: statsSvc, ChunkSvc: chunkSvc, LogDetail: logDetail}).RegisterRoutes(v1)

	anthropic := e.Group("/anthropic")
	anthropic.Use(common.ProxyMiddleware())
	(&handlers.GatewayHandler{BaseHandler: base, ModelRouter: modelRouter.(*router.ModelRouter), StatsSvc: statsSvc, ChunkSvc: chunkSvc, LogDetail: logDetail}).RegisterRoutes(anthropic)

	// Stats + Request Logs — uses LeMiddleware (JWT token)
	apiGroup := e.Group(statsPrefix)
	apiGroup.Use(common.LeMiddleware(common.LeMiddlewareConfig{
		TokenManager: tokenManager,
	}))
	(&handlers.StatsHandler{BaseHandler: base}).RegisterRoutes(apiGroup)
	(&handlers.RequestLogHandler{BaseHandler: base}).RegisterRoutes(apiGroup)

	// Health check
	e.GET("/health", func(c echo.Context) error {
		return c.JSON(http.StatusOK, map[string]interface{}{"status": "ok"})
	})
	e.GET("/health/ready", func(c echo.Context) error {
		sqlDB, _ := db.DB()
		if err := sqlDB.Ping(); err != nil {
			return c.JSON(http.StatusServiceUnavailable, map[string]interface{}{"status": "not ready"})
		}
		return c.JSON(http.StatusOK, map[string]interface{}{"status": "ready"})
	})

	return e
}
