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
	println("Log Mode: ", lo.If(cfg.LogMode != "", cfg.LogMode).Else("console"))
	println("Log Level: ", lo.If(cfg.LogLevel != "", cfg.LogLevel).Else("info"))
}

func StartServer(cfg *core.Config) {
	service.InitSlog(cfg.LogMode, cfg.LogLevel, filepath.Join(cfg.DataDir, "logs"))
	printWelcome(cfg)

	// 初始化数据库
	db, conn := database.InitDB(cfg.DataDir)
	defer func(conn *sql.DB) {
		err := conn.Close()
		if err != nil {
			slog.Error("Error closing application database connection", "error", err.Error())
		}
	}(conn)
	// 设置全局对象
	core.DB = db
	// 首次启动自动创建默认用户
	database.SeedDefaultUser(db)

	// 初始化分析时序库
	store := database.InitStore(cfg.DataDir)

	webServer, store := web.InitHttpServer(db, store, cfg)
	defer func() {
		store.Close()
		_ = webServer.Close()
	}()
	err := webServer.Start(cfg.HttpAddr)
	if err != nil {
		slog.Error("failed to start web server", "error", err)
		os.Exit(1)
	}
}
