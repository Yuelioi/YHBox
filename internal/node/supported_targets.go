package node

import (
	"github.com/yottaapp/yotta/internal/automation/controller"
	"github.com/yottaapp/yotta/internal/automation/target"
)

var publicTargetKinds = []string{
	target.KindWin32Window,
	target.KindAndroidADB,
}

const (
	SupportedTargetWin32Window = target.KindWin32Window
	SupportedTargetAndroidADB  = target.KindAndroidADB
)

var targetSelectionNodeKinds = map[string]string{
	"Win32WindowTarget": target.KindWin32Window,
	"AndroidTarget":     target.KindAndroidADB,
}

// PopulateSupportedTargets returns a copy of spec with SupportedTargets derived
// from the node's existing target/window metadata. Browser CDP is intentionally
// omitted here because it is currently an internal controller, not a product
// target selection node.
func PopulateSupportedTargets(spec Spec) Spec {
	spec.SupportedTargets = SupportedTargetsForSpec(spec)
	return spec
}

func SupportedTargetsForSpec(spec Spec) []string {
	if len(spec.PlatformTargets) > 0 {
		return append([]string(nil), spec.PlatformTargets...)
	}
	if kind, ok := targetSelectionNodeKinds[spec.Kind]; ok {
		return []string{kind}
	}
	if spec.NeedsWindow {
		return []string{target.KindWin32Window}
	}
	if !spec.NeedsTarget {
		return nil
	}
	out := make([]string, 0, len(publicTargetKinds))
	for _, kind := range publicTargetKinds {
		profile, ok := controller.DefaultProfileForTargetKind(kind)
		if !ok {
			continue
		}
		if profileSupportsAll(profile, spec.TargetCapabilities) {
			out = append(out, kind)
		}
	}
	return out
}

func profileSupportsAll(profile controller.BackendProfile, caps []TargetCapability) bool {
	for _, cap := range caps {
		if !profile.Capabilities.Has(controller.Capability(cap)) {
			return false
		}
	}
	return true
}
