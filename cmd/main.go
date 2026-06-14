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
				Value:   "./db",
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
			cfg := &core.Config{
				DataDir:  cmd.String("data-dir"),
				HttpAddr: cmd.String("http-addr"),
				LogMode:  cmd.String("log"),
				LogLevel: cmd.String("log-level"),
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
