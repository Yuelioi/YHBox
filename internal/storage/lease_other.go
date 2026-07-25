//go:build !windows

package storage

import (
	"errors"
	"fmt"
	"os"

	"golang.org/x/sys/unix"
)

type lease struct{ file *os.File }

func acquireLease(path string) (*lease, error) {
	file, err := os.OpenFile(path, os.O_CREATE|os.O_RDWR, 0o600)
	if err != nil {
		return nil, fmt.Errorf("open storage writer lease: %w", err)
	}
	if err := unix.Flock(int(file.Fd()), unix.LOCK_EX|unix.LOCK_NB); err != nil {
		_ = file.Close()
		if errors.Is(err, unix.EWOULDBLOCK) {
			return nil, ErrRootInUse
		}
		return nil, fmt.Errorf("acquire storage writer lease: %w", err)
	}
	return &lease{file: file}, nil
}

func (l *lease) close() error {
	if l == nil || l.file == nil {
		return nil
	}
	file := l.file
	l.file = nil
	return errors.Join(unix.Flock(int(file.Fd()), unix.LOCK_UN), file.Close())
}
