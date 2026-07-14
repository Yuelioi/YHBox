package services

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestConnectionListsModels(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"object":"list","data":[{"id":"m-b","object":"model"},{"id":"m-a","object":"model"}]}`))
	}))
	defer srv.Close()
	res := NewAIService().TestConnection(TestConnReq{
		Connection: AIConnection{Protocol: "openai", BaseURL: srv.URL},
		APIKey:     "k",
	})
	if !res.Ok {
		t.Fatalf("want ok, err=%s", res.Error)
	}
	if len(res.Models) != 2 || res.Models[0] != "m-a" {
		t.Errorf("models not sorted/returned: %v", res.Models)
	}
}

func TestConnectionFallbackChat(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasSuffix(r.URL.Path, "/models") {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"choices":[{"message":{"content":"ok"}}]}`))
	}))
	defer srv.Close()
	res := NewAIService().TestConnection(TestConnReq{
		Connection: AIConnection{Protocol: "openai", BaseURL: srv.URL},
		TestModel:  "m",
		APIKey:     "k",
	})
	if !res.Ok {
		t.Fatalf("fallback chat should pass, err=%s", res.Error)
	}
}

func TestConnectionNoModelNoFallback(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer srv.Close()
	res := NewAIService().TestConnection(TestConnReq{
		Connection: AIConnection{Protocol: "openai", BaseURL: srv.URL},
		APIKey:     "k",
	})
	if res.Ok {
		t.Error("no models + no testModel -> not ok")
	}
	if res.Error == "" {
		t.Error("should carry guidance error")
	}
}

func TestConnectionUsesStoredCredentialWithoutReturningIt(t *testing.T) {
	store := newFakeSecretStore()
	secrets := NewAISecrets(store)
	if err := secrets.Set("primary", "stored-secret"); err != nil {
		t.Fatal(err)
	}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, request *http.Request) {
		if got := request.Header.Get("Authorization"); got != "Bearer stored-secret" {
			t.Errorf("Authorization = %q", got)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"object":"list","data":[]}`))
	}))
	defer server.Close()

	service := NewAIService(secrets)
	result := service.TestConnection(TestConnReq{Connection: AIConnection{
		ID: "primary", Protocol: "openai", BaseURL: server.URL,
	}})
	if !result.Ok {
		t.Fatalf("stored credential test failed: %s", result.Error)
	}
	status := service.SecretStatus([]string{"primary", "missing"})
	if !status["primary"] || status["missing"] {
		t.Fatalf("presence metadata = %#v", status)
	}
}
