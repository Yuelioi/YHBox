//go:build windows

package scriptengine

import (
	"context"
	"errors"
	"fmt"
	"io"
	"sync"
	"time"

	"github.com/yottaapp/yotta/internal/processsandbox"
)

const (
	appContainerProfileName = "Yotta.Script.Worker.3_1"
	workerStartupAllowance  = 5 * time.Second
)

type windowsRuntime struct {
	sandbox *processsandbox.Runner
	image   processsandbox.Image
	mu      sync.Mutex
}

type processResult struct {
	exitCode uint32
	err      error
}

type responseResult struct {
	response Response
	err      error
}

func newPlatformRuntime(options RuntimeOptions) (platformRuntime, error) {
	image, err := processsandbox.OpenImage(options.Executable)
	if err != nil {
		return nil, fmt.Errorf("open script worker image: %w", err)
	}
	sandbox, err := processsandbox.New(processsandbox.Options{
		ProfileName: appContainerProfileName, DisplayName: "Yotta Script Worker",
		Description:        "Isolated zero-authority JavaScript worker",
		ProcessMemoryBytes: options.ProcessMemoryBytes, JobMemoryBytes: options.JobMemoryBytes,
	})
	if err != nil {
		return nil, err
	}
	return &windowsRuntime{sandbox: sandbox, image: image}, nil
}

func (*windowsRuntime) HostFeatures() []string { return []string{IsolationHostFeatureID} }

func (runtime *windowsRuntime) Execute(ctx context.Context, request Request) (Response, error) {
	if err := request.Validate(); err != nil {
		return Response{}, err
	}
	if err := ctx.Err(); err != nil {
		return Response{}, err
	}
	runtime.mu.Lock()
	defer runtime.mu.Unlock()
	response, failureCode, err := runtime.execute(ctx, request)
	if err == nil {
		return response, nil
	}
	if ctxErr := ctx.Err(); ctxErr != nil {
		return Response{}, ctxErr
	}
	message := "script worker failed"
	if failureCode == CodeIsolationUnavailable {
		message = "script isolation is unavailable on this host"
	} else if failureCode == CodeRunnerProtocolViolation {
		message = "script worker violated its protocol"
	}
	return failedResponse(request.AttemptID, failureCode, message), nil
}

func (runtime *windowsRuntime) execute(parent context.Context, request Request) (Response, string, error) {
	attemptContext, cancel := context.WithTimeout(parent, time.Duration(request.TimeoutMillis)*time.Millisecond+workerStartupAllowance)
	defer cancel()
	process, err := runtime.sandbox.Start(attemptContext, processsandbox.Request{
		Image: runtime.image, Args: []string{WorkerArgument},
		Timeout: time.Duration(request.TimeoutMillis)*time.Millisecond + workerStartupAllowance,
	})
	if err != nil {
		return Response{}, CodeIsolationUnavailable, err
	}
	defer process.Close()

	writeDone := make(chan error, 1)
	readDone := make(chan responseResult, 1)
	processDone := make(chan processResult, 1)
	stderrDone := make(chan struct{}, 1)
	go func() {
		err := WriteRequest(process.Stdin(), request)
		closeErr := process.Stdin().Close()
		if err == nil {
			err = closeErr
		}
		writeDone <- err
	}()
	go func() {
		response, err := ReadResponse(process.Stdout())
		readDone <- responseResult{response: response, err: err}
	}()
	go func() {
		_, _ = io.Copy(io.Discard, process.Stderr())
		stderrDone <- struct{}{}
	}()
	go func() {
		result, err := process.Wait()
		processDone <- processResult{exitCode: result.ExitCode, err: err}
	}()

	var (
		writeErr                     error
		readResult                   responseResult
		waitResult                   processResult
		cancelErr                    error
		wrote, read, waited, drained bool
		attemptDone                  = attemptContext.Done()
	)
	for !(wrote && read && waited && drained) {
		select {
		case <-attemptDone:
			_ = process.Terminate()
			_ = process.Stdin().Close()
			_ = process.Stdout().Close()
			_ = process.Stderr().Close()
			cancelErr = attemptContext.Err()
			attemptDone = nil
		case writeErr = <-writeDone:
			wrote = true
			if writeErr != nil {
				_ = process.Terminate()
			}
		case readResult = <-readDone:
			read = true
		case waitResult = <-processDone:
			waited = true
		case <-stderrDone:
			drained = true
		}
	}
	if cancelErr != nil {
		return Response{}, CodeRunnerCrashed, cancelErr
	}
	if writeErr != nil || waitResult.err != nil || waitResult.exitCode != WorkerExitOK {
		return Response{}, CodeRunnerCrashed, errors.Join(writeErr, waitResult.err, exitCodeError(waitResult.exitCode))
	}
	if readResult.err != nil {
		return Response{}, CodeRunnerProtocolViolation, readResult.err
	}
	if readResult.response.AttemptID != request.AttemptID {
		return Response{}, CodeRunnerProtocolViolation, errors.New("script worker attempt identity mismatch")
	}
	return readResult.response, "", nil
}

func exitCodeError(code uint32) error {
	if code == WorkerExitOK {
		return nil
	}
	return fmt.Errorf("script worker exited with code %d", code)
}
