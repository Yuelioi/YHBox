package controller

import (
	"testing"

	"github.com/yottaapp/yotta/internal/automation/target"
)

func TestProfileWin32(t *testing.T) {
	profile, ok := Profile(BackendWin32)
	if !ok {
		t.Fatalf("Profile(%q) not found", BackendWin32)
	}
	if profile.Backend != BackendWin32 {
		t.Fatalf("backend = %q, want %q", profile.Backend, BackendWin32)
	}
	if !profile.Capabilities.Click || !profile.Capabilities.KeyState || !profile.Capabilities.MouseButton || !profile.Capabilities.Drag || !profile.Capabilities.MoveRelative || profile.Capabilities.StartApp {
		t.Fatalf("unexpected win32 capabilities: %#v", profile.Capabilities)
	}
	if !hasTargetKind(profile, target.KindWin32Window) {
		t.Fatalf("win32 profile target kinds = %#v", profile.TargetKinds)
	}
	if !hasCoordinateSpace(profile, target.SpaceWindowClient) {
		t.Fatalf("win32 profile coordinate spaces = %#v", profile.CoordinateSpaces)
	}
}

func TestProfileAndroidADB(t *testing.T) {
	profile, ok := DefaultProfileForTargetKind(target.KindAndroidADB)
	if !ok {
		t.Fatalf("DefaultProfileForTargetKind(%q) not found", target.KindAndroidADB)
	}
	if profile.Backend != BackendAndroidADB {
		t.Fatalf("backend = %q, want %q", profile.Backend, BackendAndroidADB)
	}
	if !profile.Capabilities.Screenshot || !profile.Capabilities.Click || !profile.Capabilities.Drag || !profile.Capabilities.StartApp {
		t.Fatalf("unexpected android capabilities: %#v", profile.Capabilities)
	}
	if profile.Capabilities.KeyState || profile.Capabilities.MouseButton || profile.Capabilities.MoveRelative {
		t.Fatalf("android adb should not claim key-state capability: %#v", profile.Capabilities)
	}
	if !hasCoordinateSpace(profile, target.SpaceAndroidDevice) {
		t.Fatalf("android profile coordinate spaces = %#v", profile.CoordinateSpaces)
	}
}

func TestProfileBrowserCDP(t *testing.T) {
	profiles := ProfilesForTargetKind(target.KindBrowserCDP)
	if len(profiles) != 1 {
		t.Fatalf("browser profiles len = %d, want 1", len(profiles))
	}
	profile := profiles[0]
	if profile.Backend != BackendBrowserCDP {
		t.Fatalf("backend = %q, want %q", profile.Backend, BackendBrowserCDP)
	}
	if !profile.Capabilities.KeyChord || !profile.Capabilities.Text || !profile.Capabilities.MouseButton || !profile.Capabilities.Drag || profile.Capabilities.MoveRelative || profile.Capabilities.StartApp {
		t.Fatalf("unexpected browser capabilities: %#v", profile.Capabilities)
	}
	if !hasCoordinateSpace(profile, target.SpaceBrowserView) {
		t.Fatalf("browser profile coordinate spaces = %#v", profile.CoordinateSpaces)
	}
}

func TestProfileUnknowns(t *testing.T) {
	if _, ok := Profile("missing"); ok {
		t.Fatalf("unknown backend returned ok")
	}
	if profiles := ProfilesForTargetKind("missing-target"); len(profiles) != 0 {
		t.Fatalf("unknown target profiles = %#v", profiles)
	}
	if _, ok := DefaultProfileForTargetKind("missing-target"); ok {
		t.Fatalf("unknown target default returned ok")
	}
}

func TestProfilesReturnsStableOrder(t *testing.T) {
	profiles := Profiles()
	want := []BackendKind{BackendWin32, BackendAndroidADB, BackendBrowserCDP, BackendDebugReplay, BackendMock}
	if len(profiles) != len(want) {
		t.Fatalf("profiles len = %d, want %d", len(profiles), len(want))
	}
	for i := range want {
		if profiles[i].Backend != want[i] {
			t.Fatalf("profiles[%d] = %q, want %q", i, profiles[i].Backend, want[i])
		}
	}
}

func TestProfileReturnsCopies(t *testing.T) {
	profile, ok := Profile(BackendWin32)
	if !ok {
		t.Fatalf("Profile(%q) not found", BackendWin32)
	}
	profile.TargetKinds[0] = "mutated"
	profile.CoordinateSpaces[0] = "mutated-space"

	next, ok := Profile(BackendWin32)
	if !ok {
		t.Fatalf("Profile(%q) not found on second lookup", BackendWin32)
	}
	if next.TargetKinds[0] == "mutated" || next.CoordinateSpaces[0] == "mutated-space" {
		t.Fatalf("profile lookup returned mutable backing slices: %#v", next)
	}
}

func hasTargetKind(profile BackendProfile, kind string) bool {
	for _, candidate := range profile.TargetKinds {
		if candidate == kind {
			return true
		}
	}
	return false
}

func hasCoordinateSpace(profile BackendProfile, space target.CoordinateSpace) bool {
	for _, candidate := range profile.CoordinateSpaces {
		if candidate == space {
			return true
		}
	}
	return false
}
