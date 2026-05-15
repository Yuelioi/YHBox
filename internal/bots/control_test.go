package bots

import (
	"sync"
	"testing"
	"time"

	"yhbox/pkg/runctl"
)

func TestBotControl_ImplementsControl(t *testing.T) {
	var _ runctl.Control = NewBotControl()
}

func TestBotControl_InitialState(t *testing.T) {
	c := NewBotControl()
	if c.IsPaused() {
		t.Error("freshly created control should not be paused")
	}
	if c.IsBack() {
		t.Error("freshly created control should not be stopped")
	}
}

func TestBotControl_PauseResume(t *testing.T) {
	c := NewBotControl()
	c.Pause()
	if !c.IsPaused() {
		t.Fatal("IsPaused true after Pause")
	}

	done := make(chan bool, 1)
	go func() { done <- c.WaitUnpause() }()
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
			t.Error("WaitUnpause returned false after Resume, want true")
		}
	case <-time.After(time.Second):
		t.Fatal("WaitUnpause stuck after Resume")
	}
	if c.IsPaused() {
		t.Error("IsPaused should be false after Resume")
	}
}

func TestBotControl_StopUnblocksAndSetsBack(t *testing.T) {
	c := NewBotControl()
	c.Pause()

	done := make(chan bool, 1)
	go func() { done <- c.WaitUnpause() }()
	time.Sleep(50 * time.Millisecond)
	c.Stop()

	select {
	case ok := <-done:
		if ok {
			t.Error("WaitUnpause after Stop should return false")
		}
	case <-time.After(time.Second):
		t.Fatal("WaitUnpause stuck after Stop")
	}
	if !c.IsBack() {
		t.Error("IsBack should be true after Stop")
	}
}

func TestBotControl_ConcurrentPauseResume(t *testing.T) {
	c := NewBotControl()
	var wg sync.WaitGroup
	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			c.Pause()
			c.Resume()
		}()
	}
	wg.Wait()
	// 不崩就行
}

func TestBotControl_Reset(t *testing.T) {
	c := NewBotControl()
	c.Pause()
	c.Stop()
	if !c.IsBack() {
		t.Fatal("Stop should set IsBack")
	}
	c.Reset()
	if c.IsBack() {
		t.Error("Reset should clear IsBack")
	}
	if c.IsPaused() {
		t.Error("Reset should clear IsPaused")
	}
}
