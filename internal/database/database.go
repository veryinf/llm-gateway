package database

import (
	"database/sql"
	"fmt"
	"log/slog"
	"path/filepath"

	"llm-gateway/internal/model"

	"github.com/glebarez/sqlite"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
	gormlogger "gorm.io/gorm/logger"
)

func InitDB(dataDir string) (*gorm.DB, *sql.DB) {
	dbPath := filepath.Join(dataDir, "application.db")

	db, err := gorm.Open(sqlite.Open(dbPath), &gorm.Config{
		Logger: gormlogger.Default.LogMode(gormlogger.Info),
	})
	if err != nil {
		slog.Error("failed to open sqlite database", "error", err)
		panic(err)
	}

	if err := db.Exec("PRAGMA journal_mode=WAL").Error; err != nil {
		slog.Error("failed to enable WAL mode", "error", err)
		panic(err)
	}

	if err := db.Exec("PRAGMA foreign_keys=ON").Error; err != nil {
		slog.Error("failed to enable foreign keys", "error", err)
		panic(err)
	}

	if err := db.AutoMigrate(
		&model.User{},
		&model.APIKey{},
		&model.Provider{},
		&model.Model{},
		&model.Config{},
	); err != nil {
		slog.Error("failed to auto migrate sqlite", "error", err)
		panic(err)
	}

	sqlDB, err := db.DB()
	if err != nil {
		slog.Error("failed to get underlying sql.DB", "error", err)
		panic(err)
	}

	slog.Info("sqlite initialized", "path", dbPath)
	return db, sqlDB
}

func SeedDefaultAdmin(db *gorm.DB, adminPassword string) {
	if adminPassword == "" {
		slog.Warn("admin_password not set, default admin not created")
		return
	}

	var count int64
	if err := db.Model(&model.User{}).Count(&count).Error; err != nil {
		return
	}
	if count > 0 {
		return
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(adminPassword), bcrypt.DefaultCost)
	if err != nil {
		slog.Error("failed to hash admin password", "error", err)
		return
	}

	user := &model.User{
		Username:     "admin",
		PasswordHash: string(hash),
		Name:         "Administrator",
		Role:         model.RoleAdmin,
		IsActive:     true,
	}

	if err := db.Create(user).Error; err != nil {
		slog.Error("failed to create default admin", "error", err)
		return
	}

	slog.Info("default admin user initialized")
}

// unused import guard
var _ = fmt.Sprintf
