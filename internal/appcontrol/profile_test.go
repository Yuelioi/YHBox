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
	profile, err := SealProfile(ProfileDraft{Executable: inspection.Executable, ExecutableDigest: inspection.Digest, Arguments: []string{"--fixed", "value with spaces"}})
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
	profile, _ := testProfile(t)
	digest := profile.Machine().ExecutableDigest
	for _, draft := range []ProfileDraft{
		{Executable: "relative.exe", ExecutableDigest: digest},
		{Executable: filepath.Join(t.TempDir(), "cmd.exe"), ExecutableDigest: digest},
		{Executable: filepath.Join(t.TempDir(), "tool.bat"), ExecutableDigest: digest},
		{Executable: filepath.Join(t.TempDir(), "tool.exe"), ExecutableDigest: digest, Arguments: []string{strings.Repeat("x", MaxArgumentBytes+1)}},
	} {
		if _, err := SealProfile(draft); err == nil {
			t.Fatalf("SealProfile(%#v) succeeded", draft)
		}
	}
}

func TestVerifyProfileDetectsExecutableDrift(t *testing.T) {
	profile, path := testProfile(t)
	if err := VerifyProfile(profile); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte("installed-tool-v2"), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := VerifyProfile(profile); err == nil || !strings.Contains(err.Error(), "identity changed") {
		t.Fatalf("VerifyProfile() error = %v", err)
	}
}

func TestInstallationRequiresExactConsent(t *testing.T) {
	if !PlatformSupported() {
		t.Skip("application lifecycle provider is intentionally unavailable")
	}
	profile, _ := testProfile(t)
	draft := InstallationDraft{Slot: "after-effects", Profile: profile.Machine()}
	withoutConsent, err := Install([]InstallationDraft{draft})
	if err != nil || len(withoutConsent.Entries()) != 1 || withoutConsent.Entries()[0].Consent != "" {
		t.Fatalf("Install without consent = %#v, %v", withoutConsent, err)
	}
	consent, err := WorkflowConsentDigest(draft.Slot, profile)
	if err != nil {
		t.Fatal(err)
	}
	draft.Consent = consent
	installed, err := Install([]InstallationDraft{draft})
	if err != nil || installed.Entries()[0].Consent != consent {
		t.Fatalf("Install with consent = %#v, %v", installed, err)
	}
	draft.Consent = profile.Digest()
	if _, err := Install([]InstallationDraft{draft}); err == nil {
		t.Fatal("Install accepted stale consent")
	}
}
