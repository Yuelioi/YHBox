//go:build windows

package appcontrol

import (
	"context"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"syscall"
	"unsafe"

	"golang.org/x/sys/windows"
)

var errUnsupportedHost = errors.New("installed application lifecycle is unavailable on this host")

type windowsController struct{}

func newPlatformController() platformController { return windowsController{} }
func PlatformSupported() bool                   { return true }

func (windowsController) Launch(ctx context.Context, profile Profile) (uint32, error) {
	if err := ctx.Err(); err != nil {
		return 0, err
	}
	draft := profile.Machine()
	command := exec.Command(draft.Executable, draft.Arguments...)
	command.Dir = filepath.Dir(draft.Executable)
	command.Env = os.Environ()
	command.Stdin, command.Stdout, command.Stderr = nil, nil, nil
	command.SysProcAttr = &syscall.SysProcAttr{HideWindow: false}
	if err := command.Start(); err != nil {
		return 0, err
	}
	pid := uint32(command.Process.Pid)
	if err := command.Process.Release(); err != nil {
		_ = command.Process.Kill()
		return 0, err
	}
	return pid, nil
}

func (windowsController) Terminate(ctx context.Context, profile Profile) (int, error) {
	configured, err := os.Stat(profile.Machine().Executable)
	if err != nil {
		return 0, err
	}
	snapshot, err := windows.CreateToolhelp32Snapshot(windows.TH32CS_SNAPPROCESS, 0)
	if err != nil {
		return 0, err
	}
	defer windows.CloseHandle(snapshot)
	entry := windows.ProcessEntry32{Size: uint32(unsafe.Sizeof(windows.ProcessEntry32{}))}
	if err := windows.Process32First(snapshot, &entry); err != nil {
		return 0, err
	}
	current := uint32(os.Getpid())
	terminated := 0
	for {
		if err := ctx.Err(); err != nil {
			return terminated, err
		}
		if entry.ProcessID != 0 && entry.ProcessID != current {
			matched, terminateErr := terminateMatchingProcess(entry.ProcessID, configured)
			if terminateErr != nil {
				return terminated, terminateErr
			}
			if matched {
				terminated++
			}
		}
		if err := windows.Process32Next(snapshot, &entry); err != nil {
			if errors.Is(err, syscall.ERROR_NO_MORE_FILES) {
				break
			}
			return terminated, err
		}
	}
	return terminated, nil
}

func terminateMatchingProcess(pid uint32, configured os.FileInfo) (bool, error) {
	handle, err := windows.OpenProcess(windows.PROCESS_QUERY_LIMITED_INFORMATION|windows.PROCESS_TERMINATE, false, pid)
	if err != nil {
		return false, nil
	}
	defer windows.CloseHandle(handle)
	buffer := make([]uint16, 32_768)
	size := uint32(len(buffer))
	if err := windows.QueryFullProcessImageName(handle, 0, &buffer[0], &size); err != nil || size == 0 || size > uint32(len(buffer)) {
		return false, nil
	}
	path := windows.UTF16ToString(buffer[:size])
	info, err := os.Stat(path)
	if err != nil || !os.SameFile(configured, info) {
		return false, nil
	}
	if err := windows.TerminateProcess(handle, 0x59303131); err != nil {
		return false, err
	}
	return true, nil
}
