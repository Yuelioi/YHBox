//go:build windows

package services

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

const autostartTaskName = "Yotta"

type autostartExecutable func() (string, error)
type autostartTaskRunner func(args ...string) ([]byte, error)

func ApplyAutostart(enabled bool) error {
	return applyAutostart(enabled, os.Executable, runAutostartTask)
}

func applyAutostart(enabled bool, executable autostartExecutable, run autostartTaskRunner) error {
	if !enabled {
		if _, err := run("/Query", "/TN", autostartTaskName); err != nil {
			return nil
		}
		output, err := run("/Delete", "/TN", autostartTaskName, "/F")
		if err != nil {
			return taskCommandError("delete", output, err)
		}
		return nil
	}

	path, err := executable()
	if err != nil {
		return fmt.Errorf("locate Yotta executable: %w", err)
	}
	if strings.ContainsAny(path, "\"\r\n") {
		return errors.New("yotta executable path contains unsupported characters")
	}
	output, err := run(
		"/Create",
		"/TN", autostartTaskName,
		"/TR", `"`+path+`"`,
		"/SC", "ONLOGON",
		"/IT",
		"/RL", "HIGHEST",
		"/F",
	)
	if err != nil {
		return taskCommandError("create", output, err)
	}
	return nil
}

func runAutostartTask(args ...string) ([]byte, error) {
	windowsRoot := os.Getenv("SystemRoot")
	if windowsRoot == "" {
		windowsRoot = os.Getenv("WINDIR")
	}
	if windowsRoot == "" {
		return nil, errors.New("windows directory is unavailable")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	command := exec.CommandContext(ctx, filepath.Join(windowsRoot, "System32", "schtasks.exe"), args...)
	output, err := command.CombinedOutput()
	if ctx.Err() != nil {
		return output, fmt.Errorf("schtasks timed out: %w", ctx.Err())
	}
	return output, err
}

func taskCommandError(operation string, output []byte, err error) error {
	detail := strings.TrimSpace(string(output))
	if detail == "" {
		return fmt.Errorf("%s Yotta startup task: %w", operation, err)
	}
	return fmt.Errorf("%s Yotta startup task: %w: %s", operation, err, detail)
}
