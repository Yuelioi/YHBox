package installed

import (
	"context"
	"errors"
	"fmt"
	"slices"

	"github.com/yottaapp/yotta/internal/artifact"
)

const (
	AdapterKindWin32              = "win32"
	AdapterKindAndroidADB         = "android-adb"
	AdapterKindBrowserCDP         = "browser-cdp"
	IdentityKindWindowsExecutable = "windows-executable"
	IdentityKindADBDevice         = "adb-device"
	IdentityKindBrowserPage       = "browser-page"

	CapabilityInputID        = "https://schemas.yotta.dev/capabilities/automation/input/v1"
	CapabilityDesktopInputID = "https://schemas.yotta.dev/capabilities/automation/desktop-input/v1"
	CapabilityKeyInputID     = "https://schemas.yotta.dev/capabilities/automation/key-input/v1"
	CapabilityHeldInputID    = "https://schemas.yotta.dev/capabilities/automation/held-input/v1"
	CapabilityWindowID       = "https://schemas.yotta.dev/capabilities/automation/window/v1"
	CapabilityAppLifecycleID = "https://schemas.yotta.dev/capabilities/automation/app-lifecycle/v1"
	CapabilityCaptureID      = "https://schemas.yotta.dev/capabilities/automation/capture/v1"
	CapabilityPlaybackID     = "https://schemas.yotta.dev/capabilities/automation/playback/v1"

	InstallationManifestFormat  = "yotta.automation-installation"
	InstallationManifestVersion = 1
	ProfileVersionV1            = "1"

	// AdapterKindTest exists only to prove the seam with a second adapter in
	// conformance tests. The production registry never installs it.
	AdapterKindTest = "test"
)

// CapabilityDescriptor is the adapter-owned mapping from workflow authority
// to the provider resource and operations that implement it. Host projection,
// consent and policy all consume this exact record; none infer capabilities
// from a resource kind or an operation name.
type CapabilityDescriptor struct {
	CapabilityID string   `json:"capabilityId"`
	ResourceKind string   `json:"resourceKind"`
	Operations   []string `json:"operations"`
}

// ProfileFieldDescriptor is transport-neutral editor metadata. It describes
// installation intent without exposing native handles or runtime driver types.
type ProfileFieldDescriptor struct {
	ID       string   `json:"id"`
	Kind     string   `json:"kind"`
	Required bool     `json:"required"`
	Options  []string `json:"options,omitempty"`
}

// TargetTypeDescriptor is the public, adapter-owned installation schema.
// ResourceKinds and Operations are derived from Capabilities during registry
// sealing and exist only as convenient read projections.
type TargetTypeDescriptor struct {
	TargetKind               string                   `json:"targetKind"`
	AdapterKind              string                   `json:"adapterKind"`
	ProfileKind              string                   `json:"profileKind"`
	ProfileVersion           string                   `json:"profileVersion"`
	HostAvailable            bool                     `json:"hostAvailable"`
	Capabilities             []CapabilityDescriptor   `json:"capabilities"`
	ResourceKinds            []string                 `json:"resourceKinds"`
	Operations               []string                 `json:"operations"`
	Fields                   []ProfileFieldDescriptor `json:"fields"`
	InputBackends            []string                 `json:"inputBackends"`
	CaptureBackends          []string                 `json:"captureBackends"`
	ApplicationIdentityKinds []string                 `json:"applicationIdentityKinds"`
}

// InstallationManifestDocument is the canonical, versioned fact set for one
// installed target generation. It is projected into authoring, admission,
// provider and policy views; those projections must not add authority.
type InstallationManifestDocument struct {
	Format           string                 `json:"format"`
	Version          int                    `json:"version"`
	Slot             string                 `json:"slot"`
	Label            string                 `json:"label"`
	TargetID         string                 `json:"targetId"`
	TargetKind       string                 `json:"targetKind"`
	AdapterKind      string                 `json:"adapterKind"`
	ProfileKind      string                 `json:"profileKind"`
	ProfileVersion   string                 `json:"profileVersion"`
	ProfileDigest    artifact.Digest        `json:"profileDigest"`
	ProviderID       string                 `json:"providerId"`
	ProviderABI      string                 `json:"providerAbi"`
	ProviderArtifact artifact.Digest        `json:"providerArtifact"`
	Capabilities     []CapabilityDescriptor `json:"capabilities"`
}

