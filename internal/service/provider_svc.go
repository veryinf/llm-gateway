package service

import (
	"fmt"
	"log/slog"

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

type adapterEntry struct {
	id      string
	adapter provider.LLMProvider
}

// LoadProvidersFromDB loads all active providers, creates adapters per supported API type,
// and builds downstream model routing mappings.
func (s *ProviderService) LoadProvidersFromDB() error {
	var providers []model.Provider
	if err := s.db.Where("is_active = ?", true).Order("provider_id ASC").Find(&providers).Error; err != nil {
		return fmt.Errorf("list active providers: %w", err)
	}

	// adapterID -> provider_title mapping for routing
	adapterToProvider := make(map[string]string)

	for _, p := range providers {
		entries := s.createAdapters(p)
		for _, e := range entries {
			if err := s.registry.Register(e.adapter); err != nil {
				slog.Warn("skip adapter", "id", e.id, "error", err)
				continue
			}
			adapterToProvider[e.id] = p.Title
		}
	}

	// Build downstream model routing: downstream_name -> adapterID
	providerModels := make(map[string][]string)

	var upstreamModels []model.ProviderModel
	if err := s.db.Where("is_active = ?", true).Find(&upstreamModels).Error; err != nil {
		slog.Warn("list upstream models", "error", err)
	}

	// upstream_model_id -> adapterID lookup
	upstreamToAdapter := make(map[uint]string, len(upstreamModels))
	for _, m := range upstreamModels {
		for _, p := range providers {
			if p.ProviderID == m.ProviderID {
				// 使用 provider 的 preferred_api 作为默认 api_type
				apiType := p.PreferredAPI
				if apiType == "" {
					apiType = "openai"
				}
				adapterID := fmt.Sprintf("%s#%s", p.Title, apiType)
				upstreamToAdapter[m.ModelID] = adapterID
				break
			}
		}
	}

	var userModels []model.UserModel
	if err := s.db.Where("is_active = ?", true).Find(&userModels).Error; err != nil {
		slog.Warn("list user models", "error", err)
	}

	for _, dm := range userModels {
		adapterID, ok := upstreamToAdapter[dm.UpstreamModelID]
		if !ok {
			continue
		}
		providerModels[adapterID] = append(providerModels[adapterID], dm.Name)
	}

	s.modelRouter.LoadModelMap(providerModels)

	downstreamCount := len(userModels)
	slog.Info("loaded providers and user models", "providers", len(providers), "user_models", downstreamCount)
	return nil
}

// ReloadProviders clears the registry and re-loads all active providers.
func (s *ProviderService) ReloadProviders() error {
	s.modelRouter.Clear()
	s.registry.Clear()
	return s.LoadProvidersFromDB()
}

// createAdapters creates one adapter per supported API type for a provider.
func (s *ProviderService) createAdapters(p model.Provider) []adapterEntry {
	var entries []adapterEntry

	if p.SupportOpenAI {
		url := p.OpenAIBaseURL
		if url == "" {
			url = p.BaseURL + "/v1"
		}
		adapter := provider.NewOpenAICompatibleProvider(p.Title, url, p.APIKey)
		entries = append(entries, adapterEntry{
			id:      fmt.Sprintf("%s#%s", p.Title, model.APITypeOpenAI),
			adapter: adapter,
		})
	}

	if p.SupportAnthropic {
		url := p.AnthropicBaseURL
		if url == "" {
			url = p.BaseURL + "/anthropic/v1"
		}
		adapter := provider.NewAnthropicProvider(p.Title, url, p.APIKey)
		entries = append(entries, adapterEntry{
			id:      fmt.Sprintf("%s#%s", p.Title, model.APITypeAnthropic),
			adapter: adapter,
		})
	}

	if len(entries) == 0 {
		slog.Warn("provider has no supported API types", "title", p.Title)
	}

	return entries
}
