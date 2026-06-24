package service

import (
	"log/slog"
	"time"

	"llm-gateway/internal/model"

	"github.com/jmoiron/sqlx"
	"github.com/tidwall/gjson"
)

type RequestLogService struct {
	store *sqlx.DB
}

func NewRequestLogService(store *sqlx.DB) *RequestLogService {
	return &RequestLogService{store: store}
}

// isDetailEnabled 从数据库读取配置判断是否记录详细请求/响应
func (s *RequestLogService) isDetailEnabled() bool {
	return GetConfigString(model.ConfigKeyRequestLogDetail) == "true"
}

// extractSummary 从请求体中提取最后的用户问题作为摘要
func extractSummary(reqBytes []byte) string {
	if len(reqBytes) == 0 {
		return ""
	}
	result := gjson.ParseBytes(reqBytes)
	messages := result.Get("messages")
	if !messages.IsArray() {
		return ""
	}
	// 从后往前找最后一条 user 消息
	arr := messages.Array()
	for i := len(arr) - 1; i >= 0; i-- {
		if arr[i].Get("role").String() == "user" {
			content := arr[i].Get("content").String()
			if len(content) > 100 {
				return content[:100]
			}
			return content
		}
	}
	return ""
}

// RecordRequest 记录请求日志到 DuckDB
func (s *RequestLogService) RecordRequest(traceID string, userID, apiKeyID uint,
	userModel, providerModel, userApiType, providerApiType, passthroughLevel string, isStream bool,
	promptTokens, completionTokens, totalTokens, cachedTokens int,
	statusCode int, errMsg string, duration int64,
	ipAddress, userAgent string,
	reqBytes, respBytes []byte, chunks []*model.RequestChunk) {

	if s.store == nil {
		return
	}

	logDetail := s.isDetailEnabled()
	summary := extractSummary(reqBytes)

	// 插入请求日志
	_, err := s.store.Exec(`INSERT INTO request_logs
		(trace_id, user_id, api_key_id, user_model, provider_model, user_api_type, provider_api_type, passthrough_level,
		 summary, is_stream, prompt_tokens, completion_tokens, total_tokens, cached_tokens,
		 is_detail, status_code, error_message, duration,
		 ip_address, user_agent, created_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		traceID, userID, apiKeyID, userModel, providerModel, userApiType, providerApiType, passthroughLevel,
		summary, isStream, promptTokens, completionTokens, totalTokens, cachedTokens,
		logDetail, statusCode, TruncateStr(errMsg, 4096), duration,
		ipAddress, TruncateStr(userAgent, 512), time.Now(),
	)
	if err != nil {
		slog.Error("failed to insert request log", "error", err)
	}

	// 插入详细请求/响应
	if logDetail {
		_, err := s.store.Exec(`INSERT INTO request_details
			(trace_id, request_body, response_body)
			VALUES (?, ?, ?)`,
			traceID,
			TruncateStr(string(reqBytes), 65536),
			TruncateStr(string(respBytes), 65536),
		)
		if err != nil {
			slog.Error("failed to insert request detail", "error", err)
		}
	}

	// 插入流式 chunks
	if logDetail && len(chunks) > 0 {
		for _, chunk := range chunks {
			_, err := s.store.Exec(`INSERT INTO request_chunks
				(chunk_id, trace_id, index, data, created_at)
				VALUES (?, ?, ?, ?, ?)`,
				chunk.ChunkID, chunk.TraceID, chunk.Index, chunk.Data, chunk.CreatedAt,
			)
			if err != nil {
				slog.Error("failed to insert request chunk", "error", err)
			}
		}
	}
}

// StreamChunkCollector 收集流式响应的 chunks
type StreamChunkCollector struct {
	traceID   string
	logDetail bool
	chunks    []*model.RequestChunk
	index     int
}

// NewChunkCollector 创建 chunk 收集器
func (s *RequestLogService) NewChunkCollector(traceID string) *StreamChunkCollector {
	return &StreamChunkCollector{
		traceID:   traceID,
		logDetail: s.isDetailEnabled(),
	}
}

// Add 添加一个 chunk
func (sc *StreamChunkCollector) Add(data []byte) {
	if !sc.logDetail {
		return
	}
	sc.chunks = append(sc.chunks, &model.RequestChunk{
		TraceID:   sc.traceID,
		Index:     sc.index,
		Data:      string(data),
		CreatedAt: time.Now(),
	})
	sc.index++
}

// Chunks 返回收集的 chunks
func (sc *StreamChunkCollector) Chunks() []*model.RequestChunk {
	if !sc.logDetail {
		return nil
	}
	return sc.chunks
}

// TruncateStr 截断字符串到指定最大长度
func TruncateStr(s string, maxLen int) string {
	if len(s) > maxLen {
		return s[:maxLen]
	}
	return s
}
