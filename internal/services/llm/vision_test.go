package llm

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestOpenAIChat_Vision(t *testing.T) {
	var gotContent any
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body map[string]any
		json.NewDecoder(r.Body).Decode(&body)
		if msgs, ok := body["messages"].([]any); ok && len(msgs) > 0 {
			m, _ := msgs[len(msgs)-1].(map[string]any)
			gotContent = m["content"]
		}
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"choices":[{"message":{"content":"ok"}}]}`))
	}))
	defer srv.Close()

	p := newOpenAIProvider(ConnectionConfig{Protocol: "openai", BaseURL: srv.URL, APIKey: "k"})
	_, err := p.Chat(context.Background(), ChatRequest{Model: "m", Messages: []Message{{
		Role: RoleUser, Content: "what is this", Images: []ImageData{{Format: "png", Data: []byte("PNGDATA")}},
	}}})
	if err != nil {
		t.Fatalf("Chat err: %v", err)
	}
	arr, ok := gotContent.([]any)
	if !ok || len(arr) < 2 {
		t.Fatalf("vision content should be a parts array (text+image), got %T", gotContent)
	}
	raw, _ := json.Marshal(arr)
	if !strings.Contains(string(raw), "image_url") || !strings.Contains(string(raw), "data:image/png;base64,") {
		t.Errorf("missing image_url data-uri part: %s", raw)
	}
}

func TestAnthropicChat_Vision(t *testing.T) {
	var gotContent any
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body map[string]any
		json.NewDecoder(r.Body).Decode(&body)
		if msgs, ok := body["messages"].([]any); ok && len(msgs) > 0 {
			m, _ := msgs[0].(map[string]any)
			gotContent = m["content"]
		}
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"content":[{"type":"text","text":"ok"}]}`))
	}))
	defer srv.Close()

	p := newAnthropicProvider(ConnectionConfig{Protocol: "anthropic", BaseURL: srv.URL, APIKey: "k"})
	_, err := p.Chat(context.Background(), ChatRequest{Model: "claude-x", Messages: []Message{{
		Role: RoleUser, Content: "what is this", Images: []ImageData{{Format: "jpeg", Data: []byte("JPEGDATA")}},
	}}})
	if err != nil {
		t.Fatalf("Chat err: %v", err)
	}
	raw, _ := json.Marshal(gotContent)
	if !strings.Contains(string(raw), `"type":"image"`) ||
		!strings.Contains(string(raw), "image/jpeg") ||
		!strings.Contains(string(raw), "base64") {
		t.Errorf("missing anthropic base64 image block: %s", raw)
	}
}
