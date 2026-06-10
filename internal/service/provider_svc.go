package service

import (
	"fmt"
	"log"

	"llm-gateway/internal/model"
	"llm-gateway/internal/provider"
	"llm-gateway/internal/router"

	"gorm.io/gorm"
)

type ProviderService struct {
	db          *gorm.DB
	registry    *provider.Registry
	modelRouter *router.ModelRouter
}

func NewProviderService(db *gorm.DB, registry *provider.Registry, modelRouter *router.ModelRouter) *ProviderService {
	return &ProviderService{
		db:          db,
		registry:    registry,
		modelRouter: modelRouter,
	}
}

// LoadProvidersFromDB loads all active providers from the database, creates adapters,
// registers them, and sets up model routing mappings.
func (s *ProviderService) LoadProvidersFromDB() error {
	var providers []model.Provider
	if err := s.db.Where("is_active = ?", true).Order("priority desc").Find(&providers).Error; err != nil {
		return fmt.Errorf("list active providers: %w", err)
	}

	providerModels := make(map[string][]string)

	for _, p := range providers {
		adapter, err := s.createAdapter(p)
		if err != nil {
			log.Printf("skip provider %s: %v", p.Name, err)
			continue
		}

		if err := s.registry.Register(adapter); err != nil {
			log.Printf("skip provider %s: %v", p.Name, err)
			continue
		}

		var models []model.Model
		if err := s.db.Where("provider_id = ? AND is_active = ?", p.ID, true).Find(&models).Error; err != nil {
			log.Printf("list models for provider %s: %v", p.Name, err)
			continue
		}

		modelNames := make([]string, len(models))
		for i, m := range models {
			modelNames[i] = m.Name
		}
		providerModels[p.Name] = modelNames
	}

	s.modelRouter.LoadModelMap(providerModels)
	log.Printf("loaded %d providers", len(providers))
	return nil
}

// ReloadProviders clears the registry and re-loads all active providers.
func (s *ProviderService) ReloadProviders() error {
	s.modelRouter.Clear()
	s.registry.Clear()
	return s.LoadProvidersFromDB()
}

// createAdapter creates a provider adapter based on the provider type.
func (s *ProviderService) createAdapter(p model.Provider) (provider.LLMProvider, error) {
	var adapter provider.LLMProvider

	switch p.Type {
	case model.ProviderTypeOpenAI:
		adapter = provider.NewOpenAIProvider(p.Name, p.BaseURL, p.APIKey)
	case model.ProviderTypeAnthropic:
		adapter = provider.NewAnthropicProvider(p.Name, p.BaseURL, p.APIKey)
	case model.ProviderTypeOpenAICompatible, model.ProviderTypeAzure, model.ProviderTypeOllama:
		adapter = provider.NewOpenAICompatibleProvider(p.Name, p.BaseURL, p.APIKey)
	default:
		return nil, fmt.Errorf("unsupported provider type: %s", p.Type)
	}

	if p.RateLimitQPM > 0 {
		adapter = provider.NewRateLimitedProvider(adapter, p.RateLimitQPM, p.RateLimitBurst)
	}

	return adapter, nil
}
