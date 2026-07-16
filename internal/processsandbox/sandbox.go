// Package processsandbox launches untrusted executables behind the strongest
// process boundary implemented by the current platform.
package processsandbox

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"
)

const (
	MinProcessMemoryBytes = 64 << 20
	MaxProcessMemoryBytes = 1 << 30
	DefaultMemoryBytes    = 256 << 20
	MaxImageBytes         = 512 << 20
	MaxExecutionTime      = 24 * time.Hour
)

var (
	ErrIsolationUnavailable = errors.New("process isolation is unavailable")
	ErrNotAppContainer      = errors.New("sandbox process is not an AppContainer")
	ErrNotLPAC              = errors.New("sandbox process is not a less-privileged AppContainer")
)

var (
	profileNamePattern = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9._-]{0,63}$`)
	imageNamePattern   = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9._-]{0,127}\.exe$`)
)

// Image is an immutable executable image. The launcher stages these exact
// bytes inside the sandbox profile and verifies the digest after staging.
type Image struct {
	name    string
	content []byte
	digest  string
}

func NewImage(name string, content []byte) (Image, error) {
	trimmed := strings.TrimSpace(name)
	if filepath.Base(trimmed) != trimmed || !imageNamePattern.MatchString(trimmed) {
		return Image{}, errors.New("sandbox image name must be a simple .exe filename")
	}
	name = trimmed
	if len(content) == 0 || len(content) > MaxImageBytes {
		return Image{}, fmt.Errorf("sandbox image must be within 1..%d bytes", MaxImageBytes)
	}
	sealed := append([]byte(nil), content...)
	digest := sha256.Sum256(sealed)
	return Image{name: name, content: sealed, digest: hex.EncodeToString(digest[:])}, nil
}

func OpenImage(path string) (Image, error) {
	if path == "" || !filepath.IsAbs(path) {
		return Image{}, errors.New("sandbox image path must be absolute")
	}
	file, err := os.Open(path)
	if err != nil {
		return Image{}, err
	}
	defer file.Close()
	info, err := file.Stat()
	if err != nil {
		return Image{}, err
	}
	if !info.Mode().IsRegular() || info.Size() <= 0 || info.Size() > MaxImageBytes {
		return Image{}, errors.New("sandbox image is not a bounded regular file")
	}
	content, err := io.ReadAll(io.LimitReader(file, MaxImageBytes+1))
	if err != nil {
		return Image{}, err
	}
	if int64(len(content)) != info.Size() {
		return Image{}, errors.New("sandbox image changed while reading")
	}
	return NewImage(filepath.Base(path), content)
}

func (image Image) valid() bool {
	if !imageNamePattern.MatchString(image.name) || len(image.content) == 0 || len(image.content) > MaxImageBytes {
		return false
	}
	digest := sha256.Sum256(image.content)
	return image.digest == hex.EncodeToString(digest[:])
}

type Options struct {
	ProfileName        string
	DisplayName        string
	Description        string
	ProcessMemoryBytes uint64
	JobMemoryBytes     uint64
}

type Request struct {
	Image   Image
	Args    []string
	Timeout time.Duration
}

type Result struct {
	ExitCode uint32
}

type platformRunner interface {
	available() bool
	start(context.Context, Request) (platformProcess, error)
}

type platformProcess interface {
	stdin() io.WriteCloser
	stdout() io.ReadCloser
	stderr() io.ReadCloser
	wait() (Result, error)
	terminate() error
	close() error
}

// Runner is sealed around the platform implementation: callers cannot swap
// the strong boundary for an ordinary subprocess launcher.
type Runner struct {
	platform platformRunner
}

func New(options Options) (*Runner, error) {
	if !profileNamePattern.MatchString(options.ProfileName) {
		return nil, errors.New("sandbox profile name is invalid")
	}
	if strings.TrimSpace(options.DisplayName) == "" || len(options.DisplayName) > 128 ||
		strings.TrimSpace(options.Description) == "" || len(options.Description) > 256 {
		return nil, errors.New("sandbox profile display name and description are required and bounded")
	}
	if options.ProcessMemoryBytes < MinProcessMemoryBytes || options.ProcessMemoryBytes > MaxProcessMemoryBytes {
		return nil, fmt.Errorf("sandbox process memory must be within %d..%d bytes", MinProcessMemoryBytes, MaxProcessMemoryBytes)
	}
	if options.JobMemoryBytes < options.ProcessMemoryBytes || options.JobMemoryBytes > MaxProcessMemoryBytes {
		return nil, fmt.Errorf("sandbox job memory must be within process memory..%d bytes", MaxProcessMemoryBytes)
	}
	return &Runner{platform: newPlatformRunner(options)}, nil
}

func (runner *Runner) Start(ctx context.Context, request Request) (*Process, error) {
	if runner == nil || runner.platform == nil {
		return nil, errors.New("process sandbox is not initialized")
	}
	if ctx == nil {
		return nil, errors.New("process sandbox context is required")
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if !request.Image.valid() {
		return nil, errors.New("process sandbox image is invalid")
	}
	if request.Timeout <= 0 || request.Timeout > MaxExecutionTime {
		return nil, fmt.Errorf("process sandbox timeout must be within 1ns..%s", MaxExecutionTime)
	}
	for _, argument := range request.Args {
		if strings.IndexByte(argument, 0) >= 0 || len(argument) > 32<<10 {
			return nil, errors.New("process sandbox argument is invalid")
		}
	}
	platform, err := runner.platform.start(ctx, request)
	if err != nil {
		return nil, err
	}
	process := &Process{platform: platform}
	process.stopCancellation = context.AfterFunc(ctx, func() { _ = process.Terminate() })
	return process, nil
}

func (runner *Runner) Available() bool {
	return runner != nil && runner.platform != nil && runner.platform.available()
}

type Process struct {
	platform         platformProcess
	stopCancellation func() bool
}

func (process *Process) Stdin() io.WriteCloser { return process.platform.stdin() }
func (process *Process) Stdout() io.ReadCloser { return process.platform.stdout() }
func (process *Process) Stderr() io.ReadCloser { return process.platform.stderr() }

func (process *Process) Wait() (Result, error) {
	if process == nil || process.platform == nil {
		return Result{}, errors.New("sandbox process is not initialized")
	}
	result, err := process.platform.wait()
	if process.stopCancellation != nil {
		process.stopCancellation()
	}
	return result, err
}

func (process *Process) Terminate() error {
	if process == nil || process.platform == nil {
		return nil
	}
	return process.platform.terminate()
}

func (process *Process) Close() error {
	if process == nil || process.platform == nil {
		return nil
	}
	if process.stopCancellation != nil {
		process.stopCancellation()
	}
	return process.platform.close()
}
