//go:build windows

package processsandbox

import (
	"os"
	"testing"

	"golang.org/x/sys/windows"
)

func TestWindowsStagingRepairsTamperedImage(t *testing.T) {
	executable, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	image, err := OpenImage(executable)
	if err != nil {
		t.Fatal(err)
	}
	options := Options{
		ProfileName: "Yotta.ProcessSandbox.Test", DisplayName: "Yotta Process Sandbox Test",
		Description: "Tests exact image staging", ProcessMemoryBytes: DefaultMemoryBytes, JobMemoryBytes: DefaultMemoryBytes,
	}
	sid, err := appContainerSID(options)
	if err != nil {
		t.Fatal(err)
	}
	defer windows.FreeSid(sid)
	target, _, err := stageImage(sid, image)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(target, []byte("tampered"), 0o600); err != nil {
		t.Fatal(err)
	}
	repaired, _, err := stageImage(sid, image)
	if err != nil {
		t.Fatal(err)
	}
	if repaired != target || !matchesImage(repaired, image) {
		t.Fatal("stageImage did not repair the tampered image")
	}
}
