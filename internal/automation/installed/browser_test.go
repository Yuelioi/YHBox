package installed

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"

	"github.com/coder/websocket"
	"github.com/coder/websocket/wsjson"
	"github.com/yottaapp/yotta/internal/resource"
)

func TestBrowserProfileStoresConfiguredPage(t *testing.T) {
	payload := BrowserProfilePayload{
		BrowserEndpoint: "127.0.0.1:9222", BrowserTargetID: "page-1",
		BrowserWebSocketURL: "ws://127.0.0.1:9222/devtools/page/page-1",
		BrowserTitle:        "Fixture", BrowserURL: "https://example.test/", ResolveTimeoutMilliseconds: 1000,
	}
	profile, err := SealProfile(NewBrowserProfileDraft(payload))
	if err != nil {
		t.Fatal(err)
	}
	sealed, ok := BrowserProfile(profile)
	if !ok || sealed.BrowserEndpoint != "http://127.0.0.1:9222" {
		t.Fatalf("profile = %#v", profile.Machine())
	}
	payload.BrowserEndpoint = "http://example.test:9222"
	if _, err := SealProfile(NewBrowserProfileDraft(payload)); err != nil {
		t.Fatalf("remote browser endpoint: %v", err)
	}
}

func TestBrowserEndpointOrPageChangeRotatesProfileDigest(t *testing.T) {
	first, err := SealProfile(NewBrowserProfileDraft(BrowserProfilePayload{
		BrowserEndpoint: "http://127.0.0.1:9222", BrowserTargetID: "page-1",
		BrowserWebSocketURL: "ws://127.0.0.1:9222/devtools/page/page-1",
		BrowserTitle:        "Fixture", BrowserURL: "https://example.test/", ResolveTimeoutMilliseconds: 1000,
	}))
	if err != nil {
		t.Fatal(err)
	}
	second, err := SealProfile(NewBrowserProfileDraft(BrowserProfilePayload{
		BrowserEndpoint: "http://127.0.0.1:9333", BrowserTargetID: "page-2",
		BrowserWebSocketURL: "ws://127.0.0.1:9333/devtools/page/page-2",
		BrowserTitle:        "Fixture 2", BrowserURL: "https://example.test/next", ResolveTimeoutMilliseconds: 1000,
	}))
	if err != nil {
		t.Fatal(err)
	}
	if first.Digest() == second.Digest() {
		t.Fatalf("browser configuration digest did not change: profiles %s/%s", first.Digest(), second.Digest())
	}
}

func TestBrowserViewportSizeRequiresPositiveMetrics(t *testing.T) {
	if _, err := browserViewportSize(map[string]any{}); err == nil {
		t.Fatal("accepted missing viewport")
	}
	if _, err := browserViewportSize(map[string]any{"cssLayoutViewport": map[string]any{"clientWidth": float64(0), "clientHeight": float64(720)}}); err == nil {
		t.Fatal("accepted non-positive viewport")
	}
}

func TestBrowserDriverResolvesExactPageAndDispatchesGenericInput(t *testing.T) {
	var server *httptest.Server
	var mu sync.Mutex
	methods := []string{}
	server = httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		switch request.URL.Path {
		case "/json":
			ws := "ws" + strings.TrimPrefix(server.URL, "http") + "/devtools/page/page-1"
			_, _ = fmt.Fprintf(writer, `[{"id":"page-1","type":"page","title":"Fixture","webSocketDebuggerUrl":%q}]`, ws)
		case "/devtools/page/page-1":
			connection, err := websocket.Accept(writer, request, &websocket.AcceptOptions{InsecureSkipVerify: true})
			if err != nil {
				t.Errorf("accept: %v", err)
				return
			}
			defer connection.CloseNow()
			for {
				var call struct {
					ID     int64  `json:"id"`
					Method string `json:"method"`
				}
				if err := wsjson.Read(context.Background(), connection, &call); err != nil {
					return
				}
				mu.Lock()
				methods = append(methods, call.Method)
				mu.Unlock()
				result := map[string]any{}
				if call.Method == "Page.getLayoutMetrics" {
					result["cssLayoutViewport"] = map[string]any{"clientWidth": 1280, "clientHeight": 720}
				}
				if err := wsjson.Write(context.Background(), connection, map[string]any{"id": call.ID, "result": result}); err != nil {
					return
				}
			}
		default:
			http.NotFound(writer, request)
		}
	}))
	defer server.Close()

	ws := "ws" + strings.TrimPrefix(server.URL, "http") + "/devtools/page/page-1"
	profile, err := SealProfile(NewBrowserProfileDraft(BrowserProfilePayload{
		BrowserEndpoint: server.URL, BrowserTargetID: "page-1", BrowserWebSocketURL: ws,
		BrowserTitle: "Fixture", ResolveTimeoutMilliseconds: 3000,
	}))
	if err != nil {
		t.Fatal(err)
	}
	opened, err := newBrowserDriver(profile)
	if err != nil {
		t.Fatal(err)
	}
	defer opened.Close()
	if err := opened.Execute(context.Background(), OperationClick, ClickRequest{
		Point: Point{X: 0.5, Y: 0.5, Unit: "ratio"}, Button: "left",
	}); err != nil {
		t.Fatal(err)
	}
	mu.Lock()
	defer mu.Unlock()
	want := []string{"Page.getLayoutMetrics", "Input.dispatchMouseEvent", "Input.dispatchMouseEvent"}
	encoded, _ := json.Marshal(methods)
	if len(methods) != len(want) {
		t.Fatalf("methods = %s", encoded)
	}
	for index := range want {
		if methods[index] != want[index] {
			t.Fatalf("methods = %s", encoded)
		}
	}
}

func TestBrowserProviderRejectsOperationsOutsideItsManifest(t *testing.T) {
	payload := BrowserProfilePayload{
		BrowserEndpoint: "http://127.0.0.1:9222", BrowserTargetID: "page-1",
		BrowserWebSocketURL: "ws://127.0.0.1:9222/devtools/page/page-1",
		BrowserTitle:        "Fixture", ResolveTimeoutMilliseconds: 1000,
	}
	profile, err := SealProfile(NewBrowserProfileDraft(payload))
	if err != nil {
		t.Fatal(err)
	}
	manifest, err := sealInstallationManifestForProfile("browser", "Browser", profile, defaultAdapterRegistry())
	if err != nil {
		t.Fatal(err)
	}
	provider, err := newProvider(profile, manifest, defaultAdapterRegistry())
	if err != nil {
		t.Fatal(err)
	}
	defer provider.CloseHost()
	if _, err := provider.Open(context.Background(), resource.ProviderOpenRequest{
		Kind: KindWindow, Operations: []string{OperationActivate}, Config: []byte(`{}`),
	}); err == nil {
		t.Fatal("browser provider opened an unsupported window operation")
	}
}
