package installed

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/yottaapp/yotta/internal/automation/controller"
	"github.com/yottaapp/yotta/internal/automation/target"
	"github.com/yottaapp/yotta/internal/resource"
)

func androidProfileDraft() ProfileDraft {
	return NewAndroidProfileDraft(AndroidProfilePayload{
		ADBSerial: "emulator-5554", ADBProduct: "sdk_gphone64_x86_64", ADBModel: "sdk_gphone64_x86_64", ADBDevice: "emu64xa",
		AndroidPackage: "dev.yotta.fixture", ResolveTimeoutMilliseconds: 1000,
	})
}

func TestParseADBDevicesPreservesStateAndExactIdentity(t *testing.T) {
	devices := parseADBDevices("List of devices attached\r\nemulator-5554 device product:sdk_gphone64_x86_64 model:sdk_gphone64_x86_64 device:emu64xa transport_id:1\r\nUSB123 unauthorized transport_id:2\r\n")
	if len(devices) != 2 || devices[0].Serial != "emulator-5554" || devices[0].Product != "sdk_gphone64_x86_64" || devices[0].Device != "emu64xa" || devices[1].State != "unauthorized" {
		t.Fatalf("devices = %#v", devices)
	}
}

func TestParseAndroidApplicationDiscoveryKeepsExactPackagesAndLabels(t *testing.T) {
	if got := parseAndroidForegroundPackage("mCurrentFocus=Window{f u0 com.example.game/.MainActivity}\n"); got != "com.example.game" {
		t.Fatalf("foreground package = %q", got)
	}
	packages := parseAndroidPackageList("package:com.example.game\npackage:com.example.tool\nnot-a-package\n")
	if len(packages) != 2 {
		t.Fatalf("packages = %#v", packages)
	}
	apps := parseAndroidLauncherApps("Activity #0:\npackageName=com.example.game\nnonLocalizedLabel=Example Game\nActivity #1:\npackageName=com.example.tool\nnonLocalizedLabel=null\n")
	if len(apps) != 2 {
		t.Fatalf("apps = %#v", apps)
	}
	byPackage := map[string]AndroidAppDescriptor{}
	for _, app := range apps {
		byPackage[app.Package] = app
	}
	if byPackage["com.example.game"].Label != "Example Game" || byPackage["com.example.tool"].Label != "com.example.tool" {
		t.Fatalf("apps = %#v", apps)
	}
}

func TestAndroidDeviceDiscoveryReconnectsKnownLocalEmulators(t *testing.T) {
	deviceLists := 0
	connects := 0
	runner := controller.ADBRunnerFunc(func(_ context.Context, serial string, args ...string) ([]byte, error) {
		if serial != "" {
			t.Fatalf("unexpected serial %q", serial)
		}
		switch strings.Join(args, " ") {
		case "devices -l":
			deviceLists++
			if deviceLists == 1 {
				return []byte("List of devices attached\n"), nil
			}
			return []byte("List of devices attached\n127.0.0.1:16384 device product:Sandy model:SDY_AN00 device:Sandy transport_id:2\n"), nil
		default:
			if len(args) == 2 && args[0] == "connect" {
				connects++
				return nil, nil
			}
			t.Fatalf("unexpected ADB command %v", args)
			return nil, nil
		}
	})
	devices, err := discoverAndroidDevices(context.Background(), runner)
	if err != nil {
		t.Fatal(err)
	}
	if len(devices) != 1 || devices[0].Serial != "127.0.0.1:16384" || connects != len(commonADBEmulatorAddresses) || deviceLists != 2 {
		t.Fatalf("devices=%#v connects=%d lists=%d", devices, connects, deviceLists)
	}
}

func TestAndroidAppDiscoveryFallsBackAndDiagnosesUnauthorizedDevice(t *testing.T) {
	runner := controller.ADBRunnerFunc(func(_ context.Context, serial string, args ...string) ([]byte, error) {
		if serial != "emulator-5554" {
			t.Fatalf("serial = %q", serial)
		}
		switch strings.Join(args, " ") {
		case "shell dumpsys window":
			return []byte("mCurrentFocus=Window{f u0 com.example.game/.MainActivity}\n"), nil
		case "shell pm list packages -3":
			return []byte("package:com.example.game\npackage:com.example.tool\n"), nil
		case "shell cmd package query-activities -a android.intent.action.MAIN -c android.intent.category.LAUNCHER":
			return nil, errors.New("query unavailable")
		default:
			t.Fatalf("unexpected ADB command %v", args)
			return nil, nil
		}
	})
	apps, err := discoverAndroidApps(context.Background(), "emulator-5554", runner)
	if err != nil {
		t.Fatal(err)
	}
	if len(apps) != 2 || apps[0].Package != "com.example.game" || !apps[0].Foreground || apps[1].Package != "com.example.tool" {
		t.Fatalf("apps = %#v", apps)
	}

	unauthorized := controller.ADBRunnerFunc(func(context.Context, string, ...string) ([]byte, error) {
		return []byte("error: device unauthorized"), errors.New("exit status 1")
	})
	if _, err := discoverAndroidApps(context.Background(), "emulator-5554", unauthorized); err == nil || !strings.Contains(err.Error(), "unauthorized") {
		t.Fatalf("unauthorized discovery error = %v", err)
	}
}

