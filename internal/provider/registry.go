package provider

import (
	"fmt"
	"strings"
	"sync"
)

// Registry Provider 注册中心，管理所有 Provider 实例
type Registry struct {
	mu        sync.RWMutex
	providers map[string]LLMProvider // key: provider name (lowercase)
}

// NewRegistry 创建注册中心
func NewRegistry() *Registry {
	return &Registry{
		providers: make(map[string]LLMProvider),
	}
}

// Register 注册 provider
func (r *Registry) Register(p LLMProvider) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	key := strings.ToLower(p.ID())
	if _, exists := r.providers[key]; exists {
		return fmt.Errorf("provider %q already registered", p.ID())
	}
	r.providers[key] = p
	return nil
}

// Get 按名称获取 provider
func (r *Registry) Get(name string) (LLMProvider, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	p, ok := r.providers[strings.ToLower(name)]
	return p, ok
}

// List 列出所有已注册的 provider
func (r *Registry) List() []LLMProvider {
	r.mu.RLock()
	defer r.mu.RUnlock()

	providers := make([]LLMProvider, 0, len(r.providers))
	for _, p := range r.providers {
		providers = append(providers, p)
	}
	return providers
}

// Clear 清空所有已注册的 provider
func (r *Registry) Clear() {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.providers = make(map[string]LLMProvider)
}

// FindByModel 根据模型名称查找对应的 provider
// modelMap: model name (lowercase) -> provider name (lowercase)
func (r *Registry) FindByModel(modelName string, modelMap map[string]string) (LLMProvider, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	providerName, ok := modelMap[strings.ToLower(modelName)]
	if !ok {
		return nil, false
	}

	p, ok := r.providers[strings.ToLower(providerName)]
	return p, ok
}
