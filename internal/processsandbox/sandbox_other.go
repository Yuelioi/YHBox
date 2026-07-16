//go:build !windows

package processsandbox

import (
	"context"
	"io"
)

type unavailableRunner struct{}

func newPlatformRunner(Options) platformRunner { return unavailableRunner{} }

func (unavailableRunner) start(context.Context, Request) (platformProcess, error) {
	return nil, ErrIsolationUnavailable
}

type unavailableProcess struct{}

func (unavailableProcess) stdin() io.WriteCloser { return nil }
func (unavailableProcess) stdout() io.ReadCloser { return nil }
func (unavailableProcess) stderr() io.ReadCloser { return nil }
func (unavailableProcess) wait() (Result, error) { return Result{}, ErrIsolationUnavailable }
func (unavailableProcess) terminate() error      { return nil }
func (unavailableProcess) close() error          { return nil }
