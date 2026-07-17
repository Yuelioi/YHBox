package installed

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/yottaapp/yotta/internal/resource"
)

func androidProfileDraft() ProfileDraft {
	return ProfileDraft{
		TargetKind: TargetKindAndroidDevice, AdapterKind: AdapterKindAndroidADB, ApplicationIdentityKind: IdentityKindADBDevice,
		ADBSerial: "emulator-5554", ADBProduct: "sdk_gphone64_x86_64", ADBModel: "sdk_gphone64_x86_64", ADBDevice: "emu64xa",
		AndroidPackage: "dev.yotta.fixture", ResolveTimeoutMilliseconds: 1000,
	}
}

func TestParseADBDevicesPreservesStateAndExactIdentity(t *testing.T) {
	devices := parseADBDevices("List of devices attached\r\nemulator-5554 device product:sdk_gphone64_x86_64 model:sdk_gphone64_x86_64 device:emu64xa transport_id:1\r\nUSB123 unauthorized transport_id:2\r\n")
	if len(devices) != 2 || devices[0].Serial != "emulator-5554" || devices[0].Product != "sdk_gphone64_x86_64" || devices[0].Device != "emu64xa" || devices[1].State != "unauthorized" {
		t.Fatalf("devices = %#v", devices)
	}
}

func TestAndroidProfilePinsDeviceAndPackageWithoutDesktopIdentity(t *testing.T) {
	profile, err := SealProfile(androidProfileDraft())
	if err != nil {
		t.Fatal(err)
	}
	if profile.TargetKind() != TargetKindAndroidDevice || profile.AdapterKind() != AdapterKindAndroidADB || profile.Machine().AndroidPackage != "dev.yotta.fixture" {
		t.Fatalf("profile = %#v", profile.Machine())
	}
	draft := androidProfileDraft()
	draft.WindowTitle = "forged desktop identity"
	if _, err := SealProfile(draft); err == nil {
		t.Fatal("accepted Android profile with desktop identity")
	}
}

func TestAndroidProviderRejectsUnsupportedLowLevelInputAtOpen(t *testing.T) {
	profile, err := SealProfile(androidProfileDraft())
	if err != nil {
		t.Fatal(err)
	}
	registry := newAdapterRegistry()
	if err := registry.register(adapterDescriptor{
		Kind: AdapterKindAndroidADB, TargetKinds: []string{TargetKindAndroidDevice},
		ResourceKinds: []string{KindInput, KindWindow, KindCapture}, Operations: androidOperations(),
	}, func(Profile) (driver, error) { return &fakeDriver{}, nil }); err != nil {
		t.Fatal(err)
	}
	provider, err := newProvider(profile, registry)
	if err != nil {
		t.Fatal(err)
	}
	scope, _ := json.Marshal(CapabilityScope{Operation: OperationPressKeys})
	if _, err := provider.Open(context.Background(), resource.ProviderOpenRequest{
		Kind: KindInput, Operations: []string{OperationPressKeys}, Config: []byte(`{}`), CapabilityScope: scope,
	}); err == nil {
		t.Fatal("Android adapter opened unsupported key-state authority")
	}
}
