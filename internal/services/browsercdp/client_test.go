package browsercdp

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/coder/websocket"
	"github.com/coder/websocket/wsjson"

	"yotta/internal/automation/target"
)

func TestWebSocketClientCall(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		conn, err := websocket.Accept(w, r, &websocket.AcceptOptions{InsecureSkipVerify: true})
		if err != nil {
			t.Errorf("accept: %v", err)
			return
		}
		defer conn.CloseNow()
		var req struct {
			ID     int64          `json:"id"`
			Method string         `json:"method"`
			Params map[string]any `json:"params"`
		}
		if err := wsjson.Read(context.Background(), conn, &req); err != nil {
			t.Errorf("read: %v", err)
			return
		}
		if req.Method != "Runtime.evaluate" {
			t.Errorf("method = %q", req.Method)
		}
		if err := wsjson.Write(context.Background(), conn, map[string]any{
			"id":     req.ID,
			"result": map[string]any{"ok": true},
		}); err != nil {
			t.Errorf("write: %v", err)
		}
	}))
	defer srv.Close()

	client, err := DialWebSocketClient(context.Background(), "ws"+srv.URL[len("http"):])
	if err != nil {
		t.Fatal(err)
	}
	defer client.Close()
	res, err := client.Call(context.Background(), "Runtime.evaluate", map[string]any{"expression": "1+1"})
	if err != nil {
		t.Fatal(err)
	}
	if res["ok"] != true {
		t.Fatalf("result = %#v", res)
	}
}

func TestClientProviderDiscoversWebSocketURL(t *testing.T) {
	wsSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		conn, err := websocket.Accept(w, r, &websocket.AcceptOptions{InsecureSkipVerify: true})
		if err != nil {
			t.Errorf("accept: %v", err)
			return
		}
		defer conn.CloseNow()
		var req struct {
			ID int64 `json:"id"`
		}
		if err := wsjson.Read(context.Background(), conn, &req); err != nil {
			t.Errorf("read: %v", err)
			return
		}
		_ = wsjson.Write(context.Background(), conn, map[string]any{"id": req.ID, "result": map[string]any{"ok": true}})
	}))
	defer wsSrv.Close()
	wsURL := "ws" + wsSrv.URL[len("http"):]

	discoverySrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`[{"id":"page-1","type":"page","title":"Home","webSocketDebuggerUrl":"` + wsURL + `"}]`))
	}))
	defer discoverySrv.Close()

	provider := NewClientProvider(NewService(discoverySrv.URL))
	client, err := provider.ClientForTarget(target.Target{
		ID:       "browser:page-1",
		Kind:     target.KindBrowserCDP,
		Ref:      target.TargetRef{BrowserID: "page-1"},
		Metadata: map[string]any{"endpoint": discoverySrv.URL},
	})
	if err != nil {
		t.Fatal(err)
	}
	res, err := client.Call(context.Background(), "Runtime.evaluate", nil)
	if err != nil {
		t.Fatal(err)
	}
	if res["ok"] != true {
		t.Fatalf("result = %#v", res)
	}
}
