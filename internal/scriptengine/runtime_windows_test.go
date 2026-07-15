//go:build windows

package scriptengine

import (
	"context"
	"errors"
	"os"
	"testing"
	"time"

	"golang.org/x/sys/windows"
)

func TestWindowsRuntimeLaunchesVerifiedLPACJobWorker(t *testing.T) {
	executable := os.Getenv("YOTTA_SCRIPT_WORKER_TEST_EXE")
	if executable == "" {
		t.Skip("set YOTTA_SCRIPT_WORKER_TEST_EXE to a freshly built script worker")
	}
	runtime, err := NewRuntime(RuntimeOptions{
		Executable:         executable,
		ProcessMemoryBytes: DefaultMemoryBytes,
		JobMemoryBytes:     DefaultMemoryBytes,
	})
	if err != nil {
		t.Fatalf("NewRuntime() error = %v", err)
	}
	request := testRequest()
	request.Source = `return {answer: input.a + 41, now: Date.now()};`
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	response, err := runtime.Execute(ctx, request)
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if response.Outcome != OutcomeSucceeded {
		t.Fatalf("Execute() response = %#v failure = %#v", response, response.Failure)
	}
	if got, want := string(response.Output), `{"answer":42,"now":1700000000123}`; got != want {
		t.Fatalf("output = %s, want %s", got, want)
	}
}

func TestWindowsRuntimeKillsWorkerWhenCallerCancels(t *testing.T) {
	runtime := testWindowsRuntime(t)
	request := testRequest()
	request.Source = `for (;;) {}`
	request.TimeoutMillis = MaxTimeoutMillis
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	started := time.Now()
	_, err := runtime.Execute(ctx, request)
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("Execute() error = %v, want context deadline", err)
	}
	if elapsed := time.Since(started); elapsed > 5*time.Second {
		t.Fatalf("cancelling isolated worker took %s", elapsed)
	}
}

func TestWindowsRuntimeRepairsTamperedStagedWorker(t *testing.T) {
	runtime := testWindowsRuntime(t).(*windowsRuntime)
	sid, err := appContainerSID()
	if err != nil {
		t.Fatalf("appContainerSID() error = %v", err)
	}
	defer windows.FreeSid(sid)
	target, _, err := runtime.prepareWorkerExecutable(sid)
	if err != nil {
		t.Fatalf("prepareWorkerExecutable() error = %v", err)
	}
	wantDigest, wantSize, err := digestExecutable(runtime.options.Executable)
	if err != nil {
		t.Fatalf("digest source worker: %v", err)
	}
	if err := os.WriteFile(target, []byte("tampered"), 0o600); err != nil {
		t.Fatalf("tamper staged worker: %v", err)
	}
	repaired, _, err := runtime.prepareWorkerExecutable(sid)
	if err != nil {
		t.Fatalf("repair staged worker: %v", err)
	}
	gotDigest, gotSize, err := digestExecutable(repaired)
	if err != nil {
		t.Fatalf("digest repaired worker: %v", err)
	}
	if gotDigest != wantDigest || gotSize != wantSize {
		t.Fatalf("repaired worker digest/size = %s/%d, want %s/%d", gotDigest, gotSize, wantDigest, wantSize)
	}
}

func testWindowsRuntime(t *testing.T) Runtime {
	t.Helper()
	executable := os.Getenv("YOTTA_SCRIPT_WORKER_TEST_EXE")
	if executable == "" {
		t.Skip("set YOTTA_SCRIPT_WORKER_TEST_EXE to a freshly built script worker")
	}
	runtime, err := NewRuntime(RuntimeOptions{
		Executable:         executable,
		ProcessMemoryBytes: DefaultMemoryBytes,
		JobMemoryBytes:     DefaultMemoryBytes,
	})
	if err != nil {
		t.Fatalf("NewRuntime() error = %v", err)
	}
	return runtime
}
