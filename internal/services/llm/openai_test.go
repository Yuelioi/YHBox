package llm

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestOpenAIChat(t *testing.T) {
	var gotModel string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.HasSuffix(r.URL.Path, "/chat/completions") {
			t.Errorf("unexpected path %q", r.URL.Path)
		}
		var body map[string]any
		json.NewDecoder(r.Body).Decode(&body)
		gotModel, _ = body["model"].(string)
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"choices":[{"message":{"role":"assistant","content":"hello"}}]}`))
	}))
	defer srv.Close()

	p := newOpenAIProvider(ConnectionConfig{Protocol: "openai", BaseURL: srv.URL, APIKey: "k"})
	resp, err := p.Chat(context.Background(), ChatRequest{Model: "test-model", Messages: []Message{{Role: RoleUser, Content: "hi"}}})
	if err != nil {
		t.Fatalf("Chat err: %v", err)
	}
	if resp.Text != "hello" {
		t.Errorf("Text = %q, want hello", resp.Text)
	}
	if gotModel != "test-model" {
		t.Errorf("model sent = %q, want test-model", gotModel)
	}
}

func TestOpenAIListModels(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"object":"list","data":[{"id":"m-b","object":"model"},{"id":"m-a","object":"model"}]}`))
	}))
	defer srv.Close()

	p := newOpenAIProvider(ConnectionConfig{Protocol: "openai", BaseURL: srv.URL, APIKey: "k"})
	models, err := p.ListModels(context.Background())
	if err != nil {
		t.Fatalf("ListModels err: %v", err)
	}
	if len(models) != 2 {
		t.Fatalf("got %d models, want 2", len(models))
	}
}

func TestOpenAIChatAuthError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		w.Write([]byte(`{"error":{"message":"bad key"}}`))
	}))
	defer srv.Close()

	p := newOpenAIProvider(ConnectionConfig{Protocol: "openai", BaseURL: srv.URL, APIKey: "k"})
	_, err := p.Chat(context.Background(), ChatRequest{Model: "m", Messages: []Message{{Role: RoleUser, Content: "x"}}})
	if err == nil {
		t.Fatal("expected error")
	}
	if KindOf(err) != KindAuth {
		t.Errorf("KindOf = %q, want auth", KindOf(err))
	}
}