type manifestState struct {
	document InstallationManifestDocument
	digest   artifact.Digest
}

// InstallationManifest is immutable after sealing. Machine returns a deep
// copy so UI and policy callers cannot mutate the installed authority.
type InstallationManifest struct{ state *manifestState }

func (manifest InstallationManifest) Valid() bool {
	return manifest.state != nil && manifest.state.digest.Valid()
}

func (manifest InstallationManifest) Digest() artifact.Digest {
	if !manifest.Valid() {
		return ""
	}
	return manifest.state.digest
}

func (manifest InstallationManifest) Machine() InstallationManifestDocument {
	if !manifest.Valid() {
		return InstallationManifestDocument{}
	}
	document := manifest.state.document
	document.Capabilities = cloneCapabilities(document.Capabilities)
	return document
}

func (manifest InstallationManifest) Descriptor() InstallationDescriptor {
	document := manifest.Machine()
	return InstallationDescriptor{
		Slot: document.Slot, Label: document.Label, TargetID: document.TargetID,
		TargetKind: document.TargetKind, AdapterKind: document.AdapterKind,
		ProviderID: document.ProviderID, ProviderABI: document.ProviderABI,
		ResourceKinds: resourceKinds(document.Capabilities), Operations: operations(document.Capabilities),
	}
}

func (manifest InstallationManifest) SupportsResourceKind(kind string) bool {
	if !manifest.Valid() || kind == "" {
		return false
	}
	for _, descriptor := range manifest.state.document.Capabilities {
		if descriptor.ResourceKind == kind {
			return true
		}
	}
	return false
}

// InstallationDescriptor is the compact workflow-facing projection exposed
// by an installed target. The versioned manifest remains its sole fact source.
type InstallationDescriptor struct {
	Slot          string   `json:"slot"`
	Label         string   `json:"label"`
	TargetID      string   `json:"targetId"`
	TargetKind    string   `json:"targetKind"`
	AdapterKind   string   `json:"adapterKind"`
	ProviderID    string   `json:"providerId"`
	ProviderABI   string   `json:"providerAbi"`
	ResourceKinds []string `json:"resourceKinds"`
	Operations    []string `json:"operations"`
}

func TargetTypes() []TargetTypeDescriptor {
	adapters := productionAdapters()
	result := make([]TargetTypeDescriptor, 0, len(adapters))
	for _, adapter := range adapters {
		result = append(result, cloneTargetType(adapter.targetType))
	}
	return result
}

func TargetType(targetKind, adapterKind string) (TargetTypeDescriptor, bool) {
	for _, adapter := range productionAdapters() {
		if adapter.targetType.TargetKind == targetKind && adapter.targetType.AdapterKind == adapterKind {
			return cloneTargetType(adapter.targetType), true
		}
	}
	return TargetTypeDescriptor{}, false
}

func RequiresApplicationIdentity(targetKind, adapterKind string) bool {
	targetType, ok := TargetType(targetKind, adapterKind)
	return ok && slices.Contains(targetType.ApplicationIdentityKinds, IdentityKindWindowsExecutable)
}

type productionAdapter struct {
	targetType TargetTypeDescriptor
	seal       func(ProfileDraft) (Profile, error)
	verify     func(Profile) error
	open       func(Profile) (driver, error)
	intent     profileIntentCodec
}

