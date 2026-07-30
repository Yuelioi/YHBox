//go:build windows

package appcontrol

import (
	"context"
	"os"
	"slices"
	"testing"
	"time"
)

func TestApplicationHelperProcess(t *testing.T) {
	if !slices.Contains(os.Args, "--appcontrol-helper") {
		return
	}
	time.Sleep(30 * time.Second)
}

func TestWindowsControllerLaunchesConfiguredExecutableAndTerminatesIt(t *testing.T) {
	executable, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	profile, err := SealProfile(ProfileDraft{
		Executable: executable,
		Arguments:  []string{"-test.run=^TestApplicationHelperProcess$", "--", "--appcontrol-helper"},
	})
	if err != nil {
		t.Fatal(err)
	}
	controller := windowsController{}
	pid, err := controller.Launch(context.Background(), profile)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if process, findErr := os.FindProcess(int(pid)); findErr == nil {
			_ = process.Kill()
		}
	})
	time.Sleep(250 * time.Millisecond)
	count, err := controller.Terminate(context.Background(), profile)
	if err != nil {
		t.Fatal(err)
	}
	if count != 1 {
		t.Fatalf("terminated count = %d, want 1", count)
	}
}
