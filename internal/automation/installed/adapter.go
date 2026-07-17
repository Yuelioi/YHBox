package installed

import (
	"errors"
	"fmt"
	"slices"
)

const (
	AdapterKindWin32              = "win32"
	AdapterKindAndroidADB         = "android-adb"
	AdapterKindBrowserCDP         = "browser-cdp"
	IdentityKindWindowsExecutable = "windows-executable"
	IdentityKindADBDevice         = "adb-device"
	IdentityKindBrowserPage       = "browser-page"
	// AdapterKindTest exists only to prove the seam with a second adapter in
	// conformance tests. The production registry never installs it.
	AdapterKindTest = "test"
)

// InstallationDescriptor is the complete workflow-facing authority exposed
// by an installed target. Callers do not need to know its native profile.
type InstallationDescriptor struct {
	Slot              string   `json:"slot"`
	Label             string   `json:"label"`
	TargetID          string   `json:"targetId"`
	TargetKind        string   `json:"targetKind"`
	AdapterKind       string   `json:"adapterKind"`
	ProviderID        string   `json:"providerId"`
	ProviderABI       string   `json:"providerAbi"`
	ResourceKinds     []string `json:"resourceKinds"`
	Operations        []string `json:"operations"`
	WorkflowConsented bool     `json:"workflowConsented"`
}

// TargetTypeDescriptor drives host UI and composition without exposing native
// driver types or executable/window identity.
type TargetTypeDescriptor struct {
	TargetKind               string   `json:"targetKind"`
	AdapterKind              string   `json:"adapterKind"`
	ProfileKind              string   `json:"profileKind"`
	HostAvailable            bool     `json:"hostAvailable"`
	ResourceKinds            []string `json:"resourceKinds"`
	Operations               []string `json:"operations"`
	InputBackends            []string `json:"inputBackends"`
	CaptureBackends          []string `json:"captureBackends"`
	ApplicationIdentityKinds []string `json:"applicationIdentityKinds"`
}

func TargetTypes() []TargetTypeDescriptor {
	adapters := productionAdapters()
	result := make([]TargetTypeDescriptor, 0, len(adapters))
	for _, adapter := range adapters {
		descriptor := adapter.targetType
		descriptor.ResourceKinds = append([]string(nil), descriptor.ResourceKinds...)
		descriptor.Operations = append([]string(nil), descriptor.Operations...)
		descriptor.InputBackends = append([]string(nil), descriptor.InputBackends...)
		descriptor.CaptureBackends = append([]string(nil), descriptor.CaptureBackends...)
		descriptor.ApplicationIdentityKinds = append([]string(nil), descriptor.ApplicationIdentityKinds...)
		result = append(result, descriptor)
	}
	return result
}

type productionAdapter struct {
	targetType TargetTypeDescriptor
	open       func(Profile) (driver, error)
}

func productionAdapters() []productionAdapter {
	return []productionAdapter{
		{
			targetType: TargetTypeDescriptor{
				TargetKind: TargetKindDesktopWindow, AdapterKind: AdapterKindWin32, ProfileKind: "desktop-window", HostAvailable: PlatformSupported(),
				ResourceKinds: []string{KindInput, KindWindow, KindCapture, KindPlayback}, Operations: desktopOperations(),
				InputBackends: []string{"sendinput", "postmessage"}, CaptureBackends: []string{"gdi", "wgc"},
				ApplicationIdentityKinds: []string{IdentityKindWindowsExecutable},
			},
			open: newPlatformDriver,
		},
		{
			targetType: TargetTypeDescriptor{
				TargetKind: TargetKindAndroidDevice, AdapterKind: AdapterKindAndroidADB, ProfileKind: "android-device", HostAvailable: true,
				ResourceKinds: []string{KindInput, KindWindow, KindCapture}, Operations: androidOperations(),
				ApplicationIdentityKinds: []string{IdentityKindADBDevice},
			},
			open: newAndroidDriver,
		},
		{
			targetType: TargetTypeDescriptor{
				TargetKind: TargetKindBrowserCDP, AdapterKind: AdapterKindBrowserCDP, ProfileKind: "browser-page", HostAvailable: true,
				ResourceKinds: []string{KindInput, KindCapture}, Operations: browserOperations(),
				ApplicationIdentityKinds: []string{IdentityKindBrowserPage},
			},
			open: newBrowserDriver,
		},
	}
}

