// Package appruntime owns the ordered lifecycle of background application resources.
package appruntime

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"
)

var ErrClosed = errors.New("application runtime is closed")

// Resource describes one background resource and its symmetric lifecycle.
// Start and Close must be safe to call exactly once; Runtime provides the
// ordering and idempotence around them.
type Resource struct {
	Name  string
	Start func(context.Context) error
	Close func(context.Context) error
}

type state uint8

const (
	stateNew state = iota
	stateStarting
	stateStarted
	stateClosing
	stateClosed
)

// Runtime starts resources in declaration order and closes them in reverse.
// A failed start rolls back every resource that already started successfully.
type Runtime struct {
	resources       []Resource
	rollbackTimeout time.Duration

	mu       sync.Mutex
	changed  chan struct{}
	state    state
	started  int
	startErr error
	closeErr error
}

func New(resources ...Resource) *Runtime {
	return NewWithOptions(Options{}, resources...)
}

type Options struct {
	RollbackTimeout time.Duration
}

func NewWithOptions(options Options, resources ...Resource) *Runtime {
	rollbackTimeout := options.RollbackTimeout
	if rollbackTimeout <= 0 {
		rollbackTimeout = 5 * time.Second
	}
	return &Runtime{
		resources:       append([]Resource(nil), resources...),
		rollbackTimeout: rollbackTimeout,
		changed:         make(chan struct{}),
	}
}

// Start is idempotent after success. A runtime whose startup failed is closed
// after rollback and cannot be started again.
func (r *Runtime) Start(ctx context.Context) error {
	r.mu.Lock()
	for r.state == stateStarting || r.state == stateClosing {
		changed := r.changed
		r.mu.Unlock()
		select {
		case <-changed:
		case <-ctx.Done():
			return ctx.Err()
		}
		r.mu.Lock()
	}
	switch r.state {
	case stateStarted:
		r.mu.Unlock()
		return nil
	case stateClosed:
		err := r.startErr
		if err == nil {
			err = ErrClosed
		}
		r.mu.Unlock()
		return err
	case stateNew:
		r.state = stateStarting
	}
	r.mu.Unlock()

	for index, resource := range r.resources {
		if err := validateResource(index, resource); err != nil {
			return r.rollback(ctx, 0, err)
		}
	}
	for index, resource := range r.resources {
		if err := ctx.Err(); err != nil {
			return r.rollback(ctx, index, fmt.Errorf("start %s: %w", resource.Name, err))
		}
		if err := resource.Start(ctx); err != nil {
			return r.rollback(ctx, index, fmt.Errorf("start %s: %w", resource.Name, err))
		}
		r.mu.Lock()
		r.started = index + 1
		r.mu.Unlock()
	}

	r.mu.Lock()
	r.state = stateStarted
	r.notifyLocked()
	r.mu.Unlock()
	return nil
}

func validateResource(index int, resource Resource) error {
	if resource.Name == "" {
		return fmt.Errorf("resource %d has empty name", index)
	}
	if resource.Start == nil {
		return fmt.Errorf("resource %s has nil Start", resource.Name)
	}
	if resource.Close == nil {
		return fmt.Errorf("resource %s has nil Close", resource.Name)
	}
	return nil
}

func (r *Runtime) rollback(ctx context.Context, failedIndex int, startErr error) error {
	rollbackCtx, cancelRollback := context.WithTimeout(context.WithoutCancel(ctx), r.rollbackTimeout)
	defer cancelRollback()
	errs := []error{startErr}
	for index := failedIndex - 1; index >= 0; index-- {
		resource := r.resources[index]
		if err := resource.Close(rollbackCtx); err != nil {
			errs = append(errs, fmt.Errorf("rollback %s: %w", resource.Name, err))
		}
	}
	joined := errors.Join(errs...)

	r.mu.Lock()
	r.started = 0
	r.startErr = joined
	r.closeErr = joined
	r.state = stateClosed
	r.notifyLocked()
	r.mu.Unlock()
	return joined
}

// Close is idempotent and always attempts every started resource in reverse
// order. Concurrent callers observe the same aggregated result.
func (r *Runtime) Close(ctx context.Context) error {
	r.mu.Lock()
	for r.state == stateStarting || r.state == stateClosing {
		changed := r.changed
		r.mu.Unlock()
		select {
		case <-changed:
		case <-ctx.Done():
			return ctx.Err()
		}
		r.mu.Lock()
	}
	switch r.state {
	case stateClosed:
		err := r.closeErr
		r.mu.Unlock()
		return err
	case stateNew:
		r.state = stateClosed
		r.notifyLocked()
		r.mu.Unlock()
		return nil
	case stateStarted:
		r.state = stateClosing
	}
	started := r.started
	r.mu.Unlock()

	var errs []error
	for index := started - 1; index >= 0; index-- {
		resource := r.resources[index]
		if err := resource.Close(ctx); err != nil {
			errs = append(errs, fmt.Errorf("close %s: %w", resource.Name, err))
		}
	}
	joined := errors.Join(errs...)

	r.mu.Lock()
	r.started = 0
	r.closeErr = joined
	r.state = stateClosed
	r.notifyLocked()
	r.mu.Unlock()
	return joined
}

func (r *Runtime) notifyLocked() {
	close(r.changed)
	r.changed = make(chan struct{})
}
