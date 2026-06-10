package service

import (
	"crypto/sha256"
	"encoding/hex"
	"testing"

	"llm-gateway/internal/model"

	"gorm.io/gorm"
)

func setupAPIKeyDB(t *testing.T) (*gorm.DB, *UserService, *APIKeyService) {
	db := setupTestDB(t)
	userSvc := NewUserService(db, "test-secret")
	apiKeySvc := NewAPIKeyService(db)

	_, err := userSvc.CreateUser("testuser", "password", "测试", "13800138000", "Dept", model.RoleUser)
	if err != nil {
		t.Fatalf("failed to create test user: %v", err)
	}

	return db, userSvc, apiKeySvc
}

func TestAPIKeyService_CreateAPIKey(t *testing.T) {
	_, _, svc := setupAPIKeyDB(t)

	apiKey, rawKey, err := svc.CreateAPIKey(1, "dev-key", 1000000, 60)
	if err != nil {
		t.Fatalf("failed to create api key: %v", err)
	}

	if apiKey.ID == 0 {
		t.Error("expected api key ID")
	}
	if len(rawKey) != 67 {
		t.Errorf("expected key length 67, got %d", len(rawKey))
	}
	if apiKey.Name != "dev-key" {
		t.Errorf("expected name 'dev-key', got %s", apiKey.Name)
	}
	if apiKey.QuotaLimit != 1000000 {
		t.Errorf("expected quota 1000000, got %d", apiKey.QuotaLimit)
	}
}

func TestAPIKeyService_ValidateAPIKey(t *testing.T) {
	_, _, svc := setupAPIKeyDB(t)

	apiKey, rawKey, _ := svc.CreateAPIKey(1, "test-key", 0, 60)

	h := sha256.Sum256([]byte(rawKey))
	keyHash := hex.EncodeToString(h[:])

	validated, err := svc.ValidateAPIKey(keyHash)
	if err != nil {
		t.Fatalf("validate failed: %v", err)
	}
	if validated.ID != apiKey.ID {
		t.Errorf("expected id %d, got %d", apiKey.ID, validated.ID)
	}

	_, err = svc.ValidateAPIKey("invalid-hash")
	if err == nil {
		t.Error("expected error for invalid hash")
	}
}

func TestAPIKeyService_ListAPIKeys(t *testing.T) {
	db, _, svc := setupAPIKeyDB(t)

	_, _, _ = svc.CreateAPIKey(1, "key-1", 0, 60)
	_, _, _ = svc.CreateAPIKey(1, "key-2", 0, 60)

	var keys []model.APIKey
	if err := db.Where("user_id = ?", 1).Find(&keys).Error; err != nil {
		t.Fatalf("list keys failed: %v", err)
	}
	if len(keys) != 2 {
		t.Errorf("expected 2 keys, got %d", len(keys))
	}

	var keys2 []model.APIKey
	if err := db.Where("user_id = ?", 999).Find(&keys2).Error; err != nil {
		t.Fatalf("list keys failed: %v", err)
	}
	if len(keys2) != 0 {
		t.Errorf("expected 0 keys, got %d", len(keys2))
	}
}

func TestAPIKeyService_DeleteAPIKey(t *testing.T) {
	db, _, svc := setupAPIKeyDB(t)

	apiKey, _, _ := svc.CreateAPIKey(1, "delete-me", 0, 60)

	err := svc.DeleteAPIKey(apiKey.ID, 999)
	if err == nil {
		t.Error("expected error for wrong user")
	}

	err = svc.DeleteAPIKey(apiKey.ID, 1)
	if err != nil {
		t.Fatalf("delete failed: %v", err)
	}

	var keys []model.APIKey
	if err := db.Where("user_id = ?", 1).Find(&keys).Error; err != nil {
		t.Fatalf("list keys failed: %v", err)
	}
	if len(keys) != 0 {
		t.Error("expected empty key list after deletion")
	}
}
