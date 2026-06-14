package command

import (
	"database/sql"
	"log/slog"
	"os"
	"path/filepath"
	"runtime"

	"llm-gateway/internal/core"
	"llm-gateway/internal/database"
	"llm-gateway/internal/service"
	"llm-gateway/internal/web"

	"github.com/labstack/echo/v4"
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
	println("HTTPAddr: ", cfg.HttpAddr)
}

func StartServer(cfg *core.Config) {
	logDir := filepath.Join(cfg.DataDir, "logs")
	service.InitSlog(cfg.LogMode, cfg.LogLevel, logDir)
	printWelcome(cfg)

	db, sqlDB := database.InitDB(cfg.DataDir)
	defer func(sqlDB *sql.DB) {
		err := sqlDB.Close()
		if err != nil {
			slog.Error("Error closing application database connection", "error", err.Error())
		}
	}(sqlDB)
	core.DB = db
	database.SeedDefaultUser(db)

	// 初始化分析时序库
	store := database.InitStore(cfg.DataDir)
	defer func(sqlStore *sql.DB) {
		err := store.Close()
		if err != nil {
			slog.Error("Error closing store database connection", "error", err.Error())
		}
	}(store)

	webServer := web.InitHttpServer(db, store, cfg)
	defer func(webServer *echo.Echo) {
		_ = webServer.Close()
	}(webServer)
	err := webServer.Start(cfg.HttpAddr)
	if err != nil {
		slog.Error("failed to start web server", "error", err)
		os.Exit(1)
	}
}
