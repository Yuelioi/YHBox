package scriptengine

import (
	"context"
	"errors"
	"fmt"
	"path/filepath"
)

const (
	MinProcessMemoryBytes = 64 << 20
	MaxProcessMemoryBytes = 1 << 30
	DefaultMemoryBytes    = 256 << 20
)

type Runtime interface {
	Execute(context.Context, Request) (Response, error)
}

type RuntimeOptions struct {
	Executable         string
	ProcessMemoryBytes uint64
	JobMemoryBytes     uint64
}

func NewRuntime(options RuntimeOptions) (Runtime, error) {
	if options.Executable == "" || !filepath.IsAbs(options.Executable) {
		return nil, errors.New("script worker executable must be an absolute path")
	}
	if options.ProcessMemoryBytes < MinProcessMemoryBytes || options.ProcessMemoryBytes > MaxProcessMemoryBytes {
		return nil, fmt.Errorf("script process memory must be within %d..%d bytes", MinProcessMemoryBytes, MaxProcessMemoryBytes)
	}
	if options.JobMemoryBytes < options.ProcessMemoryBytes || options.JobMemoryBytes > MaxProcessMemoryBytes {
		return nil, fmt.Errorf("script job memory must be within process memory..%d bytes", MaxProcessMemoryBytes)
	}
	return newPlatformRuntime(options), nil
}

type unavailableRuntime struct{}

func (unavailableRuntime) Execute(ctx context.Context, request Request) (Response, error) {
	if err := request.Validate(); err != nil {
		return Response{}, err
	}
	if err := ctx.Err(); err != nil {
		return Response{}, err
	}
	return failedResponse(request.AttemptID, CodeIsolationUnavailable, "script isolation is unavailable on this host"), nil
}