func TestAndroidResolutionUsesExactDeviceAndStableOfflineDiagnostic(t *testing.T) {
	profile, err := SealProfile(androidProfileDraft())
	if err != nil {
		t.Fatal(err)
	}
	runner := controller.ADBRunnerFunc(func(_ context.Context, _ string, args ...string) ([]byte, error) {
		switch strings.Join(args, " ") {
		case "devices -l":
			return []byte("List of devices attached\nother-device device product:other model:other device:other transport_id:1\nemulator-5554 device product:sdk_gphone64_x86_64 model:sdk_gphone64_x86_64 device:emu64xa transport_id:2\n"), nil
		case "shell wm size":
			return []byte("Physical size: 1080x1920\n"), nil
		default:
			t.Fatalf("unexpected ADB command %v", args)
			return nil, nil
		}
	})
	driver := &androidDriver{profile: profile, runner: runner}
	resolved, err := driver.ResolveTarget(context.Background())
	if err != nil || resolved.Ref.ADBSerial != "emulator-5554" || resolved.Resolution != (target.Size{W: 1080, H: 1920}) {
		t.Fatalf("resolved=%#v error=%v", resolved, err)
	}

	deviceLists := 0
	driver.runner = controller.ADBRunnerFunc(func(_ context.Context, _ string, args ...string) ([]byte, error) {
		if strings.Join(args, " ") == "devices -l" {
			deviceLists++
			return []byte("List of devices attached\nemulator-5554 unauthorized transport_id:2\n"), nil
		}
		if len(args) == 2 && args[0] == "connect" {
			return nil, nil
		}
		t.Fatalf("unexpected ADB command %v", args)
		return nil, nil
	})
	_, err = driver.ResolveTarget(context.Background())
	var failure *Failure
	if !errors.As(err, &failure) || failure.Code != CodeTargetNotFound || !strings.Contains(err.Error(), "unauthorized") || deviceLists != 2 {
		t.Fatalf("offline resolution error=%v lists=%d", err, deviceLists)
	}
}

func TestAndroidDriverBoundsADBEffectCommands(t *testing.T) {
	profile, err := SealProfile(androidProfileDraft())
	if err != nil {
		t.Fatal(err)
	}
	release := make(chan struct{})
	defer func() {
		select {
		case <-release:
		default:
			close(release)
		}
	}()
	runner := controller.ADBRunnerFunc(func(ctx context.Context, _ string, args ...string) ([]byte, error) {
		switch strings.Join(args, " ") {
		case "devices -l":
			return []byte("List of devices attached\nemulator-5554 device product:sdk_gphone64_x86_64 model:sdk_gphone64_x86_64 device:emu64xa transport_id:2\n"), nil
		case "shell wm size":
			return []byte("Physical size: 1080x1920\n"), nil
		case "shell dumpsys input":
			return []byte("SurfaceOrientation: 0\n"), nil
		case "shell input tap 540 960":
			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			case <-release:
				return nil, errors.New("released blocked ADB command")
			}
		default:
			t.Fatalf("unexpected ADB command %v", args)
			return nil, nil
		}
	})
	driver := &androidDriver{profile: profile, runner: runner, commandTimeout: 25 * time.Millisecond}
	result := make(chan error, 1)
	go func() {
		result <- driver.Execute(context.Background(), OperationClick, ClickRequest{
			Point: Point{X: 0.5, Y: 0.5, Unit: "ratio"}, Button: "left",
		})
	}()
	select {
	case err := <-result:
		if !errors.Is(err, context.DeadlineExceeded) {
			t.Fatalf("bounded ADB effect error = %v", err)
		}
	case <-time.After(200 * time.Millisecond):
		close(release)
		<-result
		t.Fatal("ADB effect ignored the adapter command deadline")
	}
}

