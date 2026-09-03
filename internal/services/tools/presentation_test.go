package tools

import (
	"context"
	"errors"
	"runtime"
	"sync"
	"testing"
	"time"
)

func TestShutdownClosesWindowsAndRejectsNewPresentationWork(t *testing.T) {
	presenter := &fakePresenter{ready: true}
	cleanupCalls := 0
	service := NewServiceWithOptions(nil, presenter, Options{OnCalibratorClose: func() { cleanupCalls++ }})
	if _, err := service.OpenCalibratorHUD("request"); err != nil {
		t.Fatal(err)
	}

	if err := Shutdown(context.Background(), service); err != nil {
		t.Fatal(err)
	}
	if err := Shutdown(context.Background(), service); err != nil {
		t.Fatalf("second Shutdown() error = %v", err)
	}
	if presenter.window.closeCalls != 1 {
		t.Fatalf("window close calls = %d", presenter.window.closeCalls)
	}
	if cleanupCalls != 1 {
		t.Fatalf("calibrator cleanup calls = %d", cleanupCalls)
	}
	if err := service.OpenMouseHUD("late"); err == nil {
		t.Fatal("OpenMouseHUD() after shutdown returned nil error")
	}
}

func TestShutdownWaitIsContextBoundedWhileOpenCleanupContinues(t *testing.T) {
	presenter := &blockingPresenter{
		started: make(chan struct{}, 1),
		release: make(chan struct{}),
	}
	service := NewService(nil, presenter)
	openResult := make(chan error, 1)
	go func() { openResult <- service.OpenMouseHUD("container") }()
	select {
	case <-presenter.started:
	case <-time.After(time.Second):
		t.Fatal("OpenMouseHUD() did not reach presenter")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Millisecond)
	defer cancel()
	if err := Shutdown(ctx, service); !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("Shutdown() error = %v, want deadline exceeded", err)
	}
	close(presenter.release)
	select {
	case err := <-openResult:
		if err != nil {
			t.Fatalf("cancelled OpenMouseHUD() error = %v", err)
		}
	case <-time.After(time.Second):
		t.Fatal("cancelled OpenMouseHUD() did not return")
	}
	if err := Shutdown(context.Background(), service); err != nil {
		t.Fatalf("eventual Shutdown() error = %v", err)
	}
	presenter.mu.Lock()
	window := presenter.window
	presenter.mu.Unlock()
	if window == nil || window.closeCalls != 1 {
		t.Fatalf("cancelled in-flight window = %+v", window)
	}
}

type fakePresenter struct {
	ready    bool
	requests []WindowRequest
	window   *fakeWindow
	emitted  []string
	showMain int
}

func (p *fakePresenter) Ready() bool { return p.ready }

func (p *fakePresenter) OpenWindow(request WindowRequest) (Window, error) {
	p.requests = append(p.requests, request)
	if p.window == nil {
		p.window = &fakeWindow{}
	}
	return p.window, nil
}

func (p *fakePresenter) Emit(name string, _ any) { p.emitted = append(p.emitted, name) }
func (p *fakePresenter) ShowMain() error         { p.showMain++; return nil }

type fakeWindow struct {
	focusCalls int
	showCalls  int
	hideCalls  int
	closeCalls int
	onClosing  func()
}

func (w *fakeWindow) Focus()              { w.focusCalls++ }
func (w *fakeWindow) Show()               { w.showCalls++ }
func (w *fakeWindow) Hide()               { w.hideCalls++ }
func (w *fakeWindow) Close()              { w.closeCalls++ }
func (*fakeWindow) SetAlwaysOnTop(bool)   {}
func (*fakeWindow) SetSize(int, int)      {}
func (w *fakeWindow) OnClosing(fn func()) { w.onClosing = fn }

func TestOpenMouseHUDUsesPresentationPort(t *testing.T) {
	presenter := &fakePresenter{ready: true}
	service := NewService(nil, presenter)

	if err := service.OpenMouseHUD("target-slot"); err != nil {
		t.Fatal(err)
	}
	if len(presenter.requests) != 1 {
		t.Fatalf("window requests = %+v", presenter.requests)
	}
	request := presenter.requests[0]
	if request.Kind != WindowMouseHUD || request.TargetSlot != "target-slot" {
		t.Fatalf("window request = %+v", request)
	}

	if err := service.OpenMouseHUD("other-target"); err != nil {
		t.Fatal(err)
	}
	if len(presenter.requests) != 1 || presenter.window.focusCalls != 1 {
		t.Fatalf("open calls = %d, focus calls = %d", len(presenter.requests), presenter.window.focusCalls)
	}
}

