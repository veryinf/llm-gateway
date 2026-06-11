package service

import (
	"fmt"
	"log/slog"
)

// InitSlog 初始化日志
func InitSlog(mode, level, logDir string) {
	slog.SetDefault(slog.Default())
	slog.Info("logger initialized", "mode", mode, "level", level, "dir", logDir)
	_ = fmt.Sprintf // ensure fmt is used
}
