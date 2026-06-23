package llm

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

var testSchema = JSONSchema{Fields: []SchemaField{
	{Name: "name", JSONType: "string"},
	{Name: "count", JSONType: "integer"},
}}

func TestOpenAIChatStructured_Native(t *testing.T) {
	var gotRF map[string]any
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body map[string]any
		json.NewDecoder(r.Body).Decode(&body)
		gotRF, _ = body["response_format"].(map[string]any)
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"choices":[{"message":{"content":"{\"name\":\"bob\",\"count\":3}"}}]}`))
	}))
	defer srv.Close()

	p := newOpenAIProvider(ConnectionConfig{Protocol: "openai", BaseURL: srv.URL, APIKey: "k"})
	_, fields, err := p.ChatStructured(context.Background(),
		ChatRequest{Model: "m", Messages: []Message{{Role: RoleUser, Content: "hi"}}}, testSchema, ModeNative)
	if err != nil {
		t.Fatalf("ChatStructured err: %v", err)
	}
	if gotRF["type"] != "json_schema" {
		t.Errorf("response_format.type = %v, want json_schema", gotRF["type"])
	}
	js, _ := gotRF["json_schema"].(map[string]any)
	sc, _ := js["schema"].(map[string]any)
	if sc["additionalProperties"] != false {
		t.Errorf("strict schema must set additionalProperties:false, got %v", sc["additionalProperties"])
	}
	if fields["name"] != "bob" || fields["count"].(float64) != 3 {
		t.Errorf("fields = %v, want name=bob count=3", fields)
	}
}

func TestOpenAIChatStructured_NativeUnsupported(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(`{"error":{"message":"response_format not supported"}}`))
	}))
	defer srv.Close()

	p := newOpenAIProvider(ConnectionConfig{Protocol: "openai", BaseURL: srv.URL, APIKey: "k"})
	_, _, err := p.ChatStructured(context.Background(),
		ChatRequest{Model: "m", Messages: []Message{{Role: RoleUser, Content: "hi"}}}, testSchema, ModeNative)
	if KindOf(err) != KindUnsupported {
		t.Errorf("native 400 should map to KindUnsupported, got %v (%v)", KindOf(err), err)
	}
}

func TestAnthropicChatStructured_Native(t *testing.T) {
	var hasToolChoice bool
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body map[string]any
		json.NewDecoder(r.Body).Decode(&body)
		_, hasToolChoice = body["tool_choice"]
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"content":[{"type":"tool_use","name":"result","input":{"name":"bob","count":3}}]}`))
	}))
	defer srv.Close()

	p := newAnthropicProvider(ConnectionConfig{Protocol: "anthropic", BaseURL: srv.URL, APIKey: "k"})
	_, fields, err := p.ChatStructured(context.Background(),
		ChatRequest{Model: "claude-x", Messages: []Message{{Role: RoleUser, Content: "hi"}}}, testSchema, ModeNative)
	if err != nil {
		t.Fatalf("ChatStructured err: %v", err)
	}
	if !hasToolChoice {
		t.Error("native Anthropic structured must send tool_choice")
	}
	if fields["name"] != "bob" || fields["count"].(float64) != 3 {
		t.Errorf("fields = %v, want name=bob count=3", fields)
	}
}

func TestAnthropicChatStructured_NoToolUseFails(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"content":[{"type":"text","text":"sorry, plain text"}]}`))
	}))
	defer srv.Close()

	p := newAnthropicProvider(ConnectionConfig{Protocol: "anthropic", BaseURL: srv.URL, APIKey: "k"})
	resp, fields, err := p.ChatStructured(context.Background(),
		ChatRequest{Model: "claude-x", Messages: []Message{{Role: RoleUser, Content: "hi"}}}, testSchema, ModeNative)
	if err == nil || fields != nil {
		t.Fatalf("no tool_use should fail; got fields=%v err=%v", fields, err)
	}
	if resp.Text != "sorry, plain text" {
		t.Errorf("raw text should be carried back for Fail, got %q", resp.Text)
	}
}

func TestChatStructured_PromptMode(t *testing.T) {
	var gotSystem string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body map[string]any
		json.NewDecoder(r.Body).Decode(&body)
		if msgs, ok := body["messages"].([]any); ok && len(msgs) > 0 {
			if m0, ok := msgs[0].(map[string]any); ok {
				gotSystem, _ = m0["content"].(string)
			}
		}
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte("{\"choices\":[{\"message\":{\"content\":\"```json\\n{\\\"name\\\":\\\"bob\\\",\\\"count\\\":3}\\n```\"}}]}"))
	}))
	defer srv.Close()

	p := newOpenAIProvider(ConnectionConfig{Protocol: "openai", BaseURL: srv.URL, APIKey: "k"})
	_, fields, err := p.ChatStructured(context.Background(),
		ChatRequest{Model: "m", Messages: []Message{{Role: RoleUser, Content: "hi"}}}, testSchema, ModePrompt)
	if err != nil {
		t.Fatalf("prompt-mode err: %v", err)
	}
	if !strings.Contains(gotSystem, "JSON") {
		t.Errorf("prompt mode must inject a JSON instruction into system, got %q", gotSystem)
	}
	if fields["name"] != "bob" || fields["count"].(float64) != 3 {
		t.Errorf("fields (fence-stripped) = %v, want name=bob count=3", fields)
	}
}

func TestParseStructuredText(t *testing.T) {
	cases := []struct {
		raw     string
		wantErr bool
		key     string
	}{
		{`{"x":1}`, false, "x"},
		{"```json\n{\"y\":2}\n```", false, "y"},
		{"prefix {\"z\":3} suffix", false, "z"},
		{"no json here", true, ""},
	}
	for _, c := range cases {
		m, err := parseStructuredText(c.raw)
		if c.wantErr {
			if err == nil {
				t.Errorf("%q: want err", c.raw)
			}
			continue
		}
		if err != nil || m[c.key] == nil {
			t.Errorf("%q: parsed=%v err=%v", c.raw, m, err)
		}
	}
}
