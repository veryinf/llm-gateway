package database

import (
	"database/sql"
	"fmt"

	"llm-gateway/internal/config"
	"llm-gateway/internal/model"

	_ "github.com/duckdb/duckdb-go/v2"
	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
	gormlogger "gorm.io/gorm/logger"
)

func NewSQLiteDB(cfg *config.DatabaseConfig) (*gorm.DB, error) {
	db, err := gorm.Open(sqlite.Open(cfg.Path), &gorm.Config{
		Logger: gormlogger.Default.LogMode(gormlogger.Info),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to open sqlite database: %w", err)
	}

	if err := db.Exec("PRAGMA journal_mode=WAL").Error; err != nil {
		return nil, fmt.Errorf("failed to enable WAL mode: %w", err)
	}

	if err := db.Exec("PRAGMA foreign_keys=ON").Error; err != nil {
		return nil, fmt.Errorf("failed to enable foreign keys: %w", err)
	}

	if err := db.AutoMigrate(
		&model.User{},
		&model.APIKey{},
		&model.Provider{},
		&model.Model{},
		&model.Config{},
	); err != nil {
		return nil, fmt.Errorf("failed to auto migrate sqlite: %w", err)
	}

	return db, nil
}

func NewDuckDB(path string) (*sql.DB, error) {
	db, err := sql.Open("duckdb", path)
	if err != nil {
		return nil, fmt.Errorf("failed to open duckdb database: %w", err)
	}

	db.SetMaxOpenConns(1)

	if err := initDuckDBSchema(db); err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to init duckdb schema: %w", err)
	}

	return db, nil
}

func initDuckDBSchema(db *sql.DB) error {
	ddl := `
CREATE TABLE IF NOT EXISTS request_logs (
    trace_id        VARCHAR(36),
    user_id         BIGINT,
    api_key_id      BIGINT,
    provider_id     BIGINT,
    model_name      VARCHAR(128),
    is_stream       BOOLEAN,
    prompt_tokens   INTEGER,
    completion_tokens INTEGER,
    total_tokens    INTEGER,
    status_code     INTEGER,
    error_message   VARCHAR(1024),
    latency_ms      BIGINT,
    cost            DOUBLE,
    created_at      TIMESTAMP
);

CREATE TABLE IF NOT EXISTS audit_logs (
    trace_id          VARCHAR(36),
    user_id           BIGINT,
    api_key_id        BIGINT,
    provider_id       BIGINT,
    model_name        VARCHAR(128),
    request_summary   TEXT,
    response_summary  TEXT,
    prompt_tokens     INTEGER,
    completion_tokens INTEGER,
    status_code       INTEGER,
    error_message     VARCHAR(1024),
    latency_ms        BIGINT,
    cost              DOUBLE,
    ip_address        VARCHAR(64),
    user_agent        VARCHAR(512),
    created_at        TIMESTAMP
);
`

	if _, err := db.Exec(ddl); err != nil {
		return fmt.Errorf("failed to create duckdb tables: %w", err)
	}

	return nil
}
