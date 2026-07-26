package calibration

import (
	"context"
	"strings"
	"testing"
)

func TestResetClearsCalibrationSnapshot(t *testing.T) {
	currentState.addRelative(12, 34)
	currentState.setActive(true)
	t.Cleanup(func() {
		Reset()
		currentState.setActive(false)
	})

	Reset()
	state := Get()
	if state.AbsDx != 0 || state.AbsDy != 0 || !state.Active {
		t.Fatalf("Get() after Reset = %#v, want active with zero counts", state)
	}
}

func TestServiceShutdownIsIdempotentAndRejectsNewNativeWork(t *testing.T) {
	service := NewService(nil, nil)
	if err := Shutdown(context.Background(), service); err != nil {
		t.Fatal(err)
	}
	if err := Shutdown(context.Background(), service); err != nil {
		t.Fatalf("second Shutdown() error = %v", err)
	}
	if err := service.Start(); err == nil || !strings.Contains(err.Error(), "closed") {
		t.Fatalf("Start() after shutdown error = %v, want closed", err)
	}
	if err := service.StartHotkeyWatch(); err == nil || !strings.Contains(err.Error(), "closed") {
		t.Fatalf("StartHotkeyWatch() after shutdown error = %v, want closed", err)
	}
}
