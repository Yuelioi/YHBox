//go:build windows

package storage

import (
	"errors"
	"fmt"
	"os"

	"golang.org/x/sys/windows"
)

type lease struct {
	file       *os.File
	overlapped windows.Overlapped
}

func acquireLease(path string) (*lease, error) {
	file, err := os.OpenFile(path, os.O_CREATE|os.O_RDWR, 0o600)
	if err != nil {
		return nil, fmt.Errorf("open storage writer lease: %w", err)
	}
	held := &lease{file: file}
	err = windows.LockFileEx(
		windows.Handle(file.Fd()),
		windows.LOCKFILE_EXCLUSIVE_LOCK|windows.LOCKFILE_FAIL_IMMEDIATELY,
		0, 1, 0, &held.overlapped,
	)
	if err != nil {
		_ = file.Close()
		if errors.Is(err, windows.ERROR_LOCK_VIOLATION) || errors.Is(err, windows.ERROR_IO_PENDING) {
			return nil, ErrRootInUse
		}
		return nil, fmt.Errorf("acquire storage writer lease: %w", err)
	}
	return held, nil
}

func (l *lease) close() error {
	if l == nil || l.file == nil {
		return nil
	}
	file := l.file
	l.file = nil
	unlockErr := windows.UnlockFileEx(windows.Handle(file.Fd()), 0, 1, 0, &l.overlapped)
	return errors.Join(unlockErr, file.Close())
}
