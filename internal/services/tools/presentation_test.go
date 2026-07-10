package tools

import (
	"errors"
	"runtime"
	"sync"
	"testing"
	"time"
)

type fakePresenter struct {
	ready    bool
	requests []WindowRequest
	window   *fakeWindow
	emitted  []string
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

	if err := service.OpenMouseHUD("container with spaces"); err != nil {
		t.Fatal(err)
	}
	if len(presenter.requests) != 1 {
		t.Fatalf("window requests = %+v", presenter.requests)
	}
	request := presenter.requests[0]
	if request.Kind != WindowMouseHUD || request.ContainerID != "container with spaces" {
		t.Fatalf("window request = %+v", request)
	}

	if err := service.OpenMouseHUD("ignored"); err != nil {
		t.Fatal(err)
	}
	if len(presenter.requests) != 1 || presenter.window.focusCalls != 1 {
		t.Fatalf("open calls = %d, focus calls = %d", len(presenter.requests), presenter.window.focusCalls)
	}
}

func TestCalibratorWindowClosingClearsStateAndRunsCleanup(t *testing.T) {
	presenter := &fakePresenter{ready: true}
	service := NewService(nil, presenter)
	cleanupCalls := 0
	service.SetCalibratorCloseHandler(func() { cleanupCalls++ })

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
	service := NewService(nil, presenter)
	cleanupCalls := 0
	service.SetCalibratorCloseHandler(func() { cleanupCalls++ })

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
	service := NewService(nil, presenter)
	cleanup := make(chan struct{}, 1)
	service.SetCalibratorCloseHandler(func() { cleanup <- struct{}{} })
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
	service := NewService(nil, presenter)
	type openResult struct {
		opened bool
		err    error
	}
	reentered := make(chan openResult, 1)
	service.SetCalibratorCloseHandler(func() {
		opened, err := service.OpenCalibratorHUD("reentrant")
		reentered <- openResult{opened: opened, err: err}
	})
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
