package main

import (
	"context"
	"llm-gateway/cmd/command"
	"llm-gateway/internal/core"
	"log/slog"
	"os"
	"time"

	"github.com/urfave/cli/v3"
)

func main() {
	cli.VersionPrinter = func(cmd *cli.Command) {
		command.PrintVersion(core.BuildInfo{
			Env:     core.BuildEnv,
			Time:    core.BuildTime,
			Version: core.BuildVersion,
		})
	}

	app := &cli.Command{
		Name:    "llm-gateway",
		Usage:   "Unified LLM API Gateway with multi-provider support",
		Version: core.BuildVersion,
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:    "data-dir",
				Value:   "./data",
				Usage:   "specify the data directory",
				Sources: cli.EnvVars("DATA_DIR"),
			},
			&cli.StringFlag{
				Name:    "http-addr",
				Value:   ":3001",
				Usage:   "specify the http server address",
				Sources: cli.EnvVars("HTTP_ADDR"),
			},
			&cli.StringFlag{
				Name:    "admin-password",
				Value:   "",
				Usage:   "admin password (empty = skip creation)",
				Sources: cli.EnvVars("ADMIN_PASSWORD"),
			},
			&cli.StringFlag{
				Name:    "api-key-prefix",
				Value:   "sk-",
				Usage:   "API key prefix",
				Sources: cli.EnvVars("API_KEY_PREFIX"),
			},
			&cli.IntFlag{
				Name:    "default-qpm",
				Value:   60,
				Usage:   "default QPM rate limit per API key",
				Sources: cli.EnvVars("DEFAULT_QPM"),
			},
			&cli.IntFlag{
				Name:    "global-qpm",
				Value:   10000,
				Usage:   "global QPM rate limit",
				Sources: cli.EnvVars("GLOBAL_QPM"),
			},
			&cli.IntFlag{
				Name:    "stats-buffer-size",
				Value:   1000,
				Usage:   "stats buffer size",
				Sources: cli.EnvVars("STATS_BUFFER_SIZE"),
			},
			&cli.StringFlag{
				Name:    "stats-flush-interval",
				Value:   "5s",
				Usage:   "stats flush interval",
				Sources: cli.EnvVars("STATS_FLUSH_INTERVAL"),
			},
			&cli.IntFlag{
				Name:    "stats-flush-batch",
				Value:   100,
				Usage:   "stats flush batch size",
				Sources: cli.EnvVars("STATS_FLUSH_BATCH"),
			},
			&cli.IntFlag{
				Name:    "request-log-retention-days",
				Value:   90,
				Usage:   "request log retention days",
				Sources: cli.EnvVars("REQUEST_LOG_RETENTION_DAYS"),
			},
			&cli.StringFlag{
				Name:    "log",
				Value:   "console",
				Usage:   "log output target: console | file | both",
				Sources: cli.EnvVars("LOG"),
			},
			&cli.StringFlag{
				Name:    "log-level",
				Value:   "info",
				Usage:   "log level: debug | info | warn | error",
				Sources: cli.EnvVars("LOG_LEVEL"),
			},
		},
		Action: func(ctx context.Context, cmd *cli.Command) error {
			flushInterval, _ := time.ParseDuration(cmd.String("stats-flush-interval"))

			cfg := &core.Config{
				DataDir:                 cmd.String("data-dir"),
				HTTPAddr:                cmd.String("http-addr"),
				AdminPassword:           cmd.String("admin-password"),
				APIKeyPrefix:            cmd.String("api-key-prefix"),
				DefaultQPM:              int(cmd.Int("default-qpm")),
				GlobalQPM:               int(cmd.Int("global-qpm")),
				StatsBufferSize:         int(cmd.Int("stats-buffer-size")),
				StatsFlushInterval:      flushInterval,
				StatsFlushBatch:         int(cmd.Int("stats-flush-batch")),
				RequestLogRetentionDays: int(cmd.Int("request-log-retention-days")),
				Build: core.BuildInfo{
					Env:     core.BuildEnv,
					Time:    core.BuildTime,
					Version: core.BuildVersion,
				},
				StartTime: time.Now(),
			}
			command.StartServer(cfg)

			return nil
		},
	}

	if err := app.Run(context.Background(), os.Args); err != nil {
		slog.Error("Error", "error", err)
	}
}
