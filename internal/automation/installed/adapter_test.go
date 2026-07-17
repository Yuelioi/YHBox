package installed

import "testing"

func TestInstallationInterfaceUsesRegisteredAdapterDescriptor(t *testing.T) {
	profile, _ := testProfile(t)
	draft := profile.Machine()
	draft.AdapterKind = AdapterKindTest
	sealed, err := SealProfile(draft)
	if err != nil {
		t.Fatal(err)
	}

	registry := defaultAdapterRegistry()
	if _, ok := registry.byKind[AdapterKindWin32]; !ok {
		t.Fatal("production Win32 adapter is not registered")
	}
	opened := &fakeDriver{}
	if err := registry.register(adapterDescriptor{
		Kind: AdapterKindTest, TargetKinds: []string{TargetKindDesktopWindow}, ResourceKinds: []string{KindInput, KindWindow, KindCapture, KindPlayback}, Operations: desktopOperations(),
	}, func(Profile) (driver, error) { return opened, nil }); err != nil {
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
}

func TestRegistryRejectsAdapterTargetKindMismatch(t *testing.T) {
	profile, _ := testProfile(t)
	registry := newAdapterRegistry()
	if err := registry.register(adapterDescriptor{Kind: AdapterKindWin32, TargetKinds: []string{"different-kind"}, ResourceKinds: []string{KindInput}, Operations: []string{OperationClick}}, func(Profile) (driver, error) {
		return &fakeDriver{}, nil
	}); err != nil {
		t.Fatal(err)
	}
	if _, err := registry.open(profile); err == nil {
		t.Fatal("registry opened an adapter for an undeclared target kind")
	}
}
