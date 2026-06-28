package browsercdp

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"yotta/internal/node"
)

func TestRegisterNodeAsyncSourceListsBrowserTargets(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`[{"id":"page-1","type":"page","title":"Home","url":"https://example.test/","webSocketDebuggerUrl":"ws://example/ws"}]`))
	}))
	defer srv.Close()

	nodeSvc := node.NewService()
	RegisterNodeAsyncSource(nodeSvc, NewService(""))
	opts, err := nodeSvc.AsyncOptions("", "BrowserTarget", AsyncSourceTargets, map[string]any{"Endpoint": srv.URL})
	if err != nil {
		t.Fatal(err)
	}
	if len(opts) != 1 {
		t.Fatalf("options len = %d, want 1: %#v", len(opts), opts)
	}
	if opts[0].Value != "page-1" || opts[0].Label == "" {
		t.Fatalf("option = %#v", opts[0])
	}
}
