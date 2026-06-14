package database

import (
	"database/sql"
	"log/slog"
	"path/filepath"
	"strings"

	"llm-gateway/internal/model"

	"github.com/glebarez/sqlite"
	"github.com/google/uuid"
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
		&model.DownstreamModel{},
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

const (
	defaultUsername = "root"
	defaultPassword = "123456"
)

// SeedDefaultUser 首次启动时自动创建默认 admin 用户。
func SeedDefaultUser(db *gorm.DB) {
	var count int64
	db.Model(&model.User{}).Count(&count)
	if count > 0 {
		return
	}

	hashed, err := bcrypt.GenerateFromPassword([]byte(defaultPassword), bcrypt.DefaultCost)
	if err != nil {
		slog.Error("failed to hash default password", "error", err)
		return
	}

	ak := strings.ReplaceAll(uuid.New().String(), "-", "")[:16]
	sk := strings.ReplaceAll(uuid.New().String(), "-", "")

	user := model.User{
		Username:  defaultUsername,
		Password:  string(hashed),
		Status:    "active",
		AccessKey: ak,
		SecretKey: sk,
	}
	if err := db.Create(&user).Error; err != nil {
		slog.Error("failed to create default user", "error", err)
		return
	}

	slog.Info("created default user", "username", user.Username)
}
