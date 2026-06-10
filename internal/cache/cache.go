package cache

import (
	"log"
	"sync"
	"time"

	"llm-gateway/internal/model"

	"gorm.io/gorm"
)

// GatewayCache 内存缓存，减少高频查询的数据库 I/O
type GatewayCache struct {
	db *gorm.DB

	mu        sync.RWMutex
	apiKeys   map[string]*model.APIKey // keyHash -> APIKey
	users     map[uint]*model.User     // userID -> User
	providers map[string]*model.Provider // name -> Provider

	refreshInterval time.Duration
	stopCh          chan struct{}
}

func New(db *gorm.DB, refreshInterval time.Duration) *GatewayCache {
	return &GatewayCache{
		db:              db,
		apiKeys:         make(map[string]*model.APIKey),
		users:           make(map[uint]*model.User),
		providers:       make(map[string]*model.Provider),
		refreshInterval: refreshInterval,
		stopCh:          make(chan struct{}),
	}
}

// Start 启动定期刷新
func (c *GatewayCache) Start() {
	c.refresh()
	go c.loop()
}

// Stop 停止刷新
func (c *GatewayCache) Stop() {
	close(c.stopCh)
}

func (c *GatewayCache) loop() {
	ticker := time.NewTicker(c.refreshInterval)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			c.refresh()
		case <-c.stopCh:
			return
		}
	}
}

func (c *GatewayCache) refresh() {
	c.loadUsers()
	c.loadAPIKeys()
	c.loadProviders()
	log.Printf("cache refreshed: users=%d api_keys=%d providers=%d",
		len(c.users), len(c.apiKeys), len(c.providers))
}

func (c *GatewayCache) loadUsers() {
	var users []model.User
	if err := c.db.Find(&users).Error; err != nil {
		log.Printf("cache: failed to load users: %v", err)
		return
	}

	m := make(map[uint]*model.User, len(users))
	for i := range users {
		m[users[i].ID] = &users[i]
	}

	c.mu.Lock()
	c.users = m
	c.mu.Unlock()
}

func (c *GatewayCache) loadAPIKeys() {
	var keys []model.APIKey
	if err := c.db.Where("is_active = ?", true).Find(&keys).Error; err != nil {
		log.Printf("cache: failed to load api keys: %v", err)
		return
	}

	m := make(map[string]*model.APIKey, len(keys))
	for i := range keys {
		m[keys[i].KeyHash] = &keys[i]
	}

	c.mu.Lock()
	c.apiKeys = m
	c.mu.Unlock()
}

func (c *GatewayCache) loadProviders() {
	var providers []model.Provider
	if err := c.db.Where("is_active = ?", true).Find(&providers).Error; err != nil {
		log.Printf("cache: failed to load providers: %v", err)
		return
	}

	m := make(map[string]*model.Provider, len(providers))
	for i := range providers {
		m[providers[i].Name] = &providers[i]
	}

	c.mu.Lock()
	c.providers = m
	c.mu.Unlock()
}

// GetAPIKey 根据 keyHash 从缓存获取 API Key
func (c *GatewayCache) GetAPIKey(keyHash string) *model.APIKey {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.apiKeys[keyHash]
}

// GetUser 根据 ID 从缓存获取用户
func (c *GatewayCache) GetUser(userID uint) *model.User {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.users[userID]
}

// GetProvider 根据名称从缓存获取 Provider
func (c *GatewayCache) GetProvider(name string) *model.Provider {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.providers[name]
}
