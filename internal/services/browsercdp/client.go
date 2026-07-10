package browsercdp

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/coder/websocket"
	"github.com/coder/websocket/wsjson"

	"github.com/yottaapp/yotta/internal/automation/controller"
	"github.com/yottaapp/yotta/internal/automation/target"
)

type WebSocketClient struct {
	url  string
	conn *websocket.Conn
	mu   sync.Mutex
	next int64
}

func DialWebSocketClient(ctx context.Context, wsURL string) (*WebSocketClient, error) {
	conn, _, err := websocket.Dial(ctx, wsURL, nil)
	if err != nil {
		return nil, err
	}
	conn.SetReadLimit(16 << 20)
	return &WebSocketClient{url: wsURL, conn: conn}, nil
}

func (c *WebSocketClient) Call(ctx context.Context, method string, params map[string]any) (map[string]any, error) {
	if c == nil || c.conn == nil {
		return nil, fmt.Errorf("cdp websocket client is nil")
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	c.next++
	id := c.next
	if params == nil {
		params = map[string]any{}
	}
	if err := wsjson.Write(ctx, c.conn, map[string]any{
		"id":     id,
		"method": method,
		"params": params,
	}); err != nil {
		return nil, err
	}
	for {
		var msg struct {
			ID     int64          `json:"id"`
			Result map[string]any `json:"result"`
			Error  *struct {
				Message string `json:"message"`
				Code    int    `json:"code"`
			} `json:"error"`
		}
		if err := wsjson.Read(ctx, c.conn, &msg); err != nil {
			return nil, err
		}
		if msg.ID != id {
			continue
		}
		if msg.Error != nil {
			if msg.Error.Message != "" {
				return nil, fmt.Errorf("cdp %s failed: %s", method, msg.Error.Message)
			}
			return nil, fmt.Errorf("cdp %s failed with code %d", method, msg.Error.Code)
		}
		if msg.Result == nil {
			return map[string]any{}, nil
		}
		return msg.Result, nil
	}
}

func (c *WebSocketClient) Close() error {
	if c == nil || c.conn == nil {
		return nil
	}
	return c.conn.Close(websocket.StatusNormalClosure, "")
}

type ClientProvider struct {
	Service *Service

	mu      sync.Mutex
	clients map[string]*clientEntry
}

type clientEntry struct {
	wsURL  string
	client *WebSocketClient
}

type managedClient struct {
	provider *ClientProvider
	key      string
	client   *WebSocketClient
}

func NewClientProvider(svc *Service) *ClientProvider {
	return &ClientProvider{Service: svc}
}

func (p *ClientProvider) ClientForTarget(tg target.Target) (controller.CDPClient, error) {
	if tg.Kind != target.KindBrowserCDP {
		return nil, fmt.Errorf("cdp client requires browser target, got %s", tg.Kind)
	}
	if tg.Ref.BrowserID == "" {
		return nil, fmt.Errorf("browser target requires browser id")
	}
	wsURL, err := p.webSocketURL(tg)
	if err != nil {
		return nil, err
	}
	key := tg.Ref.BrowserID
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.clients == nil {
		p.clients = map[string]*clientEntry{}
	}
	if e := p.clients[key]; e != nil && e.wsURL == wsURL && e.client != nil {
		return &managedClient{provider: p, key: key, client: e.client}, nil
	}
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	client, err := DialWebSocketClient(ctx, wsURL)
	if err != nil {
		return nil, err
	}
	if old := p.clients[key]; old != nil && old.client != nil {
		_ = old.client.Close()
	}
	p.clients[key] = &clientEntry{wsURL: wsURL, client: client}
	return &managedClient{provider: p, key: key, client: client}, nil
}

func (p *ClientProvider) webSocketURL(tg target.Target) (string, error) {
	if wsURL := metadataString(tg.Metadata, "webSocketDebuggerUrl"); wsURL != "" {
		return wsURL, nil
	}
	if wsURL := metadataString(tg.Metadata, "webSocketDebuggerURL"); wsURL != "" {
		return wsURL, nil
	}
	endpoint := metadataString(tg.Metadata, "endpoint")
	svc := p.Service
	if svc == nil {
		svc = NewService(endpoint)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	info, ok, err := svc.TargetByID(ctx, endpoint, tg.Ref.BrowserID)
	if err != nil {
		return "", err
	}
	if !ok {
		return "", fmt.Errorf("browser cdp target %q not found", tg.Ref.BrowserID)
	}
	if strings.TrimSpace(info.WebSocketDebuggerURL) == "" {
		return "", fmt.Errorf("browser cdp target %q has no websocket debugger url", tg.Ref.BrowserID)
	}
	return info.WebSocketDebuggerURL, nil
}

func metadataString(metadata map[string]any, key string) string {
	if metadata == nil {
		return ""
	}
	if v, ok := metadata[key].(string); ok {
		return strings.TrimSpace(v)
	}
	return ""
}

func (c *managedClient) Call(ctx context.Context, method string, params map[string]any) (map[string]any, error) {
	if c == nil || c.client == nil {
		return nil, fmt.Errorf("managed cdp client is nil")
	}
	res, err := c.client.Call(ctx, method, params)
	if err != nil && c.provider != nil {
		c.provider.invalidate(c.key, c.client)
	}
	return res, err
}

func (p *ClientProvider) invalidate(key string, client *WebSocketClient) {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.clients == nil {
		return
	}
	entry := p.clients[key]
	if entry == nil || entry.client != client {
		return
	}
	delete(p.clients, key)
	_ = client.Close()
}
