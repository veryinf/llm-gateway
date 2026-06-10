package service

import (
	"log"
	"sync"
	"time"

	"llm-gateway/internal/model"

	"gorm.io/gorm"
)

type StatsService struct {
	db         *gorm.DB
	buffer     chan *model.RequestLog
	done       chan struct{}
	flushBatch int
	bufferSize int
	mu         sync.Mutex
	stopped    bool
}

func NewStatsService(db *gorm.DB, bufferSize int) *StatsService {
	return &StatsService{
		db:         db,
		buffer:     make(chan *model.RequestLog, bufferSize),
		done:       make(chan struct{}),
		bufferSize: bufferSize,
	}
}

// Start launches a background goroutine that periodically flushes buffered request logs to the database.
func (s *StatsService) Start(flushInterval time.Duration, flushBatch int) {
	s.flushBatch = flushBatch
	go s.worker(flushInterval)
}

// Stop signals the background worker to flush remaining logs and exit gracefully.
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

// Record enqueues a request log without blocking. If the buffer is full the log is dropped.
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
		// drop when buffer is full to avoid blocking the request path
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
		if err := s.db.Create(batch).Error; err != nil {
			log.Printf("stats flush error: %v", err)
		}
		batch = batch[:0]
	}

	for {
		select {
		case <-s.done:
			// drain remaining buffer before exit
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