func TestLauncherOpenIsIdempotentAndEmptyStateCanRevealSettings(t *testing.T) {
	presenter := &fakePresenter{ready: true}
	shown, hidden := 0, 0
	service := NewServiceWithOptions(nil, presenter, Options{
		OnLauncherShown: func() { shown++ }, OnLauncherHidden: func() { hidden++ },
	})
	if err := service.OpenLauncher(); err != nil {
		t.Fatal(err)
	}
	if err := service.OpenLauncher(); err != nil {
		t.Fatal(err)
	}
	if len(presenter.requests) != 1 || presenter.window.showCalls != 1 || presenter.window.focusCalls != 1 {
		t.Fatalf("launcher requests=%d show=%d focus=%d", len(presenter.requests), presenter.window.showCalls, presenter.window.focusCalls)
	}
	if shown != 2 {
		t.Fatalf("launcher shown callbacks=%d", shown)
	}
	if err := service.HideLauncher(); err != nil {
		t.Fatal(err)
	}
	if presenter.window.hideCalls != 1 {
		t.Fatalf("launcher hide calls=%d", presenter.window.hideCalls)
	}
	if hidden != 1 {
		t.Fatalf("launcher hidden callbacks=%d", hidden)
	}
	if err := service.OpenLauncher(); err != nil {
		t.Fatal(err)
	}
	if len(presenter.requests) != 1 || presenter.window.showCalls != 2 || presenter.window.focusCalls != 2 {
		t.Fatalf("launcher was not reused after hide: requests=%d show=%d focus=%d", len(presenter.requests), presenter.window.showCalls, presenter.window.focusCalls)
	}
	if err := service.RefreshLauncherHotkeys(); err != nil {
		t.Fatal(err)
	}
	if shown != 4 {
		t.Fatalf("launcher shown/refresh callbacks=%d", shown)
	}
	if err := service.OpenLauncherSettings(); err != nil {
		t.Fatal(err)
	}
	if presenter.showMain != 1 || len(presenter.emitted) != 1 || presenter.emitted[0] != "main:navigate" {
		t.Fatalf("main show=%d events=%v", presenter.showMain, presenter.emitted)
	}
}

func TestCalibratorWindowClosingClearsStateAndRunsCleanup(t *testing.T) {
	presenter := &fakePresenter{ready: true}
	cleanupCalls := 0
	service := NewServiceWithOptions(nil, presenter, Options{OnCalibratorClose: func() { cleanupCalls++ }})

	opened, err := service.OpenCalibratorHUD("request-1")
	if err != nil || !opened {
		t.Fatalf("opened = %v, err = %v", opened, err)
	}
	if presenter.window.onClosing == nil {
		t.Fatal("closing callback was not registered")
	}
	presenter.window.onClosing()

	if cleanupCalls != 1 {
		t.Fatalf("cleanup calls = %d", cleanupCalls)
	}
	if service.calibratorHUD.window != nil {
		t.Fatal("calibrator window state was not cleared")
	}
}

func TestConcurrentWindowOpenCreatesOneGeneration(t *testing.T) {
	presenter := &blockingPresenter{
		started: make(chan struct{}, 1),
		release: make(chan struct{}),
	}
	service := NewService(nil, presenter)
	errs := make(chan error, 2)
	go func() { errs <- service.OpenMouseHUD("container-1") }()
	<-presenter.started
	go func() { errs <- service.OpenMouseHUD("container-1") }()
	close(presenter.release)

	for range 2 {
		if err := <-errs; err != nil {
			t.Fatal(err)
		}
	}
	presenter.mu.Lock()
	defer presenter.mu.Unlock()
	if presenter.openCalls != 1 {
		t.Fatalf("open calls = %d", presenter.openCalls)
	}
}

func TestCloseCancelsCurrentAndQueuedWindowOpens(t *testing.T) {
	presenter := &blockingPresenter{
		started: make(chan struct{}, 1),
		release: make(chan struct{}),
	}
	service := NewService(nil, presenter)
	errs := make(chan error, 2)
	go func() { errs <- service.OpenRecordingHUD() }()
	<-presenter.started
	go func() { errs <- service.OpenRecordingHUD() }()

	deadline := time.Now().Add(time.Second)
	for {
		service.mu.Lock()
		waiters := 0
		if service.recordingHUD.opening != nil {
			waiters = service.recordingHUD.opening.waiters
		}
		service.mu.Unlock()
		if waiters == 1 {
			break
		}
		if time.Now().After(deadline) {
			t.Fatal("second open did not enter the generation wait queue")
		}
		runtime.Gosched()
	}
	service.CloseRecordingHUD()
	close(presenter.release)
	for range 2 {
		if err := <-errs; err != nil {
			t.Fatal(err)
		}
	}

	presenter.mu.Lock()
	openCalls := presenter.openCalls
	presenter.mu.Unlock()
	if openCalls != 1 {
		t.Fatalf("open calls after close = %d", openCalls)
	}
	if service.recordingHUD.window != nil {
		t.Fatal("recording window reopened after close")
	}
}