func productionAdapters() []productionAdapter {
	return []productionAdapter{
		{
			targetType: targetTypeDescriptor(
				TargetKindDesktopWindow, AdapterKindWin32, "desktop-window", PlatformSupported(),
				[]CapabilityDescriptor{
					capability(CapabilityInputID, KindInput, OperationClick, OperationDrag, OperationMove, OperationScroll, OperationTypeText),
					capability(CapabilityDesktopInputID, KindInput, OperationMoveRelative),
					capability(CapabilityKeyInputID, KindInput, OperationPressKeys),
					capability(CapabilityHeldInputID, KindHeldInput, OperationHoldKeys, OperationHoldButton, OperationReleaseHeld),
					capability(CapabilityWindowID, KindWindow, OperationActivate, OperationCloseWindow, OperationGetWindowState, OperationMoveResizeWindow, OperationSetWindowState, OperationWaitWindow, OperationWaitWindowGone),
					capability(CapabilityCaptureID, KindCapture, OperationCapture, OperationReadCapture),
					capability(CapabilityPlaybackID, KindPlayback, OperationPlayEvent, OperationReleaseHeld),
				},
				[]ProfileFieldDescriptor{
					field("applicationSlot", "installation-slot", true), field("windowTitle", "string", false),
					fieldOptions("windowTitleMatch", true, "exact", "regex"), fieldOptions("windowSelection", true, "unique", "topmost"),
					field("windowClass", "string", false), fieldOptions("inputBackend", true, "sendinput", "postmessage"),
					fieldOptions("captureBackend", true, "gdi", "wgc"), field("mouseCounts360", "integer", false),
					field("resolveTimeoutMilliseconds", "duration-ms", true),
				},
				[]string{IdentityKindWindowsExecutable},
			),
			seal: sealDesktopProfile, verify: verifyDesktopProfile,
			open: newPlatformDriver, intent: desktopProfileIntentCodec(),
		},
		{
			targetType: targetTypeDescriptor(
				TargetKindAndroidDevice, AdapterKindAndroidADB, "android-device", true,
				[]CapabilityDescriptor{
					capability(CapabilityInputID, KindInput, OperationClick, OperationDrag, OperationMove, OperationScroll, OperationTypeText),
					capability(CapabilityWindowID, KindWindow, OperationActivate),
					capability(CapabilityAppLifecycleID, KindWindow, OperationStopApp),
					capability(CapabilityCaptureID, KindCapture, OperationCapture, OperationReadCapture),
					capability(CapabilityPlaybackID, KindPlayback, OperationPlayEvent, OperationReleaseHeld),
				},
				[]ProfileFieldDescriptor{
					field("adbSerial", "string", true), field("adbProduct", "string", true), field("adbModel", "string", true),
					field("adbDevice", "string", true), field("androidPackage", "string", true),
					field("resolveTimeoutMilliseconds", "duration-ms", true),
				},
				[]string{IdentityKindADBDevice},
			),
			seal: sealAndroidProfile, verify: verifyPortableProfile,
			open: newAndroidDriver, intent: androidProfileIntentCodec(),
		},
		{
			targetType: targetTypeDescriptor(
				TargetKindBrowserCDP, AdapterKindBrowserCDP, "browser-page", true,
				[]CapabilityDescriptor{
					capability(CapabilityInputID, KindInput, OperationClick, OperationDrag, OperationMove, OperationScroll, OperationTypeText),
					capability(CapabilityKeyInputID, KindInput, OperationPressKeys),
					capability(CapabilityCaptureID, KindCapture, OperationCapture, OperationReadCapture),
				},
				[]ProfileFieldDescriptor{
					field("browserEndpoint", "url", true), field("browserTargetId", "string", true),
					field("browserWebSocketUrl", "url", true), field("browserTitle", "string", false),
					field("browserUrl", "url", false), field("resolveTimeoutMilliseconds", "duration-ms", true),
				},
				[]string{IdentityKindBrowserPage},
			),
			seal: sealBrowserProfile, verify: verifyPortableProfile,
			open: newBrowserDriver, intent: browserProfileIntentCodec(),
		},
	}
}

func capability(id, resourceKind string, operations ...string) CapabilityDescriptor {
	return CapabilityDescriptor{CapabilityID: id, ResourceKind: resourceKind, Operations: append([]string(nil), operations...)}
}

func field(id, kind string, required bool) ProfileFieldDescriptor {
	return ProfileFieldDescriptor{ID: id, Kind: kind, Required: required}
}

func fieldOptions(id string, required bool, options ...string) ProfileFieldDescriptor {
	return ProfileFieldDescriptor{ID: id, Kind: "enum", Required: required, Options: append([]string(nil), options...)}
}

func targetTypeDescriptor(targetKind, adapterKind, profileKind string, hostAvailable bool, capabilities []CapabilityDescriptor, fields []ProfileFieldDescriptor, identityKinds []string) TargetTypeDescriptor {
	return TargetTypeDescriptor{
		TargetKind: targetKind, AdapterKind: adapterKind, ProfileKind: profileKind, ProfileVersion: ProfileVersionV1,
		HostAvailable: hostAvailable, Capabilities: cloneCapabilities(capabilities),
		ResourceKinds: resourceKinds(capabilities), Operations: operations(capabilities), Fields: cloneFields(fields),
		InputBackends: optionsForField(fields, "inputBackend"), CaptureBackends: optionsForField(fields, "captureBackend"),
		ApplicationIdentityKinds: append([]string(nil), identityKinds...),
	}
}

