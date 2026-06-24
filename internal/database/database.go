package database

import (
	"database/sql"
	"llm-gateway/internal/model"
	"log/slog"
	"os"
	"path/filepath"
	"strings"

	"github.com/glebarez/sqlite"
	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

func InitDB(dataDir string) (*gorm.DB, *sql.DB) {
	dbPath := filepath.Join(dataDir, "application.db")

	db, err := gorm.Open(sqlite.Open(dbPath), &gorm.Config{
		//Logger: logger.NewSlogLogger(slog.Default(), logger.Config{
		//	SlowThreshold: 200 * time.Millisecond,
		//	LogLevel:      logger.Info,
		//	Colorful:      false,
		//}),
	})
	if err != nil {
		slog.Error("failed to connect application database", "error", err)
		os.Exit(1)
	}
	if err := db.Exec("PRAGMA journal_mode=WAL").Error; err != nil {
		slog.Error("failed to enable WAL mode", "error", err)
		os.Exit(1)
	}
	if err := db.Exec("PRAGMA foreign_keys=ON").Error; err != nil {
		slog.Error("failed to enable foreign keys", "error", err)
		os.Exit(1)
	}
	if err := db.Exec("PRAGMA mmap_size=67108864").Error; err != nil {
		slog.Error("failed to enable mmap", "error", err)
		os.Exit(1)
	}
	conn, err := db.DB()
	if err != nil {
		slog.Error("failed to get application database connection", "error", err)
		os.Exit(1)
	}

	// 自动迁移（创建表结构）
	if err := db.AutoMigrate(
		&model.Config{},
		&model.UserKey{},
		&model.Provider{},
		&model.ProviderModel{},
		&model.UserModel{},
		&model.UserModelRouter{},
		&model.User{},
	); err != nil {
		slog.Error("failed to migrate application database", "error", err)
		os.Exit(1)
	}

	slog.Info("sqlite initialized", "path", dbPath)
	return db, conn
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
		Username:   defaultUsername,
		Password:   string(hashed),
		Name:       "管理员",
		Phone:      "13012345678",
		Department: "系统管理",
		Role:       model.RoleAdmin,
		Status:     "active",
		AccessKey:  ak,
		SecretKey:  sk,
	}
	if err := db.Create(&user).Error; err != nil {
		slog.Error("failed to create default user", "error", err)
		return
	}

	slog.Info("created default user", "username", user.Username)
}
