package worker

import (
	"context"
	"database/sql"
	"time"
)

type CleanupWorker struct {
	db *sql.DB
}

func NewCleanupWorker(db *sql.DB) *CleanupWorker {
	return &CleanupWorker{db: db}
}

func (w *CleanupWorker) Start(ctx context.Context, auditRetentionDays, statsRetentionDays int) {
	ticker := time.NewTicker(24 * time.Hour)
	defer ticker.Stop()

	w.cleanup(auditRetentionDays, statsRetentionDays)

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			w.cleanup(auditRetentionDays, statsRetentionDays)
		}
	}
}

func (w *CleanupWorker) cleanup(auditRetentionDays, statsRetentionDays int) {
	if auditRetentionDays > 0 {
		before := time.Now().AddDate(0, 0, -auditRetentionDays)
		w.db.Exec("DELETE FROM audit_logs WHERE created_at < ?", before)
	}

	if statsRetentionDays > 0 {
		before := time.Now().AddDate(0, 0, -statsRetentionDays)
		w.db.Exec("DELETE FROM request_logs WHERE created_at < ?", before)
	}
}
