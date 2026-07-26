package browsercdp

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestCanonicalEndpointRejectsNonLoopbackAndAmbientURLParts(t *testing.T) {
	for _, raw := range []string{
		"http://example.com:9222", "https://127.0.0.1:9222", "http://localhost:9222",
		"http://127.0.0.1:9222/path", "http://user@127.0.0.1:9222", "http://127.0.0.1:9222?target=x",
	} {
		if _, err := CanonicalEndpoint(raw); err == nil {
			t.Fatalf("CanonicalEndpoint(%q) accepted unsafe authority", raw)
		}
	}
	got, err := CanonicalEndpoint("127.0.0.1:9333")
	if err != nil || got != "http://127.0.0.1:9333" {
		t.Fatalf("CanonicalEndpoint() = %q, %v", got, err)
	}
}

func TestValidateWebSocketURLRequiresExactEndpointAndPageIdentity(t *testing.T) {
	valid, err := ValidateWebSocketURL("ws://127.0.0.1:9222/devtools/page/page-1", DefaultEndpoint, "page-1")
	if err != nil || valid == "" {
		t.Fatalf("valid websocket = %q, %v", valid, err)
	}
	for _, raw := range []string{
		"ws://127.0.0.1:9333/devtools/page/page-1",
		"ws://127.0.0.1:9222/devtools/page/page-2",
		"ws://192.168.1.5:9222/devtools/page/page-1",
	} {
		if _, err := ValidateWebSocketURL(raw, DefaultEndpoint, "page-1"); err == nil {
			t.Fatalf("ValidateWebSocketURL(%q) accepted drifted identity", raw)
		}
	}
}

func TestServiceDiscoversOnlyExactLoopbackPages(t *testing.T) {
	var server *httptest.Server
	server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, request *http.Request) {
		if request.URL.Path != "/json" {
			t.Errorf("path = %q", request.URL.Path)
		}
		ws := "ws" + strings.TrimPrefix(server.URL, "http") + "/devtools/page/page-1"
		_, _ = fmt.Fprintf(w, `[{"id":"page-1","type":"page","title":"Home","url":"https://example.test/","webSocketDebuggerUrl":%q}]`, ws)
	}))
	defer server.Close()

	targets, err := NewService(server.URL).ListTargets(context.Background(), "")
	if err != nil {
		t.Fatal(err)
	}
	if len(targets) != 1 || targets[0].ID != "page-1" || targets[0].Title != "Home" {
		t.Fatalf("targets = %#v", targets)
	}
}

func TestServiceRejectsDriftedDiscoveryWebSocket(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`[{"id":"page-1","type":"page","webSocketDebuggerUrl":"ws://127.0.0.1:1/devtools/page/page-1"}]`))
	}))
	defer server.Close()
	if _, err := NewService(server.URL).ListTargets(context.Background(), ""); err == nil {
		t.Fatal("expected drifted websocket authority to be rejected")
	}
}

func TestParseTargetsJSONFiltersNonPagesAndIncompleteEntries(t *testing.T) {
	targets, err := ParseTargetsJSON([]byte(`[
		{"id":"page-1","type":"page","title":"Home","webSocketDebuggerUrl":"ws://127.0.0.1:9222/devtools/page/page-1"},
		{"id":"worker-1","type":"service_worker","webSocketDebuggerUrl":"ws://127.0.0.1:9222/devtools/page/worker-1"},
		{"id":"page-2","type":"page"}
	]`))
	if err != nil || len(targets) != 1 || targets[0].ID != "page-1" {
		t.Fatalf("targets = %#v, error = %v", targets, err)
	}
}
