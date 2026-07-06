package database

import (
	"context"
	"database/sql"
	"embed"
	"fmt"
	"io/fs"
	"log/slog"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	"llm-gateway/internal/model"

	"github.com/duckdb/duckdb-go/v2"
	"github.com/jmoiron/sqlx"
)

//go:embed migrations/duckdb/*.sql
var embedMigrations embed.FS

const (
	FlushInterval = 5 * time.Second // 刷新间隔
	FlushSize     = 1000            // 触发刷新的记录数
)

// Store DuckDB 存储引擎
type Store struct {
	db *sqlx.DB

	logs    []*model.RequestLog
	details []*model.RequestDetail
	chunks  []*model.RequestChunk
	mu      sync.Mutex
	cancel  context.CancelFunc
}

// InitStore 初始化 DuckDB 存储
func InitStore(dataDir string) *Store {
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
	db.SetMaxOpenConns(10)
	db.SetMaxIdleConns(5)

	if err := runMigrations(db.DB); err != nil {
		slog.Error("failed to run duckdb migrations", "error", err)
		os.Exit(1)
	}

	ctx, cancel := context.WithCancel(context.Background())
	store := &Store{
		db:     db,
		cancel: cancel,
	}
	go store.flushLoop(ctx)

	slog.Info("duckdb initialized", "path", dataDir)
	return store
}

// Close 停止后台刷新并关闭数据库
func (s *Store) Close() {
	s.cancel()
	s.flush()
	if err := s.db.Close(); err != nil {
		slog.Error("failed to close duckdb", "error", err)
	}
}

// DB 获取数据库连接（用于查询）
func (s *Store) DB() *sqlx.DB {
	return s.db
}

// flushLoop 后台定时刷新
func (s *Store) flushLoop(ctx context.Context) {
	ticker := time.NewTicker(FlushInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			s.flush()
		}
	}
}

// flush 批量写入所有缓冲数据
func (s *Store) flush() {
	s.mu.Lock()
	logs := s.logs
	details := s.details
	chunks := s.chunks
	s.logs = nil
	s.details = nil
	s.chunks = nil
	s.mu.Unlock()

	if len(logs) == 0 && len(details) == 0 && len(chunks) == 0 {
		return
	}

	if len(logs) > 0 {
		s.flushLogs(logs)
	}
	if len(details) > 0 {
		s.flushDetails(details)
	}
	if len(chunks) > 0 {
		s.flushChunks(chunks)
	}
}

// flushLogs 使用 Appender 批量写入 request_logs
func (s *Store) flushLogs(logs []*model.RequestLog) {
	conn, err := s.db.DB.Conn(context.Background())
	if err != nil {
		slog.Error("failed to get connection for logs", "error", err)
		return
	}
	defer conn.Close()

	var appender *duckdb.Appender
	err = conn.Raw(func(driverConn any) error {
		c := driverConn.(*duckdb.Conn)
		appender, err = duckdb.NewAppenderFromConn(c, "", "request_logs")
		return err
	})
	if err != nil {
		slog.Error("failed to create appender for logs", "error", err)
		return
	}
	defer appender.Close()

	for _, log := range logs {
		err := appender.AppendRow(
			log.TraceID,
			log.UserID,
			log.APIKeyID,
			log.UserModel,
			log.ProviderModel,
			log.ResponseModel,
			string(log.UserApiType),
			string(log.ProviderApiType),
			string(log.PassthroughLevel),
			log.Summary,
			log.IsStream,
			log.PromptTokens,
			log.CompletionTokens,
			log.ReasoningTokens,
			log.TotalTokens,
			log.CachedTokens,
			log.IsDetail,
			log.StatusCode,
			log.ErrorMessage,
			log.Duration,
			log.IPAddress,
			log.UserAgent,
			log.CreatedAt,
		)
		if err != nil {
			slog.Error("failed to append log row", "error", err)
			return
		}
	}

	if err := appender.Flush(); err != nil {
		slog.Error("failed to flush logs", "error", err)
	}

	// 预聚合到 stats_hourly
	s.aggregateHourly(logs)
}

// flushDetails 使用 Appender 批量写入 request_details
func (s *Store) flushDetails(details []*model.RequestDetail) {
	conn, err := s.db.DB.Conn(context.Background())
	if err != nil {
		slog.Error("failed to get connection for details", "error", err)
		return
	}
	defer conn.Close()

	var appender *duckdb.Appender
	err = conn.Raw(func(driverConn any) error {
		c := driverConn.(*duckdb.Conn)
		appender, err = duckdb.NewAppenderFromConn(c, "", "request_details")
		return err
	})
	if err != nil {
		slog.Error("failed to create appender for details", "error", err)
		return
	}
	defer appender.Close()

	for _, detail := range details {
		err := appender.AppendRow(
			detail.TraceID,
			detail.Request,
			detail.RequestRaw,
			detail.Response,
			detail.ResponseRaw,
			detail.Reasoning,
		)
		if err != nil {
			slog.Error("failed to append detail row", "error", err)
			return
		}
	}

	if err := appender.Flush(); err != nil {
		slog.Error("failed to flush details", "error", err)
	}
}

// flushChunks 使用 Appender 批量写入 request_chunks
func (s *Store) flushChunks(chunks []*model.RequestChunk) {
	conn, err := s.db.DB.Conn(context.Background())
	if err != nil {
		slog.Error("failed to get connection for chunks", "error", err)
		return
	}
	defer conn.Close()

	var appender *duckdb.Appender
	err = conn.Raw(func(driverConn any) error {
		c := driverConn.(*duckdb.Conn)
		appender, err = duckdb.NewAppenderFromConn(c, "", "request_chunks")
		return err
	})
	if err != nil {
		slog.Error("failed to create appender for chunks", "error", err)
		return
	}
	defer appender.Close()

	for _, chunk := range chunks {
		err := appender.AppendRow(
			chunk.ChunkID,
			chunk.TraceID,
			chunk.Index,
			string(chunk.Type),
			chunk.Data,
			chunk.CreatedAt,
		)
		if err != nil {
			slog.Error("failed to append chunk row", "error", err)
			return
		}
	}

	if err := appender.Flush(); err != nil {
		slog.Error("failed to flush chunks", "error", err)
	}
}

// RecordRequest 缓冲请求日志
func (s *Store) RecordRequest(log *model.RequestLog) {
	if s.db == nil || log == nil {
		return
	}

	s.mu.Lock()
	s.logs = append(s.logs, log)
	shouldFlush := len(s.logs) >= FlushSize
	s.mu.Unlock()

	if shouldFlush {
		go s.flush()
	}
}

// RecordDetail 缓冲请求详情
func (s *Store) RecordDetail(traceID string, detail *model.RequestDetail) {
	if s.db == nil {
		return
	}
	detail.TraceID = traceID

	s.mu.Lock()
	s.details = append(s.details, detail)
	shouldFlush := len(s.details) >= FlushSize
	s.mu.Unlock()

	if shouldFlush {
		go s.flush()
	}
}

// RecordChunks 缓冲流式响应 chunks
func (s *Store) RecordChunks(chunks []*model.RequestChunk) {
	if s.db == nil || len(chunks) == 0 {
		return
	}

	s.mu.Lock()
	s.chunks = append(s.chunks, chunks...)
	shouldFlush := len(s.chunks) >= FlushSize
	s.mu.Unlock()

	if shouldFlush {
		go s.flush()
	}
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
