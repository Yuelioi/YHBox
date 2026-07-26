package appruntime

import (
	"context"
	"errors"
	"net"
	"net/http"
	"sync"
)

// HTTPServer adapts an http.Handler into a synchronously-started lifecycle
// resource. Listen errors are returned from Start instead of being lost in a
// detached goroutine.
type HTTPServer struct {
	address string
	handler http.Handler

	mu        sync.Mutex
	server    *http.Server
	listener  net.Listener
	done      chan struct{}
	doneOnce  sync.Once
	serveErr  error
	cancel    context.CancelFunc
	started   bool
	closed    bool
	closeErr  error
	closeOnce sync.Once
	closeDone chan struct{}
}

func NewHTTPServer(address string, handler http.Handler) *HTTPServer {
	return &HTTPServer{
		address:   address,
		handler:   handler,
		done:      make(chan struct{}),
		closeDone: make(chan struct{}),
	}
}

func (s *HTTPServer) Start(ctx context.Context) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.started {
		return nil
	}
	if s.closed {
		return ErrClosed
	}
	if s.handler == nil {
		return errors.New("HTTP server handler is nil")
	}

	listener, err := (&net.ListenConfig{}).Listen(ctx, "tcp", s.address)
	if err != nil {
		return err
	}
	lifetime, cancel := context.WithCancel(context.Background())
	server := &http.Server{
		Handler: s.handler,
		BaseContext: func(net.Listener) context.Context {
			return lifetime
		},
	}
	s.server = server
	s.listener = listener
	s.cancel = cancel
	s.started = true
	go func() {
		err := server.Serve(listener)
		if errors.Is(err, http.ErrServerClosed) {
			err = nil
		}
		s.finish(err)
	}()
	return nil
}

// Close gracefully stops the server. If the context expires, it force-closes
// the listener so no serving goroutine is left behind.
func (s *HTTPServer) Close(ctx context.Context) error {
	s.closeOnce.Do(func() {
		s.mu.Lock()
		s.closed = true
		s.mu.Unlock()
		go s.shutdown(ctx)
	})
	select {
	case <-s.closeDone:
		s.mu.Lock()
		defer s.mu.Unlock()
		return s.closeErr
	case <-ctx.Done():
		return ctx.Err()
	}
}

func (s *HTTPServer) shutdown(ctx context.Context) {
	s.mu.Lock()
	server := s.server
	cancel := s.cancel
	done := s.done
	s.mu.Unlock()

	var joined error
	if server == nil {
		s.finish(nil)
	} else {
		if cancel != nil {
			cancel()
		}
		shutdownErr := server.Shutdown(ctx)
		var forceCloseErr error
		if shutdownErr != nil {
			forceCloseErr = server.Close()
		}
		<-done
		joined = errors.Join(shutdownErr, forceCloseErr, s.Err())
	}
	s.mu.Lock()
	s.closeErr = joined
	s.mu.Unlock()
	close(s.closeDone)
}

func (s *HTTPServer) finish(err error) {
	s.doneOnce.Do(func() {
		s.mu.Lock()
		s.serveErr = err
		s.mu.Unlock()
		close(s.done)
	})
}

func (s *HTTPServer) Addr() net.Addr {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.listener == nil {
		return nil
	}
	return s.listener.Addr()
}

func (s *HTTPServer) Done() <-chan struct{} {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.done
}

func (s *HTTPServer) Err() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.serveErr
}

func (s *HTTPServer) Resource(name string) Resource {
	return Resource{Name: name, Start: s.Start, Close: s.Close}
}
