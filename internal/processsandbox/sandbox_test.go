package processsandbox

import (
	"os"
	"path/filepath"
	"testing"
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
