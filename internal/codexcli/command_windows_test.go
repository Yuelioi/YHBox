//go:build windows

package codexcli

import (
	"context"
	"os/exec"
	"testing"
)

func TestCommandContextHidesWindowsConsole(t *testing.T) {
	command := exec.CommandContext(context.Background(), "codex.exe", "--version")
	configureCommand(command)
	if command.SysProcAttr == nil || !command.SysProcAttr.HideWindow {
		t.Fatalf("CommandContext SysProcAttr = %#v, want HideWindow", command.SysProcAttr)
	}
}