type adapterRegistration struct {
	targetType TargetTypeDescriptor
	seal       func(ProfileDraft) (Profile, error)
	verify     func(Profile) error
	open       func(Profile) (driver, error)
	intent     profileIntentCodec
}

// adapterRegistry is an internal seam: production and conformance tests use
// the same manifest interface while selecting different native adapters.
type adapterRegistry struct {
	byKind map[string]adapterRegistration
}

func newAdapterRegistry() adapterRegistry {
	return adapterRegistry{byKind: make(map[string]adapterRegistration)}
}

func (registry adapterRegistry) register(targetType TargetTypeDescriptor, seal func(ProfileDraft) (Profile, error), verify func(Profile) error, open func(Profile) (driver, error), intent profileIntentCodec) error {
	sealed, err := sealTargetType(targetType)
	if err != nil || seal == nil || verify == nil || open == nil || intent.draft == nil || intent.applicationSlot == nil {
		return errors.Join(errors.New("automation adapter registration is incomplete"), err)
	}
	if _, exists := registry.byKind[sealed.AdapterKind]; exists {
		return fmt.Errorf("automation adapter %q is already registered", sealed.AdapterKind)
	}
	registry.byKind[sealed.AdapterKind] = adapterRegistration{targetType: sealed, seal: seal, verify: verify, open: open, intent: intent}
	return nil
}

func (registry adapterRegistry) open(profile Profile) (driver, error) {
	registered, err := registry.registration(profile)
	if err != nil {
		return nil, err
	}
	return registered.open(profile)
}

func (registry adapterRegistry) registration(profile Profile) (adapterRegistration, error) {
	if !profile.Valid() {
		return adapterRegistration{}, errors.New("automation adapter requires a sealed profile")
	}
	registered, ok := registry.byKind[profile.AdapterKind()]
	if !ok || registered.targetType.TargetKind != profile.TargetKind() {
		return adapterRegistration{}, failure(CodeUnsupportedHost, fmt.Errorf("automation adapter %q for %q is unavailable on this host", profile.AdapterKind(), profile.TargetKind()))
	}
	return registered, nil
}

func defaultAdapterRegistry() adapterRegistry {
	registry := newAdapterRegistry()
	for _, adapter := range productionAdapters() {
		_ = registry.register(adapter.targetType, adapter.seal, adapter.verify, adapter.open, adapter.intent)
	}
	return registry
}

func sealInstallationManifest(slot, label, targetID, providerID string, providerArtifact artifact.Digest, profile Profile, registered adapterRegistration) (InstallationManifest, error) {
	if slot == "" || targetID == "" || providerID == "" || !providerArtifact.Valid() || !profile.Valid() {
		return InstallationManifest{}, errors.New("automation installation manifest identity is incomplete")
	}
	targetType := registered.targetType
	document := InstallationManifestDocument{
		Format: InstallationManifestFormat, Version: InstallationManifestVersion,
		Slot: slot, Label: label, TargetID: targetID, TargetKind: targetType.TargetKind, AdapterKind: targetType.AdapterKind,
		ProfileKind: targetType.ProfileKind, ProfileVersion: targetType.ProfileVersion, ProfileDigest: profile.Digest(),
		ProviderID: providerID, ProviderABI: ProviderABI, ProviderArtifact: providerArtifact,
		Capabilities: cloneCapabilities(targetType.Capabilities),
	}
	raw, err := artifact.Marshal(document)
	if err != nil {
		return InstallationManifest{}, err
	}
	digest, err := artifact.Sum("yotta/automation-installation-manifest/v1", raw)
	if err != nil {
		return InstallationManifest{}, err
	}
	return InstallationManifest{state: &manifestState{document: document, digest: digest}}, nil
}

