package service

import (
	"log"
	"sync"
	"time"

	"llm-gateway/internal/model"

	"gorm.io/gorm"
)

type AuditService struct {
	db         *gorm.DB
	buffer     chan *model.AuditLog
	done       chan struct{}
	flushBatch int
	mu         sync.Mutex
	stopped    bool
}

func NewAuditService(db *gorm.DB, bufferSize int) *AuditService {
	return &AuditService{
		db:     db,
		buffer: make(chan *model.AuditLog, bufferSize),
		done:   make(chan struct{}),
	}
}

// Start launches a background goroutine that periodically flushes buffered audit logs.
func (s *AuditService) Start(flushInterval time.Duration, flushBatch int) {
	s.flushBatch = flushBatch
	go s.worker(flushInterval)
}

// Stop signals the background worker to flush remaining logs and exit.
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

// Record enqueues an audit log without blocking.
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
		// drop when buffer is full
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
		if err := s.db.Create(batch).Error; err != nil {
			log.Printf("audit flush error: %v", err)
		}
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
