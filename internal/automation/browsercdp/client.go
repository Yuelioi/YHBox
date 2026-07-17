package browsercdp

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/coder/websocket"
	"github.com/coder/websocket/wsjson"
	"github.com/yottaapp/yotta/internal/automation/controller"
	"github.com/yottaapp/yotta/internal/automation/target"
)

type WebSocketClient struct {
	conn *websocket.Conn
	mu   sync.Mutex
	next int64
}

func DialWebSocketClient(ctx context.Context, wsURL string) (*WebSocketClient, error) {
	validated, err := ValidateLoopbackWebSocketURL(wsURL)
	if err != nil {
		return nil, err
	}
	conn, _, err := websocket.Dial(ctx, validated, &websocket.DialOptions{HTTPClient: &http.Client{
		CheckRedirect: func(_ *http.Request, _ []*http.Request) error {
			return fmt.Errorf("browser CDP websocket redirects are not allowed")
		},
	}})
	if err != nil {
		return nil, err
	}
	conn.SetReadLimit(16 << 20)
	return &WebSocketClient{conn: conn}, nil
}

func (c *WebSocketClient) Call(ctx context.Context, method string, params map[string]any) (map[string]any, error) {
	if c == nil || c.conn == nil {
		return nil, fmt.Errorf("CDP websocket client is nil")
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	c.next++
	id := c.next
	if params == nil {
		params = map[string]any{}
	}
	if err := wsjson.Write(ctx, c.conn, map[string]any{"id": id, "method": method, "params": params}); err != nil {
		return nil, err
	}
	for {
		var message struct {
			ID     int64          `json:"id"`
			Result map[string]any `json:"result"`
			Error  *struct {
				Message string `json:"message"`
				Code    int    `json:"code"`
			} `json:"error"`
		}
		if err := wsjson.Read(ctx, c.conn, &message); err != nil {
			return nil, err
		}
		if message.ID != id {
			continue
		}
		if message.Error != nil {
			return nil, fmt.Errorf("cdp %s failed (%d): %s", method, message.Error.Code, message.Error.Message)
		}
		if message.Result == nil {
			return map[string]any{}, nil
		}
		return message.Result, nil
	}
}

func (c *WebSocketClient) Close() error {
	if c == nil || c.conn == nil {
		return nil
	}
	return c.conn.Close(websocket.StatusNormalClosure, "")
}

type ClientProvider struct{ Service *Service }

func NewClientProvider(service *Service) *ClientProvider { return &ClientProvider{Service: service} }

func (p *ClientProvider) ClientForTarget(tg target.Target) (controller.CDPClient, error) {
	if tg.Kind != target.KindBrowserCDP || tg.Ref.BrowserID == "" {
		return nil, fmt.Errorf("CDP client requires an exact browser page target")
	}
	endpoint := metadataString(tg.Metadata, "endpoint")
	service := p.Service
	if service == nil {
		service = NewService(endpoint)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	info, ok, err := service.TargetByID(ctx, endpoint, tg.Ref.BrowserID)
	if err != nil {
		return nil, err
	}
	if !ok {
		return nil, fmt.Errorf("browser CDP page %q is offline or stale", tg.Ref.BrowserID)
	}
	expected := metadataString(tg.Metadata, "webSocketDebuggerUrl")
	if expected != "" && expected != info.WebSocketDebuggerURL {
		return nil, fmt.Errorf("browser CDP page %q websocket identity changed", tg.Ref.BrowserID)
	}
	return DialWebSocketClient(ctx, info.WebSocketDebuggerURL)
}

func metadataString(metadata map[string]any, key string) string {
	if value, ok := metadata[key].(string); ok {
		return strings.TrimSpace(value)
	}
	return ""
}
