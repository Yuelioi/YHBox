package runctl

import (
	"sync"
	"testing"
	"time"
)

// 简单测试 mock Control 实现行为正确（实际 GUI 实现在 cmd/yhbox/gui/control.go）。
// 这里给一个 mock 验证 interface 契约。

type mockCtrl struct {
	mu          sync.Mutex
	cond        *sync.Cond
	paused      bool
	stopped     bool
	initialized bool
}

func newMock() *mockCtrl {
	m := &mockCtrl{}
	m.cond = sync.NewCond(&m.mu)
	m.initialized = true
	return m
}

func (m *mockCtrl) IsPaused() bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.paused
}

func (m *mockCtrl) IsBack() bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.stopped
}

func (m *mockCtrl) Pause() {
	m.mu.Lock()
	m.paused = true
	m.mu.Unlock()
}

func (m *mockCtrl) Resume() {
	m.mu.Lock()
	m.paused = false
	m.cond.Broadcast()
	m.mu.Unlock()
}

func (m *mockCtrl) Stop() {
	m.mu.Lock()
	m.stopped = true
	m.cond.Broadcast()
	m.mu.Unlock()
}

func (m *mockCtrl) WaitUnpause() bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	for m.paused && !m.stopped {
		m.cond.Wait()
	}
	return !m.stopped
}

// TestControl_InterfaceImplemented 编译期断言 mock 实现 Control。
func TestControl_InterfaceImplemented(t *testing.T) {
	var _ Control = (*mockCtrl)(nil)
}

// TestControl_PauseResumeFlow Pause → WaitUnpause 阻塞 → Resume 解阻塞。
func TestControl_PauseResumeFlow(t *testing.T) {
	c := newMock()
	c.Pause()
	if !c.IsPaused() {
		t.Fatal("IsPaused should be true after Pause()")
	}

	done := make(chan bool, 1)
	go func() { done <- c.WaitUnpause() }()

	// 给 goroutine 进入 Wait 的时间
	time.Sleep(50 * time.Millisecond)
	select {
	case <-done:
		t.Fatal("WaitUnpause returned before Resume")
	default:
	}

	c.Resume()
	select {
	case ok := <-done:
		if !ok {
			t.Errorf("WaitUnpause returned false, want true (not stopped)")
		}
	case <-time.After(time.Second):
		t.Fatal("WaitUnpause stuck after Resume")
	}
}

// TestControl_StopUnblocksWait 暂停中调 Stop 应解阻塞 WaitUnpause 并返回 false。
func TestControl_StopUnblocksWait(t *testing.T) {
	c := newMock()
	c.Pause()

	done := make(chan bool, 1)
	go func() { done <- c.WaitUnpause() }()

	time.Sleep(50 * time.Millisecond)
	c.Stop()

	select {
	case ok := <-done:
		if ok {
			t.Errorf("WaitUnpause returned true after Stop, want false")
		}
	case <-time.After(time.Second):
		t.Fatal("WaitUnpause stuck after Stop")
	}

	if !c.IsBack() {
		t.Error("IsBack should be true after Stop")
	}
}
