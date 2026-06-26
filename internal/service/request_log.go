package service

import (
	"log/slog"
	"time"

	"llm-gateway/internal/model"

	"github.com/jmoiron/sqlx"
)

type RequestLogService struct {
	store *sqlx.DB
}

func NewRequestLogService(store *sqlx.DB) *RequestLogService {
	return &RequestLogService{store: store}
}

// IsDetailEnabled 判断是否记录详细请求/响应
func (s *RequestLogService) IsDetailEnabled() bool {
	return GetConfigString(model.ConfigKeyRequestLogDetail) == "true"
}

// RecordRequest 保存请求日志到数据库
func (s *RequestLogService) RecordRequest(log *model.RequestLog) {
	if s.store == nil || log == nil {
		return
	}

	log.IsDetail = s.IsDetailEnabled()
	log.CreatedAt = time.Now()
	log.ErrorMessage = TruncateStr(log.ErrorMessage, 4096)
	log.UserAgent = TruncateStr(log.UserAgent, 512)

	_, err := s.store.NamedExec(`INSERT INTO request_logs
		(trace_id, user_id, api_key_id, user_model, provider_model, response_model, user_api_type, provider_api_type, passthrough_level,
		 summary, is_stream, prompt_tokens, completion_tokens, reasoning_tokens, total_tokens, cached_tokens,
		 is_detail, status_code, error_message, duration, ip_address, user_agent, created_at)
		VALUES (:trace_id, :user_id, :api_key_id, :user_model, :provider_model, :response_model, :user_api_type, :provider_api_type, :passthrough_level,
		 :summary, :is_stream, :prompt_tokens, :completion_tokens, :reasoning_tokens, :total_tokens, :cached_tokens,
		 :is_detail, :status_code, :error_message, :duration, :ip_address, :user_agent, :created_at)`, log)
	if err != nil {
		slog.Error("failed to insert request log", "error", err)
	}
}

// RecordDetail 记录请求详情
func (s *RequestLogService) RecordDetail(traceID string, reqBytes, respBytes []byte) {
	if s.store == nil {
		return
	}
	_, err := s.store.Exec(`INSERT INTO request_details (trace_id, request, response) VALUES (?, ?, ?)`,
		traceID, TruncateStr(string(reqBytes), 65536), TruncateStr(string(respBytes), 65536))
	if err != nil {
		slog.Error("failed to insert request detail", "error", err)
	}
}

// RecordChunks 记录流式响应 chunks
func (s *RequestLogService) RecordChunks(chunks []*model.RequestChunk) {
	if s.store == nil || len(chunks) == 0 {
		return
	}

	for _, chunk := range chunks {
		_, err := s.store.Exec(`INSERT INTO request_chunks (chunk_id, trace_id, index, data, created_at) VALUES (?, ?, ?, ?, ?)`,
			chunk.ChunkID, chunk.TraceID, chunk.Index, chunk.Data, chunk.CreatedAt)
		if err != nil {
			slog.Error("failed to insert request chunk", "error", err)
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
		logDetail: s.IsDetailEnabled(),
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
