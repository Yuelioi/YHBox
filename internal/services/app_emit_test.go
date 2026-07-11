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

// shouldMirrorToRootLog: container:* 事件镜像到 rootLog 做 post-mortem,
// 但高频/内部 plumbing (node-enter + node-dump 家族) 不镜像 — 否则每次节点执行刷一行 SYS "runtime event".
func TestShouldMirrorToRootLog(t *testing.T) {
	cases := []struct {
		name string
		want bool
	}{
		{"container:warning", true},
		{"container:node-error", true},
		{"container:log", true},
		{"container:node-enter", false},
		{"container:node-dump", false},       // 每节点执行一发 → 不能镜像 (用户撞到的刷屏)
		{"container:node-dump-batch", false}, // merger 已把它发前端; 文件另有 AppendDumpLine
		{"container:node-dump-flush", false}, // run 停止内部信号
		{"container:action-trace", false},    // 专用脱敏文件行; generic mirror 会泄漏 raw payload
		{"log:lines", false},                 // 非 container: 前缀
		{"hotkey:changed", false},
		{"", false},
	}
	for _, c := range cases {
		if got := shouldMirrorToRootLog(c.name); got != c.want {
			t.Errorf("shouldMirrorToRootLog(%q) = %v, want %v", c.name, got, c.want)
		}
	}
}

func TestAppEmitUsesAttachedTransport(t *testing.T) {
	app := NewApp(t.TempDir()+"/settings.json", nil, zerolog.Nop())
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

func TestAppNodeEnterBatchUsesAttachedTransport(t *testing.T) {
	app := NewApp(t.TempDir()+"/settings.json", nil, zerolog.Nop())
	received := make(chan string, 1)
	if err := app.AttachEmitter(func(name string, _ any) { received <- name }); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(app.Shutdown)
	app.Emit("container:node-enter", map[string]any{"nodeId": "n1", "nodeKind": "Sleep"})
	app.flushNodeEnter()

	select {
	case name := <-received:
		if name != "container:node-enter-batch" {
			t.Fatalf("event = %q", name)
		}
	case <-time.After(time.Second):
		t.Fatal("node-enter batch was not emitted")
	}
}

func TestAttachEmitterPublishesMergerAtomicallyAndOnlyOnce(t *testing.T) {
	app := NewApp(t.TempDir()+"/settings.json", nil, zerolog.Nop())
	received := make(chan string, 1)
	if err := app.AttachEmitter(func(name string, _ any) { received <- name }); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(app.Shutdown)

	app.Emit("container:node-dump", map[string]any{
		"containerId": "c1", "nodeId": "n1", "nodeKind": "Sleep",
		"line": "failed", "lineKey": "k1", "isError": true,
	})
	select {
	case name := <-received:
		if name != "container:node-dump-batch" {
			t.Fatalf("event = %q", name)
		}
	case <-time.After(time.Second):
		t.Fatal("node dump was lost during emitter attachment")
	}

	if err := app.AttachEmitter(func(string, any) {}); err == nil {
		t.Fatal("second emitter attachment succeeded")
	}
}

func TestAppPresentationLifecycleCannotReopenAfterShutdown(t *testing.T) {
	app := NewApp(t.TempDir()+"/settings.json", nil, zerolog.Nop())
	if err := app.AttachEmitter(func(string, any) {}); err != nil {
		t.Fatal(err)
	}
	app.Shutdown()
	if err := app.AttachEmitter(func(string, any) {}); err == nil {
		t.Fatal("emitter attachment succeeded after shutdown")
	}
}

func TestAppShutdownClearsNodeEnterTimerAndBuffer(t *testing.T) {
	app := NewApp(t.TempDir()+"/settings.json", nil, zerolog.Nop())
	if err := app.AttachEmitter(func(string, any) {}); err != nil {
		t.Fatal(err)
	}
	app.Emit("container:node-enter", map[string]any{"nodeId": "n1", "nodeKind": "Sleep"})
	if err := app.ShutdownContext(context.Background()); err != nil {
		t.Fatal(err)
	}
	app.nodeEnterMu.Lock()
	defer app.nodeEnterMu.Unlock()
	if app.nodeEnterTimer != nil || len(app.nodeEnterBuf) != 0 {
		t.Fatalf("node-enter lifecycle not cleared: timer=%v buffer=%v", app.nodeEnterTimer, app.nodeEnterBuf)
	}
}

func TestAppShutdownDeadlineDoesNotLeaveLogFileOpen(t *testing.T) {
	dir := t.TempDir()
	callbackStarted := make(chan struct{})
	releaseCallback := make(chan struct{})
	sink := NewLogSink(func(LogLinesEvent) {
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

func TestAppShutdownDeadlineClosesFileBehindBlockedMergerEmitter(t *testing.T) {
	dir := t.TempDir()
	sink := NewLogSink(nil)
	sink.SetFileWriter(dir)
	callbackStarted := make(chan struct{})
	releaseCallback := make(chan struct{})
	app := NewApp(filepath.Join(dir, "settings.json"), sink, zerolog.Nop())
	if err := app.AttachEmitter(func(name string, _ any) {
		if name == "container:node-dump-batch" {
			close(callbackStarted)
			<-releaseCallback
		}
	}); err != nil {
		t.Fatal(err)
	}
	emitDone := make(chan struct{})
	go func() {
		app.Emit("container:node-dump", map[string]any{
			"containerId": "c1", "nodeId": "n1", "nodeKind": "Sleep",
			"line": "blocked", "lineKey": "k", "isError": true,
		})
		close(emitDone)
	}()
	select {
	case <-callbackStarted:
	case <-time.After(time.Second):
		t.Fatal("merger callback did not start")
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
		t.Fatalf("blocked merger left log file open: %v", err)
	}
	close(releaseCallback)
	select {
	case <-emitDone:
	case <-time.After(time.Second):
		t.Fatal("blocked merger emit did not finish")
	}
	if err := app.ShutdownContext(context.Background()); err != nil {
		t.Fatal(err)
	}
}
