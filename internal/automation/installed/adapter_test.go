package installed

import (
	"context"
	"slices"
	"testing"
)

func TestProductionManifestsOwnCapabilityAndEditorProjection(t *testing.T) {
	targetTypes := TargetTypes()
	if len(targetTypes) != 3 {
		t.Fatalf("target types = %#v", targetTypes)
	}
	for _, targetType := range targetTypes {
		sealed, err := sealTargetType(targetType)
		if err != nil {
			t.Fatalf("seal %s: %v", targetType.AdapterKind, err)
		}
		if !slices.Equal(sealed.ResourceKinds, resourceKinds(sealed.Capabilities)) || !slices.Equal(sealed.Operations, operations(sealed.Capabilities)) {
			t.Fatalf("%s carries a second resource/operation fact source: %#v", targetType.AdapterKind, sealed)
		}
		if sealed.ProfileVersion != ProfileVersionV1 || len(sealed.Fields) == 0 {
			t.Fatalf("%s has no versioned editor schema: %#v", targetType.AdapterKind, sealed)
		}
	}

	desktop := targetTypes[0]
	if !slices.Contains(desktop.Operations, OperationPressKeys) ||
		!hasCapability(desktop.Capabilities, CapabilityKeyInputID, KindInput, OperationPressKeys) ||
		!slices.Equal(optionsForField(desktop.Fields, "windowTitleMatch"), []string{"exact", "regex"}) {
		t.Fatalf("desktop manifest lost key input or selector schema: %#v", desktop)
	}
	android := targetTypes[1]
	if slices.Contains(android.Operations, OperationPressKeys) || !hasCapability(android.Capabilities, CapabilityAppLifecycleID, KindWindow, OperationStopApp) {
		t.Fatalf("android manifest authority = %#v", android)
	}
	browser := targetTypes[2]
	if !hasCapability(browser.Capabilities, CapabilityKeyInputID, KindInput, OperationPressKeys) || slices.Contains(browser.ResourceKinds, KindWindow) {
		t.Fatalf("browser manifest authority = %#v", browser)
	}
}

func hasCapability(capabilities []CapabilityDescriptor, capabilityID, resourceKind, operation string) bool {
	for _, descriptor := range capabilities {
		if descriptor.CapabilityID == capabilityID && descriptor.ResourceKind == resourceKind && slices.Contains(descriptor.Operations, operation) {
			return true
		}
	}
	return false
}

func TestInstallationInterfaceUsesRegisteredAdapterDescriptor(t *testing.T) {
	profile, _ := testProfile(t)
	draft := profile.Machine()
	draft.AdapterKind = AdapterKindTest
	registry := defaultAdapterRegistry()
	if _, ok := registry.byKind[AdapterKindWin32]; !ok {
		t.Fatal("production Win32 adapter is not registered")
	}
	opened := &fakeDriver{}
	targetType := cloneTargetType(productionAdapters()[0].targetType)
	targetType.AdapterKind = AdapterKindTest
	if err := registry.register(targetType, sealDesktopProfile, verifyDesktopProfile, func(Profile) (driver, error) { return opened, nil }, desktopProfileIntentCodec()); err != nil {
		t.Fatal(err)
	}
	sealed, err := sealProfileWithRegistry(draft, registry)
	if err != nil {
		t.Fatal(err)
	}

	installations, err := installWithRegistry([]InstallationDraft{{
		Slot: "editor", Label: "Editor", Profile: sealed.Machine(),
	}}, registry)
	if err != nil {
		t.Fatal(err)
	}
	defer installations.Close()
	entries := installations.Entries()
	if len(entries) != 1 {
		t.Fatalf("entries = %#v", entries)
	}
	descriptor := entries[0].Descriptor
	if descriptor.TargetKind != TargetKindDesktopWindow || descriptor.AdapterKind != AdapterKindTest ||
		descriptor.ProviderABI != ProviderABI || len(descriptor.ResourceKinds) != 4 || len(descriptor.Operations) == 0 {
		t.Fatalf("descriptor = %#v", descriptor)
	}
	manifest := entries[0].Manifest.Machine()
	if manifest.Format != InstallationManifestFormat || manifest.Version != InstallationManifestVersion ||
		manifest.ProfileVersion != ProfileVersionV1 || manifest.ProviderID != descriptor.ProviderID || len(manifest.Capabilities) != 6 {
		t.Fatalf("manifest = %#v", manifest)
	}
	provider, ok := entries[0].Provider.(*provider)
	if !ok || !slices.Equal(provider.operations, operations(manifest.Capabilities)) {
		t.Fatalf("provider operations = %#v, manifest = %#v", provider, manifest.Capabilities)
	}
	if err := CheckInstallationHealth(context.Background(), entries[0]); err != nil {
		t.Fatalf("manifest health = %v", err)
	}
	manifest.Capabilities[0].Operations[0] = "forged"
	if entries[0].Manifest.Machine().Capabilities[0].Operations[0] == "forged" {
		t.Fatal("installation manifest exposed mutable authority")
	}
}

func TestRegistryRejectsAdapterTargetKindMismatch(t *testing.T) {
	profile, _ := testProfile(t)
	registry := newAdapterRegistry()
	targetType := cloneTargetType(productionAdapters()[0].targetType)
	targetType.TargetKind = "different-kind"
	if err := registry.register(targetType, sealDesktopProfile, verifyDesktopProfile, func(Profile) (driver, error) {
		return &fakeDriver{}, nil
	}, desktopProfileIntentCodec()); err != nil {
		t.Fatal(err)
	}
	if _, err := registry.open(profile); err == nil {
		t.Fatal("registry opened an adapter for an undeclared target kind")
	}
}
