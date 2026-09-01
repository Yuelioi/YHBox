package mcpserver

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
	"sync"
	"time"

	"github.com/mark3labs/mcp-go/server"
	appcore "github.com/yottaapp/yotta/internal/application"
)

// RuntimeConfig deliberately exposes only a loopback port. Yotta's desktop
// MCP endpoint must never become a LAN listener through persisted settings.
type RuntimeConfig struct {
	Enabled bool
	Port    int
}

type runtimeInstance struct {
	config   RuntimeConfig
	listener net.Listener
	server   *http.Server
}

// Runtime owns the optional Streamable HTTP transport for the desktop process.
type Runtime struct {
	application *appcore.Application
	mu          sync.Mutex
	current     *runtimeInstance
}

func NewRuntime(application *appcore.Application) (*Runtime, error) {
	if application == nil {
		return nil, errors.New("MCP runtime requires Application")
	}
	return &Runtime{application: application}, nil
}

// Prepare reserves the requested port before settings are committed. Commit
// atomically swaps the live endpoint; Abort releases the reserved listener.
func (r *Runtime) Prepare(config RuntimeConfig) (commit func() error, abort func(), err error) {
	r.mu.Lock()
	if r.current != nil && r.current.config == config {
		r.mu.Unlock()
		return nil, nil, nil
	}
	r.mu.Unlock()

	var next *runtimeInstance
	if config.Enabled {
		listener, listenErr := net.Listen("tcp", fmt.Sprintf("127.0.0.1:%d", config.Port))
		if listenErr != nil {
			return nil, nil, fmt.Errorf("listen on MCP loopback port %d: %w", config.Port, listenErr)
		}
		protocol, buildErr := BuildProtocol(r.application)
		if buildErr != nil {
			_ = listener.Close()
			return nil, nil, buildErr
		}
		transport := server.NewStreamableHTTPServer(protocol)
		mux := http.NewServeMux()
		mux.Handle("/mcp", transport)
		next = &runtimeInstance{
			config: config, listener: listener,
			server: &http.Server{Handler: mux, ReadHeaderTimeout: 5 * time.Second},
		}
	}

	var once sync.Once
	abort = func() {
		once.Do(func() {
			if next != nil {
				_ = next.listener.Close()
			}
		})
	}
	commit = func() error {
		committed := false
		once.Do(func() { committed = true })
		if !committed {
			return errors.New("MCP settings activation was already finalized")
		}
		r.mu.Lock()
		previous := r.current
		r.current = next
		r.mu.Unlock()
		if next != nil {
			go func() {
				if serveErr := next.server.Serve(next.listener); serveErr != nil && !errors.Is(serveErr, http.ErrServerClosed) {
					// The listener was successfully reserved during Prepare. Runtime
					// failures after commit are surfaced by the next connection attempt.
				}
			}()
		}
		if previous != nil {
			ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
			defer cancel()
			return previous.server.Shutdown(ctx)
		}
		return nil
	}
	return commit, abort, nil
}

func (r *Runtime) Start(config RuntimeConfig) error {
	commit, abort, err := r.Prepare(config)
	if err != nil {
		return err
	}
	if abort != nil {
		defer abort()
	}
	if commit != nil {
		return commit()
	}
	return nil
}

func (r *Runtime) Close(ctx context.Context) error {
	r.mu.Lock()
	current := r.current
	r.current = nil
	r.mu.Unlock()
	if current == nil {
		return nil
	}
	return current.server.Shutdown(ctx)
}
