package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/coder/websocket"
	"github.com/coder/websocket/wsjson"
	"github.com/yottaapp/yotta/internal/storage"
	"github.com/yottaapp/yotta/internal/storage/catalog"
	"github.com/yottaapp/yotta/internal/workflowinstallation"
)

func TestSeedRecoveryFixtureUsesCurrentCatalogAuthority(t *testing.T) {
	root := filepath.Join(t.TempDir(), "profile")
	if err := seedRecoveryFixture(context.Background(), root); err != nil {
		t.Fatal(err)
	}
	profile, err := storage.Open(context.Background(), storage.OpenOptions{Root: root})
	if err != nil {
		t.Fatal(err)
	}
	defer profile.Close()
	if raw, err := os.ReadFile(profile.Roots.ManifestFile()); err != nil ||
		!strings.Contains(string(raw), `"version": "2"`) {
		t.Fatalf("root manifest = %q, %v", raw, err)
	}
	foundation, err := catalog.Open(context.Background(), profile.Roots)
	if err != nil {
		t.Fatal(err)
	}
	defer foundation.Close()
	recoveries, err := foundation.Workflows().ListQuarantine(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if len(recoveries) != 1 || recoveries[0].OriginalName != "damaged-workflow.json" ||
		string(recoveries[0].Artifact) != `{"format":"yotta.workflow","version":"1",` {
		t.Fatalf("recoveries = %#v", recoveries)
	}
	installations, err := workflowinstallation.New(
		foundation.WorkflowInstallations(),
		workflowinstallation.Options{},
	)
	if err != nil {
		t.Fatal(err)
	}
	listed, err := installations.List(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if len(listed) != 1 || listed[0].ID != smokeInstallationID ||
		listed[0].Name != "Installed smoke workflow" {
		t.Fatalf("installations = %#v", listed)
	}
}

func TestWorkflowEditorUIFailures(t *testing.T) {
	t.Run("accepts the dark non-overlapping Nuxt UI flow", func(t *testing.T) {
		failures := workflowEditorUIFailures(
			pageState{GraphChromeDark: true, RunStarted: true, MinimapToggle: true},
			pageState{ConfirmDialog: true},
			pageState{SaveInlineFeedback: true},
		)
		if len(failures) != 0 {
			t.Fatalf("unexpected failures: %v", failures)
		}
	})

	t.Run("reports every guarded regression", func(t *testing.T) {
		failures := workflowEditorUIFailures(
			pageState{HandleOverlaps: 8},
			pageState{NativeConfirmCalls: 1},
			pageState{SaveToast: true},
		)
		want := []string{
			"Vue Flow controls or minimap use a light background",
			"workflow canvas omitted the minimap toggle",
			"8 workflow handles overlap their labels",
			"new workflow omitted the RunStarted root",
			"workflow navigation called window.confirm",
			"workflow navigation did not open the shared confirm dialog",
			"workflow save displayed a success toast",
			"workflow save omitted inline success feedback",
		}
		if !reflect.DeepEqual(failures, want) {
			t.Fatalf("failures = %v, want %v", failures, want)
		}
	})
}

func TestWorkflowEditorHash(t *testing.T) {
	t.Run("extracts the durable editor route", func(t *testing.T) {
		got, err := workflowEditorHash("http://wails.localhost/#/workflows/workflow-1/edit")
		if err != nil || got != "#/workflows/workflow-1/edit" {
			t.Fatalf("workflowEditorHash() = %q, %v", got, err)
		}
	})

	t.Run("rejects unrelated and incomplete routes", func(t *testing.T) {
		for _, href := range []string{
			"http://wails.localhost/#/assets",
			"http://wails.localhost/#/workflows//edit",
			"http://wails.localhost/#/workflows/workflow-1",
		} {
			if _, err := workflowEditorHash(href); err == nil {
				t.Fatalf("workflowEditorHash(%q) succeeded", href)
			}
		}
	})
}

func TestRunCompletesWorkflowEditorJourney(t *testing.T) {
	base := pageState{
		Href: "http://wails.localhost/#/workflows/test/edit", CanvasNodes: 3,
		RunStarted: true, GraphChromeDark: true, CurrentGraph: "main", MinimapToggle: true,
		DebugStart: true, NodeAddTrigger: true, WorkspaceTools: 5, GraphManager: true,
	}
	oneNode := withState(base, func(state *pageState) { state.CanvasNodes = 1 })
	twoNodes := withState(base, func(state *pageState) { state.CanvasNodes = 2 })
	postDelete := withState(base, func(state *pageState) { state.CanvasNodes = 1 })
	connected := withState(base, func(state *pageState) { state.CanvasNodes, state.CanvasEdges = 2, 1 })
	states := []pageState{
		{RecoveryPanel: true, InstallationRows: 1, LauncherButton: true},
		{RecoveryPanel: true, InstallationRows: 1, InstallationSettings: true, LauncherButton: true},
		{RecoveryPanel: true, InstallationRows: 1, LauncherButton: true},
		{RecoveryPanel: true, InstallationRows: 1, LauncherButton: true, ConfirmDialog: true},
		base,
		{RecoveryPanel: true, InstallationRows: 1, LauncherButton: true},
		{CreateInput: true, RecoveryPanel: true, InstallationRows: 1, LauncherButton: true},
		{},
		oneNode,
		withState(base, func(state *pageState) { state.CanvasNodes, state.MinimapOpen = 1, true }),
		withState(base, func(state *pageState) { state.CanvasNodes = 1 }),
		oneNode,
		twoNodes,
		oneNode,
		oneNode,
		twoNodes,
		oneNode,
		oneNode,
		twoNodes,
		twoNodes,
		base,
		base,
		base,
		withState(base, func(state *pageState) { state.SelectedNodes, state.SelectionToolbar = 2, true }),
		base,
		withState(base, func(state *pageState) { state.SelectedNodes, state.SelectionToolbar = 2, true }),
		withState(base, func(state *pageState) { state.SelectedNodes, state.SelectionToolbar = 2, true }),
		postDelete,
		postDelete,
		withState(postDelete, func(state *pageState) { state.ConnectionMenu = true }),
		withState(postDelete, func(state *pageState) { state.ConnectionMenu, state.ConnectionCandidates = true, 1 }),
		connected,
		connected,
		withState(connected, func(state *pageState) { state.ConfirmDialog = true }),
		withState(connected, func(state *pageState) { state.ConfirmDialog = true }),
		withState(connected, func(state *pageState) { state.SaveInlineFeedback = true }),
		withState(connected, func(state *pageState) { state.SaveInlineFeedback = true }),
		withState(connected, func(state *pageState) { state.Breakpoints = 1 }),
		withState(connected, func(state *pageState) {
			state.Debugger, state.DebugPaused, state.DebugCurrent, state.DebugNode = true, true, 1, "run-started"
		}),
		withState(connected, func(state *pageState) {
			state.Debugger, state.DebugPaused, state.DebugBusy, state.DebugCurrent, state.DebugNode = true, true, true, 1, "inserted-delay"
		}),
		withState(connected, func(state *pageState) {
			state.Debugger, state.DebugPaused, state.DebugCurrent, state.DebugNode = true, true, 1, "inserted-delay"
		}),
		withState(connected, func(state *pageState) {
			state.Debugger, state.DebugCompleted = true, true
		}),
		withState(connected, func(state *pageState) {
			state.Debugger, state.DebugPaused, state.DebugCurrent, state.DebugNode = true, true, 1, "run-started"
		}),
		withState(connected, func(state *pageState) {
			state.Debugger, state.DebugCompleted = true, true
		}),
		withState(connected, func(state *pageState) { state.WorkflowState = true }),
		withState(connected, func(state *pageState) { state.GraphNameInput = true }),
		withState(connected, func(state *pageState) {
			state.CurrentGraph, state.CanvasNodes, state.CanvasEdges, state.GraphBoundaries = "child", 0, 0, 1
		}),
		withState(connected, func(state *pageState) {
			state.CurrentGraph, state.CanvasNodes, state.CanvasEdges, state.GraphBoundaries = "child", 0, 0, 1
		}),
		withState(connected, func(state *pageState) {
			state.CurrentGraph, state.CanvasNodes, state.CanvasEdges, state.GraphBoundaries = "child", 1, 0, 1
		}),
		withState(connected, func(state *pageState) {
			state.CurrentGraph, state.CanvasNodes, state.CanvasEdges, state.GraphBoundaries, state.ConfirmDialog = "child", 1, 0, 1, true
		}),
		withState(connected, func(state *pageState) {
			state.CurrentGraph, state.CanvasNodes, state.CanvasEdges, state.GraphBoundaries, state.GraphInterface = "child", 1, 0, 3, true
		}),
		connected,
		withState(connected, func(state *pageState) { state.Reroutes = 1 }),
		withState(connected, func(state *pageState) { state.CallMenuOptions = 1 }),
		withState(connected, func(state *pageState) { state.GraphCalls = 1 }),
		withState(connected, func(state *pageState) { state.GraphCalls, state.Annotations = 1, 1 }),
		withState(connected, func(state *pageState) { state.GraphCalls, state.Annotations, state.SaveInlineFeedback = 1, 1, true }),
		withState(connected, func(state *pageState) { state.AIReview = true }),
		withState(connected, func(state *pageState) { state.AIReview = true }),
		withState(connected, func(state *pageState) { state.AIReview = true }),
		withState(connected, func(state *pageState) { state.NodeContextMenu = true }),
		withState(connected, func(state *pageState) { state.SnippetModal = true }),
		withState(connected, func(state *pageState) { state.SnippetDock, state.SnippetItems = true, 1 }),
		withState(connected, func(state *pageState) { state.SnippetDock, state.SnippetItems = true, 1 }),
		withState(connected, func(state *pageState) {
			state.SnippetDock, state.SnippetItems, state.CanvasNodes, state.SelectedNodes = true, 1, connected.CanvasNodes+1, 1
		}),
		withState(connected, func(state *pageState) { state.SnippetDock, state.SnippetItems = true, 1 }),
		withState(connected, func(state *pageState) {
			state.SnippetDock, state.SnippetItems, state.SaveInlineFeedback = true, 1, true
		}),
		withState(connected, func(state *pageState) {
			state.SnippetDock, state.SnippetItems, state.ConfirmDialog = true, 1, true
		}),
		withState(connected, func(state *pageState) { state.SnippetDock = true }),
		withState(connected, func(state *pageState) {
			state.AIReview, state.ResourceDock, state.ResourceCreate, state.ResourceKind = true, true, true, "macro"
		}),
		withState(connected, func(state *pageState) {
			state.ResourceDock, state.ResourceCreate, state.ResourceKind = true, true, "clip"
		}),
		withState(connected, func(state *pageState) {
			state.ResourceDock, state.ResourceCreate, state.ResourceKind = true, true, "template"
			state.ResourceScope, state.ResourceScopeActive = "workflow", 1
			state.ResourceScopeContrast, state.ResourceFiltersFill = true, true
		}),
		withState(connected, func(state *pageState) {
			state.ResourceDock, state.ResourceCreate, state.ResourceKind = true, true, "template"
			state.ResourceScope, state.ResourceScopeActive = "library", 1
			state.ResourceScopeContrast, state.ResourceFiltersFill = true, true
		}),
		connected,
		withState(connected, func(state *pageState) { state.AssetsView, state.AssetsRecording = true, true }),
		withState(connected, func(state *pageState) { state.AssetsView, state.AssetsRecording = true, true }),
		withState(connected, func(state *pageState) {
			state.CurrentGraph, state.GraphCalls, state.Annotations = "main", 1, 1
		}),
	}

	var serverURL string
	var lastState pageState
	probeReads := 0
	quickSelectionPending := false
	analyzeColorUsedQuickAdd := false
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, request *http.Request) {
		if request.URL.Path == "/json" {
			wsURL := "ws" + strings.TrimPrefix(serverURL, "http") + "/devtools/page/page-1"
			_ = json.NewEncoder(w).Encode([]map[string]any{{
				"id": "page-1", "type": "page", "webSocketDebuggerUrl": wsURL,
			}})
			return
		}
		connection, err := websocket.Accept(w, request, &websocket.AcceptOptions{InsecureSkipVerify: true})
		if err != nil {
			t.Errorf("accept websocket: %v", err)
			return
		}
		defer connection.CloseNow()
		for {
			var call struct {
				ID     int64          `json:"id"`
				Method string         `json:"method"`
				Params map[string]any `json:"params"`
			}
			if err := wsjson.Read(context.Background(), connection, &call); err != nil {
				return
			}
			result := map[string]any{}
			expression := stringValue(call.Params["expression"])
			if call.Method == "Runtime.evaluate" &&
				strings.Contains(expression, "node quick add did not become ready") &&
				strings.Contains(expression, "vision/analyze-color") {
				if err := wsjson.Write(context.Background(), connection, map[string]any{
					"id": call.ID,
					"error": map[string]any{
						"code": -32000, "message": "Promise was collected",
					},
				}); err != nil {
					return
				}
				continue
			}
			if call.Method == "Runtime.evaluate" &&
				strings.Contains(expression, "workflow-quick-add-item") &&
				strings.Contains(expression, "vision/analyze-color") {
				analyzeColorUsedQuickAdd = true
			}
			if call.Method == "Runtime.evaluate" && strings.Contains(expression, "const probe = document.createElement") {
				if quickSelectionPending {
					selected := lastState
					selected.SelectedNodes = 1
					raw, _ := json.Marshal(selected)
					quickSelectionPending = false
					result = map[string]any{"result": map[string]any{"value": string(raw)}}
					if err := wsjson.Write(context.Background(), connection, map[string]any{
						"id": call.ID, "result": result,
					}); err != nil {
						return
					}
					continue
				}
				if len(states) == 0 {
					t.Errorf("unexpected extra page state request")
					return
				}
				lastState = states[0]
				raw, _ := json.Marshal(lastState)
				states = states[1:]
				result = map[string]any{"result": map[string]any{"value": string(raw)}}
			} else if call.Method == "Runtime.evaluate" && strings.Contains(expression, "workflow-debug-step") && lastState.DebugBusy {
				t.Errorf("debug Step was clicked before the previous control request settled")
			} else if call.Method == "Runtime.evaluate" && strings.Contains(expression, "Analyze Color node ergonomics probe") {
				probeReads++
				zoom := 2.0
				if probeReads == 3 {
					zoom = 1.5
				} else if probeReads > 3 {
					zoom = 1
				}
				value := fmt.Sprintf(`{"centerX":100,"centerY":100,"blankX":300,"blankY":300,"width":320,"height":240,"zoom":%v}`, zoom)
				result = map[string]any{"result": map[string]any{"value": value}}
			} else if call.Method == "Runtime.evaluate" &&
				(strings.Contains(expression, "quick add search ready") ||
					strings.Contains(expression, "quick add item ready")) {
				result = map[string]any{"result": map[string]any{"value": "true"}}
			} else if call.Method == "Runtime.evaluate" && strings.Contains(expression, "JSON.stringify") {
				value := `{"start":{"x":10,"y":10},"end":{"x":20,"y":20}}`
				if strings.Contains(expression, "quick-added click template node header") {
					quickSelectionPending = true
					value = `{"x":15,"y":15}`
				} else if strings.Contains(expression, "batch delete needs two non-root workflow nodes") {
					value = `[{"x":15,"y":15},{"x":25,"y":25}]`
				} else if strings.Contains(expression, "multi-selection needs") {
					value = `{"start":{"x":10,"y":10},"end":{"x":20,"y":20}}`
				} else if strings.Contains(expression, "Delay.in connection candidate") {
					value = `{"x":15,"y":15}`
				}
				result = map[string]any{"result": map[string]any{"value": value}}
			} else if call.Method == "Page.captureScreenshot" {
				result = map[string]any{"data": "cG5n"}
			}
			if err := wsjson.Write(context.Background(), connection, map[string]any{
				"id": call.ID, "result": result,
			}); err != nil {
				return
			}
		}
	}))
	serverURL = server.URL
	defer server.Close()

	dir := t.TempDir()
	screenshot := filepath.Join(dir, "workflow.png")
	assetsScreenshot := filepath.Join(dir, "assets.png")
	nodeMenuScreenshot := filepath.Join(dir, "node-context-menu.png")
	quickAddScreenshot := filepath.Join(dir, "quick-add.png")
	runStateScreenshot := filepath.Join(dir, "run-state.png")
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := run(ctx, server.URL, screenshot, assetsScreenshot, "", "", "", ""); err != nil {
		t.Fatal(err)
	}
	if !analyzeColorUsedQuickAdd {
		t.Fatal("Analyze Color smoke insertion bypassed the explicit quick-add entry")
	}
	if len(states) != 0 {
		t.Fatalf("unconsumed page states: %d", len(states))
	}
	for _, path := range []string{
		screenshot,
		assetsScreenshot,
		nodeMenuScreenshot,
		quickAddScreenshot,
		runStateScreenshot,
	} {
		if raw, err := os.ReadFile(path); err != nil || string(raw) != "png" {
			t.Fatalf("screenshot %s = %q, %v", path, raw, err)
		}
	}
	currentScreenshot := filepath.Join(dir, "current.png")
	if err := captureCurrent(ctx, server.URL, "", currentScreenshot); err != nil {
		t.Fatal(err)
	}
	if raw, err := os.ReadFile(currentScreenshot); err != nil || string(raw) != "png" {
		t.Fatalf("current screenshot = %q, %v", raw, err)
	}
}

func withState(base pageState, update func(*pageState)) pageState {
	update(&base)
	return base
}

func stringValue(value any) string {
	text, _ := value.(string)
	return text
}
