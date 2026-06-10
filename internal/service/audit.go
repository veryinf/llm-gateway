package service

import (
	"context"
	"database/sql"
	"log"
	"sync"
	"time"

	"llm-gateway/internal/model"
)

const insertAuditLogSQL = `INSERT INTO audit_logs
	(trace_id, user_id, api_key_id, provider_id, model_name,
	 request_summary, response_summary, prompt_tokens, completion_tokens,
	 status_code, error_message, latency_ms, cost, ip_address, user_agent,
	 created_at)
	VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`

type AuditService struct {
	db         *sql.DB
	buffer     chan *model.AuditLog
	done       chan struct{}
	flushBatch int
	mu         sync.Mutex
	stopped    bool
}

func NewAuditService(db *sql.DB, bufferSize int) *AuditService {
	return &AuditService{
		db:     db,
		buffer: make(chan *model.AuditLog, bufferSize),
		done:   make(chan struct{}),
	}
}

func (s *AuditService) Start(flushInterval time.Duration, flushBatch int) {
	s.flushBatch = flushBatch
	go s.worker(flushInterval)
}

func (s *AuditService) Stop() {
	s.mu.Lock()
	if s.stopped {
		s.mu.Unlock()
		return
	}
	s.stopped = true
	s.mu.Unlock()

	close(s.done)
}

func (s *AuditService) Record(log *model.AuditLog) {
	s.mu.Lock()
	stopped := s.stopped
	s.mu.Unlock()

	if stopped {
		return
	}

	select {
	case s.buffer <- log:
	default:
	}
}

func (s *AuditService) worker(flushInterval time.Duration) {
	ticker := time.NewTicker(flushInterval)
	defer ticker.Stop()

	batch := make([]*model.AuditLog, 0, s.flushBatch)

	flush := func() {
		if len(batch) == 0 {
			return
		}
		s.flush(batch)
		batch = batch[:0]
	}

	for {
		select {
		case <-s.done:
			for {
				select {
				case log := <-s.buffer:
					batch = append(batch, log)
				default:
					flush()
					return
				}
			}
		case log := <-s.buffer:
			batch = append(batch, log)
			if len(batch) >= s.flushBatch {
				flush()
			}
		case <-ticker.C:
			flush()
		}
	}
}

func (s *AuditService) flush(batch []*model.AuditLog) {
	if len(batch) == 0 {
		return
	}

	ctx := context.Background()
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		log.Printf("audit flush begin error: %v", err)
		return
	}

	stmt, err := tx.PrepareContext(ctx, insertAuditLogSQL)
	if err != nil {
		tx.Rollback()
		log.Printf("audit flush prepare error: %v", err)
		return
	}
	defer stmt.Close()

	for _, l := range batch {
		if _, err := stmt.ExecContext(ctx,
			l.TraceID, l.UserID, l.APIKeyID, l.ProviderID, l.ModelName,
			l.RequestSummary, l.ResponseSummary, l.PromptTokens, l.CompletionTokens,
			l.StatusCode, l.ErrorMessage, l.LatencyMs, l.Cost,
			l.IPAddress, l.UserAgent, l.CreatedAt,
		); err != nil {
			tx.Rollback()
			log.Printf("audit flush exec error: %v", err)
			return
		}
	}

	if err := tx.Commit(); err != nil {
		log.Printf("audit flush commit error: %v", err)
	}
}
