package main

import (
	"context"
	"encoding/json"
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
)

func TestWorkflowEditorUIFailures(t *testing.T) {
	t.Run("accepts the dark non-overlapping Nuxt UI flow", func(t *testing.T) {
		failures := workflowEditorUIFailures(
			pageState{GraphChromeDark: true, RunStarted: true},
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

func TestRunCompletesWorkflowEditorJourney(t *testing.T) {
	base := pageState{
		Href: "http://wails.localhost/#/workflows/test/edit", Catalog: 100, CanvasNodes: 3,
		RunStarted: true, GraphChromeDark: true, CurrentGraph: "main",
	}
	connected := withState(base, func(state *pageState) { state.CanvasNodes, state.CanvasEdges = 4, 1 })
	states := []pageState{
		{CreateInput: true},
		{Catalog: 0},
		{Catalog: 100, CanvasNodes: 1, RunStarted: true},
		withState(base, func(state *pageState) { state.CanvasNodes, state.Breakpoints = 1, 1 }),
		withState(base, func(state *pageState) {
			state.CanvasNodes, state.Debugger, state.DebugPaused, state.DebugCurrent, state.DebugNode = 1, true, true, 1, "run-started"
		}),
		withState(base, func(state *pageState) {
			state.CanvasNodes, state.Debugger, state.DebugCompleted = 1, true, true
		}),
		withState(base, func(state *pageState) {
			state.CanvasNodes, state.Debugger, state.DebugPaused, state.DebugCurrent, state.DebugNode = 1, true, true, 1, "run-started"
		}),
		withState(base, func(state *pageState) {
			state.CanvasNodes, state.Debugger, state.DebugCompleted = 1, true, true
		}),
		{Catalog: 100, CanvasNodes: 1, RunStarted: true},
		{Catalog: 100, CanvasNodes: 2, RunStarted: true},
		{Catalog: 100, CanvasNodes: 2, RunStarted: true},
		base,
		base,
		base,
		withState(base, func(state *pageState) { state.SelectedNodes, state.SelectionToolbar = 2, true }),
		base,
		base,
		withState(base, func(state *pageState) { state.ConnectionMenu = true }),
		withState(base, func(state *pageState) { state.ConnectionMenu, state.ConnectionCandidates = true, 1 }),
		connected,
		connected,
		withState(connected, func(state *pageState) { state.Catalog = 1 }),
		connected,
		withState(connected, func(state *pageState) { state.ConfirmDialog = true }),
		withState(connected, func(state *pageState) { state.ConfirmDialog = true }),
		withState(connected, func(state *pageState) { state.SaveInlineFeedback = true }),
		withState(connected, func(state *pageState) { state.SaveInlineFeedback = true }),
		withState(connected, func(state *pageState) { state.WorkflowState = true }),
		withState(connected, func(state *pageState) { state.GraphNameInput = true }),
		withState(connected, func(state *pageState) { state.CurrentGraph, state.CanvasNodes, state.CanvasEdges = "child", 0, 0 }),
		withState(connected, func(state *pageState) { state.CurrentGraph, state.CanvasNodes, state.CanvasEdges = "child", 1, 0 }),
		connected,
		withState(connected, func(state *pageState) { state.Reroutes = 1 }),
		withState(connected, func(state *pageState) { state.CallMenuOptions = 1 }),
		withState(connected, func(state *pageState) { state.GraphCalls = 1 }),
		withState(connected, func(state *pageState) { state.GraphCalls, state.Annotations = 1, 1 }),
		withState(connected, func(state *pageState) { state.GraphCalls, state.Annotations, state.SaveInlineFeedback = 1, 1, true }),
		withState(connected, func(state *pageState) { state.AIReview = true }),
		withState(connected, func(state *pageState) { state.AIReview = true }),
		withState(connected, func(state *pageState) { state.AssetsView, state.AssetsRecording = true, true }),
		withState(connected, func(state *pageState) { state.AssetsView, state.AssetsRecording = true, true }),
	}

	var serverURL string
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
			if call.Method == "Runtime.evaluate" && strings.Contains(expression, "const probe = document.createElement") {
				if len(states) == 0 {
					t.Errorf("unexpected extra page state request")
					return
				}
				raw, _ := json.Marshal(states[0])
				states = states[1:]
				result = map[string]any{"result": map[string]any{"value": string(raw)}}
			} else if call.Method == "Runtime.evaluate" && strings.Contains(expression, "JSON.stringify") {
				value := `{"start":{"x":10,"y":10},"end":{"x":20,"y":20}}`
				if strings.Contains(expression, "multi-selection needs") {
					value = `[{"x":10,"y":10},{"x":20,"y":20}]`
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
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := run(ctx, server.URL, screenshot, assetsScreenshot); err != nil {
		t.Fatal(err)
	}
	if len(states) != 0 {
		t.Fatalf("unconsumed page states: %d", len(states))
	}
	for _, path := range []string{screenshot, assetsScreenshot} {
		if raw, err := os.ReadFile(path); err != nil || string(raw) != "png" {
			t.Fatalf("screenshot %s = %q, %v", path, raw, err)
		}
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
