package main

import (
	"context"
	"embed"
	"io/fs"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"llm-gateway/internal/cache"
	"llm-gateway/internal/config"
	"llm-gateway/internal/database"
	"llm-gateway/internal/handler"
	"llm-gateway/internal/middleware"
	"llm-gateway/internal/provider"
	"llm-gateway/internal/router"
	"llm-gateway/internal/service"
	"llm-gateway/internal/worker"

	"github.com/labstack/echo/v4"
	"go.uber.org/zap"
)

//go:embed dist/*
var frontendFS embed.FS

func main() {
	cfg := config.ParseFlags()

	// Initialize logger
	logger, err := zap.NewProduction()
	if err != nil {
		log.Fatalf("failed to create logger: %v", err)
	}
	defer logger.Sync()

	// Initialize SQLite (basic data: users, apikeys, providers, models)
	sqliteDB, err := database.NewSQLiteDB(&cfg.Database)
	if err != nil {
		logger.Fatal("failed to init sqlite database", zap.Error(err))
	}
	logger.Info("sqlite initialized", zap.String("path", cfg.Database.Path))

	// Initialize DuckDB (analytics data: request_logs, audit_logs)
	duckDB, err := database.NewDuckDB(cfg.Database.DuckDBPath)
	if err != nil {
		logger.Fatal("failed to init duckdb database", zap.Error(err))
	}
	logger.Info("duckdb initialized", zap.String("path", cfg.Database.DuckDBPath))
	defer duckDB.Close()

	// Initialize cache (loads from SQLite)
	gwCache := cache.New(sqliteDB, 30*time.Second)
	gwCache.Start()
	defer gwCache.Stop()
	logger.Info("cache started")

	// Create services (SQLite-based)
	userSvc := service.NewUserService(sqliteDB, cfg.Auth.JWTSecret)
	apiKeySvc := service.NewAPIKeyService(sqliteDB)

	// Create services (DuckDB-based)
	statsSvc := service.NewStatsService(duckDB, cfg.Stats.BufferSize)
	auditSvc := service.NewAuditService(duckDB, cfg.Stats.BufferSize)

	// Create provider registry and model router
	registry := provider.NewRegistry()
	modelRouter := router.NewModelRouter(registry)

	// Load providers from SQLite
	providerSvc := service.NewProviderService(sqliteDB, registry, modelRouter)
	if err := providerSvc.LoadProvidersFromDB(); err != nil {
		logger.Warn("failed to load providers from DB", zap.Error(err))
	}

	// Start background workers
	statsSvc.Start(cfg.Stats.FlushInterval, cfg.Stats.FlushBatch)
	auditSvc.Start(cfg.Stats.FlushInterval, cfg.Stats.FlushBatch)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	cleanupWorker := worker.NewCleanupWorker(duckDB)
	go cleanupWorker.Start(ctx, cfg.Audit.RetentionDays, cfg.Audit.StatsRetentionDays)

	// Create default admin user
	if cfg.Auth.AdminPassword != "" {
		if _, err := userSvc.CreateDefaultAdmin("admin", cfg.Auth.AdminPassword); err != nil {
			logger.Error("failed to create default admin", zap.Error(err))
		} else {
			logger.Info("default admin user initialized")
		}
	} else {
		logger.Warn("admin_password not set, default admin not created")
	}

	// Create handlers
	gatewayHandler := handler.NewGatewayHandler(modelRouter, statsSvc, auditSvc)
	adminHandler := handler.NewAdminHandler(userSvc, apiKeySvc, providerSvc, sqliteDB)
	statsHandler := handler.NewStatsHandler(duckDB, sqliteDB)
	auditHandler := handler.NewAuditHandler(duckDB)

	// Setup Echo
	e := echo.New()
	e.HideBanner = true
	e.HidePort = true

	// Global middleware
	e.Use(middleware.Recovery(logger))
	e.Use(middleware.Logger(logger))
	e.Use(middleware.CORS())

	// Health check
	e.GET("/health", func(c echo.Context) error {
		return c.JSON(http.StatusOK, map[string]interface{}{"status": "ok"})
	})
	e.GET("/health/ready", func(c echo.Context) error {
		sqlDB, _ := sqliteDB.DB()
		if err := sqlDB.Ping(); err != nil {
			return c.JSON(http.StatusServiceUnavailable, map[string]interface{}{"status": "not ready", "error": "sqlite: " + err.Error()})
		}
		if err := duckDB.Ping(); err != nil {
			return c.JSON(http.StatusServiceUnavailable, map[string]interface{}{"status": "not ready", "error": "duckdb: " + err.Error()})
		}
		return c.JSON(http.StatusOK, map[string]interface{}{"status": "ready"})
	})

	// Serve admin frontend from embedded dist or disk (dev fallback)
	frontendFileServer := func() http.Handler {
		if subFS, err := fs.Sub(frontendFS, "dist"); err == nil {
			if _, err := fs.Stat(subFS, "index.html"); err == nil {
				logger.Info("serving embedded frontend from binary")
				return http.FileServer(http.FS(subFS))
			}
		}
		if _, err := os.Stat("web/dist/index.html"); err == nil {
			logger.Info("serving frontend from web/dist on disk")
			return http.FileServer(http.Dir("web/dist"))
		}
		logger.Info("frontend not built, admin UI not available (run: pnpm build)")
		return nil
	}()

	if frontendFileServer != nil {
		adminGroup := e.Group("/admin")
		adminGroup.GET("/*", echo.WrapHandler(frontendFileServer))
		adminGroup.GET("", func(c echo.Context) error {
			return c.Redirect(http.StatusMovedPermanently, "/admin/")
		})
	}

	// LLM Gateway API (API Key auth)
	v1 := e.Group("/v1")
	v1.Use(middleware.AuthMiddleware(gwCache))
	{
		v1.POST("/chat/completions", gatewayHandler.HandleChatCompletion)
		v1.POST("/messages", gatewayHandler.HandleMessages)
		v1.GET("/models", gatewayHandler.HandleListModels)
	}

	// Admin API (JWT auth)
	admin := e.Group("/api/admin")
	{
		admin.POST("/login", adminHandler.Login)

		protected := admin.Group("")
		protected.Use(middleware.AdminAuthMiddleware(cfg.Auth.JWTSecret, gwCache))
		{
			// Profile
			protected.GET("/profile", adminHandler.Profile)

			// Users
			protected.GET("/users", adminHandler.ListUsers)
			protected.POST("/users", adminHandler.CreateUser)
			protected.PUT("/users/:id", adminHandler.UpdateUser)
			protected.DELETE("/users/:id", adminHandler.DeleteUser)
			protected.GET("/users/:id/api-keys", adminHandler.ListAPIKeys)
			protected.POST("/users/:id/api-keys", adminHandler.CreateAPIKey)
			protected.DELETE("/users/:id/api-keys/:kid", adminHandler.DeleteAPIKey)

			// AKSK
			protected.POST("/users/:id/aksk", adminHandler.GenerateAKSK)
			protected.GET("/users/:id/aksk", adminHandler.GetAKSK)

			// API Keys (global)
			protected.GET("/api-keys", adminHandler.ListAllAPIKeys)
			protected.DELETE("/api-keys/:id", adminHandler.DeleteAPIKeyByID)
			protected.PUT("/api-keys/:id/toggle", adminHandler.ToggleAPIKey)

			// Providers
			protected.GET("/providers", adminHandler.ListProviders)
			protected.POST("/providers", adminHandler.CreateProvider)
			protected.PUT("/providers/:id", adminHandler.UpdateProvider)
			protected.DELETE("/providers/:id", adminHandler.DeleteProvider)
			protected.PUT("/providers/:id/toggle", adminHandler.ToggleProvider)

			// Models
			protected.GET("/models", adminHandler.ListModels)
			protected.POST("/models", adminHandler.CreateModel)
			protected.PUT("/models/:id", adminHandler.UpdateModel)

			// Configs
			protected.GET("/configs", adminHandler.ListConfigs)
			protected.PUT("/configs", adminHandler.UpdateConfig)
		}
	}

	// Stats API (JWT + AKSK auth)
	stats := e.Group("/api")
	stats.Use(middleware.AdminAuthMiddleware(cfg.Auth.JWTSecret, gwCache))
	{
		stats.GET("/stats/tokens", statsHandler.TokenStats)
		stats.GET("/stats/requests", statsHandler.RequestStats)
		stats.GET("/stats/costs", statsHandler.CostStats)
		stats.GET("/stats/behavior", statsHandler.BehaviorStats)
		stats.GET("/dashboard/overview", statsHandler.DashboardOverview)
	}

	// Audit API (JWT + AKSK auth)
	audit := e.Group("/api")
	audit.Use(middleware.AdminAuthMiddleware(cfg.Auth.JWTSecret, gwCache))
	{
		audit.GET("/audit/logs", auditHandler.ListAuditLogs)
		audit.GET("/audit/logs/:trace_id", auditHandler.GetAuditLogByTrace)
	}

	// Start server
	addr := cfg.Server.GetAddr()
	go func() {
		logger.Info("server starting", zap.String("addr", addr))
		if err := e.Start(addr); err != nil && err != http.ErrServerClosed {
			logger.Fatal("server failed", zap.Error(err))
		}
	}()

	// Graceful shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	logger.Info("shutting down server...")

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer shutdownCancel()

	if err := e.Shutdown(shutdownCtx); err != nil {
		logger.Error("server forced to shutdown", zap.Error(err))
	}

	// Stop workers
	statsSvc.Stop()
	auditSvc.Stop()
	cancel()

	logger.Info("server stopped")
}
