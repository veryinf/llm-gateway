package worker

import (
	"context"
	"time"

	"llm-gateway/internal/model"

	"gorm.io/gorm"
)

type CleanupWorker struct {
	db *gorm.DB
}

func NewCleanupWorker(db *gorm.DB) *CleanupWorker {
	return &CleanupWorker{db: db}
}

// Start runs a periodic cleanup that deletes audit logs older than retentionDays.
func (w *CleanupWorker) Start(ctx context.Context, retentionDays int) {
	ticker := time.NewTicker(24 * time.Hour)
	defer ticker.Stop()

	w.cleanup(retentionDays)

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			w.cleanup(retentionDays)
		}
	}
}

func (w *CleanupWorker) cleanup(retentionDays int) {
	before := time.Now().AddDate(0, 0, -retentionDays)
	w.db.Where("created_at < ?", before).Delete(&model.AuditLog{})
}
