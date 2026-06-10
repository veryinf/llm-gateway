package config

import (
	"flag"
	"fmt"
	"time"
)

type Config struct {
	Server    ServerConfig    `mapstructure:"server"`
	Database  DatabaseConfig  `mapstructure:"database"`
	Auth      AuthConfig      `mapstructure:"auth"`
	RateLimit RateLimitConfig `mapstructure:"rate_limit"`
	Stats     StatsConfig     `mapstructure:"stats"`
	Audit     AuditConfig     `mapstructure:"audit"`
}

type ServerConfig struct {
	Port int    `mapstructure:"port"`
	Host string `mapstructure:"host"`
}

type DatabaseConfig struct {
	Path string `mapstructure:"path"`
}

type AuthConfig struct {
	JWTSecret     string `mapstructure:"jwt_secret"`
	APIKeyPrefix  string `mapstructure:"api_key_prefix"`
	AdminPassword string `mapstructure:"admin_password"`
}

func (s *ServerConfig) GetAddr() string {
	if s.Host == "" {
		s.Host = "0.0.0.0"
	}
	return fmt.Sprintf("%s:%d", s.Host, s.Port)
}

type RateLimitConfig struct {
	DefaultQPM int `mapstructure:"default_qpm"`
	GlobalQPM  int `mapstructure:"global_qpm"`
}

type StatsConfig struct {
	BufferSize    int           `mapstructure:"buffer_size"`
	FlushInterval time.Duration `mapstructure:"flush_interval"`
	FlushBatch    int           `mapstructure:"flush_batch"`
}

type AuditConfig struct {
	RetentionDays int `mapstructure:"retention_days"`
}

func ParseFlags() *Config {
	cfg := &Config{}

	flag.IntVar(&cfg.Server.Port, "port", 3001, "server port")
	flag.StringVar(&cfg.Server.Host, "host", "0.0.0.0", "server host")
	flag.StringVar(&cfg.Database.Path, "db-path", "./data/llm_gateway.db", "SQLite database path")

	flag.StringVar(&cfg.Auth.AdminPassword, "admin-password", "", "admin password (empty = skip creation)")
	flag.StringVar(&cfg.Auth.JWTSecret, "jwt-secret", "change-me-in-production", "JWT signing secret")
	flag.StringVar(&cfg.Auth.APIKeyPrefix, "api-key-prefix", "sk-", "API key prefix")

	flag.IntVar(&cfg.RateLimit.DefaultQPM, "default-qpm", 60, "default QPM rate limit per API key")
	flag.IntVar(&cfg.RateLimit.GlobalQPM, "global-qpm", 10000, "global QPM rate limit")

	flag.IntVar(&cfg.Stats.BufferSize, "stats-buffer-size", 1000, "stats buffer size")
	flushInterval := flag.Duration("stats-flush-interval", 5*time.Second, "stats flush interval")
	flag.IntVar(&cfg.Stats.FlushBatch, "stats-flush-batch", 100, "stats flush batch size")

	flag.IntVar(&cfg.Audit.RetentionDays, "audit-retention-days", 90, "audit log retention days")

	flag.Parse()

	cfg.Stats.FlushInterval = *flushInterval

	return cfg
}
