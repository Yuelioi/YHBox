package appcontrol

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func testProfile(t *testing.T) (Profile, string) {
	t.Helper()
	path := filepath.Join(t.TempDir(), "tool.exe")
	if err := os.WriteFile(path, []byte("installed-tool-v1"), 0o700); err != nil {
		t.Fatal(err)
	}
	inspection, err := InspectExecutable(path)
	if err != nil {
		t.Fatal(err)
	}
	profile, err := SealProfile(ProfileDraft{Executable: inspection.Executable, Arguments: []string{"--fixed", "value with spaces"}})
	if err != nil {
		t.Fatal(err)
	}
	return profile, path
}

func TestProfileSealsExactExecutableAndFixedArguments(t *testing.T) {
	profile, _ := testProfile(t)
	opened, err := OpenProfile(profile.Bytes(), profile.Digest())
	if err != nil || opened.Digest() != profile.Digest() {
		t.Fatalf("OpenProfile() = %#v, %v", opened, err)
	}
	machine := opened.Machine()
	if !filepath.IsAbs(machine.Executable) || len(machine.Arguments) != 2 || machine.Arguments[1] != "value with spaces" {
		t.Fatalf("profile machine = %#v", machine)
	}
	machine.Arguments[0] = "mutated"
	if opened.Machine().Arguments[0] != "--fixed" {
		t.Fatal("profile arguments were mutable")
	}
}

func TestProfileRejectsAmbientAndScriptEntrypoints(t *testing.T) {
	for _, draft := range []ProfileDraft{
		{Executable: "relative.exe"},
		{Executable: filepath.Join(t.TempDir(), "cmd.exe")},
		{Executable: filepath.Join(t.TempDir(), "tool.bat")},
		{Executable: filepath.Join(t.TempDir(), "tool.exe"), Arguments: []string{strings.Repeat("x", MaxArgumentBytes+1)}},
	} {
		if _, err := SealProfile(draft); err == nil {
			t.Fatalf("SealProfile(%#v) succeeded", draft)
		}
	}
}

func TestConfiguredInstallationSurvivesExecutableUpdate(t *testing.T) {
	profile, path := testProfile(t)
	if err := VerifyProfile(profile); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte("installed-tool-v2"), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := VerifyProfile(profile); err != nil {
		t.Fatalf("normal application update invalidated the authorized profile: %v", err)
	}
	installed, err := Install([]InstallationDraft{{Slot: "tool", Profile: profile.Machine()}})
	if err != nil || len(installed.Entries()) != 1 {
		t.Fatalf("updated configured application did not install: %#v, %v", installed, err)
	}
}

func TestInstallationDefersMissingExecutableToInvocation(t *testing.T) {
	profile, path := testProfile(t)
	if err := os.Remove(path); err != nil {
		t.Fatal(err)
	}
	installed, err := Install([]InstallationDraft{{Slot: "tool", Profile: profile.Machine()}})
	if err != nil || len(installed.Entries()) != 1 {
		t.Fatalf("missing executable prevented application startup composition: %#v, %v", installed, err)
	}
}

func TestConfiguredInstallationIsImmediatelyAvailable(t *testing.T) {
	if !PlatformSupported() {
		t.Skip("application lifecycle provider is intentionally unavailable")
	}
	profile, _ := testProfile(t)
	draft := InstallationDraft{Slot: "after-effects", Profile: profile.Machine()}
	installed, err := Install([]InstallationDraft{draft})
	if err != nil || len(installed.Entries()) != 1 ||
		installed.Entries()[0].TargetID != TargetID(draft.Slot) ||
		installed.Entries()[0].Provider == nil {
		t.Fatalf("configured application = %#v, %v", installed, err)
	}
}
