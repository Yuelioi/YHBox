//go:build windows

package scriptengine

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"

	"golang.org/x/sys/windows"
)

func TestMain(m *testing.M) {
	if os.Getenv("YOTTA_SCRIPT_WORKER_TEST_EXE") != "" {
		os.Exit(m.Run())
	}
	repositoryRoot, err := filepath.Abs(filepath.Join("..", ".."))
	if err != nil {
		fmt.Fprintf(os.Stderr, "resolve repository root for script worker tests: %v\n", err)
		os.Exit(1)
	}
	temporary, err := os.MkdirTemp("", "yotta-script-worker-test-*")
	if err != nil {
		fmt.Fprintf(os.Stderr, "create script worker test directory: %v\n", err)
		os.Exit(1)
	}
	executable := filepath.Join(temporary, WorkerExecutableName)
	command := exec.Command("go", "build", "-mod=readonly", "-trimpath", "-buildvcs=false", "-ldflags=-w -s -H windowsgui", "-o", executable, "./cmd/yotta-script-worker")
	command.Dir = repositoryRoot
	if output, buildErr := command.CombinedOutput(); buildErr != nil {
		fmt.Fprintf(os.Stderr, "build isolated script worker for tests: %v\n%s", buildErr, output)
		os.Exit(1)
	}
	if err := os.Setenv("YOTTA_SCRIPT_WORKER_TEST_EXE", executable); err != nil {
		fmt.Fprintf(os.Stderr, "publish isolated script worker test path: %v\n", err)
		os.Exit(1)
	}
	code := m.Run()
	if err := os.RemoveAll(temporary); err != nil && code == 0 {
		fmt.Fprintf(os.Stderr, "remove script worker test directory: %v\n", err)
		code = 1
	}
	os.Exit(code)
}

func TestWindowsRuntimeLaunchesVerifiedLPACJobWorker(t *testing.T) {
	executable := os.Getenv("YOTTA_SCRIPT_WORKER_TEST_EXE")
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

func TestWindowsRuntimeAdvertisesExactIsolationFeature(t *testing.T) {
	runtime := testWindowsRuntime(t)
	features := runtime.HostFeatures()
	if len(features) != 1 || features[0] != IsolationHostFeatureID {
		t.Fatalf("HostFeatures() = %#v", features)
	}
}

func TestWindowsRuntimeFailsClosedWhenUninitialized(t *testing.T) {
	var runtime *Runtime
	if features := runtime.HostFeatures(); features != nil {
		t.Fatalf("uninitialized HostFeatures() = %#v", features)
	}
	if _, err := runtime.Execute(context.Background(), testRequest()); err == nil {
		t.Fatal("uninitialized Runtime.Execute accepted a request")
	}
}

func TestWindowsRuntimeRejectsInvalidExecutableAndStatusHelpers(t *testing.T) {
	if err := checkHRESULT("test", 0); err != nil {
		t.Fatalf("zero HRESULT = %v", err)
	}
	if err := checkHRESULT("test", 0x80004005); err == nil {
		t.Fatal("failing HRESULT was accepted")
	}
	if err := exitCodeError(WorkerExitOK); err != nil {
		t.Fatalf("successful worker exit = %v", err)
	}
	if err := exitCodeError(WorkerExitProtocol); err == nil {
		t.Fatal("failing worker exit was accepted")
	}
	if _, _, err := digestExecutable(t.TempDir()); err == nil {
		t.Fatal("directory was accepted as a worker executable")
	}
	empty := filepath.Join(t.TempDir(), WorkerExecutableName)
	if err := os.WriteFile(empty, nil, 0o600); err != nil {
		t.Fatal(err)
	}
	if _, _, err := digestExecutable(empty); err == nil {
		t.Fatal("empty file was accepted as a worker executable")
	}
	var handle windows.Handle
	closeHandle(nil)
	closeHandle(&handle)
	handle = windows.InvalidHandle
	closeHandle(&handle)
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
	runtime := testWindowsRuntime(t).platform.(*windowsRuntime)
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

func testWindowsRuntime(t *testing.T) *Runtime {
	t.Helper()
	executable := os.Getenv("YOTTA_SCRIPT_WORKER_TEST_EXE")
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