// CheckInstallationHealth resolves a target through the provider sealed by
// the same manifest used by authoring, admission, policy, and execution.
func CheckInstallationHealth(ctx context.Context, installation Installation) error {
	if ctx == nil {
		return errors.New("automation target health context is required")
	}
	document := installation.Manifest.Machine()
	provider, ok := installation.Provider.(*provider)
	if !installation.Profile.Valid() || !installation.Manifest.Valid() || !ok || provider == nil ||
		document.ProfileDigest != installation.Profile.Digest() || document.ProviderID != installation.ProviderID ||
		document.ProviderArtifact != installation.ProviderArtifact {
		return errors.New("automation target health installation is invalid")
	}
	if err := provider.verifyProfile(); err != nil {
		return err
	}
	_, err := provider.driver.ResolveTarget(ctx)
	return err
}

func sealTargetType(targetType TargetTypeDescriptor) (TargetTypeDescriptor, error) {
	if targetType.TargetKind == "" || targetType.AdapterKind == "" || targetType.ProfileKind == "" || targetType.ProfileVersion == "" || len(targetType.Capabilities) == 0 || len(targetType.Fields) == 0 {
		return TargetTypeDescriptor{}, errors.New("automation target type identity or schema is incomplete")
	}
	seenCapabilities := make(map[string]struct{}, len(targetType.Capabilities))
	for _, descriptor := range targetType.Capabilities {
		if descriptor.CapabilityID == "" || descriptor.ResourceKind == "" || len(descriptor.Operations) == 0 {
			return TargetTypeDescriptor{}, errors.New("automation target capability is incomplete")
		}
		key := descriptor.CapabilityID + "\x00" + descriptor.ResourceKind
		if _, exists := seenCapabilities[key]; exists {
			return TargetTypeDescriptor{}, errors.New("automation target capability is duplicated")
		}
		seenCapabilities[key] = struct{}{}
	}
	targetType.Capabilities = cloneCapabilities(targetType.Capabilities)
	targetType.ResourceKinds = resourceKinds(targetType.Capabilities)
	targetType.Operations = operations(targetType.Capabilities)
	targetType.Fields = cloneFields(targetType.Fields)
	targetType.InputBackends = optionsForField(targetType.Fields, "inputBackend")
	targetType.CaptureBackends = optionsForField(targetType.Fields, "captureBackend")
	targetType.ApplicationIdentityKinds = append([]string(nil), targetType.ApplicationIdentityKinds...)
	return targetType, nil
}

func cloneTargetType(source TargetTypeDescriptor) TargetTypeDescriptor {
	source.Capabilities = cloneCapabilities(source.Capabilities)
	source.ResourceKinds = append([]string(nil), source.ResourceKinds...)
	source.Operations = append([]string(nil), source.Operations...)
	source.Fields = cloneFields(source.Fields)
	source.InputBackends = append([]string(nil), source.InputBackends...)
	source.CaptureBackends = append([]string(nil), source.CaptureBackends...)
	source.ApplicationIdentityKinds = append([]string(nil), source.ApplicationIdentityKinds...)
	return source
}

func cloneCapabilities(source []CapabilityDescriptor) []CapabilityDescriptor {
	result := make([]CapabilityDescriptor, len(source))
	for index, descriptor := range source {
		result[index] = descriptor
		result[index].Operations = append([]string(nil), descriptor.Operations...)
	}
	return result
}

func cloneFields(source []ProfileFieldDescriptor) []ProfileFieldDescriptor {
	result := make([]ProfileFieldDescriptor, len(source))
	for index, descriptor := range source {
		result[index] = descriptor
		result[index].Options = append([]string(nil), descriptor.Options...)
	}
	return result
}

func optionsForField(fields []ProfileFieldDescriptor, id string) []string {
	for _, descriptor := range fields {
		if descriptor.ID == id {
			return append([]string(nil), descriptor.Options...)
		}
	}
	return nil
}

func resourceKinds(capabilities []CapabilityDescriptor) []string {
	result := make([]string, 0, len(capabilities))
	for _, descriptor := range capabilities {
		if !slices.Contains(result, descriptor.ResourceKind) {
			result = append(result, descriptor.ResourceKind)
		}
	}
	return result
}

func operations(capabilities []CapabilityDescriptor) []string {
	result := make([]string, 0)
	for _, descriptor := range capabilities {
		for _, operation := range descriptor.Operations {
			if !slices.Contains(result, operation) {
				result = append(result, operation)
			}
		}
	}
	return result
}
