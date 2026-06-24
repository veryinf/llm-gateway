package database

import (
	"database/sql"
	"embed"
	"fmt"
	"io/fs"
	"log/slog"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/jmoiron/sqlx"
	_ "github.com/duckdb/duckdb-go/v2"
)

//go:embed migrations/duckdb/*.sql
var embedMigrations embed.FS

const (
	StatsBufferSize    = 1000
	StatsFlushInterval = 5 * time.Second
	StatsFlushBatch    = 100
)

func InitStore(dataDir string) *sqlx.DB {
	dbPath := filepath.Join(dataDir, "store.duckdb")
	db, err := sqlx.Open("duckdb", dbPath)
	if err != nil {
		slog.Error("failed to open duckdb database", "error", err)
		os.Exit(1)
	}
	if err := db.Ping(); err != nil {
		slog.Error("failed to ping duckdb database", "error", err)
		os.Exit(1)
	}

	// 配置连接池
	db.SetMaxOpenConns(10)  // 最大打开连接数
	db.SetMaxIdleConns(5)   // 最大空闲连接数

	if err := runMigrations(db.DB); err != nil {
		slog.Error("failed to run duckdb migrations", "error", err)
		os.Exit(1)
	}

	slog.Info("duckdb initialized", "path", dataDir)
	return db
}

func runMigrations(db *sql.DB) error {
	entries, err := fs.ReadDir(embedMigrations, "migrations/duckdb")
	if err != nil {
		return fmt.Errorf("failed to read migrations: %w", err)
	}

	sort.Slice(entries, func(i, j int) bool {
		return entries[i].Name() < entries[j].Name()
	})

	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".sql") {
			continue
		}

		content, err := fs.ReadFile(embedMigrations, "migrations/duckdb/"+entry.Name())
		if err != nil {
			return fmt.Errorf("failed to read migration %s: %w", entry.Name(), err)
		}

		for _, stmt := range strings.Split(string(content), ";") {
			stmt = strings.TrimSpace(stmt)
			if stmt == "" {
				continue
			}
			if _, err := db.Exec(stmt); err != nil {
				return fmt.Errorf("failed to execute migration %s: %w\nSQL: %s", entry.Name(), err, stmt)
			}
		}

		slog.Info("applied duckdb migration", "file", entry.Name())
	}

	return nil
}
