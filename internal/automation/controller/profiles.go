package controller

import "github.com/yottaapp/yotta/internal/automation/target"

type BackendKind string

const (
	BackendWin32       BackendKind = "win32"
	BackendAndroidADB  BackendKind = "android-adb"
	BackendBrowserCDP  BackendKind = "browser-cdp"
	BackendDebugReplay BackendKind = "debug-replay"
	BackendMock        BackendKind = "mock"
)

type BackendProfile struct {
	Backend          BackendKind
	TargetKinds      []string
	CoordinateSpaces []target.CoordinateSpace
	Capabilities     CapabilitySet
}

var backendProfileOrder = []BackendKind{
	BackendWin32,
	BackendAndroidADB,
	BackendBrowserCDP,
	BackendDebugReplay,
	BackendMock,
}

var backendProfiles = map[BackendKind]BackendProfile{
	BackendWin32: {
		Backend:          BackendWin32,
		TargetKinds:      []string{target.KindWin32Window},
		CoordinateSpaces: []target.CoordinateSpace{target.SpaceNormalized, target.SpaceWindowClient, target.SpaceCaptureFrame, target.SpaceScreen},
		Capabilities: CapabilitySet{
			Screenshot:      true,
			Click:           true,
			Move:            true,
			Scroll:          true,
			MouseButton:     true,
			Drag:            true,
			MoveRelative:    true,
			PointerPosition: true,
			KeyChord:        true,
			KeyState:        true,
			Text:            true,
		},
	},
	BackendAndroidADB: {
		Backend:          BackendAndroidADB,
		TargetKinds:      []string{target.KindAndroidADB},
		CoordinateSpaces: []target.CoordinateSpace{target.SpaceNormalized, target.SpaceAndroidDevice, target.SpaceCaptureFrame},
		Capabilities: CapabilitySet{
			Screenshot: true,
			Click:      true,
			Move:       true,
			Scroll:     true,
			Drag:       true,
			Text:       true,
			StartApp:   true,
			StopApp:    true,
		},
	},
	BackendBrowserCDP: {
		Backend:          BackendBrowserCDP,
		TargetKinds:      []string{target.KindBrowserCDP},
		CoordinateSpaces: []target.CoordinateSpace{target.SpaceNormalized, target.SpaceBrowserView, target.SpaceCaptureFrame},
		Capabilities: CapabilitySet{
			Screenshot:  true,
			Click:       true,
			Move:        true,
			Scroll:      true,
			MouseButton: true,
			Drag:        true,
			KeyChord:    true,
			KeyState:    true,
			Text:        true,
		},
	},
	BackendDebugReplay: {
		Backend:          BackendDebugReplay,
		TargetKinds:      []string{target.KindDebugReplay},
		CoordinateSpaces: []target.CoordinateSpace{target.SpaceCaptureFrame},
		Capabilities: CapabilitySet{
			Screenshot: true,
		},
	},
	BackendMock: {
		Backend:          BackendMock,
		TargetKinds:      []string{target.KindMock},
		CoordinateSpaces: []target.CoordinateSpace{target.SpaceNormalized, target.SpaceCaptureFrame},
		Capabilities: CapabilitySet{
			Screenshot:   true,
			Click:        true,
			Move:         true,
			Scroll:       true,
			MouseButton:  true,
			Drag:         true,
			MoveRelative: true,
			KeyChord:     true,
			KeyState:     true,
			Text:         true,
		},
	},
}

func Profile(kind BackendKind) (BackendProfile, bool) {
	profile, ok := backendProfiles[kind]
	if !ok {
		return BackendProfile{}, false
	}
	return cloneProfile(profile), true
}

func Profiles() []BackendProfile {
	out := make([]BackendProfile, 0, len(backendProfiles))
	for _, kind := range backendProfileOrder {
		if profile, ok := backendProfiles[kind]; ok {
			out = append(out, cloneProfile(profile))
		}
	}
	return out
}

func ProfilesForTargetKind(kind string) []BackendProfile {
	out := []BackendProfile{}
	for _, backend := range backendProfileOrder {
		if profile, ok := backendProfiles[backend]; ok && supportsTargetKind(profile, kind) {
			out = append(out, cloneProfile(profile))
		}
	}
	return out
}

func DefaultProfileForTargetKind(kind string) (BackendProfile, bool) {
	for _, backend := range backendProfileOrder {
		if profile, ok := backendProfiles[backend]; ok && supportsTargetKind(profile, kind) {
			return cloneProfile(profile), true
		}
	}
	return BackendProfile{}, false
}

func cloneProfile(profile BackendProfile) BackendProfile {
	profile.TargetKinds = append([]string(nil), profile.TargetKinds...)
	profile.CoordinateSpaces = append([]target.CoordinateSpace(nil), profile.CoordinateSpaces...)
	return profile
}

func supportsTargetKind(profile BackendProfile, kind string) bool {
	for _, candidate := range profile.TargetKinds {
		if candidate == kind {
			return true
		}
	}
	return false
}
