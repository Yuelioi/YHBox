package adbexec

import (
	"context"
	"testing"
)

func TestCommandContextHidesWindow(t *testing.T) {
	cmd := CommandContext(context.Background(), "version")
	if cmd.SysProcAttr == nil || !cmd.SysProcAttr.HideWindow {
		t.Fatalf("CommandContext SysProcAttr = %#v, want HideWindow", cmd.SysProcAttr)
	}
}
