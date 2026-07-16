package processsandbox

import (
	"bytes"
	"context"
	"io"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestImageSealsExecutableBytes(t *testing.T) {
	content := []byte("executable")
	image, err := NewImage("worker.exe", content)
	if err != nil {
		t.Fatal(err)
	}
	content[0] = 'X'
	if !image.valid() || string(image.content) != "executable" {
		t.Fatal("Image did not seal its source bytes")
	}
	if _, err := NewImage("../worker.exe", []byte("x")); err == nil {
		t.Fatal("Image accepted a path instead of a simple filename")
	}
}

func TestRunnerOwnsPlatformProcessLifecycle(t *testing.T) {
	image, err := NewImage("worker.exe", []byte("worker"))
	if err != nil {
		t.Fatal(err)
	}
	platform := &fakePlatformRunner{process: newFakePlatformProcess()}
	runner := &Runner{platform: platform}
	if !runner.Available() {
		t.Fatal("fake platform should be available")
	}
	process, err := runner.Start(context.Background(), Request{Image: image, Args: []string{"--run"}, Timeout: time.Second})
	if err != nil {
		t.Fatal(err)
	}
	if process.Stdin() == nil || process.Stdout() == nil || process.Stderr() == nil {
		t.Fatal("process streams were not exposed")
	}
	if result, err := process.Wait(); err != nil || result.ExitCode != 7 {
		t.Fatalf("Wait = %#v, %v", result, err)
	}
	if err := process.Terminate(); err != nil {
		t.Fatalf("Terminate = %v", err)
	}
	if err := process.Close(); err != nil {
		t.Fatalf("Close = %v", err)
	}
	if !platform.started || !platform.process.terminated || !platform.process.closed {
		t.Fatal("platform lifecycle was not forwarded")
	}
}

func TestRunnerRejectsInvalidRequestsBeforePlatformStart(t *testing.T) {
	image, err := NewImage("worker.exe", []byte("worker"))
	if err != nil {
		t.Fatal(err)
	}
	runner := &Runner{platform: &fakePlatformRunner{process: newFakePlatformProcess()}}
	cancelled, cancel := context.WithCancel(context.Background())
	cancel()
	cases := []struct {
		ctx context.Context
		req Request
	}{
		{nil, Request{Image: image, Timeout: time.Second}},
		{cancelled, Request{Image: image, Timeout: time.Second}},
		{context.Background(), Request{Timeout: time.Second}},
		{context.Background(), Request{Image: image}},
		{context.Background(), Request{Image: image, Timeout: MaxExecutionTime + time.Nanosecond}},
		{context.Background(), Request{Image: image, Timeout: time.Second, Args: []string{"bad\x00arg"}}},
	}
	for _, test := range cases {
		if _, err := runner.Start(test.ctx, test.req); err == nil {
			t.Fatalf("Start accepted invalid request %#v", test.req)
		}
	}
	if _, err := (*Runner)(nil).Start(context.Background(), Request{}); err == nil {
		t.Fatal("nil Runner accepted Start")
	}
	if _, err := (*Process)(nil).Wait(); err == nil {
		t.Fatal("nil Process accepted Wait")
	}
	if err := (*Process)(nil).Terminate(); err != nil || (*Process)(nil).Close() != nil {
		t.Fatal("nil Process cleanup should be idempotent")
	}
}

type fakePlatformRunner struct {
	process *fakePlatformProcess
	started bool
}

func (runner *fakePlatformRunner) available() bool { return true }
func (runner *fakePlatformRunner) start(context.Context, Request) (platformProcess, error) {
	runner.started = true
	return runner.process, nil
}

type fakePlatformProcess struct {
	in          io.WriteCloser
	out, errOut io.ReadCloser
	terminated  bool
	closed      bool
}

func newFakePlatformProcess() *fakePlatformProcess {
	return &fakePlatformProcess{in: nopWriteCloser{&bytes.Buffer{}}, out: io.NopCloser(bytes.NewReader(nil)), errOut: io.NopCloser(bytes.NewReader(nil))}
}
func (process *fakePlatformProcess) stdin() io.WriteCloser { return process.in }
func (process *fakePlatformProcess) stdout() io.ReadCloser { return process.out }
func (process *fakePlatformProcess) stderr() io.ReadCloser { return process.errOut }
func (process *fakePlatformProcess) wait() (Result, error) { return Result{ExitCode: 7}, nil }
func (process *fakePlatformProcess) terminate() error      { process.terminated = true; return nil }
func (process *fakePlatformProcess) close() error          { process.closed = true; return nil }

type nopWriteCloser struct{ io.Writer }

func (nopWriteCloser) Close() error { return nil }

func TestOpenImageRejectsUnboundedSources(t *testing.T) {
	if _, err := OpenImage("relative.exe"); err == nil {
		t.Fatal("OpenImage accepted a relative path")
	}
	if _, err := OpenImage(t.TempDir()); err == nil {
		t.Fatal("OpenImage accepted a directory")
	}
	empty := filepath.Join(t.TempDir(), "empty.exe")
	if err := os.WriteFile(empty, nil, 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := OpenImage(empty); err == nil {
		t.Fatal("OpenImage accepted an empty file")
	}
}

func TestRunnerOptionsFailClosed(t *testing.T) {
	valid := Options{
		ProfileName: "Yotta.Test", DisplayName: "Yotta Test", Description: "Test sandbox",
		ProcessMemoryBytes: DefaultMemoryBytes, JobMemoryBytes: DefaultMemoryBytes,
	}
	if _, err := New(valid); err != nil {
		t.Fatal(err)
	}
	invalid := valid
	invalid.ProcessMemoryBytes = MinProcessMemoryBytes - 1
	if _, err := New(invalid); err == nil {
		t.Fatal("New accepted an undersized process memory limit")
	}
}
