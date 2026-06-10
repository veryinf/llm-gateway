package router

import (
	"testing"

	"llm-gateway/internal/provider"
)

func TestModelRouter_Resolve(t *testing.T) {
	registry := provider.NewRegistry()
	registry.Register(provider.NewOpenAIProvider("openai", "https://api.openai.com", "test-key"))
	registry.Register(provider.NewOpenAICompatibleProvider("deepseek", "https://api.deepseek.com", "test-key"))

	router := NewModelRouter(registry)
	router.LoadModelMap(map[string][]string{
		"openai":   {"gpt-4o", "gpt-4o-mini"},
		"deepseek": {"deepseek-chat", "deepseek-reasoner"},
	})

	p, err := router.Resolve("gpt-4o")
	if err != nil {
		t.Fatalf("resolve failed: %v", err)
	}
	if p.ID() != "openai" {
		t.Errorf("expected openai, got %s", p.ID())
	}

	p, err = router.Resolve("deepseek-chat")
	if err != nil {
		t.Fatalf("resolve failed: %v", err)
	}
	if p.ID() != "deepseek" {
		t.Errorf("expected deepseek, got %s", p.ID())
	}

	// Case insensitive
	p, err = router.Resolve("GPT-4O")
	if err != nil {
		t.Fatalf("case-insensitive resolve failed: %v", err)
	}
	if p.ID() != "openai" {
		t.Errorf("expected openai for GPT-4O, got %s", p.ID())
	}

	// Passthrough: unknown model falls back to first available provider
	p, err = router.Resolve("unknown-model")
	if err != nil {
		t.Fatalf("passthrough resolve failed: %v", err)
	}
	if p.ID() != "openai" {
		t.Errorf("expected fallback to openai for unknown model, got %s", p.ID())
	}
}

func TestModelRouter_GetAllModels(t *testing.T) {
	registry := provider.NewRegistry()
	registry.Register(provider.NewOpenAIProvider("openai", "https://api.openai.com", "test-key"))

	router := NewModelRouter(registry)
	router.LoadModelMap(map[string][]string{
		"openai": {"gpt-4o", "gpt-4o-mini", "gpt-4-turbo"},
	})

	models := router.GetAllModels()
	if len(models) != 3 {
		t.Errorf("expected 3 models, got %d", len(models))
	}

	modelNames := make(map[string]bool)
	for _, m := range models {
		modelNames[m.ID] = true
	}

	for _, name := range []string{"gpt-4o", "gpt-4o-mini", "gpt-4-turbo"} {
		if !modelNames[name] {
			t.Errorf("missing model %s", name)
		}
	}
}

func TestModelRouter_Clear(t *testing.T) {
	registry := provider.NewRegistry()
	registry.Register(provider.NewOpenAIProvider("openai", "https://api.openai.com", "test-key"))

	router := NewModelRouter(registry)
	router.LoadModelMap(map[string][]string{
		"openai": {"gpt-4o"},
	})

	router.Clear()

	// After clear, modelMap is empty but fallback still works
	p, err := router.Resolve("gpt-4o")
	if err != nil {
		t.Fatalf("passthrough should still work after clear: %v", err)
	}
	if p.ID() != "openai" {
		t.Errorf("expected fallback to openai after clear, got %s", p.ID())
	}

	models := router.GetAllModels()
	if len(models) != 0 {
		t.Errorf("expected 0 models after clear, got %d", len(models))
	}

	// No provider at all should error
	emptyRegistry := provider.NewRegistry()
	emptyRouter := NewModelRouter(emptyRegistry)
	_, err = emptyRouter.Resolve("anything")
	if err == nil {
		t.Error("expected error when no provider available")
	}
}
