package browsercdp

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestCanonicalEndpointAcceptsConfiguredHTTPURLs(t *testing.T) {
	got, err := CanonicalEndpoint("127.0.0.1:9333")
	if err != nil || got != "http://127.0.0.1:9333" {
		t.Fatalf("CanonicalEndpoint() = %q, %v", got, err)
	}
	got, err = CanonicalEndpoint("https://example.com/cdp/")
	if err != nil || got != "https://example.com/cdp" {
		t.Fatalf("CanonicalEndpoint(remote) = %q, %v", got, err)
	}
}

func TestValidateWebSocketURLAcceptsConfiguredRemoteURLs(t *testing.T) {
	valid, err := ValidateWebSocketURL("wss://example.com/devtools/page/current", DefaultEndpoint, "page-1")
	if err != nil || valid == "" {
		t.Fatalf("valid websocket = %q, %v", valid, err)
	}
	if _, err := ValidateWebSocketURL("not-a-websocket", "", ""); err == nil {
		t.Fatal("accepted malformed websocket URL")
	}
}

func TestServiceDiscoversConfiguredPages(t *testing.T) {
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

func TestServiceAcceptsWebSocketReturnedByConfiguredEndpoint(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`[{"id":"page-1","type":"page","webSocketDebuggerUrl":"ws://127.0.0.1:1/devtools/page/page-1"}]`))
	}))
	defer server.Close()
	targets, err := NewService(server.URL).ListTargets(context.Background(), "")
	if err != nil || len(targets) != 1 {
		t.Fatalf("targets = %#v, error = %v", targets, err)
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