func TestConcurrentOpenWaitersShareFirstAttemptError(t *testing.T) {
	wantErr := errors.New("window creation failed")
	presenter := &blockingPresenter{
		started: make(chan struct{}, 1),
		release: make(chan struct{}),
		err:     wantErr,
	}
	service := NewService(nil, presenter)
	errs := make(chan error, 3)
	go func() { errs <- service.OpenRecordingHUD() }()
	<-presenter.started
	go func() { errs <- service.OpenRecordingHUD() }()
	go func() { errs <- service.OpenRecordingHUD() }()

	deadline := time.Now().Add(time.Second)
	for {
		service.mu.Lock()
		waiters := 0
		if service.recordingHUD.opening != nil {
			waiters = service.recordingHUD.opening.waiters
		}
		service.mu.Unlock()
		if waiters == 2 {
			break
		}
		if time.Now().After(deadline) {
			t.Fatal("concurrent opens did not join the first attempt")
		}
		runtime.Gosched()
	}
	close(presenter.release)
	for range 3 {
		if err := <-errs; !errors.Is(err, wantErr) {
			t.Fatalf("open error = %v, want %v", err, wantErr)
		}
	}

	presenter.mu.Lock()
	defer presenter.mu.Unlock()
	if presenter.openCalls != 1 {
		t.Fatalf("open calls = %d", presenter.openCalls)
	}
}

func TestOldClosingCallbackCannotClearNewWindowGeneration(t *testing.T) {
	presenter := &fakePresenter{ready: true}
	cleanupCalls := 0
	service := NewServiceWithOptions(nil, presenter, Options{OnCalibratorClose: func() { cleanupCalls++ }})

	if _, err := service.OpenCalibratorHUD("request-1"); err != nil {
		t.Fatal(err)
	}
	oldWindow := presenter.window
	oldClosing := oldWindow.onClosing
	if err := service.CloseCalibratorHUD(); err != nil {
		t.Fatal(err)
	}
	presenter.window = &fakeWindow{}
	if _, err := service.OpenCalibratorHUD("request-2"); err != nil {
		t.Fatal(err)
	}
	newWindow := presenter.window

	oldClosing()
	if service.calibratorHUD.window != newWindow {
		t.Fatal("old closing callback cleared the new window generation")
	}
	if cleanupCalls != 1 {
		t.Fatalf("cleanup calls = %d", cleanupCalls)
	}
}

func TestClosingCalibratorWhileOpeningStillRunsCleanup(t *testing.T) {
	presenter := &blockingPresenter{
		started: make(chan struct{}, 1),
		release: make(chan struct{}),
	}
	cleanup := make(chan struct{}, 1)
	service := NewServiceWithOptions(nil, presenter, Options{OnCalibratorClose: func() { cleanup <- struct{}{} }})
	type openResult struct {
		opened bool
		err    error
	}
	result := make(chan openResult, 1)
	go func() {
		opened, err := service.OpenCalibratorHUD("request-1")
		result <- openResult{opened: opened, err: err}
	}()
	<-presenter.started
	if err := service.CloseCalibratorHUD(); err != nil {
		t.Fatal(err)
	}
	close(presenter.release)
	got := <-result
	if got.err != nil {
		t.Fatal(got.err)
	}
	if got.opened {
		t.Fatal("cancelled calibrator open reported opened=true")
	}

	select {
	case <-cleanup:
	default:
		t.Fatal("calibrator cleanup did not run for a cancelled open")
	}
	presenter.mu.Lock()
	defer presenter.mu.Unlock()
	if presenter.window == nil || presenter.window.closeCalls != 1 {
		t.Fatalf("cancelled window = %+v", presenter.window)
	}
}

func TestCancelledOpenCleanupCanReenterWithoutDeadlock(t *testing.T) {
	presenter := &blockingPresenter{
		started: make(chan struct{}, 1),
		release: make(chan struct{}),
	}
	type openResult struct {
		opened bool
		err    error
	}
	reentered := make(chan openResult, 1)
	var service *Service
	service = NewServiceWithOptions(nil, presenter, Options{OnCalibratorClose: func() {
		opened, err := service.OpenCalibratorHUD("reentrant")
		reentered <- openResult{opened: opened, err: err}
	}})
	initial := make(chan openResult, 1)
	go func() {
		opened, err := service.OpenCalibratorHUD("initial")
		initial <- openResult{opened: opened, err: err}
	}()
	<-presenter.started
	if err := service.CloseCalibratorHUD(); err != nil {
		t.Fatal(err)
	}
	close(presenter.release)

	for name, results := range map[string]<-chan openResult{
		"initial": initial, "reentrant": reentered,
	} {
		select {
		case result := <-results:
			if result.err != nil || result.opened {
				t.Fatalf("%s result = %+v", name, result)
			}
		case <-time.After(time.Second):
			t.Fatalf("%s open deadlocked", name)
		}
	}
	presenter.mu.Lock()
	defer presenter.mu.Unlock()
	if presenter.openCalls != 1 {
		t.Fatalf("open calls = %d", presenter.openCalls)
	}
}

type blockingPresenter struct {
	started   chan struct{}
	release   chan struct{}
	mu        sync.Mutex
	window    *fakeWindow
	openCalls int
	err       error
}

func (*blockingPresenter) Ready() bool { return true }

func (p *blockingPresenter) OpenWindow(WindowRequest) (Window, error) {
	p.mu.Lock()
	p.openCalls++
	p.mu.Unlock()
	p.started <- struct{}{}
	<-p.release
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.err != nil {
		return nil, p.err
	}
	p.window = &fakeWindow{}
	return p.window, nil
}

func (*blockingPresenter) Emit(string, any) {}
func (*blockingPresenter) ShowMain() error  { return nil }
