package services

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/rs/zerolog"
)

func TestAppEmitUsesAttachedTransport(t *testing.T) {
	app := NewApp(filepath.Join(t.TempDir(), "settings.json"), nil, zerolog.Nop())
	var mu sync.Mutex
	var names []string
	if err := app.AttachEmitter(func(name string, _ any) {
		mu.Lock()
		names = append(names, name)
		mu.Unlock()
	}); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(app.Shutdown)

	app.Emit("settings:changed", map[string]any{})
	mu.Lock()
	defer mu.Unlock()
	if len(names) != 1 || names[0] != "settings:changed" {
		t.Fatalf("events = %v", names)
	}
}

func TestAppAttachEmitterIsSingleAssignment(t *testing.T) {
	app := NewApp(filepath.Join(t.TempDir(), "settings.json"), nil, zerolog.Nop())
	if err := app.AttachEmitter(func(string, any) {}); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(app.Shutdown)
	if err := app.AttachEmitter(func(string, any) {}); err == nil {
		t.Fatal("second emitter attachment succeeded")
	}
}

func TestAppPresentationLifecycleCannotReopenAfterShutdown(t *testing.T) {
	app := NewApp(filepath.Join(t.TempDir(), "settings.json"), nil, zerolog.Nop())
	if err := app.AttachEmitter(func(string, any) {}); err != nil {
		t.Fatal(err)
	}
	app.Shutdown()
	if err := app.AttachEmitter(func(string, any) {}); err == nil {
		t.Fatal("emitter attachment succeeded after shutdown")
	}
}

func TestAppShutdownDeadlineDoesNotLeaveLogFileOpen(t *testing.T) {
	dir := t.TempDir()
	callbackStarted := make(chan struct{})
	releaseCallback := make(chan struct{})
	sink := NewLogSink(func(LogBatchEvent) {
		close(callbackStarted)
		<-releaseCallback
	})
	sink.SetFileWriter(dir)
	app := NewApp(filepath.Join(dir, "settings.json"), sink, zerolog.Nop())
	if err := app.AttachEmitter(func(string, any) {}); err != nil {
		t.Fatal(err)
	}
	if _, err := sink.Write([]byte("blocking delivery\n")); err != nil {
		t.Fatal(err)
	}
	sink.Flush()
	select {
	case <-callbackStarted:
	case <-time.After(time.Second):
		t.Fatal("log callback did not start")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Millisecond)
	defer cancel()
	if err := app.ShutdownContext(ctx); !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("ShutdownContext error = %v", err)
	}
	files, err := filepath.Glob(filepath.Join(dir, "yotta-*.log"))
	if err != nil || len(files) != 1 {
		t.Fatalf("log files = %v, err = %v", files, err)
	}
	if err := os.Remove(files[0]); err != nil {
		t.Fatalf("shutdown deadline left log file open: %v", err)
	}
	close(releaseCallback)
	if err := app.ShutdownContext(context.Background()); err != nil {
		t.Fatal(err)
	}
}
