package llm

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestAnthropicChatDefaultsMaxTokens(t *testing.T) {
	var gotMax float64 = -1
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body map[string]any
		json.NewDecoder(r.Body).Decode(&body)
		if v, ok := body["max_tokens"].(float64); ok {
			gotMax = v
		}
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"content":[{"type":"text","text":"hi there"}]}`))
	}))
	defer srv.Close()

	p := newAnthropicProvider(ConnectionConfig{Protocol: "anthropic", BaseURL: srv.URL, APIKey: "k"})
	resp, err := p.Chat(context.Background(), ChatRequest{Model: "claude-x", Messages: []Message{{Role: RoleUser, Content: "hi"}}})
	if err != nil {
		t.Fatalf("Chat err: %v", err)
	}
	if resp.Text != "hi there" {
		t.Errorf("Text = %q", resp.Text)
	}
	if gotMax != 1024 {
		t.Errorf("max_tokens sent = %v, want 1024 (API requires it, 0 -> default)", gotMax)
	}
}

func TestAnthropicListModels(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.HasSuffix(r.URL.Path, "/models") {
			t.Errorf("unexpected path %q", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"data":[{"id":"claude-x","type":"model"}],"has_more":false}`))
	}))
	defer srv.Close()

	p := newAnthropicProvider(ConnectionConfig{Protocol: "anthropic", BaseURL: srv.URL, APIKey: "k"})
	models, err := p.ListModels(context.Background())
	if err != nil {
		t.Fatalf("ListModels err: %v", err)
	}
	if len(models) != 1 || models[0] != "claude-x" {
		t.Errorf("models = %v, want [claude-x]", models)
	}
}
