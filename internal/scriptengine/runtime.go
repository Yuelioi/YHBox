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

	IsolationHostFeatureID = "https://schemas.yotta.dev/host-features/script-isolation/lpac-appcontainer-job/v1"
)

type platformRuntime interface {
	Execute(context.Context, Request) (Response, error)
	HostFeatures() []string
}

// Runtime is the sealed production launcher. Its platform implementation is
// not injectable outside this package, so application composition cannot
// replace isolation with an in-process or ordinary subprocess evaluator.
type Runtime struct {
	platform platformRuntime
}

type RuntimeOptions struct {
	Executable         string
	ProcessMemoryBytes uint64
	JobMemoryBytes     uint64
}

func NewRuntime(options RuntimeOptions) (*Runtime, error) {
	if options.Executable == "" || !filepath.IsAbs(options.Executable) {
		return nil, errors.New("script worker executable must be an absolute path")
	}
	if options.ProcessMemoryBytes < MinProcessMemoryBytes || options.ProcessMemoryBytes > MaxProcessMemoryBytes {
		return nil, fmt.Errorf("script process memory must be within %d..%d bytes", MinProcessMemoryBytes, MaxProcessMemoryBytes)
	}
	if options.JobMemoryBytes < options.ProcessMemoryBytes || options.JobMemoryBytes > MaxProcessMemoryBytes {
		return nil, fmt.Errorf("script job memory must be within process memory..%d bytes", MaxProcessMemoryBytes)
	}
	return &Runtime{platform: newPlatformRuntime(options)}, nil
}

func (runtime *Runtime) Execute(ctx context.Context, request Request) (Response, error) {
	if runtime == nil || runtime.platform == nil {
		return Response{}, errors.New("script runtime is not initialized")
	}
	return runtime.platform.Execute(ctx, request)
}

// HostFeatures returns the exact trusted host features implemented by this
// sealed runtime. Application composition uses this for admission discovery.
func (runtime *Runtime) HostFeatures() []string {
	if runtime == nil || runtime.platform == nil {
		return nil
	}
	return append([]string(nil), runtime.platform.HostFeatures()...)
}

type unavailableRuntime struct{}

func (unavailableRuntime) HostFeatures() []string { return []string{} }

func (unavailableRuntime) Execute(ctx context.Context, request Request) (Response, error) {
	if err := request.Validate(); err != nil {
		return Response{}, err
	}
	if err := ctx.Err(); err != nil {
		return Response{}, err
	}
	return failedResponse(request.AttemptID, CodeIsolationUnavailable, "script isolation is unavailable on this host"), nil
}
