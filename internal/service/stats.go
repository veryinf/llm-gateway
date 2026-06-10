package service

import (
	"context"
	"database/sql"
	"log"
	"sync"
	"time"

	"llm-gateway/internal/model"
)

const insertRequestLogSQL = `INSERT INTO request_logs
	(trace_id, user_id, api_key_id, provider_id, model_name, is_stream,
	 prompt_tokens, completion_tokens, total_tokens, status_code,
	 error_message, latency_ms, cost, created_at)
	VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`

type StatsService struct {
	db         *sql.DB
	buffer     chan *model.RequestLog
	done       chan struct{}
	flushBatch int
	bufferSize int
	mu         sync.Mutex
	stopped    bool
}

func NewStatsService(db *sql.DB, bufferSize int) *StatsService {
	return &StatsService{
		db:         db,
		buffer:     make(chan *model.RequestLog, bufferSize),
		done:       make(chan struct{}),
		bufferSize: bufferSize,
	}
}

func (s *StatsService) Start(flushInterval time.Duration, flushBatch int) {
	s.flushBatch = flushBatch
	go s.worker(flushInterval)
}

func (s *StatsService) Stop() {
	s.mu.Lock()
	if s.stopped {
		s.mu.Unlock()
		return
	}
	s.stopped = true
	s.mu.Unlock()

	close(s.done)
}

func (s *StatsService) Record(log *model.RequestLog) {
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

func (s *StatsService) worker(flushInterval time.Duration) {
	ticker := time.NewTicker(flushInterval)
	defer ticker.Stop()

	batch := make([]*model.RequestLog, 0, s.flushBatch)

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

func (s *StatsService) flush(batch []*model.RequestLog) {
	if len(batch) == 0 {
		return
	}

	ctx := context.Background()
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		log.Printf("stats flush begin error: %v", err)
		return
	}

	stmt, err := tx.PrepareContext(ctx, insertRequestLogSQL)
	if err != nil {
		tx.Rollback()
		log.Printf("stats flush prepare error: %v", err)
		return
	}
	defer stmt.Close()

	for _, l := range batch {
		if _, err := stmt.ExecContext(ctx,
			l.TraceID, l.UserID, l.APIKeyID, l.ProviderID,
			l.ModelName, l.IsStream, l.PromptTokens, l.CompletionTokens,
			l.TotalTokens, l.StatusCode, l.ErrorMessage, l.LatencyMs,
			l.Cost, l.CreatedAt,
		); err != nil {
			tx.Rollback()
			log.Printf("stats flush exec error: %v", err)
			return
		}
	}

	if err := tx.Commit(); err != nil {
		log.Printf("stats flush commit error: %v", err)
	}
}
