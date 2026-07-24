//go:build windows

package appcontrol

import (
	"context"
	"os"
	"slices"
	"strings"
	"testing"
	"time"
)

func TestApplicationHelperProcess(t *testing.T) {
	if !slices.Contains(os.Args, "--appcontrol-helper") {
		return
	}
	time.Sleep(30 * time.Second)
}

func TestWindowsControllerLaunchesWithoutAmbientEnvironmentAndTerminatesExactExecutable(t *testing.T) {
	t.Setenv("YOTTA_TEST_SECRET", "must-not-be-inherited")
	for _, entry := range applicationEnvironment() {
		upper := strings.ToUpper(entry)
		if strings.HasPrefix(upper, "PATH=") || strings.HasPrefix(upper, "YOTTA_TEST_SECRET=") || strings.Contains(upper, "PROXY=") {
			t.Fatalf("application environment leaked ambient entry %q", entry)
		}
	}
	executable, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	inspection, err := InspectExecutable(executable)
	if err != nil {
		t.Fatal(err)
	}
	profile, err := SealProfile(ProfileDraft{
		Executable: inspection.Executable,
		Arguments:  []string{"-test.run=^TestApplicationHelperProcess$", "--", "--appcontrol-helper"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := VerifyProfile(profile); err == nil || !strings.Contains(err.Error(), "yotta host executable") {
		t.Fatalf("VerifyProfile(current executable) error = %v", err)
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
