package services

import "testing"

func aiTestSettings() *Settings {
	return &Settings{AI: AISettings{
		Connections: []AIConnection{
			{ID: "c1", Label: "L1", Protocol: "openai", BaseURL: "http://local/v1", APIKey: "k"},
		},
		Default: "c1",
	}}
}

func TestAIProviderCache_HitAndFingerprintRebuild(t *testing.T) {
	s := aiTestSettings()
	cache := NewAIProviderCache(func() *Settings { return s })

	p1, err := cache.Provider("c1")
	if err != nil {
		t.Fatalf("Provider(c1): %v", err)
	}
	p2, err := cache.Provider("c1")
	if err != nil {
		t.Fatalf("Provider(c1) again: %v", err)
	}
	if p1 != p2 {
		t.Error("expected cache hit (same provider instance) when fingerprint unchanged")
	}

	// 改 endpoint → 指纹变 → 重建新实例。
	s.AI.Connections[0].BaseURL = "http://other/v1"
	p3, err := cache.Provider("c1")
	if err != nil {
		t.Fatalf("Provider(c1) after change: %v", err)
	}
	if p3 == p1 {
		t.Error("expected rebuild (new provider instance) after fingerprint change")
	}
}

func TestAIProviderCache_DefaultAndNotFound(t *testing.T) {
	s := aiTestSettings()
	cache := NewAIProviderCache(func() *Settings { return s })

	if _, err := cache.Provider(""); err != nil {
		t.Errorf("empty id should resolve ai.default, got err: %v", err)
	}
	if _, err := cache.Provider("ghost"); err == nil {
		t.Error("unknown connectionID should error (notFound)")
	}
}

func TestAIProviderCache_LabelChangeDoesNotRebuild(t *testing.T) {
	s := aiTestSettings()
	cache := NewAIProviderCache(func() *Settings { return s })

	p1, _ := cache.Provider("c1")
	// 改 Label(非 llm 相关字段)不该使指纹变、不该重建。
	s.AI.Connections[0].Label = "renamed"
	p2, _ := cache.Provider("c1")
	if p1 != p2 {
		t.Error("Label change must not evict the cached provider (fingerprint excludes Label)")
	}
}
