// Package serviceproblem contains the shared mechanics for projecting a
// service-boundary cause without replacing an existing domain Problem.
package serviceproblem

import (
	"errors"

	"github.com/yottaapp/yotta/internal/apperr"
)

type projected struct {
	id, category string
	params       map[string]any
	retryable    bool
}

func (e projected) Error() string { return e.id }
func (e projected) RPCErrorEnvelope() apperr.Envelope {
	return apperr.Envelope{ID: e.id, Category: e.category, Params: e.params, Retryable: e.retryable}
}

func Wrap(id, category string, params map[string]any, retryable bool, cause error) error {
	if cause == nil {
		return nil
	}
	var provider apperr.EnvelopeProvider
	if errors.As(cause, &provider) {
		return cause
	}
	return errors.Join(projected{id: id, category: category, params: params, retryable: retryable}, cause)
}
