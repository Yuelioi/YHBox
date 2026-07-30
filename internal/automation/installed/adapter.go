package installed

import (
	"context"
	"errors"
	"fmt"
	"slices"

	"github.com/yottaapp/yotta/internal/artifact"
)

const (
	AdapterKindWin32      = "win32"
	AdapterKindAndroidADB = "android-adb"
	AdapterKindBrowserCDP = "browser-cdp"

	InstallationManifestFormat  = "yotta.automation-installation"
	InstallationManifestVersion = 1
	ProfileVersionV1            = "1"

	// AdapterKindTest exists only to prove the seam with a second adapter in
	// conformance tests. The production registry never installs it.
	AdapterKindTest = "test"
)

// ResourceDescriptor lists the sessions and operations implemented by one
// configured target adapter. It is descriptive runtime configuration, not an
// authorization, consent, or grant record.
type ResourceDescriptor struct {
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
// ResourceKinds and Operations are derived from Resources during registry
// sealing and exist only as convenient read projections.
type TargetTypeDescriptor struct {
	TargetKind      string                   `json:"targetKind"`
	AdapterKind     string                   `json:"adapterKind"`
	ProfileKind     string                   `json:"profileKind"`
	ProfileVersion  string                   `json:"profileVersion"`
	HostAvailable   bool                     `json:"hostAvailable"`
	Resources       []ResourceDescriptor     `json:"resources"`
	ResourceKinds   []string                 `json:"resourceKinds"`
	Operations      []string                 `json:"operations"`
	Fields          []ProfileFieldDescriptor `json:"fields"`
	InputBackends   []string                 `json:"inputBackends"`
	CaptureBackends []string                 `json:"captureBackends"`
}

// InstallationManifestDocument is the canonical, versioned fact set for one
// configured target generation. It is projected into authoring and runtime
// views without an authorization layer.
type InstallationManifestDocument struct {
	Format           string               `json:"format"`
	Version          int                  `json:"version"`
	Slot             string               `json:"slot"`
	Label            string               `json:"label"`
	TargetID         string               `json:"targetId"`
	TargetKind       string               `json:"targetKind"`
	AdapterKind      string               `json:"adapterKind"`
	ProfileKind      string               `json:"profileKind"`
	ProfileVersion   string               `json:"profileVersion"`
	ProfileDigest    artifact.Digest      `json:"profileDigest"`
	ProviderID       string               `json:"providerId"`
	ProviderABI      string               `json:"providerAbi"`
	ProviderArtifact artifact.Digest      `json:"providerArtifact"`
	Resources        []ResourceDescriptor `json:"resources"`
}

type manifestState struct {
	document InstallationManifestDocument
	digest   artifact.Digest
}

// InstallationManifest is immutable after sealing. Machine returns a deep
// copy so callers cannot mutate the installed target description.
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
	document.Resources = cloneResources(document.Resources)
	return document
}

func (manifest InstallationManifest) Descriptor() InstallationDescriptor {
	document := manifest.Machine()
	return InstallationDescriptor{
		Slot: document.Slot, Label: document.Label, TargetID: document.TargetID,
		TargetKind: document.TargetKind, AdapterKind: document.AdapterKind,
		ProviderID: document.ProviderID, ProviderABI: document.ProviderABI,
		ResourceKinds: resourceKinds(document.Resources), Operations: operations(document.Resources),
	}
}

