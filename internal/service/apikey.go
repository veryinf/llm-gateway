package service

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"fmt"

	"llm-gateway/internal/model"

	"gorm.io/gorm"
)

type APIKeyService struct {
	db *gorm.DB
}

func NewAPIKeyService(db *gorm.DB) *APIKeyService {
	return &APIKeyService{db: db}
}

// CreateAPIKey generates a new API key, stores its SHA256 hash, and returns the raw key (visible only once).
func (s *APIKeyService) CreateAPIKey(userID uint, name string, quotaLimit int64, rateLimitQPM int) (*model.APIKey, string, error) {
	raw := make([]byte, 32)
	if _, err := rand.Read(raw); err != nil {
		return nil, "", fmt.Errorf("generate random key: %w", err)
	}

	rawKey := "sk-" + hex.EncodeToString(raw)
	hash := sha256.Sum256([]byte(rawKey))
	keyHash := hex.EncodeToString(hash[:])

	key := &model.APIKey{
		UserID:       userID,
		KeyHash:      keyHash,
		KeyPrefix:    rawKey[:10],
		Name:         name,
		QuotaLimit:   quotaLimit,
		QuotaUsed:    0,
		RateLimitQPM: rateLimitQPM,
		IsActive:     true,
	}

	if err := s.db.Create(key).Error; err != nil {
		return nil, "", err
	}
	return key, rawKey, nil
}

// ValidateAPIKey looks up an API key by its SHA256 hash and validates it is active.
func (s *APIKeyService) ValidateAPIKey(keyHash string) (*model.APIKey, error) {
	var apikey model.APIKey
	if err := s.db.Where("key_hash = ?", keyHash).First(&apikey).Error; err != nil {
		return nil, err
	}
	return &apikey, nil
}

// DeleteAPIKey deletes an API key, verifying ownership via userID.
func (s *APIKeyService) DeleteAPIKey(id, userID uint) error {
	var key model.APIKey
	if err := s.db.Where("id = ? AND user_id = ?", id, userID).First(&key).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return fmt.Errorf("api key not found")
		}
		return err
	}
	return s.db.Delete(&model.APIKey{}, id).Error
}
