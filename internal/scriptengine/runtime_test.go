package scriptengine

import (
	"context"
	"path/filepath"
	"testing"
)

func TestNewRuntimeRejectsIncompleteContainmentLimits(t *testing.T) {
	tests := []RuntimeOptions{
		{},
		{Executable: "relative.exe", ProcessMemoryBytes: DefaultMemoryBytes, JobMemoryBytes: DefaultMemoryBytes},
		{Executable: filepath.Join(t.TempDir(), "worker"), ProcessMemoryBytes: MinProcessMemoryBytes - 1, JobMemoryBytes: DefaultMemoryBytes},
		{Executable: filepath.Join(t.TempDir(), "worker"), ProcessMemoryBytes: DefaultMemoryBytes, JobMemoryBytes: DefaultMemoryBytes - 1},
	}
	for _, options := range tests {
		if _, err := NewRuntime(options); err == nil {
			t.Fatalf("NewRuntime(%#v) error = nil", options)
		}
	}
}

func TestUnavailableRuntimeFailsClosed(t *testing.T) {
	response, err := (unavailableRuntime{}).Execute(context.Background(), testRequest())
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if response.Failure == nil || response.Failure.Code != CodeIsolationUnavailable {
		t.Fatalf("response = %#v", response)
	}
}