type adapterDescriptor struct {
	Kind          string
	TargetKinds   []string
	ResourceKinds []string
	Operations    []string
}

type adapterRegistration struct {
	descriptor adapterDescriptor
	open       func(Profile) (driver, error)
}

// adapterRegistry is an internal seam: production and conformance tests use
// the same installation interface while selecting different native adapters.
type adapterRegistry struct {
	byKind map[string]adapterRegistration
}

func newAdapterRegistry() adapterRegistry {
	return adapterRegistry{byKind: make(map[string]adapterRegistration)}
}

func (r adapterRegistry) register(descriptor adapterDescriptor, open func(Profile) (driver, error)) error {
	if descriptor.Kind == "" || len(descriptor.TargetKinds) == 0 || len(descriptor.ResourceKinds) == 0 || len(descriptor.Operations) == 0 || open == nil {
		return errors.New("automation adapter registration is incomplete")
	}
	if _, exists := r.byKind[descriptor.Kind]; exists {
		return fmt.Errorf("automation adapter %q is already registered", descriptor.Kind)
	}
	descriptor.TargetKinds = append([]string(nil), descriptor.TargetKinds...)
	descriptor.ResourceKinds = append([]string(nil), descriptor.ResourceKinds...)
	descriptor.Operations = append([]string(nil), descriptor.Operations...)
	r.byKind[descriptor.Kind] = adapterRegistration{descriptor: descriptor, open: open}
	return nil
}

func (r adapterRegistry) open(profile Profile) (driver, error) {
	registered, err := r.registration(profile)
	if err != nil {
		return nil, err
	}
	return registered.open(profile)
}

func (r adapterRegistry) registration(profile Profile) (adapterRegistration, error) {
	if !profile.Valid() {
		return adapterRegistration{}, errors.New("automation adapter requires a sealed profile")
	}
	registered, ok := r.byKind[profile.AdapterKind()]
	if !ok || !slices.Contains(registered.descriptor.TargetKinds, profile.TargetKind()) {
		return adapterRegistration{}, failure(CodeUnsupportedHost, fmt.Errorf("automation adapter %q for %q is unavailable on this host", profile.AdapterKind(), profile.TargetKind()))
	}
	return registered, nil
}

func defaultAdapterRegistry() adapterRegistry {
	registry := newAdapterRegistry()
	for _, adapter := range productionAdapters() {
		targetType := adapter.targetType
		_ = registry.register(adapterDescriptor{
			Kind: targetType.AdapterKind, TargetKinds: []string{targetType.TargetKind},
			ResourceKinds: targetType.ResourceKinds, Operations: targetType.Operations,
		}, adapter.open)
	}
	return registry
}

func descriptorFor(slot, label, targetID, providerID string, profile Profile, adapter adapterDescriptor, consented bool) InstallationDescriptor {
	return InstallationDescriptor{
		Slot: slot, Label: label, TargetID: targetID, TargetKind: profile.TargetKind(), AdapterKind: profile.AdapterKind(),
		ProviderID: providerID, ProviderABI: ProviderABI,
		ResourceKinds: append([]string(nil), adapter.ResourceKinds...), Operations: append([]string(nil), adapter.Operations...), WorkflowConsented: consented,
	}
}

func desktopOperations() []string {
	operations := Operations()
	return slices.DeleteFunc(operations, func(operation string) bool { return operation == OperationStopApp })
}

func androidOperations() []string {
	return []string{OperationActivate, OperationCapture, OperationClick, OperationDrag, OperationMove, OperationReadCapture, OperationScroll, OperationStopApp, OperationTypeText}
}

func browserOperations() []string {
	return []string{OperationCapture, OperationClick, OperationDrag, OperationMove, OperationPressKeys, OperationReadCapture, OperationScroll, OperationTypeText}
}
