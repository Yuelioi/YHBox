package appruntime

import (
	"context"
	"errors"
	"io"
	"net"
	"net/http"
	"testing"
	"time"
)

func TestHTTPServerOwnsListenerLifecycle(t *testing.T) {
	server := NewHTTPServer("127.0.0.1:0", http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = io.WriteString(w, "ok")
	}))
	if err := server.Start(context.Background()); err != nil {
		t.Fatalf("Start: %v", err)
	}
	if err := server.Start(context.Background()); err != nil {
		t.Fatalf("second Start: %v", err)
	}

	response, err := testHTTPClient().Get("http://" + server.Addr().String())
	if err != nil {
		t.Fatalf("GET: %v", err)
	}
	body, err := io.ReadAll(response.Body)
	_ = response.Body.Close()
	if err != nil {
		t.Fatalf("read response: %v", err)
	}
	if string(body) != "ok" {
		t.Fatalf("body=%q, want ok", body)
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	if err := server.Close(ctx); err != nil {
		t.Fatalf("Close: %v", err)
	}
	if err := server.Close(ctx); err != nil {
		t.Fatalf("second Close: %v", err)
	}
}

func TestHTTPServerReportsListenFailureSynchronously(t *testing.T) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("Listen: %v", err)
	}
	defer listener.Close()

	server := NewHTTPServer(listener.Addr().String(), http.NotFoundHandler())
	if err := server.Start(context.Background()); err == nil {
		t.Fatal("Start succeeded on an occupied address")
	}
	if err := server.Close(context.Background()); err != nil {
		t.Fatalf("Close after failed Start: %v", err)
	}
}

func TestHTTPServerCompletionBroadcastPreservesServeError(t *testing.T) {
	server := NewHTTPServer("127.0.0.1:0", http.NotFoundHandler())
	if err := server.Start(context.Background()); err != nil {
		t.Fatalf("Start: %v", err)
	}
	server.mu.Lock()
	listener := server.listener
	server.mu.Unlock()
	if err := listener.Close(); err != nil {
		t.Fatalf("listener Close: %v", err)
	}

	select {
	case <-server.Done():
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for serve completion")
	}
	// Done is a broadcast signal: multiple lifecycle observers may wait on it.
	<-server.Done()
	if server.Err() == nil {
		t.Fatal("unexpected listener close did not preserve serve error")
	}
	if err := server.Close(context.Background()); err == nil {
		t.Fatal("Close did not return preserved serve error")
	}
}

func TestHTTPServerRequestLifetimeOutlivesStartDeadline(t *testing.T) {
	server := NewHTTPServer("127.0.0.1:0", http.HandlerFunc(func(w http.ResponseWriter, request *http.Request) {
		if err := request.Context().Err(); err != nil {
			http.Error(w, err.Error(), http.StatusServiceUnavailable)
			return
		}
		w.WriteHeader(http.StatusNoContent)
	}))
	startCtx, cancelStart := context.WithTimeout(context.Background(), 10*time.Millisecond)
	defer cancelStart()
	if err := server.Start(startCtx); err != nil {
		t.Fatalf("Start: %v", err)
	}
	<-startCtx.Done()

	response, err := testHTTPClient().Get("http://" + server.Addr().String())
	if err != nil {
		t.Fatalf("GET: %v", err)
	}
	_ = response.Body.Close()
	if response.StatusCode != http.StatusNoContent {
		t.Fatalf("status=%d, want %d", response.StatusCode, http.StatusNoContent)
	}
	if err := server.Close(context.Background()); err != nil {
		t.Fatalf("Close: %v", err)
	}
}

func TestHTTPServerConcurrentCloseWaitHonorsContext(t *testing.T) {
	handlerStarted := make(chan struct{})
	releaseHandler := make(chan struct{})
	server := NewHTTPServer("127.0.0.1:0", http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
		close(handlerStarted)
		<-releaseHandler
	}))
	if err := server.Start(context.Background()); err != nil {
		t.Fatalf("Start: %v", err)
	}
	requestDone := make(chan error, 1)
	go func() {
		response, err := testHTTPClient().Get("http://" + server.Addr().String())
		if response != nil {
			_ = response.Body.Close()
		}
		requestDone <- err
	}()
	waitSignal(t, "HTTP handler start", handlerStarted)
	firstDone := make(chan error, 1)
	go func() { firstDone <- server.Close(context.Background()) }()
	deadline := time.Now().Add(time.Second)
	for {
		server.mu.Lock()
		closing := server.closed
		server.mu.Unlock()
		if closing {
			break
		}
		if time.Now().After(deadline) {
			t.Fatal("first Close did not initiate shutdown")
		}
		time.Sleep(time.Millisecond)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Millisecond)
	defer cancel()
	if err := server.Close(ctx); !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("second Close error=%v, want deadline exceeded", err)
	}
	close(releaseHandler)
	if err := waitError(t, "first HTTP Close", firstDone); err != nil {
		t.Fatalf("first Close: %v", err)
	}
	_ = waitError(t, "HTTP request completion", requestDone)
}

func testHTTPClient() *http.Client {
	return &http.Client{Timeout: time.Second}
}

func TestHTTPServerDoneIsStableBeforeStart(t *testing.T) {
	server := NewHTTPServer("127.0.0.1:0", http.NotFoundHandler())
	done := server.Done()
	if done == nil {
		t.Fatal("Done returned nil before Start")
	}
	if err := server.Start(context.Background()); err != nil {
		t.Fatalf("Start: %v", err)
	}
	if err := server.Close(context.Background()); err != nil {
		t.Fatalf("Close: %v", err)
	}
	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("pre-Start Done channel did not observe completion")
	}
}
