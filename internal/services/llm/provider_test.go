package llm

import (
	"errors"
	"testing"
)

func TestNewUnknownProtocol(t *testing.T) {
	if _, err := New(ConnectionConfig{Protocol: "grok"}); err == nil {
		t.Fatal("expected error for unknown protocol")
	}
}

func TestNewOfficialEmptyKey(t *testing.T) {
	_, err := New(ConnectionConfig{Protocol: "openai", BaseURL: "", APIKey: ""})
	if !errors.Is(err, ErrAPIKeyRequired) {
		t.Fatalf("err = %v, want ErrAPIKeyRequired", err)
	}
}

func TestNewLocalEmptyKeyOK(t *testing.T) {
	if _, err := New(ConnectionConfig{Protocol: "openai", BaseURL: "http://localhost:11434/v1", APIKey: ""}); err != nil {
		t.Fatalf("local empty key should be allowed, got %v", err)
	}
}
