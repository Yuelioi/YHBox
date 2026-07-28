package appbootstrap_test

import (
	"bytes"
	"context"
	"encoding/json"
	"image"
	"image/draw"
	"image/png"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/yottaapp/yotta/internal/appbootstrap"
	"github.com/yottaapp/yotta/internal/automation/browsercdp"
	automationinstalled "github.com/yottaapp/yotta/internal/automation/installed"
	"github.com/yottaapp/yotta/internal/blob"
	"github.com/yottaapp/yotta/internal/nodes"
	runrecord "github.com/yottaapp/yotta/internal/run"
	"github.com/yottaapp/yotta/internal/services/inputclip"
	"github.com/yottaapp/yotta/internal/services/workflow"
	"github.com/yottaapp/yotta/internal/workflow/authoring"
	"github.com/yottaapp/yotta/internal/workflow/schema"
)

const (
	browserSmokeSlot = "browser-smoke"
	androidSmokeSlot = "android-smoke"
)

func TestBrowserCDPWorkflowSmoke(t *testing.T) {
	if os.Getenv("YOTTA_BROWSER_CDP_SMOKE") != "1" {
		t.Skip("set YOTTA_BROWSER_CDP_SMOKE=1 with a controlled loopback Chrome or Edge endpoint")
	}
	endpoint := strings.TrimSpace(os.Getenv("YOTTA_BROWSER_CDP_ENDPOINT"))
	if endpoint == "" {
		endpoint = "http://127.0.0.1:9337"
	}
	ctx, cancel := context.WithTimeout(context.Background(), 45*time.Second)
	defer cancel()
	discovery := browsercdp.NewService(endpoint)
	targets, err := discovery.ListTargets(ctx, endpoint)
	if err != nil || len(targets) == 0 {
		t.Fatalf("discover controlled browser page: targets=%d error=%v", len(targets), err)
	}
	page := targets[0]
	client, err := browsercdp.DialWebSocketClient(ctx, page.WebSocketDebuggerURL)
	if err != nil {
		t.Fatal(err)
	}
	defer client.Close()
	if _, err := client.Call(ctx, "Runtime.evaluate", map[string]any{
		"expression": `document.body.innerHTML='<label>Yotta workflow smoke <input id="yotta-platform-smoke"></label>';document.querySelector('#yotta-platform-smoke').focus()`,
	}); err != nil {
		t.Fatal(err)
	}

	profileDraft := automationinstalled.NewBrowserProfileDraft(automationinstalled.BrowserProfilePayload{
		BrowserEndpoint: endpoint, BrowserTargetID: page.ID, BrowserWebSocketURL: page.WebSocketDebuggerURL,
		BrowserTitle: page.Title, BrowserURL: page.URL, ResolveTimeoutMilliseconds: 5000,
	})
	installations := configuredAutomationInstallations(t, browserSmokeSlot, "Browser smoke", profileDraft)
	runtime, service := buildPlatformSmokeRuntime(t, installations)
	source, err := service.CreateSource("Browser CDP platform smoke")
	if err != nil {
		t.Fatal(err)
	}
	patched, err := service.ApplyPatch(source.WorkflowID, source.Revision, []authoring.Command{
		addNode("type", nodes.TypeTextNodeID, 320),
		setSlot("type", browserSmokeSlot),
		bindValue("type", "text", "yotta-browser-workflow"),
		addNode("capture", nodes.CaptureWindowNodeID, 640),
		setSlot("capture", browserSmokeSlot),
		connect("run-started", "started", "$type", "in"),
		connect("$type", "completed", "$capture", "in"),
	})
	if err != nil {
		t.Fatal(err)
	}
	run := startAndWaitForPlatformRun(t, service, patched.Source.WorkflowID)
	assertPlatformActions(t, run, "automation.type-text", "automation.capture-window")
	value, err := client.Call(ctx, "Runtime.evaluate", map[string]any{
		"expression": "document.querySelector('#yotta-platform-smoke').value", "returnByValue": true,
	})
	if err != nil || !runtimeValueEquals(value, "yotta-browser-workflow") {
		t.Fatalf("browser text side effect = %#v, %v", value, err)
	}
	if _, err := runtime.AuthoringTargets().ResolveTarget(ctx, browserSmokeSlot); err != nil {
		t.Fatalf("browser authoring target = %v", err)
	}
}

