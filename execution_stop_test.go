package main

import (
	"errors"
	"testing"

	"github.com/rs/zerolog"
)

func TestStopAllForHotkeyCallsUnifiedStopAll(t *testing.T) {
	called := false
	stopAllForHotkey(func() error {
		called = true
		return nil
	}, zerolog.Nop())

	if !called {
		t.Fatal("stopAllForHotkey did not call StopAll")
	}
}

func TestStopAllForHotkeyDoesNotPanicOnError(t *testing.T) {
	stopAllForHotkey(func() error {
		return errors.New("stop failed")
	}, zerolog.Nop())
}
