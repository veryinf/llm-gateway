package service

import (
	"testing"

	"llm-gateway/internal/model"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

func setupTestDB(t *testing.T) *gorm.DB {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("failed to open test db: %v", err)
	}

	if err := db.AutoMigrate(&model.User{}, &model.APIKey{}); err != nil {
		t.Fatalf("failed to migrate: %v", err)
	}

	return db
}

func TestUserService_CreateDefaultAdmin(t *testing.T) {
	db := setupTestDB(t)
	svc := NewUserService(db, "test-secret")

	admin, err := svc.CreateDefaultAdmin("admin", "password123")
	if err != nil {
		t.Fatalf("failed to create admin: %v", err)
	}
	if admin == nil {
		t.Fatal("expected admin user, got nil")
	}
	if admin.Role != model.RoleAdmin {
		t.Errorf("expected role admin, got %s", admin.Role)
	}

	admin2, err := svc.CreateDefaultAdmin("admin2", "password456")
	if err != nil {
		t.Fatalf("failed on second call: %v", err)
	}
	if admin2 != nil {
		t.Error("expected nil for second call")
	}
}

func TestUserService_Login(t *testing.T) {
	db := setupTestDB(t)
	svc := NewUserService(db, "test-secret")

	_, err := svc.CreateDefaultAdmin("admin", "password123")
	if err != nil {
		t.Fatalf("failed to create admin: %v", err)
	}

	token, err := svc.Login("admin", "password123")
	if err != nil {
		t.Fatalf("login failed: %v", err)
	}
	if token == "" {
		t.Error("expected token, got empty string")
	}

	_, err = svc.Login("admin", "wrongpass")
	if err == nil {
		t.Error("expected error for wrong password")
	}

	_, err = svc.Login("nonexistent", "password")
	if err == nil {
		t.Error("expected error for non-existent user")
	}
}

func TestUserService_CreateUser(t *testing.T) {
	db := setupTestDB(t)
	svc := NewUserService(db, "test-secret")

	user, err := svc.CreateUser("testuser", "pass123", "test@example.com", "Engineering", model.RoleUser)
	if err != nil {
		t.Fatalf("failed to create user: %v", err)
	}
	if user.ID == 0 {
		t.Error("expected user ID")
	}
	if user.Role != model.RoleUser {
		t.Errorf("expected role user, got %s", user.Role)
	}

	_, err = svc.CreateUser("testuser", "pass456", "test2@example.com", "Engineering", model.RoleUser)
	if err == nil {
		t.Error("expected error for duplicate username")
	}
}

func TestUserService_ListUsers(t *testing.T) {
	db := setupTestDB(t)
	svc := NewUserService(db, "test-secret")

	_, _ = svc.CreateUser("user1", "pass1", "u1@test.com", "DeptA", model.RoleUser)
	_, _ = svc.CreateUser("user2", "pass2", "u2@test.com", "DeptB", model.RoleViewer)

	var users []model.User
	if err := db.Find(&users).Error; err != nil {
		t.Fatalf("list users failed: %v", err)
	}
	if len(users) != 2 {
		t.Errorf("expected 2 users, got %d", len(users))
	}
}

func TestUserService_UpdateUser(t *testing.T) {
	db := setupTestDB(t)
	svc := NewUserService(db, "test-secret")

	user, _ := svc.CreateUser("update-user", "pass", "up@test.com", "Dept", model.RoleUser)

	err := db.Model(&model.User{}).Where("id = ?", user.ID).Updates(map[string]interface{}{
		"department": "NewDept",
		"email":      "updated@test.com",
	}).Error
	if err != nil {
		t.Fatalf("update failed: %v", err)
	}

	updated, _ := svc.GetUser(user.ID)
	if updated.Department != "NewDept" {
		t.Errorf("expected NewDept, got %s", updated.Department)
	}
}

func TestGenerateAPIKey(t *testing.T) {
	rawKey, prefix, hash, err := GenerateAPIKey()
	if err != nil {
		t.Fatalf("GenerateAPIKey failed: %v", err)
	}
	if len(rawKey) != 67 {
		t.Errorf("expected key length 67, got %d", len(rawKey))
	}
	if prefix == "" {
		t.Error("expected prefix")
	}
	if hash == "" {
		t.Error("expected hash")
	}
}
