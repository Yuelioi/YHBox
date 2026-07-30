package installed

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/yottaapp/yotta/internal/appcontrol"
	"github.com/yottaapp/yotta/internal/artifact"
)

func testProfile(t *testing.T) (Profile, string) {
	t.Helper()
	path := filepath.Join(t.TempDir(), "target.exe")
	if err := os.WriteFile(path, []byte("installed-automation-target-v1"), 0o700); err != nil {
		t.Fatal(err)
	}
	profile, err := SealProfile(NewDesktopProfileDraft(DesktopProfilePayload{
		Application: appcontrol.ProfileDraft{Executable: path, Arguments: []string{}},
		WindowTitle: "Editor", WindowTitleMatch: "exact", WindowSelection: "unique", WindowClass: "ExampleWindow",
		InputBackend: "postmessage", CaptureBackend: "gdi", ResolveTimeoutMilliseconds: 250,
	}))
	if err != nil {
		t.Fatal(err)
	}
	return profile, path
}

func desktopPayload(t *testing.T, profile Profile) DesktopProfilePayload {
	t.Helper()
	payload, ok := DesktopProfile(profile)
	if !ok {
		t.Fatal("profile has no desktop payload")
	}
	return payload
}

func TestProfileIsCanonicalVersionedAndImmutable(t *testing.T) {
	profile, _ := testProfile(t)
	opened, err := OpenProfile(profile.Bytes(), profile.Digest())
	if err != nil {
		t.Fatal(err)
	}
	payload := desktopPayload(t, opened)
	payload.WindowTitle = "forged"
	payload.Application.Arguments = append(payload.Application.Arguments, "--forged")
	stable := desktopPayload(t, opened)
	if stable.WindowTitle != "Editor" || len(stable.Application.Arguments) != 0 {
		t.Fatal("profile payload view mutated sealed state")
	}
	draft := opened.Machine()
	draft.ProfileVersion = "999"
	if _, err := SealProfile(draft); err == nil {
		t.Fatal("profile accepted an adapter-unknown payload version")
	}
}

func TestAdapterOwnsSettingsIntentDecodingAndApplicationResolution(t *testing.T) {
	profile, _ := testProfile(t)
	desktop := desktopPayload(t, profile)
	intent := DesktopProfileIntent{
		ApplicationSlot: "editor", WindowTitle: desktop.WindowTitle, WindowTitleMatch: desktop.WindowTitleMatch,
		WindowSelection: desktop.WindowSelection, WindowClass: desktop.WindowClass, InputBackend: desktop.InputBackend,
		CaptureBackend: desktop.CaptureBackend, MouseCounts360: desktop.MouseCounts360,
		ResolveTimeoutMilliseconds: desktop.ResolveTimeoutMilliseconds,
	}
	raw, err := artifact.Marshal(intent)
	if err != nil {
		t.Fatal(err)
	}
	draft, err := ProfileDraftFromIntent(TargetKindDesktopWindow, AdapterKindWin32, ProfileVersionV1, raw, func(slot string) (appcontrol.ProfileDraft, bool) {
		return desktop.Application, slot == "editor"
	})
	if err != nil {
		t.Fatal(err)
	}
	sealed, err := SealProfile(draft)
	if err != nil || sealed.Digest() != profile.Digest() {
		t.Fatalf("intent profile = %q, %v; want %q", sealed.Digest(), err, profile.Digest())
	}
	if slot, err := ApplicationSlotFromIntent(TargetKindDesktopWindow, AdapterKindWin32, ProfileVersionV1, raw); err != nil || slot != "editor" {
		t.Fatalf("application slot = %q, %v", slot, err)
	}
	forged := append(append([]byte(nil), raw[:len(raw)-1]...), []byte(`,"forged":true}`)...)
	if _, err := ProfileDraftFromIntent(TargetKindDesktopWindow, AdapterKindWin32, ProfileVersionV1, forged, func(string) (appcontrol.ProfileDraft, bool) {
		return desktop.Application, true
	}); err == nil {
		t.Fatal("adapter accepted an unknown settings intent field")
	}
}

func TestProfileRejectsInvalidTargetConfiguration(t *testing.T) {
	profile, _ := testProfile(t)
	base := desktopPayload(t, profile)
	for _, mutate := range []func(*DesktopProfilePayload){
		func(payload *DesktopProfilePayload) {
			payload.Application.Arguments = []string{"--workflow-controlled"}
		},
		func(payload *DesktopProfilePayload) { payload.InputBackend = "auto" },
		func(payload *DesktopProfilePayload) { payload.CaptureBackend = "auto" },
		func(payload *DesktopProfilePayload) { payload.MouseCounts360 = 10_000_001 },
		func(payload *DesktopProfilePayload) { payload.WindowTitle = "   " },
	} {
		payload := base
		mutate(&payload)
		if _, err := SealProfile(NewDesktopProfileDraft(payload)); err == nil {
			t.Fatalf("SealProfile(%#v) succeeded", payload)
		}
	}
}

func TestProfilePreservesExactWindowIdentityWhitespace(t *testing.T) {
	profile, _ := testProfile(t)
	payload := desktopPayload(t, profile)
	payload.WindowTitle = "异环  "
	sealed, err := SealProfile(NewDesktopProfileDraft(payload))
	if err != nil {
		t.Fatal(err)
	}
	if got := desktopPayload(t, sealed).WindowTitle; got != payload.WindowTitle {
		t.Fatalf("window title = %q, want exact captured identity %q", got, payload.WindowTitle)
	}
}

func TestProfileAcceptsExactAndRegexWindowTitleModes(t *testing.T) {
	profile, _ := testProfile(t)
	for _, test := range []struct {
		mode  string
		title string
	}{
		{mode: "exact", title: "异环  "},
		{mode: "regex", title: `^异环\s*$`},
	} {
		payload := desktopPayload(t, profile)
		payload.WindowTitleMatch = test.mode
		payload.WindowTitle = test.title
		sealed, err := SealProfile(NewDesktopProfileDraft(payload))
		if err != nil {
			t.Fatalf("SealProfile(%s): %v", test.mode, err)
		}
		if got := desktopPayload(t, sealed).WindowTitleMatch; got != test.mode {
			t.Fatalf("title match = %q, want %q", got, test.mode)
		}
	}
	payload := desktopPayload(t, profile)
	payload.WindowTitleMatch = "regex"
	payload.WindowTitle = "[invalid"
	if _, err := SealProfile(NewDesktopProfileDraft(payload)); err == nil {
		t.Fatal("invalid title regex was accepted")
	}
}

func TestProfileRequiresExplicitSupportedWindowSelectionPolicy(t *testing.T) {
	profile, _ := testProfile(t)
	payload := desktopPayload(t, profile)
	if payload.WindowSelection != "unique" {
		t.Fatalf("window selection = %q, want unique", payload.WindowSelection)
	}
	payload.WindowSelection = "topmost"
	sealed, err := SealProfile(NewDesktopProfileDraft(payload))
	if err != nil || desktopPayload(t, sealed).WindowSelection != "topmost" {
		t.Fatalf("topmost selection = %q, %v", desktopPayload(t, sealed).WindowSelection, err)
	}
	payload.WindowSelection = "random"
	if _, err := SealProfile(NewDesktopProfileDraft(payload)); err == nil {
		t.Fatal("unsupported window selection policy was accepted")
	}
}
