package service

import (
	"fmt"

	"llm-gateway/internal/model"
	"llm-gateway/internal/provider"

	"gorm.io/gorm"
)

// RouterResult 模型解析结果
type RouterResult struct {
	Adapter           *provider.Adapter
	UserModelName     string
	ProviderModelName string                 // Provider 模型名称
	Level             model.PassthroughLevel //实际生效的透传级别
}

type RouterService struct {
	db *gorm.DB
}

func NewRouterService(db *gorm.DB) *RouterService {
	return &RouterService{db: db}
}

// ResolveProvider 根据模型名称解析到对应的 Provider
func (s *RouterService) ResolveProvider(inputModel string) (*RouterResult, error) {
	level := s.getPassthroughLevel()

	var userModel model.UserModel
	if err := s.db.Where("name = ? AND is_active = ?", inputModel, true).First(&userModel).Error; err != nil {
		if level == model.PassthroughLevelNone {
			return nil, fmt.Errorf("model %q not found", inputModel)
		}
		return s.resolveByPassthrough(inputModel, level)
	}

	var userModelRouter model.UserModelRouter
	if err := s.db.Where("user_model_id = ? AND is_active = ?", userModel.UserModelID, true).Order("priority ASC, router_id ASC").First(&userModelRouter).Error; err != nil {
		if level == model.PassthroughLevelNone {
			return nil, fmt.Errorf("no router entry for model %q", inputModel)
		}
		return s.resolveByPassthrough(inputModel, level)
	}

	var providerModel model.ProviderModel
	if err := s.db.Where("model_id = ? AND is_active = ?", userModelRouter.ProviderModelID, true).First(&providerModel).Error; err != nil {
		if level != model.PassthroughLevelProvider {
			return nil, fmt.Errorf("provider model %d not found", userModelRouter.ProviderModelID)
		}
		return s.resolveByDefaultProvider(inputModel)
	}

	var prov model.Provider
	if err := s.db.Where("provider_id = ? AND is_active = ?", providerModel.ProviderID, true).First(&prov).Error; err != nil {
		if level != model.PassthroughLevelProvider {
			return nil, fmt.Errorf("provider %d not found", providerModel.ProviderID)
		}
		return s.resolveByDefaultProvider(inputModel)
	}

	adapter, err := provider.NewAdapter(&prov)
	if err != nil {
		return nil, err
	}

	return &RouterResult{
		Adapter:           adapter,
		UserModelName:     inputModel,
		ProviderModelName: providerModel.Name,
		Level:             model.PassthroughLevelNone,
	}, nil
}

// getPassthroughLevel 获取透传级别配置
func (s *RouterService) getPassthroughLevel() model.PassthroughLevel {
	level := GetConfigStringOrDefault(model.ConfigKeyRouterPassthrough, "none")
	switch level {
	case "user", "provider":
		return model.PassthroughLevel(level)
	default:
		return model.PassthroughLevelNone
	}
}

// resolveByPassthrough 透传模式：直接匹配 ProviderModel
func (s *RouterService) resolveByPassthrough(inputModel string, level model.PassthroughLevel) (*RouterResult, error) {
	var providerModel model.ProviderModel
	if err := s.db.Where("name = ? AND is_active = ?", inputModel, true).First(&providerModel).Error; err != nil {
		if level != model.PassthroughLevelProvider {
			return nil, fmt.Errorf("model %q not found in provider models", inputModel)
		}
		return s.resolveByDefaultProvider(inputModel)
	}

	var prov model.Provider
	if err := s.db.Where("provider_id = ? AND is_active = ?", providerModel.ProviderID, true).First(&prov).Error; err != nil {
		if level != model.PassthroughLevelProvider {
			return nil, fmt.Errorf("provider %d not found", providerModel.ProviderID)
		}
		return s.resolveByDefaultProvider(inputModel)
	}

	adapter, err := provider.NewAdapter(&prov)
	if err != nil {
		return nil, err
	}

	return &RouterResult{
		Adapter:           adapter,
		UserModelName:     inputModel,
		ProviderModelName: providerModel.Name,
		Level:             model.PassthroughLevelUser,
	}, nil
}

// resolveByDefaultProvider 二级透传：使用 default Provider
func (s *RouterService) resolveByDefaultProvider(inputModel string) (*RouterResult, error) {
	var prov model.Provider
	if err := s.db.Where("is_default = ? AND is_active = ?", true, true).First(&prov).Error; err != nil {
		return nil, fmt.Errorf("no default provider configured for model %q", inputModel)
	}

	adapter, err := provider.NewAdapter(&prov)
	if err != nil {
		return nil, err
	}

	return &RouterResult{
		Adapter:           adapter,
		UserModelName:     inputModel,
		ProviderModelName: inputModel,
		Level:             model.PassthroughLevelProvider,
	}, nil
}
