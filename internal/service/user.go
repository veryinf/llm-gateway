package service

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"

	"llm-gateway/internal/model"

	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

type UserService struct {
	db *gorm.DB
}

func NewUserService(db *gorm.DB) *UserService {
	return &UserService{db: db}
}

// CreateDefaultAdmin checks if any user exists; if not, creates a default admin account.
func (s *UserService) CreateDefaultAdmin(username, password string) (*model.User, error) {
	var count int64
	if err := s.db.Model(&model.User{}).Count(&count).Error; err != nil {
		return nil, err
	}
	if count > 0 {
		return nil, nil
	}

	return s.CreateUser(username, password, "", "", "", model.RoleAdmin)
}

// Login validates username and password.
func (s *UserService) Login(username, password string) (*model.User, error) {
	var user model.User
	if err := s.db.Where("username = ?", username).First(&user).Error; err != nil {
		return nil, err
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(password)); err != nil {
		return nil, err
	}

	return &user, nil
}

// CreateUser creates a new user with a bcrypt-hashed password.
func (s *UserService) CreateUser(username, password, name, phone, department string, role model.Role) (*model.User, error) {
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return nil, err
	}

	user := &model.User{
		Username:   username,
		Password:   string(hash),
		Name:       name,
		Phone:      phone,
		Department: department,
		Role:       role,
		Status:     "active",
	}

	if err := s.db.Create(user).Error; err != nil {
		return nil, err
	}
	return user, nil
}

// GetUser retrieves a user by ID.
func (s *UserService) GetUser(id uint) (*model.User, error) {
	var user model.User
	if err := s.db.First(&user, id).Error; err != nil {
		return nil, err
	}
	return &user, nil
}

// GenerateAPIKeyRecord creates and stores a new API key record with plaintext key.
func GenerateAPIKeyRecord(db *gorm.DB, userID uint, name string, quotaLimit int64, rateLimitQPM int) (*model.APIKey, string, error) {
	raw := make([]byte, 32)
	if _, err := rand.Read(raw); err != nil {
		return nil, "", fmt.Errorf("generate random key: %w", err)
	}

	rawKey := "sk-" + hex.EncodeToString(raw)

	key := &model.APIKey{
		UserID:       userID,
		Key:          rawKey,
		Name:         name,
		QuotaLimit:   quotaLimit,
		QuotaUsed:    0,
		RateLimitQPM: rateLimitQPM,
		IsActive:     true,
	}

	if err := db.Create(key).Error; err != nil {
		return nil, "", err
	}
	return key, rawKey, nil
}

// GenerateAKSK generates AKSK key pair for a user.
func GenerateAKSK(db *gorm.DB, userID uint) (accessKey, secretKey string, err error) {
	ak := make([]byte, 24)
	if _, err = rand.Read(ak); err != nil {
		return "", "", err
	}
	accessKey = hex.EncodeToString(ak)

	sk := make([]byte, 32)
	if _, err = rand.Read(sk); err != nil {
		return "", "", err
	}
	secretKey = hex.EncodeToString(sk)

	if err = db.Model(&model.User{}).Where("id = ?", userID).
		Updates(map[string]interface{}{
			"access_key": accessKey,
			"secret_key": secretKey,
		}).Error; err != nil {
		return "", "", err
	}

	return accessKey, secretKey, nil
}
