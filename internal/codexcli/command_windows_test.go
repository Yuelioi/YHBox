//go:build windows

package codexcli

import (
	"context"
	"testing"
)

func TestCommandContextHidesWindowsConsole(t *testing.T) {
	command, err := CommandContext(context.Background(), "--version")
	if err != nil {
		t.Fatal(err)
	}
	if command.SysProcAttr == nil || !command.SysProcAttr.HideWindow {
		t.Fatalf("CommandContext SysProcAttr = %#v, want HideWindow", command.SysProcAttr)
	}
}