func TestAndroidADBWorkflowSmoke(t *testing.T) {
	if os.Getenv("YOTTA_ADB_SMOKE") != "1" {
		t.Skip("set YOTTA_ADB_SMOKE=1 with an authorized emulator or device")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
	defer cancel()
	devices, err := automationinstalled.DiscoverAndroidDevices(ctx)
	if err != nil {
		t.Fatal(err)
	}
	wantedSerial := strings.TrimSpace(os.Getenv("YOTTA_ADB_SERIAL"))
	var selected *automationinstalled.AndroidDeviceDescriptor
	for index := range devices {
		device := &devices[index]
		if device.State == "device" && device.Product != "" && device.Model != "" && device.Device != "" &&
			(wantedSerial == "" || device.Serial == wantedSerial) {
			selected = device
			break
		}
	}
	if selected == nil {
		t.Fatalf("no ready ADB device with complete identity (wanted %q): %#v", wantedSerial, devices)
	}
	androidPackage := strings.TrimSpace(os.Getenv("YOTTA_ADB_SMOKE_PACKAGE"))
	if androidPackage == "" {
		androidPackage = "com.android.settings"
	}
	profileDraft := automationinstalled.NewAndroidProfileDraft(automationinstalled.AndroidProfilePayload{
		ADBSerial: selected.Serial, ADBProduct: selected.Product, ADBModel: selected.Model, ADBDevice: selected.Device,
		AndroidPackage: androidPackage, ResolveTimeoutMilliseconds: 5000,
	})
	installations := configuredAutomationInstallations(t, androidSmokeSlot, "Android smoke", profileDraft)
	runtime, service := buildPlatformSmokeRuntime(t, installations)
	if err := runtime.AuthoringTargets().Activate(ctx, androidSmokeSlot); err != nil {
		t.Fatalf("activate Android fixture: %v", err)
	}
	time.Sleep(750 * time.Millisecond)
	appDiscovered := false
	for attempt := 0; attempt < 12 && !appDiscovered; attempt++ {
		apps, err := automationinstalled.DiscoverAndroidApps(ctx, selected.Serial)
		if err != nil {
			t.Fatalf("discover applications after activation: %v", err)
		}
		for _, app := range apps {
			if app.Package == androidPackage && app.Foreground {
				appDiscovered = true
				break
			}
		}
		if !appDiscovered {
			time.Sleep(250 * time.Millisecond)
		}
	}
	if !appDiscovered {
		t.Fatalf("activated Android package %q was not discoverable as the foreground application", androidPackage)
	}
	resolved, err := runtime.AuthoringTargets().ResolveTarget(ctx, androidSmokeSlot)
	if err != nil {
		t.Fatal(err)
	}
	frame, err := runtime.AuthoringTargets().CapturePNG(ctx, androidSmokeSlot)
	if err != nil {
		t.Fatal(err)
	}
	templatePNG := centerTemplatePNG(t, frame)
	templateRef, err := runtime.BlobStore.Put(ctx, "image/png", bytes.NewReader(templatePNG))
	if err != nil {
		t.Fatal(err)
	}
	clipRef := androidTouchClip(t, runtime.BlobStore, resolved.Resolution.W, resolved.Resolution.H)

	source, err := service.CreateSource("Android ADB platform smoke")
	if err != nil {
		t.Fatal(err)
	}
	commands := []authoring.Command{
		addNode("activate", nodes.ActivateWindowNodeID, 240), setSlot("activate", androidSmokeSlot),
		addNode("capture", nodes.CaptureWindowNodeID, 480), setSlot("capture", androidSmokeSlot),
		addNode("template", nodes.ClickTemplateNodeID, 720), setSlot("template", androidSmokeSlot), bindBlob("template", "template", templateRef),
		addNode("drag", nodes.DragPointerNodeID, 960), setSlot("drag", androidSmokeSlot),
		bindValue("drag", "from", map[string]any{"x": 0.5, "y": 0.65, "unit": "ratio"}),
		bindValue("drag", "to", map[string]any{"x": 0.5, "y": 0.55, "unit": "ratio"}),
		addNode("clip", nodes.PlayInputClipNodeID, 1200), setSlot("clip", androidSmokeSlot), bindBlob("clip", "clip", clipRef),
		addNode("stop", nodes.StopTargetAppNodeID, 1440), setSlot("stop", androidSmokeSlot),
		connect("run-started", "started", "$activate", "in"),
		connect("$activate", "completed", "$capture", "in"),
		connect("$capture", "completed", "$template", "in"),
		connect("$template", "completed", "$drag", "in"),
		connect("$drag", "completed", "$clip", "in"),
		connect("$clip", "completed", "$stop", "in"),
	}
	patched, err := service.ApplyPatch(source.WorkflowID, source.Revision, commands)
	if err != nil {
		t.Fatal(err)
	}
	run := startAndWaitForPlatformRun(t, service, patched.Source.WorkflowID)
	assertPlatformActions(t, run,
		"automation.activate-window", "automation.capture-window", "automation.click-template",
		"automation.drag", "automation.play-input-clip", "automation.stop-target-app",
	)
}

func configuredAutomationInstallations(t *testing.T, slot, label string, draft automationinstalled.ProfileDraft) automationinstalled.Installations {
	t.Helper()
	installations, err := automationinstalled.Install([]automationinstalled.InstallationDraft{{Slot: slot, Label: label, Profile: draft}})
	if err != nil {
		t.Fatal(err)
	}
	return installations
}

func buildPlatformSmokeRuntime(t *testing.T, installations automationinstalled.Installations) (*appbootstrap.Runtime, *workflow.Service) {
	t.Helper()
	stores := newTestWorkflowStorage(t)
	runtime, err := appbootstrap.Build(appbootstrap.Config{
		DataRoot: stores.roots.Data, ProgramCacheRoot: filepath.Join(stores.roots.Cache, "programs"),
		WorkflowRepository: stores.foundation.Workflows(),
		RunRepository:      stores.foundation.Runs(),
		BlobStore:          stores.blobs, Limits: testLimits(),
		AIInstallations: emptyAIInstallations(t), HTTPInstallations: emptyHTTPInstallations(t),
		ApplicationInstallations: emptyApplicationInstallations(t), AutomationInstallations: installations,
		ScriptRuntime: bootstrapScriptRuntime(t), LogEmitter: discardWorkflowLog{},
		GrantTTL: 5 * time.Minute, OwnerCloseTimeout: 5 * time.Second, Now: time.Now,
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := runtime.Application.Start(context.Background()); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		if err := runtime.Close(ctx); err != nil {
			t.Errorf("close platform smoke runtime: %v", err)
		}
	})
	service, err := workflow.NewService(runtime.Application)
	if err != nil {
		t.Fatal(err)
	}
	return runtime, service
}

func addNode(handle, nodeTypeID string, x float64) authoring.Command {
	return authoring.Command{Kind: authoring.CommandAddNode, AddNode: &authoring.AddNodeCommand{
		GraphID: "main", NodeTypeID: nodeTypeID, Handle: handle, Position: schema.Position{X: x, Y: 160},
	}}
}

func setSlot(handle, slot string) authoring.Command {
	return authoring.Command{Kind: authoring.CommandSetConfig, SetConfig: &authoring.SetConfigCommand{
		GraphID: "main", NodeID: "$" + handle, FieldID: "slot", Value: slot,
	}}
}

func bindValue(handle, port string, value any) authoring.Command {
	return authoring.Command{Kind: authoring.CommandBindValue, BindValue: &authoring.BindValueCommand{
		GraphID: "main", NodeID: "$" + handle, PortID: port, Value: value,
	}}
}

func bindBlob(handle, port string, ref blob.BlobRef) authoring.Command {
	return authoring.Command{Kind: authoring.CommandBindBlob, BindBlob: &authoring.BindBlobCommand{
		GraphID: "main", NodeID: "$" + handle, PortID: port, Blob: ref,
	}}
}

func connect(fromNode, fromPort, toNode, toPort string) authoring.Command {
	return authoring.Command{Kind: authoring.CommandConnect, Connect: &authoring.EdgeCommand{GraphID: "main", Edge: authoring.PatchEdgeFromSource(schema.Edge{
		Channel: schema.EdgeExec, From: schema.Endpoint{NodeID: fromNode, PortID: fromPort}, To: schema.Endpoint{NodeID: toNode, PortID: toPort},
	})}}
}

func startAndWaitForPlatformRun(t *testing.T, service *workflow.Service, workflowID string) workflow.RunView {
	t.Helper()
	started, err := service.StartRun(workflowID)
	if err != nil || started.Run == nil {
		t.Fatalf("start platform workflow: %#v, %v", started, err)
	}
	runTimeout := 60 * time.Second
	if configured := strings.TrimSpace(os.Getenv("YOTTA_PLATFORM_SMOKE_RUN_TIMEOUT")); configured != "" {
		parsed, err := time.ParseDuration(configured)
		if err != nil || parsed <= 0 {
			t.Fatalf("invalid YOTTA_PLATFORM_SMOKE_RUN_TIMEOUT %q", configured)
		}
		runTimeout = parsed
	}
	deadline := time.Now().Add(runTimeout)
	last := workflow.RunView{}
	for time.Now().Before(deadline) {
		current, err := service.GetRunTimeline(started.Run.RunID)
		if err != nil {
			t.Fatal(err)
		}
		last = current
		switch current.Status {
		case string(runrecord.StatusSucceeded):
			return current
		case string(runrecord.StatusFailed), string(runrecord.StatusCancelled), string(runrecord.StatusInterrupted):
			encoded, _ := json.Marshal(current)
			t.Fatalf("platform workflow ended as %s: %s", current.Status, encoded)
		}
		time.Sleep(50 * time.Millisecond)
	}
	encoded, _ := json.Marshal(last)
	t.Fatalf("platform workflow did not finish within %s: %s", runTimeout, encoded)
	return workflow.RunView{}
}

func assertPlatformActions(t *testing.T, run workflow.RunView, expected ...string) {
	t.Helper()
	found := map[string]bool{}
	for _, entry := range run.Timeline {
		if entry.Action != "" && entry.ActionOutcome == string(runrecord.ActionSucceeded) {
			found[entry.Action] = true
		}
	}
	for _, action := range expected {
		if !found[action] {
			t.Fatalf("successful action %q missing from timeline: %#v", action, run.Timeline)
		}
	}
}

func runtimeValueEquals(response map[string]any, expected string) bool {
	result, ok := response["result"].(map[string]any)
	if !ok {
		return false
	}
	value, ok := result["value"].(string)
	return ok && value == expected
}

func centerTemplatePNG(t *testing.T, raw []byte) []byte {
	t.Helper()
	decoded, err := png.Decode(bytes.NewReader(raw))
	if err != nil {
		t.Fatal(err)
	}
	bounds := decoded.Bounds()
	width, height := min(160, bounds.Dx()), min(160, bounds.Dy())
	if width < 32 || height < 32 {
		t.Fatalf("Android capture is too small for a template: %v", bounds)
	}
	left, top := bounds.Min.X+(bounds.Dx()-width)/2, bounds.Min.Y+(bounds.Dy()-height)/2
	template := image.NewRGBA(image.Rect(0, 0, width, height))
	draw.Draw(template, template.Bounds(), decoded, image.Point{X: left, Y: top}, draw.Src)
	var encoded bytes.Buffer
	if err := png.Encode(&encoded, template); err != nil {
		t.Fatal(err)
	}
	return encoded.Bytes()
}

func androidTouchClip(t *testing.T, store *blob.Store, width, height int) blob.BlobRef {
	t.Helper()
	if width <= 0 || height <= 0 {
		t.Fatalf("invalid Android clip resolution %dx%d", width, height)
	}
	x, y := int32(width*9/10), int32(height*9/10)
	clip := &inputclip.InputClip{
		Meta: inputclip.ClipMeta{RecordingMode: inputclip.RecordingModeSimple, MouseMode: "absolute", BaseResolution: [2]int{width, height}},
		Events: []inputclip.Event{
			{TUs: 0, Seq: 0, Type: inputclip.EventTypeMouseBtnDown, A: 0, B: x, C: y},
			{TUs: 50_000, Seq: 0, Type: inputclip.EventTypeMouseBtnUp, A: 0, B: x, C: y},
		},
	}
	clip.UpdateDuration()
	var encoded bytes.Buffer
	if err := inputclip.Encode(&encoded, clip); err != nil {
		t.Fatal(err)
	}
	ref, err := store.Put(context.Background(), inputclip.MediaType, bytes.NewReader(encoded.Bytes()))
	if err != nil {
		t.Fatal(err)
	}
	return ref
}
