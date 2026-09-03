package codexcli

import (
	"context"
	"errors"
	"os/exec"
	"path/filepath"
	"runtime"
)

func Available() bool {
	_, _, err := resolve()
	return err == nil
}

func CommandContext(ctx context.Context, args ...string) (*exec.Cmd, error) {
	program, prefix, err := resolve()
	if err != nil {
		return nil, err
	}
	command := exec.CommandContext(ctx, program, append(prefix, args...)...)
	configureCommand(command)
	return command, nil
}

func resolve() (string, []string, error) {
	if runtime.GOOS == "windows" {
		if native, err := exec.LookPath("codex.exe"); err == nil {
			return native, nil, nil
		}
		shim, err := exec.LookPath("codex.cmd")
		if err != nil {
			return "", nil, err
		}
		pattern := filepath.Join(filepath.Dir(shim), "node_modules", "@openai", "codex", "node_modules", "@openai", "codex-win32-*", "vendor", "*", "codex", "codex.exe")
		if native, _ := filepath.Glob(pattern); len(native) != 0 {
			return native[0], nil, nil
		}
		if shim == "" {
			return "", nil, errors.New("codex command path is empty")
		}
		return "cmd.exe", []string{"/d", "/s", "/c", "call", shim}, nil
	}
	path, err := exec.LookPath("codex")
	return path, nil, err
}
