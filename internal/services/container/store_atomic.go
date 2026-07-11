package container

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
)

type containerFileCommittedError struct{ err error }

func (e *containerFileCommittedError) Error() string { return e.err.Error() }
func (e *containerFileCommittedError) Unwrap() error { return e.err }

func containerFileWriteCommitted(err error) bool {
	var committed *containerFileCommittedError
	return errors.As(err, &committed)
}

func writeContainerFileAtomic(path string, data []byte) error {
	dir := filepath.Dir(path)
	mode := os.FileMode(0o644)
	if info, err := os.Stat(path); err == nil {
		mode = info.Mode().Perm()
	} else if !os.IsNotExist(err) {
		return err
	}
	tmp, err := os.CreateTemp(dir, "."+filepath.Base(path)+".tmp-*")
	if err != nil {
		return err
	}
	tmpPath := tmp.Name()
	committed := false
	defer func() {
		_ = tmp.Close()
		if !committed {
			_ = os.Remove(tmpPath)
		}
	}()
	if err := tmp.Chmod(mode); err != nil {
		return err
	}
	if _, err := tmp.Write(data); err != nil {
		return err
	}
	if err := tmp.Sync(); err != nil {
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	if err := replaceContainerFile(tmpPath, path); err != nil {
		return err
	}
	committed = true
	if err := syncContainerDir(dir); err != nil {
		return &containerFileCommittedError{err: fmt.Errorf("sync container directory: %w", err)}
	}
	return nil
}
