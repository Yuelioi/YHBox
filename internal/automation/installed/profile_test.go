package installed

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/yottaapp/yotta/internal/appcontrol"
)

func testProfile(t *testing.T) (Profile, string) {
	t.Helper()
	path := filepath.Join(t.TempDir(), "target.exe")
	if err := os.WriteFile(path, []byte("installed-automation-target-v1"), 0o700); err != nil {
		t.Fatal(err)
	}
	inspection, err := appcontrol.InspectExecutable(path)
	if err != nil {
		t.Fatal(err)
	}
	profile, err := SealProfile(ProfileDraft{
		Application: appcontrol.ProfileDraft{Executable: inspection.Executable, ExecutableDigest: inspection.Digest, Arguments: []string{}},
		WindowTitle: "Editor", WindowClass: "ExampleWindow", InputBackend: "postmessage", ResolveTimeoutMilliseconds: 250,
	})
	if err != nil {
		t.Fatal(err)
	}
	return profile, path
}

func TestProfileIsCanonicalAndImmutable(t *testing.T) {
	profile, _ := testProfile(t)
	opened, err := OpenProfile(profile.Bytes(), profile.Digest())
	if err != nil {
		t.Fatal(err)
	}
	machine := opened.Machine()
	machine.WindowTitle = "forged"
	machine.Application.Arguments = append(machine.Application.Arguments, "--forged")
	if opened.Machine().WindowTitle != "Editor" || len(opened.Machine().Application.Arguments) != 0 {
		t.Fatal("profile machine view mutated sealed state")
	}
}

func TestProfileRejectsAmbientOrBroadTargetConfiguration(t *testing.T) {
	profile, _ := testProfile(t)
	base := profile.Machine()
	for _, mutate := range []func(*ProfileDraft){
		func(draft *ProfileDraft) { draft.Application.Arguments = []string{"--workflow-controlled"} },
		func(draft *ProfileDraft) { draft.InputBackend = "auto" },
		func(draft *ProfileDraft) { draft.WindowTitle = " padded " },
		func(draft *ProfileDraft) { draft.ResolveTimeoutMilliseconds = MaxResolveTimeoutMilliseconds + 1 },
	} {
		draft := base
		mutate(&draft)
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
	if err := os.WriteFile(path, []byte("installed-automation-target-v2"), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := VerifyProfile(profile); err == nil || !strings.Contains(err.Error(), "identity changed") {
		t.Fatalf("VerifyProfile() error = %v", err)
	}
}
