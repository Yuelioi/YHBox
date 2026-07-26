package hotkey

import (
	"context"
	"testing"
	"time"
)

func newTestHotkeyManager() *HotkeyManager {
	manager := NewHotkeyManager()
	manager.runLoop = runTestHotkeyLoop
	return manager
}

func runTestHotkeyLoop(ctx context.Context, _ []HotkeySpec, _ func(int), ready chan<- error, done chan<- struct{}) {
	defer close(done)
	ready <- nil
	<-ctx.Done()
}

func TestHotkeyManagerUnregisterDoesNotDeadlockWithInFlightDispatch(t *testing.T) {
	manager := NewHotkeyManager()
	manager.runLoop = func(ctx context.Context, specs []HotkeySpec, handler func(int), ready chan<- error, done chan<- struct{}) {
		defer close(done)
		ready <- nil
		<-ctx.Done()
		handler(specs[0].ID)
	}
	id, err := manager.Register(HotkeySpec{Mods: MOD_CONTROL, VK: VK_1}, OwnerAction, func() {})
	if err != nil {
		t.Fatalf("Register() error = %v", err)
	}

	finished := make(chan error, 1)
	go func() { finished <- manager.Unregister(id) }()
	select {
	case err := <-finished:
		if err != nil {
			t.Fatalf("Unregister() error = %v", err)
		}
	case <-time.After(time.Second):
		t.Fatal("Unregister() deadlocked with in-flight dispatch")
	}
}
