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
)

func TestBrowserProfilePinsExactLoopbackPageIdentity(t *testing.T) {
	draft := ProfileDraft{
		TargetKind: TargetKindBrowserCDP, AdapterKind: AdapterKindBrowserCDP,
		BrowserEndpoint: "127.0.0.1:9222", BrowserTargetID: "page-1",
		BrowserWebSocketURL: "ws://127.0.0.1:9222/devtools/page/page-1",
		BrowserTitle:        "Fixture", BrowserURL: "https://example.test/", ResolveTimeoutMilliseconds: 1000,
	}
	profile, err := SealProfile(draft)
	if err != nil {
		t.Fatal(err)
	}
	if profile.Machine().BrowserEndpoint != "http://127.0.0.1:9222" || profile.Machine().ApplicationIdentityKind != IdentityKindBrowserPage {
		t.Fatalf("profile = %#v", profile.Machine())
	}
	draft.BrowserEndpoint = "http://example.test:9222"
	if _, err := SealProfile(draft); err == nil {
		t.Fatal("accepted non-loopback browser discovery authority")
	}
}

func TestBrowserViewportSizeRejectsMissingOrUnboundedMetrics(t *testing.T) {
	if _, err := browserViewportSize(map[string]any{}); err == nil {
		t.Fatal("accepted missing viewport")
	}
	if _, err := browserViewportSize(map[string]any{"cssLayoutViewport": map[string]any{"clientWidth": float64(200_000), "clientHeight": float64(720)}}); err == nil {
		t.Fatal("accepted unbounded viewport")
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
	profile, err := SealProfile(ProfileDraft{
		TargetKind: TargetKindBrowserCDP, AdapterKind: AdapterKindBrowserCDP,
		BrowserEndpoint: server.URL, BrowserTargetID: "page-1", BrowserWebSocketURL: ws,
		BrowserTitle: "Fixture", ResolveTimeoutMilliseconds: 3000,
	})
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
