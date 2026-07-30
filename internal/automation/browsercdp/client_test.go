package browsercdp

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/coder/websocket"
	"github.com/coder/websocket/wsjson"
	"github.com/yottaapp/yotta/internal/automation/target"
)

func TestClientProviderRediscoversAndConnectsExactPage(t *testing.T) {
	var server *httptest.Server
	server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, request *http.Request) {
		switch request.URL.Path {
		case "/json":
			ws := "ws" + strings.TrimPrefix(server.URL, "http") + "/devtools/page/page-1"
			_, _ = fmt.Fprintf(w, `[{"id":"page-1","type":"page","webSocketDebuggerUrl":%q}]`, ws)
		case "/devtools/page/page-1":
			connection, err := websocket.Accept(w, request, &websocket.AcceptOptions{InsecureSkipVerify: true})
			if err != nil {
				t.Errorf("accept: %v", err)
				return
			}
			defer connection.CloseNow()
			var call struct {
				ID int64 `json:"id"`
			}
			if err := wsjson.Read(context.Background(), connection, &call); err != nil {
				t.Errorf("read: %v", err)
				return
			}
			_ = wsjson.Write(context.Background(), connection, map[string]any{"id": call.ID, "result": map[string]any{"ok": true}})
		default:
			http.NotFound(w, request)
		}
	}))
	defer server.Close()

	client, err := NewClientProvider(NewService(server.URL)).ClientForTarget(target.Target{
		ID: "browser:page-1", Kind: target.KindBrowserCDP, Ref: target.TargetRef{BrowserID: "page-1"},
		Metadata: map[string]any{"endpoint": server.URL},
	})
	if err != nil {
		t.Fatal(err)
	}
	closer, ok := client.(*WebSocketClient)
	if !ok {
		t.Fatalf("client type = %T", client)
	}
	defer closer.Close()
	result, err := client.Call(context.Background(), "Runtime.evaluate", nil)
	if err != nil || result["ok"] != true {
		t.Fatalf("result = %#v, error = %v", result, err)
	}
}

func TestClientProviderIgnoresStoredWebSocketMetadata(t *testing.T) {
	var server *httptest.Server
	server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, request *http.Request) {
		switch request.URL.Path {
		case "/json":
			ws := "ws" + strings.TrimPrefix(server.URL, "http") + "/devtools/page/page-1"
			_, _ = fmt.Fprintf(w, `[{"id":"page-1","type":"page","webSocketDebuggerUrl":%q}]`, ws)
		case "/devtools/page/page-1":
			connection, acceptErr := websocket.Accept(w, request, &websocket.AcceptOptions{InsecureSkipVerify: true})
			if acceptErr != nil {
				t.Errorf("accept: %v", acceptErr)
				return
			}
			defer connection.CloseNow()
			<-request.Context().Done()
		default:
			http.NotFound(w, request)
		}
	}))
	defer server.Close()
	client, err := NewClientProvider(NewService(server.URL)).ClientForTarget(target.Target{
		ID: "browser:page-1", Kind: target.KindBrowserCDP, Ref: target.TargetRef{BrowserID: "page-1"},
		Metadata: map[string]any{
			"endpoint":             server.URL,
			"webSocketDebuggerUrl": "ws://127.0.0.1:1/devtools/page/page-1",
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if closer, ok := client.(*WebSocketClient); ok {
		_ = closer.Close()
	}
}
