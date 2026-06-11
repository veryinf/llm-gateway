package core

import "time"

// BuildInfo 构建信息
type BuildInfo struct {
	Env     string
	Time    string
	Version string
}

// Config 静态配置，启动后不变
type Config struct {
	DataDir           string
	HTTPAddr          string
	AdminPassword     string
	APIKeyPrefix      string
	DefaultQPM        int
	GlobalQPM         int
	StatsBufferSize   int
	StatsFlushInterval time.Duration
	StatsFlushBatch        int
	RequestLogRetentionDays int
	Build                  BuildInfo
	StartTime         time.Time
}

func (c *Config) IsDevelopment() bool {
	return c.Build.Env == "development" || c.Build.Env == "dev"
}

// 构建变量，通过 -ldflags 注入
var (
	BuildEnv     string
	BuildTime    string
	BuildVersion string
)
