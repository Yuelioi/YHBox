//go:build !windows

package desktopapp

import (
	"os"
	"os/exec"
)

func restartCurrentProcess() error {
	executable, err := os.Executable()
	if err != nil {
		return err
	}
	return exec.Command(executable, os.Args[1:]...).Start()
}
