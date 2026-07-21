package desktopapp

import (
	"bytes"
	"context"
	"errors"
	"path/filepath"
	"testing"

	"github.com/rs/zerolog"
	"github.com/wailsapp/wails/v3/pkg/application"
	appcore "github.com/yottaapp/yotta/internal/application"
	automationinstalled "github.com/yottaapp/yotta/internal/automation/installed"
	"github.com/yottaapp/yotta/internal/automation/target"
	"github.com/yottaapp/yotta/internal/hotkey"
	"github.com/yottaapp/yotta/internal/noderuntime"
	"github.com/yottaapp/yotta/internal/services"
	"github.com/yottaapp/yotta/internal/services/tools"
)

type fixedRecordingTargetResolver struct{ counts360 int }

func (resolver fixedRecordingTargetResolver) AcquireRecordingTarget(context.Context, string) (target.WindowHandle, int, func(), error) {
	return target.WindowHandle{HWND: 1, ClientW: 1280, ClientH: 720}, resolver.counts360, func() {}, nil
}

func TestRootCompositionAdaptersExposeSafeDefaultsAndLifecycle(t *testing.T) {
	missing := &recordingHkAdapter{}
	if missing.GetStartHotkeyVK() != 0x79 || missing.GetStopHotkeyVK() != 0x7B || missing.GetPauseHotkeyVK() != 0x7A {
		t.Fatal("recording hotkey adapter lost safe defaults")
	}
	emptyRegistry := hotkey.NewHotkeyRegistry(nil)
	emptyAdapter := &recordingHkAdapter{reg: emptyRegistry}
	if emptyAdapter.GetStopHotkeyVK() != 0x7B {
		t.Fatal("missing recording hotkey did not use fallback")
	}
	if err := emptyRegistry.RegisterLLHook("recording.stop", hotkey.HotkeySourceRecording, "stop", "", ""); err != nil {
		t.Fatal(err)
	}
	if emptyAdapter.GetStopHotkeyVK() != 0x7B {
		t.Fatal("empty recording hotkey did not use fallback")
	}

	registry := hotkey.NewHotkeyRegistry(nil)
	if err := registry.RegisterLLHook("recording.start", hotkey.HotkeySourceRecording, "start", "F8", ""); err != nil {
		t.Fatal(err)
	}
	if err := registry.RegisterLLHook("recording.stop", hotkey.HotkeySourceRecording, "stop", "F10", ""); err != nil {
		t.Fatal(err)
	}
	if err := registry.RegisterLLHook("recording.pause", hotkey.HotkeySourceRecording, "pause", "F9", ""); err != nil {
		t.Fatal(err)
	}
	adapter := &recordingHkAdapter{reg: registry}
	if adapter.GetStartHotkeyVK() != 0x77 || adapter.GetStopHotkeyVK() != 0x79 || adapter.GetPauseHotkeyVK() != 0x78 {
		t.Fatalf("recording hotkeys = %#x / %#x / %#x", adapter.GetStartHotkeyVK(), adapter.GetStopHotkeyVK(), adapter.GetPauseHotkeyVK())
	}

	app := services.NewApp(filepath.Join(t.TempDir(), "settings.json"), nil, zerolog.Nop())
	adapter.app = app
	if adapter.GetMouseMode() != "relative" {
		t.Fatalf("default mouse mode = %q", adapter.GetMouseMode())
	}
	if _, _, err := app.MutateSettings(func(settings *services.Settings) error {
		settings.UI.RecordingMouseMode = "absolute"
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	if adapter.GetMouseMode() != "absolute" {
		t.Fatalf("configured mouse mode = %q", adapter.GetMouseMode())
	}
	if service := newRecordingService(app, nil, nil, registry, automationinstalled.AuthoringTargets{}); service == nil {
		t.Fatal("recording composition returned nil")
	}

	scheduleRegistry := hotkey.NewHotkeyRegistry(nil)
	registrar := &scheduleHotkeyRegistrar{reg: scheduleRegistry}
	if err := registrar.Register("schedule.test", string(hotkey.HotkeySourceSchedule), "test", nil, "", "", nil); err != nil {
		t.Fatal(err)
	}
	if err := registrar.Unregister("schedule.test"); err != nil {
		t.Fatal(err)
	}
	if err := registry.Shutdown(context.Background()); err != nil {
		t.Fatal(err)
	}
	if err := scheduleRegistry.Shutdown(context.Background()); err != nil {
		t.Fatal(err)
	}

	presenter := &wailsToolsPresenter{}
	if presenter.Ready() {
		t.Fatal("detached presenter reported ready")
	}
	if _, err := presenter.OpenWindow(tools.WindowRequest{Kind: tools.WindowLauncher}); err == nil {
		t.Fatal("detached presenter opened a window")
	}
	presenter.Emit("ignored", nil)
	presenter.Attach(&application.App{})
	if !presenter.Ready() {
		t.Fatal("attached presenter did not report ready")
	}
	presenter.Detach()
	if err := emptyRegistry.Shutdown(context.Background()); err != nil {
		t.Fatal(err)
	}
}

func TestRecordingCalibrationTargetsFollowActiveProfileWhenTargetHasNoOverride(t *testing.T) {
	activeCounts360 := 4134
	resolver := &recordingCalibrationTargets{
		targets:         fixedRecordingTargetResolver{counts360: 0},
		activeCounts360: func() int { return activeCounts360 },
	}
	_, counts360, release, err := resolver.AcquireRecordingTarget(context.Background(), "window-target")
	if err != nil {
		t.Fatal(err)
	}
	release()
	if counts360 != 4134 {
		t.Fatalf("recording counts360 = %d, want active calibration 4134", counts360)
	}
	activeCounts360 = 5000
	_, counts360, release, err = resolver.AcquireRecordingTarget(context.Background(), "window-target")
	if err != nil {
		t.Fatal(err)
	}
	release()
	if counts360 != 5000 {
		t.Fatalf("recording counts360 after recalibration = %d, want 5000", counts360)
	}
}

func TestRecordingCalibrationTargetsKeepExplicitTargetOverride(t *testing.T) {
	resolver := &recordingCalibrationTargets{
		targets:         fixedRecordingTargetResolver{counts360: 8000},
		activeCounts360: func() int { return 4134 },
	}
	_, counts360, release, err := resolver.AcquireRecordingTarget(context.Background(), "window-target")
	if err != nil {
		t.Fatal(err)
	}
	release()
	if counts360 != 8000 {
		t.Fatalf("recording counts360 = %d, want target override 8000", counts360)
	}
}

func TestClipCompositionUsesTheGlobalAssetStore(t *testing.T) {
	store := newTestAssetStore(t, t.TempDir())
	if service := newClipService(store); service == nil {
		t.Fatal("clip composition returned nil")
	}
	if err := (&workflowRunStarter{}).StartWorkflow(context.Background(), "missing"); err == nil {
		t.Fatal("workflow starter accepted a missing Application")
	}
	if err := (&workflowRunStarter{application: &appcore.Application{}}).StartWorkflow(context.Background(), "missing"); err == nil {
		t.Fatal("workflow starter hid an unavailable Application")
	}
}

func TestWorkflowLogEmitterPreservesLevelAndAttribution(t *testing.T) {
	var output bytes.Buffer
	emitter := newWorkflowLogEmitter(zerolog.New(&output).Level(zerolog.DebugLevel))
	for _, level := range []string{"debug", "info", "warn", "error"} {
		if err := emitter.EmitWorkflowLog(context.Background(), noderuntime.LogEntry{
			Level: level, Message: "message-" + level, GraphID: "main", NodeID: "log", InvocationID: "invoke-1", Attempt: 2,
		}); err != nil {
			t.Fatal(err)
		}
	}
	for _, fact := range []string{"message-info", "message-warn", "message-error", `"graphId":"main"`, `"nodeId":"log"`, `"attempt":2`} {
		if !bytes.Contains(output.Bytes(), []byte(fact)) {
			t.Fatalf("workflow log output omitted %q: %s", fact, output.String())
		}
	}
	cancelled, cancel := context.WithCancel(context.Background())
	cancel()
	if err := emitter.EmitWorkflowLog(cancelled, noderuntime.LogEntry{}); !errors.Is(err, context.Canceled) {
		t.Fatalf("cancelled workflow log = %v", err)
	}
}

func TestHotkeyOrDefaultNormalizesConfiguredValues(t *testing.T) {
	if got := hotkeyOrDefault("  F10  ", "F12"); got != "F10" {
		t.Fatalf("configured hotkey = %q", got)
	}
	if got := hotkeyOrDefault("  ", "F12"); got != "F12" {
		t.Fatalf("fallback hotkey = %q", got)
	}
}

func TestRegistryHotkeyUsesExactBindingAndFallback(t *testing.T) {
	registry := hotkey.NewHotkeyRegistry(nil)
	t.Cleanup(func() { _ = registry.Shutdown(context.Background()) })
	if mods, vk := registryHotkey(registry, "capture", 0x78); mods != 0 || vk != 0x78 {
		t.Fatalf("missing hotkey = %#x/%#x", mods, vk)
	}
	if err := registry.RegisterLLHook("capture", hotkey.HotkeySourceSystem, "capture", "Ctrl+F10", ""); err != nil {
		t.Fatal(err)
	}
	mods, vk := registryHotkey(registry, "capture", 0x78)
	if mods == 0 || vk != 0x79 {
		t.Fatalf("configured hotkey = %#x/%#x", mods, vk)
	}
}
