package database

import (
	"fmt"
	"log/slog"
	"time"

	"llm-gateway/internal/model"
)

// aggKey 预聚合的维度键
type aggKey struct {
	hour          time.Time
	userID        uint
	userModel     string
	providerModel string
}

// statsHourlyRow 预聚合行
type statsHourlyRow struct {
	hour             time.Time
	userID           uint
	userModel        string
	providerModel    string
	requestCount     int
	successCount     int
	errorCount       int
	promptTokens     int64
	completionTokens int64
	reasoningTokens  int64
	totalTokens      int64
	totalDuration    int64
}

// aggregateHourly 从已缓冲的 logs 计算预聚合并 upsert 到 stats_hourly
func (s *Store) aggregateHourly(logs []*model.RequestLog) {
	if len(logs) == 0 {
		return
	}

	// 1) 在 Go 侧计算聚合
	agg := make(map[aggKey]*statsHourlyRow)

	for _, log := range logs {
		k := aggKey{
			hour:          log.CreatedAt.Truncate(time.Hour),
			userID:        log.UserID,
			userModel:     log.UserModel,
			providerModel: log.ProviderModel,
		}
		row, exists := agg[k]
		if !exists {
			row = &statsHourlyRow{
				hour:          k.hour,
				userID:        k.userID,
				userModel:     k.userModel,
				providerModel: k.providerModel,
			}
			agg[k] = row
		}
		row.requestCount++
		if log.StatusCode < 400 {
			row.successCount++
		} else {
			row.errorCount++
		}
		row.promptTokens += int64(log.PromptTokens)
		row.completionTokens += int64(log.CompletionTokens)
		row.reasoningTokens += int64(log.ReasoningTokens)
		row.totalTokens += int64(log.TotalTokens)
		row.totalDuration += log.Duration
	}

	// 2) 批量 upsert
	if err := s.upsertStatsHourly(agg); err != nil {
		slog.Error("failed to upsert stats_hourly", "error", err)
	}
}

// upsertStatsHourly 使用 INSERT ... ON CONFLICT DO UPDATE 实现批量 upsert
func (s *Store) upsertStatsHourly(agg map[aggKey]*statsHourlyRow) error {
	for _, row := range agg {
		_, err := s.db.Exec(`
			INSERT INTO stats_hourly (
				hour, user_id, user_model, provider_model,
				request_count, success_count, error_count,
				prompt_tokens, completion_tokens, reasoning_tokens,
				total_tokens, total_duration
			) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
			ON CONFLICT (hour, user_id, user_model, provider_model) DO UPDATE SET
				request_count = excluded.request_count + stats_hourly.request_count,
				success_count = excluded.success_count + stats_hourly.success_count,
				error_count = excluded.error_count + stats_hourly.error_count,
				prompt_tokens = excluded.prompt_tokens + stats_hourly.prompt_tokens,
				completion_tokens = excluded.completion_tokens + stats_hourly.completion_tokens,
				reasoning_tokens = excluded.reasoning_tokens + stats_hourly.reasoning_tokens,
				total_tokens = excluded.total_tokens + stats_hourly.total_tokens,
				total_duration = excluded.total_duration + stats_hourly.total_duration
		`,
			row.hour, row.userID, row.userModel, row.providerModel,
			row.requestCount, row.successCount, row.errorCount,
			row.promptTokens, row.completionTokens, row.reasoningTokens,
			row.totalTokens, row.totalDuration,
		)
		if err != nil {
			return fmt.Errorf("failed to upsert row: %w", err)
		}
	}
	return nil
}