package service

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"time"

	"llm-gateway/internal/model"

	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

type UserService struct {
	db        *gorm.DB
	jwtSecret string
}

func NewUserService(db *gorm.DB, jwtSecret string) *UserService {
	return &UserService{db: db, jwtSecret: jwtSecret}
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

	return s.CreateUser(username, password, "", "", model.RoleAdmin)
}

// Login validates username and password, returns a JWT token on success.
func (s *UserService) Login(username, password string) (string, error) {
	var user model.User
	if err := s.db.Where("username = ?", username).First(&user).Error; err != nil {
		return "", err
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(password)); err != nil {
		return "", err
	}

	claims := jwt.MapClaims{
		"user_id": user.ID,
		"role":    user.Role,
		"exp":     time.Now().Add(24 * time.Hour).Unix(),
		"iat":     time.Now().Unix(),
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(s.jwtSecret))
}

// CreateUser creates a new user with a bcrypt-hashed password.
func (s *UserService) CreateUser(username, password, email, department string, role model.Role) (*model.User, error) {
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return nil, err
	}

	user := &model.User{
		Username:     username,
		PasswordHash: string(hash),
		Email:        email,
		Department:   department,
		Role:         role,
		IsActive:     true,
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

// GenerateAPIKey generates a random API key with sk- prefix, SHA256 hash, and prefix for display.
// Returns rawKey (full key to give to user), prefix (for display), hash (for storage).
func GenerateAPIKey() (rawKey, prefix, hash string, err error) {
	b := make([]byte, 32)
	if _, err = rand.Read(b); err != nil {
		return "", "", "", err
	}

	rawKey = "sk-" + hex.EncodeToString(b)
	prefix = rawKey[:16] + "..."

	h := sha256.Sum256([]byte(rawKey))
	hash = hex.EncodeToString(h[:])

	return rawKey, prefix, hash, nil
}
