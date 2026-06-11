package service

import (
	"context"
	"database/sql"
	"log"
	"sync"
	"time"

	"llm-gateway/internal/model"
)

const insertRequestChunkSQL = `INSERT INTO request_chunks
	(trace_id, chunk_index, chunk_data, created_at)
	VALUES (?, ?, ?, ?)`

type RequestChunkService struct {
	db         *sql.DB
	buffer     chan *model.RequestChunk
	done       chan struct{}
	flushBatch int
	mu         sync.Mutex
	stopped    bool
}

func NewRequestChunkService(db *sql.DB, bufferSize int) *RequestChunkService {
	return &RequestChunkService{
		db:     db,
		buffer: make(chan *model.RequestChunk, bufferSize),
		done:   make(chan struct{}),
	}
}

func (s *RequestChunkService) Start(flushInterval time.Duration, flushBatch int) {
	s.flushBatch = flushBatch
	go s.worker(flushInterval)
}

func (s *RequestChunkService) Stop() {
	s.mu.Lock()
	if s.stopped {
		s.mu.Unlock()
		return
	}
	s.stopped = true
	s.mu.Unlock()

	close(s.done)
}

func (s *RequestChunkService) Record(chunk *model.RequestChunk) {
	s.mu.Lock()
	stopped := s.stopped
	s.mu.Unlock()

	if stopped {
		return
	}

	select {
	case s.buffer <- chunk:
	default:
	}
}

func (s *RequestChunkService) worker(flushInterval time.Duration) {
	ticker := time.NewTicker(flushInterval)
	defer ticker.Stop()

	batch := make([]*model.RequestChunk, 0, s.flushBatch)

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
				case chunk := <-s.buffer:
					batch = append(batch, chunk)
				default:
					flush()
					return
				}
			}
		case chunk := <-s.buffer:
			batch = append(batch, chunk)
			if len(batch) >= s.flushBatch {
				flush()
			}
		case <-ticker.C:
			flush()
		}
	}
}

func (s *RequestChunkService) flush(batch []*model.RequestChunk) {
	if len(batch) == 0 {
		return
	}

	ctx := context.Background()
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		log.Printf("request_chunk flush begin error: %v", err)
		return
	}

	stmt, err := tx.PrepareContext(ctx, insertRequestChunkSQL)
	if err != nil {
		tx.Rollback()
		log.Printf("request_chunk flush prepare error: %v", err)
		return
	}
	defer stmt.Close()

	for _, c := range batch {
		if _, err := stmt.ExecContext(ctx,
			c.TraceID, c.ChunkIndex, c.ChunkData, c.CreatedAt,
		); err != nil {
			tx.Rollback()
			log.Printf("request_chunk flush exec error: %v", err)
			return
		}
	}

	if err := tx.Commit(); err != nil {
		log.Printf("request_chunk flush commit error: %v", err)
	}
}
