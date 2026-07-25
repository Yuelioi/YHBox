//go:build windows

package desktopapp

import (
	"os"
	"os/exec"
	"syscall"
)

func restartCurrentProcess() error {
	executable, err := os.Executable()
	if err != nil {
		return err
	}
	command := exec.Command(executable, os.Args[1:]...)
	command.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}
	return command.Start()
}
