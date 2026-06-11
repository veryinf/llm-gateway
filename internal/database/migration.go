package database

import (
	"database/sql"
	"embed"
	"fmt"
	"io/fs"
	"sort"
	"strings"
)

//go:embed migrations/*.sql
var embedMigrations embed.FS

// initVersionTable 创建迁移版本追踪表（不使用 AUTO_INCREMENT，兼容 DuckDB）
func initVersionTable(db *sql.DB) error {
	_, err := db.Exec(`
		CREATE TABLE IF NOT EXISTS schema_migrations (
			id        INTEGER,
			name      VARCHAR(256),
			applied_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			PRIMARY KEY (id)
		)
	`)
	return err
}

// getAppliedMigrations 查询已执行的迁移版本
func getAppliedMigrations(db *sql.DB) (map[int64]bool, error) {
	rows, err := db.Query("SELECT id FROM schema_migrations")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	applied := make(map[int64]bool)
	for rows.Next() {
		var id int64
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		applied[id] = true
	}
	return applied, nil
}

// RunMigrations 扫描嵌入的 SQL 文件，按文件名排序，执行未应用的迁移
func RunMigrations(db *sql.DB) error {
	if err := initVersionTable(db); err != nil {
		return fmt.Errorf("failed to create schema_migrations: %w", err)
	}

	applied, err := getAppliedMigrations(db)
	if err != nil {
		return fmt.Errorf("failed to get applied migrations: %w", err)
	}

	entries, err := fs.ReadDir(embedMigrations, "migrations")
	if err != nil {
		return fmt.Errorf("failed to read migrations: %w", err)
	}

	// 按文件名排序确保顺序执行
	sort.Slice(entries, func(i, j int) bool {
		return entries[i].Name() < entries[j].Name()
	})

	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".sql") {
			continue
		}

		// 从文件名解析版本号：00001_xxx.sql → 1
		version, err := parseVersion(entry.Name())
		if err != nil {
			continue // 跳过非标准命名文件
		}

		if applied[version] {
			continue // 已执行，跳过
		}

		content, err := fs.ReadFile(embedMigrations, "migrations/"+entry.Name())
		if err != nil {
			return fmt.Errorf("failed to read migration %s: %w", entry.Name(), err)
		}

		// 提取 -- +goose Up 部分的 SQL
		sqlContent := extractUpSQL(string(content))
		statements := splitSQL(sqlContent)

		tx, err := db.Begin()
		if err != nil {
			return fmt.Errorf("failed to begin transaction for %s: %w", entry.Name(), err)
		}

		for _, stmt := range statements {
			if stmt == "" {
				continue
			}
			if _, err := tx.Exec(stmt); err != nil {
				tx.Rollback()
				return fmt.Errorf("failed to execute migration %s: %w\nSQL: %s", entry.Name(), err, stmt)
			}
		}

		// 记录已执行
		if _, err := tx.Exec("INSERT INTO schema_migrations (id, name) VALUES (?, ?)", version, entry.Name()); err != nil {
			tx.Rollback()
			return fmt.Errorf("failed to record migration %s: %w", entry.Name(), err)
		}

		if err := tx.Commit(); err != nil {
			return fmt.Errorf("failed to commit migration %s: %w", entry.Name(), err)
		}

		fmt.Printf("[goose] applied migration: %s\n", entry.Name())
	}

	return nil
}

// extractUpSQL 从 goose 风格的 SQL 文件中提取 -- +goose Up 和 -- +goose Down 之间的部分
func extractUpSQL(content string) string {
	lines := strings.Split(content, "\n")
	var result []string
	inUpSection := false

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)

		// 检测 -- +goose Up 标记
		if strings.HasPrefix(trimmed, "-- +goose Up") {
			inUpSection = true
			continue
		}

		// 检测 -- +goose Down 标记（停止收集）
		if strings.HasPrefix(trimmed, "-- +goose Down") {
			inUpSection = false
			continue
		}

		if inUpSection {
			result = append(result, line)
		}
	}

	// 如果没有 goose 标记，返回原始内容
	if len(result) == 0 {
		return content
	}

	return strings.Join(result, "\n")
}

// parseVersion 从文件名解析版本号：00001_init.sql → 1
func parseVersion(filename string) (int64, error) {
	parts := strings.SplitN(filename, "_", 2)
	if len(parts) < 2 {
		return 0, fmt.Errorf("invalid migration filename: %s", filename)
	}
	var version int64
	for _, c := range parts[0] {
		if c < '0' || c > '9' {
			return 0, fmt.Errorf("invalid version in filename: %s", filename)
		}
		version = version*10 + int64(c-'0')
	}
	return version, nil
}

// splitSQL 按分号分割 SQL，过滤空语句
func splitSQL(content string) []string {
	var statements []string
	for _, stmt := range strings.Split(content, ";") {
		stmt = strings.TrimSpace(stmt)
		if stmt != "" {
			statements = append(statements, stmt)
		}
	}
	return statements
}
