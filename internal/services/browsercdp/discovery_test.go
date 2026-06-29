package browsercdp

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestParseTargetsJSONFiltersPageTargets(t *testing.T) {
	targets, err := ParseTargetsJSON([]byte(`[
		{"id":"page-1","type":"page","title":"Home","url":"https://example.test/","webSocketDebuggerUrl":"ws://127.0.0.1/devtools/page/page-1"},
		{"id":"worker-1","type":"service_worker","title":"Worker","webSocketDebuggerUrl":"ws://127.0.0.1/devtools/page/worker-1"},
		{"id":"page-2","type":"page","title":"No WS","url":"https://missing.test/"}
	]`))
	if err != nil {
		t.Fatal(err)
	}
	if len(targets) != 1 {
		t.Fatalf("targets len = %d, want 1: %#v", len(targets), targets)
	}
	if targets[0].ID != "page-1" || targets[0].Title != "Home" || targets[0].URL != "https://example.test/" {
		t.Fatalf("target = %#v", targets[0])
	}
}

func TestServiceListTargetsFetchesJSONEndpoint(t *testing.T) {
	var gotPath string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		_, _ = w.Write([]byte(`[{"id":"page-1","type":"page","title":"Home","webSocketDebuggerUrl":"ws://example/ws"}]`))
	}))
	defer srv.Close()

	targets, err := NewService(srv.URL).ListTargets(context.Background(), "")
	if err != nil {
		t.Fatal(err)
	}
	if gotPath != "/json" {
		t.Fatalf("path = %q, want /json", gotPath)
	}
	if len(targets) != 1 || targets[0].ID != "page-1" {
		t.Fatalf("targets = %#v", targets)
	}
}

func TestTargetFromInfoCarriesBrowserMetadata(t *testing.T) {
	tg := TargetFromInfo("http://127.0.0.1:9222", TargetInfo{
		ID:                   "page-1",
		Title:                "Home",
		URL:                  "https://example.test/",
		WebSocketDebuggerURL: "ws://127.0.0.1/devtools/page/page-1",
	}, 1280, 720, "")
	if err := tg.Validate(); err != nil {
		t.Fatal(err)
	}
	if tg.ID != "browser:page-1" || tg.Ref.BrowserID != "page-1" || tg.DisplayName != "Home" {
		t.Fatalf("target = %#v", tg)
	}
	if tg.Metadata["endpoint"] != "http://127.0.0.1:9222" {
		t.Fatalf("metadata = %#v", tg.Metadata)
	}
}