func (manifest InstallationManifest) SupportsResourceKind(kind string) bool {
	if !manifest.Valid() || kind == "" {
		return false
	}
	for _, descriptor := range manifest.state.document.Resources {
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

func RequiresApplication(targetKind, adapterKind string) bool {
	targetType, ok := TargetType(targetKind, adapterKind)
	return ok && targetType.AdapterKind == AdapterKindWin32
}

type productionAdapter struct {
	targetType TargetTypeDescriptor
	seal       func(ProfileDraft) (Profile, error)
	open       func(Profile) (driver, error)
	intent     profileIntentCodec
}

func productionAdapters() []productionAdapter {
	return []productionAdapter{
		{
			targetType: targetTypeDescriptor(
				TargetKindDesktopWindow, AdapterKindWin32, "desktop-window", PlatformSupported(),
				[]ResourceDescriptor{
					targetResource(KindInput, OperationClick, OperationDrag, OperationMove, OperationScroll, OperationTypeText, OperationMoveRelative, OperationPressKeys),
					targetResource(KindHeldInput, OperationHoldKeys, OperationHoldButton, OperationReleaseHeld),
					targetResource(KindWindow, OperationActivate, OperationCloseWindow, OperationGetWindowState, OperationMoveResizeWindow, OperationSetWindowState, OperationWaitWindow, OperationWaitWindowGone),
					targetResource(KindCapture, OperationCapture, OperationReadCapture),
					targetResource(KindPlayback, OperationPlayEvent, OperationReleaseHeld),
				},
				[]ProfileFieldDescriptor{
					field("applicationSlot", "installation-slot", true), field("windowTitle", "string", false),
					fieldOptions("windowTitleMatch", true, "exact", "regex"), fieldOptions("windowSelection", true, "unique", "topmost"),
					field("windowClass", "string", false), fieldOptions("inputBackend", true, "sendinput", "postmessage"),
					fieldOptions("captureBackend", true, "gdi", "wgc"), field("mouseCounts360", "integer", false),
					field("resolveTimeoutMilliseconds", "duration-ms", true),
				},
			),
			seal: sealDesktopProfile,
			open: newPlatformDriver, intent: desktopProfileIntentCodec(),
		},
		{
			targetType: targetTypeDescriptor(
				TargetKindAndroidDevice, AdapterKindAndroidADB, "android-device", true,
				[]ResourceDescriptor{
					targetResource(KindInput, OperationClick, OperationDrag, OperationMove, OperationScroll, OperationTypeText),
					targetResource(KindWindow, OperationActivate, OperationStopApp),
					targetResource(KindCapture, OperationCapture, OperationReadCapture),
					targetResource(KindPlayback, OperationPlayEvent, OperationReleaseHeld),
				},
				[]ProfileFieldDescriptor{
					field("adbSerial", "string", true), field("adbProduct", "string", false), field("adbModel", "string", false),
					field("adbDevice", "string", false), field("androidPackage", "string", true),
					field("resolveTimeoutMilliseconds", "duration-ms", true),
				},
			),
			seal: sealAndroidProfile,
			open: newAndroidDriver, intent: androidProfileIntentCodec(),
		},
		{
			targetType: targetTypeDescriptor(
				TargetKindBrowserCDP, AdapterKindBrowserCDP, "browser-page", true,
				[]ResourceDescriptor{
					targetResource(KindInput, OperationClick, OperationDrag, OperationMove, OperationScroll, OperationTypeText, OperationPressKeys),
					targetResource(KindCapture, OperationCapture, OperationReadCapture),
				},
				[]ProfileFieldDescriptor{
					field("browserEndpoint", "url", true), field("browserTargetId", "string", true),
					field("browserWebSocketUrl", "url", true), field("browserTitle", "string", false),
					field("browserUrl", "url", false), field("resolveTimeoutMilliseconds", "duration-ms", true),
				},
			),
			seal: sealBrowserProfile,
			open: newBrowserDriver, intent: browserProfileIntentCodec(),
		},
	}
}

func targetResource(resourceKind string, operations ...string) ResourceDescriptor {
	return ResourceDescriptor{ResourceKind: resourceKind, Operations: append([]string(nil), operations...)}
}

func field(id, kind string, required bool) ProfileFieldDescriptor {
	return ProfileFieldDescriptor{ID: id, Kind: kind, Required: required}
}

func fieldOptions(id string, required bool, options ...string) ProfileFieldDescriptor {
	return ProfileFieldDescriptor{ID: id, Kind: "enum", Required: required, Options: append([]string(nil), options...)}
}

func targetTypeDescriptor(targetKind, adapterKind, profileKind string, hostAvailable bool, resources []ResourceDescriptor, fields []ProfileFieldDescriptor) TargetTypeDescriptor {
	return TargetTypeDescriptor{
		TargetKind: targetKind, AdapterKind: adapterKind, ProfileKind: profileKind, ProfileVersion: ProfileVersionV1,
		HostAvailable: hostAvailable, Resources: cloneResources(resources),
		ResourceKinds: resourceKinds(resources), Operations: operations(resources), Fields: cloneFields(fields),
		InputBackends: optionsForField(fields, "inputBackend"), CaptureBackends: optionsForField(fields, "captureBackend"),
	}
}

type adapterRegistration struct {
	targetType TargetTypeDescriptor
	seal       func(ProfileDraft) (Profile, error)
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

func (registry adapterRegistry) register(targetType TargetTypeDescriptor, seal func(ProfileDraft) (Profile, error), open func(Profile) (driver, error), intent profileIntentCodec) error {
	sealed, err := sealTargetType(targetType)
	if err != nil || seal == nil || open == nil || intent.draft == nil || intent.applicationSlot == nil {
		return errors.Join(errors.New("automation adapter registration is incomplete"), err)
	}
	if _, exists := registry.byKind[sealed.AdapterKind]; exists {
		return fmt.Errorf("automation adapter %q is already registered", sealed.AdapterKind)
	}
	registry.byKind[sealed.AdapterKind] = adapterRegistration{targetType: sealed, seal: seal, open: open, intent: intent}
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
		_ = registry.register(adapter.targetType, adapter.seal, adapter.open, adapter.intent)
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
		Resources: cloneResources(targetType.Resources),
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

// CheckInstallationHealth resolves a target through the same configured
// adapter used by authoring and execution.
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
	_, err := provider.driver.ResolveTarget(ctx)
	return err
}

func sealTargetType(targetType TargetTypeDescriptor) (TargetTypeDescriptor, error) {
	if targetType.TargetKind == "" || targetType.AdapterKind == "" || targetType.ProfileKind == "" || targetType.ProfileVersion == "" || len(targetType.Resources) == 0 || len(targetType.Fields) == 0 {
		return TargetTypeDescriptor{}, errors.New("automation target type identity or schema is incomplete")
	}
	seenResources := make(map[string]struct{}, len(targetType.Resources))
	for _, descriptor := range targetType.Resources {
		if descriptor.ResourceKind == "" || len(descriptor.Operations) == 0 {
			return TargetTypeDescriptor{}, errors.New("automation target resource is incomplete")
		}
		if _, exists := seenResources[descriptor.ResourceKind]; exists {
			return TargetTypeDescriptor{}, errors.New("automation target resource is duplicated")
		}
		seenResources[descriptor.ResourceKind] = struct{}{}
	}
	targetType.Resources = cloneResources(targetType.Resources)
	targetType.ResourceKinds = resourceKinds(targetType.Resources)
	targetType.Operations = operations(targetType.Resources)
	targetType.Fields = cloneFields(targetType.Fields)
	targetType.InputBackends = optionsForField(targetType.Fields, "inputBackend")
	targetType.CaptureBackends = optionsForField(targetType.Fields, "captureBackend")
	return targetType, nil
}

func cloneTargetType(source TargetTypeDescriptor) TargetTypeDescriptor {
	source.Resources = cloneResources(source.Resources)
	source.ResourceKinds = append([]string(nil), source.ResourceKinds...)
	source.Operations = append([]string(nil), source.Operations...)
	source.Fields = cloneFields(source.Fields)
	source.InputBackends = append([]string(nil), source.InputBackends...)
	source.CaptureBackends = append([]string(nil), source.CaptureBackends...)
	return source
}

func cloneResources(source []ResourceDescriptor) []ResourceDescriptor {
	result := make([]ResourceDescriptor, len(source))
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

func resourceKinds(resources []ResourceDescriptor) []string {
	result := make([]string, 0, len(resources))
	for _, descriptor := range resources {
		if !slices.Contains(result, descriptor.ResourceKind) {
			result = append(result, descriptor.ResourceKind)
		}
	}
	return result
}

func operations(resources []ResourceDescriptor) []string {
	result := make([]string, 0)
	for _, descriptor := range resources {
		for _, operation := range descriptor.Operations {
			if !slices.Contains(result, operation) {
				result = append(result, operation)
			}
		}
	}
	return result
}
