package common

import (
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
)

// TokenInfo 保存Token的信息
type TokenInfo struct {
	Token         string
	UID           uint
	TimeCreated   time.Time
	TimeActivated time.Time
}

// TokenManager 管理所有活动Token
type TokenManager struct {
	tokens map[string]*TokenInfo
	mutex  sync.RWMutex
}

func NewTokenManager() *TokenManager {
	return &TokenManager{
		tokens: make(map[string]*TokenInfo),
	}
}

func (tm *TokenManager) CreateToken(uid uint) string {
	tm.mutex.Lock()
	defer tm.mutex.Unlock()

	token := strings.Replace(uuid.New().String(), "-", "", -1)
	now := time.Now()

	tokenInfo := &TokenInfo{
		Token:         token,
		UID:           uid,
		TimeCreated:   now,
		TimeActivated: now,
	}

	tm.tokens[token] = tokenInfo

	return token
}

func (tm *TokenManager) ValidateToken(token string) (*TokenInfo, bool) {
	tm.mutex.Lock()
	defer tm.mutex.Unlock()
	//clean
	now := time.Now()
	for t, info := range tm.tokens {
		if now.After(info.TimeActivated.Add(24 * time.Hour)) {
			delete(tm.tokens, t)
		}
	}
	tokenInfo, exists := tm.tokens[token]

	if !exists {
		return nil, false
	}
	// 更新最后活动时间
	tokenInfo.TimeActivated = time.Now()
	return tokenInfo, true
}

// DeleteToken 删除指定Token
func (tm *TokenManager) DeleteToken(token string) {
	tm.mutex.Lock()
	defer tm.mutex.Unlock()
	delete(tm.tokens, token)
}
