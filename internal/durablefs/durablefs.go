// Package durablefs owns crash-durable replacement and removal of authoritative files.
package durablefs

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
)

type committedError struct{ err error }

func (e *committedError) Error() string { return e.err.Error() }
func (e *committedError) Unwrap() error { return e.err }

// Committed reports whether the authoritative directory entry changed before
// the operation returned an error. Callers must update their in-memory truth
// even though the durability guarantee could not be confirmed.
func Committed(err error) bool {
	var committed *committedError
	return errors.As(err, &committed)
}

// Replace atomically publishes stagedPath at destinationPath and makes the
// directory entry durable before returning.
func Replace(stagedPath, destinationPath string) error {
	if err := replaceFile(stagedPath, destinationPath); err != nil {
		return err
	}
	if err := syncDir(filepath.Dir(destinationPath)); err != nil {
		return &committedError{err: fmt.Errorf("sync destination directory: %w", err)}
	}
	return nil
}

// Remove durably removes path. A missing path is reported by the platform.
func Remove(path string) error {
	return removeFile(path)
}

// WriteFile publishes a complete file without exposing partial bytes.
func WriteFile(path string, data []byte, mode os.FileMode) error {
	tmp, err := os.CreateTemp(filepath.Dir(path), ".durable-*.tmp")
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
	if err := Replace(tmpPath, path); err != nil {
		if Committed(err) {
			committed = true
		}
		return err
	}
	committed = true
	return nil
}
