package database

import (
	"fmt"

	"llm-gateway/internal/config"
	"llm-gateway/internal/model"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func NewDB(cfg *config.DatabaseConfig) (*gorm.DB, error) {
	db, err := gorm.Open(sqlite.Open(cfg.Path), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Info),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	if err := db.Exec("PRAGMA journal_mode=WAL").Error; err != nil {
		return nil, fmt.Errorf("failed to enable WAL mode: %w", err)
	}

	if err := db.Exec("PRAGMA foreign_keys=ON").Error; err != nil {
		return nil, fmt.Errorf("failed to enable foreign keys: %w", err)
	}

	if err := db.AutoMigrate(
		&model.User{},
		&model.APIKey{},
		&model.Provider{},
		&model.Model{},
		&model.RequestLog{},
		&model.StatisticsHourly{},
		&model.PricingRule{},
		&model.AuditLog{},
	); err != nil {
		return nil, fmt.Errorf("failed to auto migrate: %w", err)
	}

	return db, nil
}
