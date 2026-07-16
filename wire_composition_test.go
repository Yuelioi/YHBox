package main

import (
	"bytes"
	"context"
	"errors"
	"path/filepath"
	"testing"

	"github.com/rs/zerolog"
	"github.com/wailsapp/wails/v3/pkg/application"
	app31 "github.com/yottaapp/yotta/internal/application"
	"github.com/yottaapp/yotta/internal/hotkey"
	"github.com/yottaapp/yotta/internal/node"
	"github.com/yottaapp/yotta/internal/nodes/control"
	"github.com/yottaapp/yotta/internal/nodes31runtime"
	"github.com/yottaapp/yotta/internal/services"
	"github.com/yottaapp/yotta/internal/services/container"
	"github.com/yottaapp/yotta/internal/services/tools"
)

func TestSubgraphCompositionUsesOneDependencyScannerForClosureAndReferrers(t *testing.T) {
	subgraphs, err := container.NewSubgraphStore(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	target := container.Subgraph{ID: "sg-target", Label: "target"}
	if err := subgraphs.Create(&target); err != nil {
		t.Fatal(err)
	}
	caller := container.Subgraph{ID: "sg-caller", Label: "caller", Graph: container.Graph{Nodes: []container.GraphNode{{
		ID: "call-target", Kind: "Subgraph", Config: map[string]any{"SubgraphID": target.ID},
	}}}}
	if err := subgraphs.Create(&caller); err != nil {
		t.Fatal(err)
	}

	registry := node.NewRegistry()
	registry.Register(&control.Start{})
	containers, err := container.NewStoreWithRegistry(t.TempDir(), registry)
	if err != nil {
		t.Fatal(err)
	}
	if err := containers.Save(&container.Container{SchemaVersion: 1, ID: "root", Name: "root", Graph: container.Graph{
		Nodes: []container.GraphNode{{ID: "start", Kind: "Start"}},
	}}); err != nil {
		t.Fatal(err)
	}

	root := &container.Container{Graph: container.Graph{Nodes: []container.GraphNode{{
		ID: "call-caller", Kind: "Subgraph", Config: map[string]any{"literal": map[string]any{"SubgraphID": caller.ID}},
	}}}}
	closure := subgraphClosureFor(root, subgraphs)
	if len(closure) != 2 || closure[0].ID != caller.ID || closure[1].ID != target.ID {
		t.Fatalf("subgraph closure = %#v", closure)
	}
	infos := depNodeInfos(root.Graph.Nodes)
	if len(infos) != 1 || infos[0].Kind != "Subgraph" {
		t.Fatalf("dependency node projection = %#v", infos)
	}
	if dependencies := nodeSubgraphDeps(&root.Graph.Nodes[0]); len(dependencies) != 1 || dependencies[0] != caller.ID {
		t.Fatalf("direct subgraph dependencies = %#v", dependencies)
	}
	if dependencies := nodeSubgraphDeps(&container.GraphNode{Kind: "unknown"}); len(dependencies) != 0 {
		t.Fatalf("unknown node dependencies = %#v", dependencies)
	}

	refs := scanSubgraphReferrers(containers, subgraphs)(target.ID)
	if len(refs) != 1 || refs[0].SubgraphID != caller.ID || refs[0].NodeID != "call-target" {
		t.Fatalf("subgraph referrers = %#v", refs)
	}
	referenced := collectReferencedSubgraphIDs(containers, subgraphs)
	if !referenced[target.ID] || referenced[caller.ID] {
		t.Fatalf("referenced subgraphs = %#v", referenced)
	}
}

func TestRootCompositionAdaptersExposeSafeDefaultsAndLifecycle(t *testing.T) {
	missing := &recordingHkAdapter{}
	if missing.GetStopHotkeyVK() != 0x7B || missing.GetPauseHotkeyVK() != 0x7A {
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
	if err := registry.RegisterLLHook("recording.stop", hotkey.HotkeySourceRecording, "stop", "F10", ""); err != nil {
		t.Fatal(err)
	}
	if err := registry.RegisterLLHook("recording.pause", hotkey.HotkeySourceRecording, "pause", "F9", ""); err != nil {
		t.Fatal(err)
	}
	adapter := &recordingHkAdapter{reg: registry}
	if adapter.GetStopHotkeyVK() != 0x79 || adapter.GetPauseHotkeyVK() != 0x78 {
		t.Fatalf("recording hotkeys = %#x / %#x", adapter.GetStopHotkeyVK(), adapter.GetPauseHotkeyVK())
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
	if service := newRecordingService(app, nil, registry); service == nil {
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

func TestClipCompositionUsesTheGlobalAssetStore(t *testing.T) {
	store := newTestAssetStore(t, t.TempDir())
	if service := newClipService(store); service == nil {
		t.Fatal("clip composition returned nil")
	}
	if err := (&workflowRunStarter{}).StartWorkflow(context.Background(), "missing"); err == nil {
		t.Fatal("workflow starter accepted a missing Application")
	}
	if err := (&workflowRunStarter{application: &app31.Application{}}).StartWorkflow(context.Background(), "missing"); err == nil {
		t.Fatal("workflow starter hid an unavailable Application")
	}
}

func TestWorkflowLogEmitterPreservesLevelAndAttribution(t *testing.T) {
	var output bytes.Buffer
	emitter := newWorkflowLogEmitter(zerolog.New(&output).Level(zerolog.DebugLevel))
	for _, level := range []string{"debug", "info", "warn", "error"} {
		if err := emitter.EmitWorkflowLog(context.Background(), nodes31runtime.LogEntry{
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
	if err := emitter.EmitWorkflowLog(cancelled, nodes31runtime.LogEntry{}); !errors.Is(err, context.Canceled) {
		t.Fatalf("cancelled workflow log = %v", err)
	}
}
