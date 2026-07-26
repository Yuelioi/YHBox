package securestore

import "errors"

var (
	ErrNotFound    = errors.New("credential not found")
	ErrUnavailable = errors.New("secure credential storage is unavailable")
)

// Store persists small application secrets outside ordinary configuration files.
type Store interface {
	Get(target string) (string, error)
	Set(target, value string) error
	Delete(target string) error
}
