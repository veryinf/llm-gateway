package command

import (
	"io/fs"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"runtime"

	"llm-gateway/internal/core"
	"llm-gateway/internal/database"
	"llm-gateway/internal/provider"
	"llm-gateway/internal/router"
	"llm-gateway/internal/service"
	"llm-gateway/internal/web"

	"github.com/samber/lo"
)

func PrintVersion(build core.BuildInfo) {
	println("Build Version: ", build.Version)
	println("Build Time: ", build.Time)
	isDev := build.Env == "development" || build.Env == "dev"
	println("Build Env: ", lo.If(isDev, "development").Else("production"))
	println("OS: ", runtime.GOOS)
	println("Arch: ", runtime.GOARCH)
}

func printWelcome(cfg *core.Config) {
	println("Welcome to LLM Gateway")
	PrintVersion(cfg.Build)
	println("DataDir: ", cfg.DataDir)
	println("HTTPAddr: ", cfg.HTTPAddr)
	println("DefaultQPM: ", cfg.DefaultQPM)
	println("GlobalQPM: ", cfg.GlobalQPM)
}

func StartServer(cfg *core.Config) {
	logDir := filepath.Join(cfg.DataDir, "logs")
	service.InitSlog("console", cfg.Build.Env, logDir)
	printWelcome(cfg)

	db, sqlDB := database.InitDB(cfg.DataDir)
	defer sqlDB.Close()
	core.DB = db

	database.SeedDefaultAdmin(db, cfg.AdminPassword)

	if err := database.RunMigrations(sqlDB); err != nil {
		slog.Warn("failed to run migrations", "error", err)
	}

	registry := provider.NewRegistry()
	modelRouter := router.NewModelRouter(registry)
	providerSvc := service.NewProviderService(db, registry, modelRouter)
	if err := providerSvc.LoadProvidersFromDB(); err != nil {
		slog.Warn("failed to load providers from DB", "error", err)
	}

	statsSvc := service.NewStatsService(sqlDB, cfg.StatsBufferSize)
	statsSvc.Start(cfg.StatsFlushInterval, cfg.StatsFlushBatch)
	defer statsSvc.Stop()

	chunkSvc := service.NewRequestChunkService(sqlDB, cfg.StatsBufferSize)
	chunkSvc.Start(cfg.StatsFlushInterval, cfg.StatsFlushBatch)
	defer chunkSvc.Stop()

	_ = fs.Sub // ensure embed is available

	webServer := web.InitHttpServer(db, sqlDB, registry, modelRouter, providerSvc, statsSvc, chunkSvc, cfg)
	defer webServer.Close()
	err := webServer.Start(cfg.HTTPAddr)
	if err != nil {
		slog.Error("failed to start web server", "error", err)
		os.Exit(1)
	}
}

// serveFrontend 返回前端静态文件服务，优先从 embed 读取，其次从磁盘读取
func serveFrontend() http.Handler {
	// 尝试从 embed 读取
	if subFS, err := fs.Sub(nil, ""); err == nil {
		_ = subFS
	}
	// 尝试从磁盘读取 web/dist
	if _, err := os.Stat("web/dist/index.html"); err == nil {
		slog.Info("serving frontend from web/dist on disk")
		return http.FileServer(http.Dir("web/dist"))
	}
	slog.Info("frontend not built, admin UI not available")
	return nil
}
