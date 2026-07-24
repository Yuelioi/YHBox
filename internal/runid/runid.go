// Package runid owns the canonical Yotta Run identifier.
package runid

import (
	"errors"

	"github.com/google/uuid"
)

func New() (string, error) {
	id, err := uuid.NewV7()
	if err != nil {
		return "", err
	}
	return id.String(), nil
}

func Validate(value string) error {
	id, err := uuid.Parse(value)
	if err != nil || id.String() != value || id.Version() != 7 || id.Variant() != uuid.RFC4122 {
		return errors.New("run ID must be a canonical UUIDv7")
	}
	return nil
}
