package services

import (
	"os"
	"path/filepath"
	"testing"
)

func TestAIValidateUniqueLabelAndID(t *testing.T) {
	s := defaultSettings()
	s.AI.Connections = []AIConnection{
		{ID: "a", Label: "X", Protocol: "openai"},
		{ID: "b", Label: "X", Protocol: "openai"},
	}
	if err := s.Validate(); err == nil {
		t.Error("duplicate Label must fail validate")
	}
	s.AI.Connections = []AIConnection{
		{ID: "a", Label: "X", Protocol: "openai"},
		{ID: "a", Label: "Y", Protocol: "openai"},
	}
	if err := s.Validate(); err == nil {
		t.Error("duplicate ID must fail validate")
	}
}

func TestAIValidateProtocolAndBaseURL(t *testing.T) {
	s := defaultSettings()
	s.AI.Connections = []AIConnection{{ID: "a", Label: "X", Protocol: "grok"}}
	if err := s.Validate(); err == nil {
		t.Error("bad protocol must fail")
	}
	s.AI.Connections = []AIConnection{{ID: "a", Label: "X", Protocol: "openai", BaseURL: "localhost:11434"}}
	if err := s.Validate(); err == nil {
		t.Error("baseURL without scheme must fail")
	}
	s.AI.Connections = []AIConnection{{ID: "a", Label: "X", Protocol: "openai", BaseURL: "file:///x"}}
	if err := s.Validate(); err == nil {
		t.Error("file:// scheme must fail")
	}
	s.AI.Connections = []AIConnection{{ID: "a", Label: "X", Protocol: "openai", BaseURL: "http://[::1]:11434/v1"}}
	if err := s.Validate(); err != nil {
		t.Errorf("IPv6 baseURL should pass, got %v", err)
	}
}

func TestAIValidateDanglingDefault(t *testing.T) {
	s := defaultSettings()
	s.AI = AISettings{
		Connections: []AIConnection{{ID: "a", Label: "X", Protocol: "openai", BaseURL: "http://localhost:1/v1"}},
		Default:     "ghost",
	}
	if err := s.Validate(); err == nil {
		t.Error("dangling default must fail validate (patch path)")
	}
}

func TestDefaultConnection(t *testing.T) {
	s := defaultSettings()
	s.AI = AISettings{Connections: []AIConnection{{ID: "a", Label: "X", Protocol: "openai"}}, Default: "a"}
	if got := s.DefaultConnection(); got == nil || got.ID != "a" {
		t.Errorf("DefaultConnection = %v, want id a", got)
	}
	s.AI.Default = ""
	if s.DefaultConnection() != nil {
		t.Error("empty default -> nil (no single-conn magic)")
	}
}

func TestLoadSettingsNormalizesDanglingDefault(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "settings.json")
	raw := `{"ai":{"connections":[{"id":"a","label":"X","protocol":"openai","baseURL":"http://localhost:1/v1"}],"default":"ghost"}}`
	os.WriteFile(path, []byte(raw), 0644)
	s := LoadSettings(path)
	if s.AI.Default != "" {
		t.Errorf("dangling default should be normalized to empty, got %q", s.AI.Default)
	}
	if len(s.AI.Connections) != 1 {
		t.Errorf("connections should survive normalization, got %d", len(s.AI.Connections))
	}
}
