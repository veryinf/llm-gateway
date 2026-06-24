package service

import (
	"fmt"

	"llm-gateway/internal/model"
	"llm-gateway/internal/provider"

	"gorm.io/gorm"
)

type RouterService struct {
	db *gorm.DB
}

func NewRouterService(db *gorm.DB) *RouterService {
	return &RouterService{db: db}
}

// ResolveProvider 根据模型名称解析到对应的 Provider
// 返回: adapter, providerModelName, passthroughLevel, error
func (s *RouterService) ResolveProvider(modelName string) (*provider.Adapter, string, string, error) {
	var userModel model.UserModel
	if err := s.db.Where("name = ? AND is_active = ?", modelName, true).First(&userModel).Error; err != nil {
		// UserModel 找不到，检查透传级别
		level := s.getPassthroughLevel()
		if level == "none" {
			return nil, "", "none", fmt.Errorf("model %q not found", modelName)
		}
		p, pmName, err := s.resolveProviderByModelName(modelName, level)
		return p, pmName, level, err
	}

	var routerEntry model.UserModelRouter
	if err := s.db.Where("user_model_id = ?", userModel.UserModelID).
		Order("priority ASC, router_id ASC").
		First(&routerEntry).Error; err != nil {
		return nil, "", "none", fmt.Errorf("no router entry for model %q", modelName)
	}

	var providerModel model.ProviderModel
	if err := s.db.Where("model_id = ? AND is_active = ?", routerEntry.ProviderModelID, true).
		First(&providerModel).Error; err != nil {
		return nil, "", "none", fmt.Errorf("provider model %d not found", routerEntry.ProviderModelID)
	}

	var prov model.Provider
	if err := s.db.Where("provider_id = ? AND is_active = ?", providerModel.ProviderID, true).
		First(&prov).Error; err != nil {
		return nil, "", "none", fmt.Errorf("provider %d not found", providerModel.ProviderID)
	}

	p, err := CreateProviderAdapter(prov)
	return p, providerModel.Name, "none", err
}

// getPassthroughLevel 获取透传级别配置
func (s *RouterService) getPassthroughLevel() string {
	var config model.Config
	if err := s.db.Where("key = ?", model.ConfigKeyRouterPassthrough).First(&config).Error; err != nil {
		return "none"
	}
	switch config.Value {
	case "user", "provider":
		return config.Value
	default:
		return "none"
	}
}

// resolveProviderByModelName 透传：直接匹配 ProviderModel
func (s *RouterService) resolveProviderByModelName(modelName, level string) (*provider.Adapter, string, error) {
	var providerModel model.ProviderModel
	if err := s.db.Where("name = ? AND is_active = ?", modelName, true).First(&providerModel).Error; err != nil {
		if level == "provider" {
			return s.resolveProviderByDefault(modelName)
		}
		return nil, "", fmt.Errorf("model %q not found in provider models", modelName)
	}

	var prov model.Provider
	if err := s.db.Where("provider_id = ? AND is_active = ?", providerModel.ProviderID, true).First(&prov).Error; err != nil {
		return nil, "", fmt.Errorf("provider %d not found", providerModel.ProviderID)
	}

	p, err := CreateProviderAdapter(prov)
	return p, providerModel.Name, err
}

// resolveProviderByDefault 二级透传：使用 default Provider
func (s *RouterService) resolveProviderByDefault(modelName string) (*provider.Adapter, string, error) {
	var prov model.Provider
	if err := s.db.Where("is_default = ? AND is_active = ?", true, true).First(&prov).Error; err != nil {
		return nil, "", fmt.Errorf("no default provider configured for model %q", modelName)
	}

	p, err := CreateProviderAdapter(prov)
	return p, modelName, err
}

// CreateProviderAdapter 根据 provider 配置创建适配器
func CreateProviderAdapter(p model.Provider) (*provider.Adapter, error) {
	return provider.NewAdapter(&p)
}
