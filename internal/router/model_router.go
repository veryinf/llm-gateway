package router

import (
	"fmt"
	"strings"
	"sync"

	"llm-gateway/internal/provider"
)

// ProviderModels 单个 provider 对应的模型列表
type ProviderModels struct {
	ProviderName string
	Models       []string
}

// ModelRouter 模型路由器，根据模型名称路由到对应的 Provider
type ModelRouter struct {
	registry *provider.Registry
	mu       sync.RWMutex
	modelMap map[string]string // model name -> provider name
}

// NewModelRouter 创建模型路由器
func NewModelRouter(registry *provider.Registry) *ModelRouter {
	return &ModelRouter{
		registry: registry,
		modelMap: make(map[string]string),
	}
}

// LoadModelMap 加载 provider→models 映射，自动构建反向索引（model→provider）
func (r *ModelRouter) LoadModelMap(providerModels map[string][]string) {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.modelMap = make(map[string]string)

	for providerName, models := range providerModels {
		for _, modelName := range models {
			r.modelMap[strings.ToLower(modelName)] = strings.ToLower(providerName)
		}
	}
}

// Clear 清空所有模型路由映射
func (r *ModelRouter) Clear() {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.modelMap = make(map[string]string)
}

// Resolve 根据模型名称解析到对应的 Provider
// 如果模型未在路由表中注册，则 fallback 到第一个可用的 Provider（透传模式）
func (r *ModelRouter) Resolve(modelName string) (provider.LLMProvider, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	providerName, ok := r.modelMap[strings.ToLower(modelName)]
	if ok {
		p, ok := r.registry.Get(providerName)
		if ok {
			return p, nil
		}
	}

	// Passthrough: fallback to first available provider
	providers := r.registry.List()
	if len(providers) == 0 {
		return nil, fmt.Errorf("no provider available")
	}
	return providers[0], nil
}

// GetAllModels 获取所有已注册的模型列表
func (r *ModelRouter) GetAllModels() []provider.ModelInfo {
	r.mu.RLock()
	defer r.mu.RUnlock()

	// 按 provider 去重，避免重复的 model name
	seen := make(map[string]bool)
	var models []provider.ModelInfo

	for modelName := range r.modelMap {
		lower := strings.ToLower(modelName)
		if seen[lower] {
			continue
		}
		seen[lower] = true
		models = append(models, provider.ModelInfo{
			ID:     modelName,
			Object: "model",
		})
	}

	return models
}

// GetModelMap 返回当前的 model→provider 映射（只读副本）
func (r *ModelRouter) GetModelMap() map[string]string {
	r.mu.RLock()
	defer r.mu.RUnlock()

	m := make(map[string]string, len(r.modelMap))
	for k, v := range r.modelMap {
		m[k] = v
	}
	return m
}