func TestAndroidProfilePinsDeviceAndPackageWithoutDesktopIdentity(t *testing.T) {
	profile, err := SealProfile(androidProfileDraft())
	if err != nil {
		t.Fatal(err)
	}
	payload, ok := AndroidProfile(profile)
	if profile.TargetKind() != TargetKindAndroidDevice || profile.AdapterKind() != AdapterKindAndroidADB || !ok || payload.AndroidPackage != "dev.yotta.fixture" {
		t.Fatalf("profile = %#v", profile.Machine())
	}
	draft := androidProfileDraft()
	draft.Payload = append(draft.Payload[:len(draft.Payload)-1], []byte(`,"windowTitle":"forged"}`)...)
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
	if err := registry.register(productionAdapters()[1].targetType, sealAndroidProfile, verifyPortableProfile, func(Profile) (driver, error) { return &fakeDriver{}, nil }, androidProfileIntentCodec()); err != nil {
		t.Fatal(err)
	}
	manifest, err := sealInstallationManifestForProfile("android", "Android", profile, registry)
	if err != nil {
		t.Fatal(err)
	}
	provider, err := newProvider(profile, manifest, registry)
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

func TestAndroidPlaybackTranslatesTouchKeyAndScrollEvents(t *testing.T) {
	profile, err := SealProfile(androidProfileDraft())
	if err != nil {
		t.Fatal(err)
	}
	commands := []string{}
	runner := controller.ADBRunnerFunc(func(_ context.Context, serial string, args ...string) ([]byte, error) {
		commands = append(commands, fmt.Sprintf("%s %v", serial, args))
		return nil, nil
	})
	resolved := target.Target{
		ID: "android:emulator-5554", Kind: target.KindAndroidADB, DisplayName: "sdk_gphone64_x86_64",
		Ref: target.TargetRef{ADBSerial: "emulator-5554"}, Resolution: target.Size{W: 1080, H: 1920},
	}
	driver := &androidDriver{
		profile: profile, runner: runner,
		playback: &androidPlaybackState{target: resolved, keys: map[uint32]struct{}{}},
	}
	point := func(x, y float64) *Point { return &Point{X: x, Y: y, Unit: "ratio"} }
	events := []PlaybackEvent{
		{Kind: PlaybackButtonDown, Point: point(0.25, 0.25), Button: "left"},
		{Kind: PlaybackMove, Point: point(0.75, 0.75)},
		{Kind: PlaybackButtonUp, Point: point(0.75, 0.75), Button: "left"},
		{Kind: PlaybackKeyDown, KeyCode: 0x41},
		{Kind: PlaybackKeyUp, KeyCode: 0x41},
		{Kind: PlaybackScroll, Point: point(0.5, 0.5), Notches: 1},
	}
	for _, event := range events {
		if err := driver.PlayEvent(context.Background(), event); err != nil {
			t.Fatalf("PlayEvent(%s) = %v", event.Kind, err)
		}
	}
	if len(commands) != 5 {
		t.Fatalf("ADB commands = %#v", commands)
	}
	if commands[0] != "emulator-5554 [shell dumpsys input]" ||
		commands[1] != "emulator-5554 [shell input swipe 270 480 810 1440 100]" ||
		commands[2] != "emulator-5554 [shell input keyevent 29]" ||
		commands[3] != "emulator-5554 [shell dumpsys input]" ||
		commands[4] != "emulator-5554 [shell input swipe 540 960 540 720 120]" {
		t.Fatalf("ADB commands = %#v", commands)
	}
	if err := driver.ReleaseInput(); err != nil || driver.playback != nil {
		t.Fatalf("ReleaseInput left playback state: %#v, %v", driver.playback, err)
	}
}

func TestAndroidPlaybackFailsClosedForUnrepresentableEvents(t *testing.T) {
	profile, err := SealProfile(androidProfileDraft())
	if err != nil {
		t.Fatal(err)
	}
	driver := &androidDriver{
		profile: profile,
		playback: &androidPlaybackState{
			target: target.Target{ID: "android:test", Kind: target.KindAndroidADB, DisplayName: "test", Ref: target.TargetRef{ADBSerial: "emulator-5554"}, Resolution: target.Size{W: 1, H: 1}},
			keys:   map[uint32]struct{}{},
		},
	}
	if err := driver.PlayEvent(context.Background(), PlaybackEvent{Kind: PlaybackMoveRelative, DeltaX: 1, SourceCounts360: 100}); err == nil {
		t.Fatal("relative desktop playback was accepted for Android")
	}
	if _, ok := androidPlaybackKeyCode(0x10); ok {
		t.Fatal("modifier key was silently mapped without chord semantics")
	}
}
