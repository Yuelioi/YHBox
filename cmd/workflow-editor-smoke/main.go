package main

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/yottaapp/yotta/internal/automation/browsercdp"
)

type pageState struct {
	Href                 string         `json:"href"`
	Catalog              int            `json:"catalog"`
	CanvasNodes          int            `json:"canvasNodes"`
	CanvasEdges          int            `json:"canvasEdges"`
	AIReview             bool           `json:"aiReview"`
	WorkflowState        bool           `json:"workflowState"`
	RunStarted           bool           `json:"runStarted"`
	AssetsView           bool           `json:"assetsView"`
	AssetsRecording      bool           `json:"assetsRecording"`
	CreateInput          bool           `json:"createInput"`
	RecoveryPanel        bool           `json:"recoveryPanel"`
	LauncherButton       bool           `json:"launcherButton"`
	GraphChromeDark      bool           `json:"graphChromeDark"`
	HandleOverlaps       int            `json:"handleOverlaps"`
	NativeConfirmCalls   int            `json:"nativeConfirmCalls"`
	ConfirmDialog        bool           `json:"confirmDialog"`
	Dirty                bool           `json:"dirty"`
	SaveInlineFeedback   bool           `json:"saveInlineFeedback"`
	SaveError            string         `json:"saveError"`
	SaveToast            bool           `json:"saveToast"`
	SelectedNodes        int            `json:"selectedNodes"`
	SelectionToolbar     bool           `json:"selectionToolbar"`
	ConnectionMenu       bool           `json:"connectionMenu"`
	ConnectionCandidates int            `json:"connectionCandidates"`
	ConnectionError      string         `json:"connectionError"`
	Debugger             bool           `json:"debugger"`
	DebugPaused          bool           `json:"debugPaused"`
	DebugCompleted       bool           `json:"debugCompleted"`
	DebugCurrent         int            `json:"debugCurrent"`
	DebugNode            string         `json:"debugNode"`
	Breakpoints          int            `json:"breakpoints"`
	CurrentGraph         string         `json:"currentGraph"`
	GraphCalls           int            `json:"graphCalls"`
	Annotations          int            `json:"annotations"`
	GraphNameInput       bool           `json:"graphNameInput"`
	CallMenuOptions      int            `json:"callMenuOptions"`
	Reroutes             int            `json:"reroutes"`
	NodeOverlaps         int            `json:"nodeOverlaps"`
	NodeGeometry         []nodeGeometry `json:"nodeGeometry"`
	Errors               []string       `json:"errors"`
}

type nodeGeometry struct {
	ID        string  `json:"id"`
	X         float64 `json:"x"`
	Y         float64 `json:"y"`
	Width     float64 `json:"width"`
	Height    float64 `json:"height"`
	Transform string  `json:"transform"`
}

type point struct {
	X float64 `json:"x"`
	Y float64 `json:"y"`
}

type connectionGesture struct {
	Start point `json:"start"`
	End   point `json:"end"`
}

func main() {
	endpoint := flag.String("endpoint", "http://127.0.0.1:9227", "WebView2 CDP endpoint")
	screenshot := flag.String("screenshot", ".task/workflow-editor-smoke.png", "PNG output path")
	assetsScreenshot := flag.String("assets-screenshot", ".task/assets-smoke.png", "asset library PNG output path")
	workflowsScreenshot := flag.String("workflows-screenshot", ".task/workflows-smoke.png", "workflow recovery PNG output path")
	launcherScreenshot := flag.String("launcher-screenshot", ".task/launcher-smoke.png", "floating launcher PNG output path")
	flag.Parse()

	ctx, cancel := context.WithTimeout(context.Background(), 180*time.Second)
	defer cancel()
	if err := run(ctx, *endpoint, *screenshot, *assetsScreenshot, *workflowsScreenshot, *launcherScreenshot); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run(ctx context.Context, endpoint, screenshot, assetsScreenshot, workflowsScreenshot, launcherScreenshot string) error {
	targets, err := browsercdp.NewService(endpoint).ListTargets(ctx, endpoint)
	if err != nil {
		return fmt.Errorf("discover Wails WebView: %w", err)
	}
	if len(targets) != 1 {
		return fmt.Errorf("expected one Wails page target, got %d", len(targets))
	}
	client, err := browsercdp.DialWebSocketClient(ctx, targets[0].WebSocketDebuggerURL)
	if err != nil {
		return fmt.Errorf("connect Wails WebView: %w", err)
	}
	defer client.Close()

	if _, err := client.Call(ctx, "Runtime.enable", nil); err != nil {
		return err
	}
	if _, err := client.Call(ctx, "Page.enable", nil); err != nil {
		return err
	}
	if err := eval(ctx, client, `(() => {
		window.__yottaSmokeErrors = [];
		window.addEventListener('error', event => window.__yottaSmokeErrors.push(String(event.error?.stack || event.message)));
		window.addEventListener('unhandledrejection', event => window.__yottaSmokeErrors.push(String(event.reason?.stack || event.reason)));
		const originalError = console.error.bind(console);
		console.error = (...args) => {
			window.__yottaSmokeErrors.push(args.map(value => { try { return String(value) } catch { return '<unprintable>' } }).join(' '));
			originalError(...args);
		};
	})()`); err != nil {
		return err
	}
	if err := waitUntil(ctx, client, func(current pageState) bool {
		return current.CreateInput && current.RecoveryPanel && current.LauncherButton
	}); err != nil {
		return fmt.Errorf("wait for workflow list hydration: %w", err)
	}
	if workflowsScreenshot != "" {
		if err := capture(ctx, client, workflowsScreenshot); err != nil {
			return fmt.Errorf("capture workflow recovery surface: %w", err)
		}
	}
	nameJSON, _ := json.Marshal("Agent UI smoke " + time.Now().UTC().Format("20060102T150405Z"))
	if err := eval(ctx, client, fmt.Sprintf(`(() => {
		const input = document.querySelector('input[data-testid="workflow-create-name"], [data-testid="workflow-create-name"] input');
		if (!input) throw new Error('workflow name input not found');
		const setter = Object.getOwnPropertyDescriptor(HTMLInputElement.prototype, 'value').set;
		setter.call(input, %s);
		input.dispatchEvent(new InputEvent('input', { bubbles: true, inputType: 'insertText' }));
		input.dispatchEvent(new Event('change', { bubbles: true }));
	})()`, nameJSON)); err != nil {
		return err
	}
	if err := waitFor(ctx, client, func(state pageState) bool { return state.Catalog == 0 }, func() error {
		return eval(ctx, client, `(() => {
			const button = document.querySelector('[data-testid="workflow-create-submit"]');
			if (!button || button.disabled) throw new Error('create workflow button is unavailable');
			button.click();
		})()`)
	}); err != nil {
		return err
	}
	if err := waitUntil(ctx, client, func(state pageState) bool { return state.Catalog > 0 }); err != nil {
		return fmt.Errorf("open workflow editor: %w", err)
	}
	if launcherScreenshot != "" {
		if err := configureLauncherWorkflow(ctx, client); err != nil {
			return err
		}
		if err := exerciseLauncher(ctx, endpoint, targets[0].ID, client, launcherScreenshot); err != nil {
			return err
		}
	}
	if err := exerciseDebugger(ctx, client); err != nil {
		return err
	}

	before, err := state(ctx, client)
	if err != nil {
		return err
	}
	if err := eval(ctx, client, `document.querySelector('[data-node-type-id="https://schemas.yotta.dev/nodes/text/concat"]')?.click()`); err != nil {
		return err
	}
	if err := waitUntil(ctx, client, func(current pageState) bool { return current.CanvasNodes == before.CanvasNodes+1 }); err != nil {
		return fmt.Errorf("click catalog node: %w", err)
	}

	afterClick, err := state(ctx, client)
	if err != nil {
		return err
	}
	if err := eval(ctx, client, `(() => {
		const item = document.querySelectorAll('[data-testid="node-catalog-item"]')[1];
		const canvas = document.querySelector('[data-testid="workflow-canvas"]');
		if (!item || !canvas) throw new Error('drag source or workflow canvas not found');
		const rect = canvas.getBoundingClientRect();
		const data = new DataTransfer();
		const point = { clientX: rect.left + rect.width * 0.62, clientY: rect.top + rect.height * 0.42 };
		item.dispatchEvent(new DragEvent('dragstart', { bubbles: true, cancelable: true, dataTransfer: data, ...point }));
		canvas.dispatchEvent(new DragEvent('dragover', { bubbles: true, cancelable: true, dataTransfer: data, ...point }));
		canvas.dispatchEvent(new DragEvent('drop', { bubbles: true, cancelable: true, dataTransfer: data, ...point }));
		item.dispatchEvent(new DragEvent('dragend', { bubbles: true, dataTransfer: data, ...point }));
	})()`); err != nil {
		return err
	}
	if err := waitUntil(ctx, client, func(current pageState) bool { return current.CanvasNodes == afterClick.CanvasNodes+1 }); err != nil {
		return fmt.Errorf("drag catalog node: %w", err)
	}

	if err := eval(ctx, client, `(() => {
		const button = document.querySelector('[data-testid="workflow-layout-lr"]');
		if (!button) throw new Error('left-to-right layout button not found');
		button.click();
	})()`); err != nil {
		return err
	}
	if err := waitUntil(ctx, client, func(current pageState) bool { return current.NodeOverlaps == 0 }); err != nil {
		return fmt.Errorf("auto-layout workflow nodes: %w", err)
	}
	if err := eval(ctx, client, `(() => {
		const pane = document.querySelector('.vue-flow__pane');
		if (!pane) throw new Error('workflow pane not found');
		pane.dispatchEvent(new MouseEvent('click', { bubbles: true, cancelable: true, view: window }));
	})()`); err != nil {
		return err
	}
	if err := waitUntil(ctx, client, func(current pageState) bool {
		return current.SelectedNodes == 0 && !current.SelectionToolbar
	}); err != nil {
		return fmt.Errorf("clear workflow selection before box select: %w", err)
	}

	var selectionPoints []point
	if err := evalJSON(ctx, client, `(() => {
		const nodes = [...document.querySelectorAll('.vue-flow__node')].slice(-2);
		if (nodes.length < 2) throw new Error('multi-selection needs two workflow nodes');
		return nodes.map(node => {
			const rect = node.getBoundingClientRect();
			return { x: rect.left + rect.width / 2, y: rect.top + rect.height / 2 };
		});
	})()`, &selectionPoints); err != nil {
		return err
	}
	if err := dispatchMultiSelectClicks(ctx, client, selectionPoints); err != nil {
		return err
	}
	if err := waitUntil(ctx, client, func(current pageState) bool {
		return current.SelectedNodes == 2 && current.SelectionToolbar
	}); err != nil {
		return fmt.Errorf("multi-select workflow nodes: %w", err)
	}
	beforeDelete, err := state(ctx, client)
	if err != nil {
		return err
	}
	if err := clickRequired(ctx, client, "workflow-selection-remove"); err != nil {
		return err
	}
	if err := waitUntil(ctx, client, func(current pageState) bool {
		return current.CanvasNodes == beforeDelete.CanvasNodes-2 && current.SelectedNodes == 0 && !current.SelectionToolbar
	}); err != nil {
		return fmt.Errorf("delete selected workflow nodes: %w", err)
	}
	beforeConnection, err := state(ctx, client)
	if err != nil {
		return err
	}

	var gesture connectionGesture
	if err := evalJSON(ctx, client, `(() => {
		const handle = document.querySelector('.vue-flow__node[data-id="run-started"] .vue-flow__handle.source');
		const canvas = document.querySelector('[data-testid="workflow-canvas"]');
		if (!handle || !canvas) throw new Error('connection handle or canvas not found');
		const h = handle.getBoundingClientRect();
		const c = canvas.getBoundingClientRect();
		const candidates = [];
		for (const y of [0.82, 0.72, 0.62, 0.52, 0.42, 0.32, 0.22]) {
			for (const x of [0.82, 0.72, 0.62, 0.52, 0.42, 0.32, 0.22]) {
				candidates.push({ x: c.left + c.width * x, y: c.top + c.height * y });
			}
		}
		const end = candidates.find(point => {
			const element = document.elementFromPoint(point.x, point.y);
			return element && canvas.contains(element) && !element.closest('.vue-flow__node, .vue-flow__edge, .vue-flow__controls, .vue-flow__minimap, [data-testid="workflow-selection-toolbar"]');
		});
		if (!end) throw new Error('blank connection drop point not found');
		return { start: { x: h.left + h.width / 2, y: h.top + h.height / 2 }, end };
	})()`, &gesture); err != nil {
		return err
	}
	if err := dispatchConnectionGesture(ctx, client, gesture); err != nil {
		return err
	}
	if err := waitUntil(ctx, client, func(current pageState) bool { return current.ConnectionMenu }); err != nil {
		return fmt.Errorf("open compatible connection menu: %w", err)
	}
	if err := eval(ctx, client, `(() => {
		const input = document.querySelector('[data-testid="workflow-connection-search"] input, input[data-testid="workflow-connection-search"]');
		if (!input) throw new Error('connection candidate search input not found');
		const setter = Object.getOwnPropertyDescriptor(HTMLInputElement.prototype, 'value').set;
		setter.call(input, 'delay');
		input.dispatchEvent(new InputEvent('input', { bubbles: true, inputType: 'insertText' }));
	})()`); err != nil {
		return err
	}
	if err := waitUntil(ctx, client, func(current pageState) bool {
		return current.ConnectionCandidates == 1
	}); err != nil {
		return fmt.Errorf("filter compatible connection candidates: %w", err)
	}
	if err := eval(ctx, client, `(() => {
		const candidate = document.querySelector('[data-testid="workflow-connection-candidate"][data-node-type-id="https://schemas.yotta.dev/nodes/control/delay"][data-port-id="in"]');
		if (!candidate) throw new Error('Delay.in connection candidate not found');
		candidate.click();
	})()`); err != nil {
		return err
	}
	if err := waitForConnectionInsert(ctx, client, beforeConnection); err != nil {
		return fmt.Errorf("insert compatible connection candidate: %w", err)
	}

	visualState, err := state(ctx, client)
	if err != nil {
		return err
	}
	if err := eval(ctx, client, `(() => {
		const input = document.querySelector('[data-testid="workflow-catalog-search"] input, input[data-testid="workflow-catalog-search"]');
		if (!input) throw new Error('catalog search input not found');
		const setter = Object.getOwnPropertyDescriptor(HTMLInputElement.prototype, 'value').set;
		setter.call(input, 'concat');
		input.dispatchEvent(new InputEvent('input', { bubbles: true, inputType: 'insertText' }));
	})()`); err != nil {
		return err
	}
	if err := waitUntil(ctx, client, func(current pageState) bool { return current.Catalog == 1 }); err != nil {
		return fmt.Errorf("filter node catalog: %w", err)
	}
	if err := eval(ctx, client, `(() => {
		const input = document.querySelector('[data-testid="workflow-catalog-search"] input, input[data-testid="workflow-catalog-search"]');
		const setter = Object.getOwnPropertyDescriptor(HTMLInputElement.prototype, 'value').set;
		setter.call(input, '');
		input.dispatchEvent(new InputEvent('input', { bubbles: true, inputType: 'deleteContentBackward' }));
	})()`); err != nil {
		return err
	}
	if err := waitUntil(ctx, client, func(current pageState) bool { return current.Catalog > 1 }); err != nil {
		return fmt.Errorf("clear node catalog search: %w", err)
	}

	if err := eval(ctx, client, `(() => {
		window.__yottaNativeConfirmCalls = 0;
		window.confirm = () => {
			window.__yottaNativeConfirmCalls++;
			return false;
		};
		const button = document.querySelector('[data-testid="workflow-editor-back"]');
		if (!button) throw new Error('workflow editor back button not found');
		button.click();
	})()`); err != nil {
		return err
	}
	if err := waitUntil(ctx, client, func(current pageState) bool {
		return current.NativeConfirmCalls > 0 || current.ConfirmDialog
	}); err != nil {
		return fmt.Errorf("request discard confirmation: %w", err)
	}
	confirmState, err := state(ctx, client)
	if err != nil {
		return err
	}
	if confirmState.ConfirmDialog {
		if err := eval(ctx, client, `document.querySelector('[data-testid="confirm-cancel"]')?.click()`); err != nil {
			return err
		}
	}

	if err := eval(ctx, client, `(() => {
		const button = document.querySelector('[data-testid="workflow-save"]');
		if (!button || button.disabled) throw new Error('workflow save button is unavailable');
		button.click();
	})()`); err != nil {
		return err
	}
	if err := waitForSave(ctx, client); err != nil {
		return fmt.Errorf("save workflow: %w", err)
	}
	saveState, err := state(ctx, client)
	if err != nil {
		return err
	}
	if err := eval(ctx, client, `document.querySelector('[data-testid="workflow-state-open"]')?.click()`); err != nil {
		return err
	}
	if err := waitUntil(ctx, client, func(current pageState) bool { return current.WorkflowState }); err != nil {
		return fmt.Errorf("open workflow state panel: %w", err)
	}
	if err := eval(ctx, client, `document.querySelector('[data-testid="workflow-state-open"]')?.click()`); err != nil {
		return err
	}
	if err := exerciseMultigraph(ctx, client); err != nil {
		return err
	}
	if err := clickRequired(ctx, client, "workflow-save"); err != nil {
		return err
	}
	if err := waitForSave(ctx, client); err != nil {
		return fmt.Errorf("save multigraph workflow: %w", err)
	}
	uiFailures := workflowEditorUIFailures(visualState, confirmState, saveState)
	if err := eval(ctx, client, `(() => {
		const button = document.querySelector('[data-testid="ai-workflow-review-open"]');
		if (!button) throw new Error('AI workflow review button not found');
		button.click();
	})()`); err != nil {
		return err
	}
	if err := waitUntil(ctx, client, func(current pageState) bool { return current.AIReview }); err != nil {
		return fmt.Errorf("open AI workflow review: %w", err)
	}

	final, err := state(ctx, client)
	if err != nil {
		return err
	}
	if len(final.Errors) != 0 {
		return fmt.Errorf("WebView reported errors: %s", strings.Join(final.Errors, " | "))
	}
	if len(uiFailures) != 0 {
		return fmt.Errorf("workflow editor UI regressions: %s", strings.Join(uiFailures, " | "))
	}
	if _, err := client.Call(ctx, "Page.bringToFront", nil); err != nil {
		return err
	}
	if _, err := client.Call(ctx, "Emulation.setFocusEmulationEnabled", map[string]any{"enabled": true}); err != nil {
		return err
	}
	if err := eval(ctx, client, `new Promise(resolve => requestAnimationFrame(() => requestAnimationFrame(() => setTimeout(resolve, 500))))`); err != nil {
		return err
	}
	if err := capture(ctx, client, screenshot); err != nil {
		return err
	}
	if err := eval(ctx, client, `location.hash = '#/assets'`); err != nil {
		return err
	}
	if err := waitUntil(ctx, client, func(current pageState) bool {
		return current.AssetsView && current.AssetsRecording
	}); err != nil {
		return fmt.Errorf("open asset library: %w", err)
	}
	assetsState, err := state(ctx, client)
	if err != nil {
		return err
	}
	if len(assetsState.Errors) != 0 {
		return fmt.Errorf("asset library reported errors: %s", strings.Join(assetsState.Errors, " | "))
	}
	if err := eval(ctx, client, `new Promise(resolve => requestAnimationFrame(() => requestAnimationFrame(() => setTimeout(resolve, 300))))`); err != nil {
		return err
	}
	if err := capture(ctx, client, assetsScreenshot); err != nil {
		return err
	}
	result, _ := json.MarshalIndent(map[string]any{
		"status": "passed", "href": final.Href, "catalogNodes": final.Catalog,
		"canvasNodes": final.CanvasNodes, "aiReview": final.AIReview, "screenshot": screenshot,
		"assetsScreenshot": assetsScreenshot, "workflowsScreenshot": workflowsScreenshot,
	}, "", "  ")
	fmt.Println(string(result))
	return nil
}

func exerciseMultigraph(ctx context.Context, client *browsercdp.WebSocketClient) error {
	if err := clickRequired(ctx, client, "workflow-graph-new"); err != nil {
		return err
	}
	if err := waitUntil(ctx, client, func(current pageState) bool { return current.GraphNameInput }); err != nil {
		return fmt.Errorf("open new subgraph dialog: %w", err)
	}
	if err := eval(ctx, client, `(() => {
		const input = document.querySelector('[data-testid="workflow-graph-name"] input, input[data-testid="workflow-graph-name"]');
		if (!input) throw new Error('subgraph name input not found');
		const setter = Object.getOwnPropertyDescriptor(HTMLInputElement.prototype, 'value').set;
		setter.call(input, 'Reusable wait');
		input.dispatchEvent(new InputEvent('input', { bubbles: true, inputType: 'insertText' }));
	})()`); err != nil {
		return err
	}
	if err := clickRequired(ctx, client, "workflow-graph-confirm"); err != nil {
		return err
	}
	if err := waitUntil(ctx, client, func(current pageState) bool {
		return current.CurrentGraph != "" && current.CurrentGraph != "main" && current.CanvasNodes == 0
	}); err != nil {
		return fmt.Errorf("enter new subgraph: %w", err)
	}
	if err := eval(ctx, client, `(() => {
		const item = document.querySelector('[data-node-type-id="https://schemas.yotta.dev/nodes/control/delay"]');
		if (!item) throw new Error('delay catalog item not found in subgraph');
		item.click();
	})()`); err != nil {
		return err
	}
	if err := waitUntil(ctx, client, func(current pageState) bool { return current.CanvasNodes == 1 }); err != nil {
		return fmt.Errorf("author subgraph node: %w", err)
	}
	if err := clickRequired(ctx, client, "workflow-graph-infer-interface"); err != nil {
		return err
	}
	if err := clickRequired(ctx, client, "workflow-graph-breadcrumb-main"); err != nil {
		return err
	}
	if err := waitUntil(ctx, client, func(current pageState) bool { return current.CurrentGraph == "main" }); err != nil {
		return fmt.Errorf("return to main graph: %w", err)
	}
	if err := eval(ctx, client, `(() => {
		const edge = document.querySelector('.vue-flow__edge .vue-flow__edge-interaction, .vue-flow__edge path');
		if (!edge) throw new Error('workflow edge not found for reroute');
		edge.dispatchEvent(new MouseEvent('click', { bubbles: true, cancelable: true, view: window }));
	})()`); err != nil {
		return err
	}
	if err := clickRequired(ctx, client, "workflow-reroute-add"); err != nil {
		return err
	}
	if err := waitUntil(ctx, client, func(current pageState) bool { return current.Reroutes > 0 }); err != nil {
		return fmt.Errorf("add edge reroute: %w", err)
	}
	if err := clickRequired(ctx, client, "workflow-graph-add-call"); err != nil {
		return err
	}
	if err := waitUntil(ctx, client, func(current pageState) bool { return current.CallMenuOptions > 0 }); err != nil {
		return fmt.Errorf("open callable subgraph menu: %w", err)
	}
	if err := eval(ctx, client, `(() => {
		const item = document.querySelector('[role="menu"] [role="menuitem"]');
		if (!item) throw new Error('callable subgraph menu item not found');
		item.click();
	})()`); err != nil {
		return err
	}
	if err := waitUntil(ctx, client, func(current pageState) bool { return current.GraphCalls == 1 }); err != nil {
		return fmt.Errorf("insert graph call: %w", err)
	}
	if err := clickRequired(ctx, client, "workflow-annotation-add"); err != nil {
		return err
	}
	if err := waitUntil(ctx, client, func(current pageState) bool { return current.Annotations == 1 }); err != nil {
		return fmt.Errorf("add graph annotation: %w", err)
	}
	return nil
}

func exerciseDebugger(ctx context.Context, client *browsercdp.WebSocketClient) error {
	if err := eval(ctx, client, `(() => {
		const button = document.querySelector('.vue-flow__node[data-id="run-started"] [data-testid="node-breakpoint"]');
		if (!button) throw new Error('RunStarted breakpoint button not found');
		button.click();
	})()`); err != nil {
		return err
	}
	if err := waitUntil(ctx, client, func(current pageState) bool { return current.Breakpoints == 1 }); err != nil {
		return fmt.Errorf("set node breakpoint: %w", err)
	}
	if err := clickRequired(ctx, client, "workflow-debug-start"); err != nil {
		return err
	}
	if err := waitUntil(ctx, client, func(current pageState) bool {
		return current.Debugger && current.DebugPaused && current.DebugCurrent == 1 && current.DebugNode == "run-started"
	}); err != nil {
		return fmt.Errorf("start paused debug Run: %w", err)
	}
	if err := clickRequired(ctx, client, "workflow-debug-step"); err != nil {
		return err
	}
	if err := waitUntil(ctx, client, func(current pageState) bool {
		return current.DebugCompleted && current.DebugCurrent == 0
	}); err != nil {
		return fmt.Errorf("step one visible node: %w", err)
	}

	if err := clickRequired(ctx, client, "workflow-debug-start"); err != nil {
		return err
	}
	if err := waitUntil(ctx, client, func(current pageState) bool {
		return current.DebugPaused
	}); err != nil {
		return fmt.Errorf("restart paused debug Run: %w", err)
	}
	if err := clickRequired(ctx, client, "workflow-debug-stop"); err != nil {
		return err
	}
	if err := waitUntil(ctx, client, func(current pageState) bool { return current.DebugCompleted }); err != nil {
		return fmt.Errorf("stop debug Run: %w", err)
	}
	return nil
}

func clickRequired(ctx context.Context, client *browsercdp.WebSocketClient, testID string) error {
	testIDJSON, _ := json.Marshal(testID)
	return eval(ctx, client, fmt.Sprintf(`(() => {
		const button = document.querySelector('[data-testid=' + %s + ']');
		if (!button) throw new Error(%s + ' button not found');
		button.click();
	})()`, testIDJSON, testIDJSON))
}

func workflowEditorUIFailures(visualState, confirmState, saveState pageState) []string {
	var failures []string
	if !visualState.GraphChromeDark {
		failures = append(failures, "Vue Flow controls or minimap use a light background")
	}
	if visualState.HandleOverlaps != 0 {
		failures = append(failures, fmt.Sprintf("%d workflow handles overlap their labels", visualState.HandleOverlaps))
	}
	if !visualState.RunStarted {
		failures = append(failures, "new workflow omitted the RunStarted root")
	}
	if confirmState.NativeConfirmCalls != 0 {
		failures = append(failures, "workflow navigation called window.confirm")
	}
	if !confirmState.ConfirmDialog {
		failures = append(failures, "workflow navigation did not open the shared confirm dialog")
	}
	if saveState.SaveToast {
		failures = append(failures, "workflow save displayed a success toast")
	}
	if !saveState.SaveInlineFeedback {
		failures = append(failures, "workflow save omitted inline success feedback")
	}
	return failures
}

func waitFor(ctx context.Context, client *browsercdp.WebSocketClient, ready func(pageState) bool, action func() error) error {
	current, err := state(ctx, client)
	if err != nil {
		return err
	}
	if !ready(current) {
		return errors.New("unexpected initial page state")
	}
	time.Sleep(100 * time.Millisecond)
	return action()
}

func waitUntil(ctx context.Context, client *browsercdp.WebSocketClient, predicate func(pageState) bool) error {
	for {
		current, err := state(ctx, client)
		if err != nil {
			return err
		}
		if predicate(current) {
			return nil
		}
		select {
		case <-ctx.Done():
			details, _ := json.Marshal(current)
			return fmt.Errorf("%w; last page state: %s", ctx.Err(), details)
		case <-time.After(100 * time.Millisecond):
		}
	}
}

func waitForConnectionInsert(ctx context.Context, client *browsercdp.WebSocketClient, before pageState) error {
	for {
		current, err := state(ctx, client)
		if err != nil {
			return err
		}
		if current.ConnectionError != "" {
			return errors.New(current.ConnectionError)
		}
		if !current.ConnectionMenu && current.CanvasNodes == before.CanvasNodes+1 && current.CanvasEdges == before.CanvasEdges+1 {
			return nil
		}
		select {
		case <-ctx.Done():
			details, _ := json.Marshal(current)
			return fmt.Errorf("%w; last page state: %s", ctx.Err(), details)
		case <-time.After(100 * time.Millisecond):
		}
	}
}

func waitForSave(ctx context.Context, client *browsercdp.WebSocketClient) error {
	for {
		current, err := state(ctx, client)
		if err != nil {
			return err
		}
		if current.SaveError != "" {
			return errors.New(current.SaveError)
		}
		if !current.Dirty && (current.SaveInlineFeedback || current.SaveToast) {
			return nil
		}
		select {
		case <-ctx.Done():
			details, _ := json.Marshal(current)
			return fmt.Errorf("%w; last page state: %s", ctx.Err(), details)
		case <-time.After(100 * time.Millisecond):
		}
	}
}

func exerciseLauncher(ctx context.Context, endpoint, mainTargetID string, mainClient *browsercdp.WebSocketClient, screenshot string) error {
	discovery := browsercdp.NewService(endpoint)
	var launcherTargetID string
	for cycle := 0; cycle < 2; cycle++ {
		if err := eval(ctx, mainClient, `(() => {
			const button = document.querySelector('[data-testid="open-launcher"]');
			if (!button || button.disabled) throw new Error('launcher button is unavailable');
			button.click();
		})()`); err != nil {
			return fmt.Errorf("open floating launcher: %w", err)
		}
		var launcher browsercdp.TargetInfo
		for !launcherTargetValid(launcher) {
			targets, err := discovery.ListTargets(ctx, endpoint)
			if err != nil {
				return fmt.Errorf("discover floating launcher: %w", err)
			}
			launcherTargets := targetsExcept(targets, mainTargetID)
			if len(launcherTargets) > 1 {
				return fmt.Errorf("floating launcher opened duplicate targets: %d", len(launcherTargets))
			}
			if len(launcherTargets) == 1 {
				launcher = launcherTargets[0]
			}
			if launcherTargetValid(launcher) {
				break
			}
			select {
			case <-ctx.Done():
				return fmt.Errorf("wait for floating launcher: %w", ctx.Err())
			case <-time.After(100 * time.Millisecond):
			}
		}
		if launcherTargetID == "" {
			launcherTargetID = launcher.ID
		} else if launcher.ID != launcherTargetID {
			return fmt.Errorf("floating launcher did not reuse its window: first target %q, next target %q", launcherTargetID, launcher.ID)
		}
		launcherClient, err := browsercdp.DialWebSocketClient(ctx, launcher.WebSocketDebuggerURL)
		if err != nil {
			return fmt.Errorf("connect floating launcher: %w", err)
		}
		if _, err := launcherClient.Call(ctx, "Runtime.enable", nil); err != nil {
			launcherClient.Close()
			return err
		}
		if _, err := launcherClient.Call(ctx, "Page.enable", nil); err != nil {
			launcherClient.Close()
			return err
		}
		for {
			var ready bool
			if err := evalJSON(ctx, launcherClient, `Boolean(document.querySelector('.launcher-content .launcher-command'))`, &ready); err == nil && ready {
				break
			}
			select {
			case <-ctx.Done():
				launcherClient.Close()
				return fmt.Errorf("wait for floating launcher content: %w", ctx.Err())
			case <-time.After(100 * time.Millisecond):
			}
		}
		if cycle == 0 {
			if err := eval(ctx, launcherClient, `(() => {
				const command = document.querySelector('.launcher-command');
				if (!command || command.disabled) throw new Error('launcher workflow command is unavailable');
				command.click();
			})()`); err != nil {
				launcherClient.Close()
				return fmt.Errorf("execute workflow from floating launcher: %w", err)
			}
			for {
				var succeeded bool
				if err := evalJSON(ctx, launcherClient, `Boolean(document.querySelector('.launcher-command--success'))`, &succeeded); err == nil && succeeded {
					break
				}
				select {
				case <-ctx.Done():
					launcherClient.Close()
					return fmt.Errorf("wait for floating launcher workflow success: %w", ctx.Err())
				case <-time.After(100 * time.Millisecond):
				}
			}
			if err := capture(ctx, launcherClient, screenshot); err != nil {
				launcherClient.Close()
				return fmt.Errorf("capture floating launcher: %w", err)
			}
		}
		if err := eval(ctx, launcherClient, `(() => {
			const buttons = [...document.querySelectorAll('.hud-shell__actions button')];
			const close = buttons.at(-1);
			if (!close || close.disabled) throw new Error('launcher hide button is unavailable');
			close.click();
		})()`); err != nil {
			launcherClient.Close()
			return fmt.Errorf("hide floating launcher: %w", err)
		}
		launcherClient.Close()

		// HideLauncher deliberately retains the WebView so query, selection and
		// window geometry survive the next invocation. Give the async click
		// handler a moment to cross the Wails bridge, then require exactly that
		// one retained target; the next cycle proves it is reused.
		select {
		case <-ctx.Done():
			return fmt.Errorf("wait for floating launcher hide: %w", ctx.Err())
		case <-time.After(250 * time.Millisecond):
		}
		targets, err := discovery.ListTargets(ctx, endpoint)
		if err != nil {
			return fmt.Errorf("check floating launcher retention: %w", err)
		}
		launcherTargets := targetsExcept(targets, mainTargetID)
		if len(launcherTargets) != 1 || launcherTargets[0].ID != launcherTargetID {
			return fmt.Errorf("floating launcher retention drifted: targets=%v", launcherTargets)
		}
	}
	return nil
}

func configureLauncherWorkflow(ctx context.Context, client *browsercdp.WebSocketClient) error {
	if err := eval(ctx, client, `(async () => {
		const match = location.hash.match(/\/workflows\/([^/]+)\/edit/);
		if (!match) throw new Error('created workflow identity is unavailable');
		const { backend } = await import('/src/lib/backend.ts');
		await backend.settings.update({ ui: { launcherItems: [{
			id: 'workflow-editor-smoke', type: 'workflow', workflowId: match[1],
			icon: 'i-tabler-player-play', label: 'Smoke workflow'
		}] } });
	})()`); err != nil {
		return fmt.Errorf("configure floating launcher workflow: %w", err)
	}
	return nil
}

func targetsExcept(targets []browsercdp.TargetInfo, excludedID string) []browsercdp.TargetInfo {
	out := make([]browsercdp.TargetInfo, 0, len(targets))
	for _, target := range targets {
		if target.ID != excludedID {
			out = append(out, target)
		}
	}
	return out
}

func launcherTargetValid(target browsercdp.TargetInfo) bool {
	return target.ID != "" && target.WebSocketDebuggerURL != ""
}

func state(ctx context.Context, client *browsercdp.WebSocketClient) (pageState, error) {
	var out pageState
	err := evalJSON(ctx, client, `(() => {
		const probe = document.createElement('div');
		probe.style.background = 'var(--ui-bg)';
		document.body.append(probe);
		const expectedBackground = getComputedStyle(probe).backgroundColor;
		probe.remove();
		const darkBackground = element => {
			if (!element) return false;
			return getComputedStyle(element).backgroundColor === expectedBackground;
		};
		const controls = document.querySelector('.vue-flow__controls');
		const controlButtons = [...document.querySelectorAll('.vue-flow__controls-button')];
		const minimap = document.querySelector('.vue-flow__minimap');
		const handleOverlaps = [...document.querySelectorAll('.workflow-node .vue-flow__handle')].filter(handle => {
			const label = handle.parentElement?.querySelector('span');
			if (!label) return false;
			const a = handle.getBoundingClientRect();
			const b = label.getBoundingClientRect();
			return a.left < b.right && a.right > b.left && a.top < b.bottom && a.bottom > b.top;
		}).length;
		const bodyText = document.body.innerText;
		const saveButtonText = document.querySelector('[data-testid="workflow-save"]')?.innerText || '';
		const nodeRects = [...document.querySelectorAll('.vue-flow__node')].map(node => node.getBoundingClientRect());
		let nodeOverlaps = 0;
		for (let left = 0; left < nodeRects.length; left++) {
			for (let right = left + 1; right < nodeRects.length; right++) {
				const a = nodeRects[left], b = nodeRects[right];
				if (a.left < b.right && a.right > b.left && a.top < b.bottom && a.bottom > b.top) nodeOverlaps++;
			}
		}
		return {
		href: location.href,
		catalog: document.querySelectorAll('[data-testid="node-catalog-item"]').length,
		canvasNodes: document.querySelectorAll('.vue-flow__node').length,
		canvasEdges: document.querySelectorAll('.vue-flow__edge').length,
		aiReview: Boolean(document.querySelector('[data-testid="ai-workflow-review-panel"]')),
		workflowState: Boolean(document.querySelector('[data-testid="workflow-state-panel"]')),
		runStarted: Boolean(document.querySelector('.vue-flow__node[data-id="run-started"]')),
		assetsView: Boolean(document.querySelector('[data-testid="assets-view"]')),
		assetsRecording: Boolean(document.querySelector('[data-testid="assets-recording-controls"]')),
		createInput: Boolean(document.querySelector('input[data-testid="workflow-create-name"], [data-testid="workflow-create-name"] input')),
		recoveryPanel: Boolean(document.querySelector('[data-testid="workflow-recovery-panel"]')),
		launcherButton: Boolean(document.querySelector('[data-testid="open-launcher"]')),
		graphChromeDark: darkBackground(controls) && controlButtons.length > 0 && controlButtons.every(darkBackground) && darkBackground(minimap),
		handleOverlaps,
		nativeConfirmCalls: window.__yottaNativeConfirmCalls || 0,
		confirmDialog: Boolean(document.querySelector('[data-testid="confirm-dialog"]')),
		dirty: Boolean(document.querySelector('[data-testid="workflow-unsaved"]')),
		saveInlineFeedback: saveButtonText.includes('已保存') || saveButtonText.includes('Saved'),
		saveError: document.querySelector('[data-testid="workflow-save-error"]')?.textContent?.trim() || '',
		saveToast: bodyText.includes('工作流已保存') || bodyText.includes('Workflow saved'),
		selectedNodes: document.querySelectorAll('.vue-flow__node.selected').length,
		selectionToolbar: Boolean(document.querySelector('[data-testid="workflow-selection-toolbar"]')),
		connectionMenu: Boolean(document.querySelector('[data-testid="workflow-connection-menu"]')),
		connectionCandidates: document.querySelectorAll('[data-testid="workflow-connection-candidate"]').length,
		connectionError: document.querySelector('[data-testid="workflow-connection-error"]')?.textContent?.trim() || '',
		debugger: Boolean(document.querySelector('[data-testid="workflow-debugger"]')),
		debugPaused: Boolean(document.querySelector('[data-testid="workflow-debugger"]')) && document.querySelector('[data-testid="workflow-debug-step"]') !== null,
		debugCompleted: Boolean(document.querySelector('[data-testid="workflow-debugger"]')) && document.querySelector('[data-testid="workflow-debug-stop"]') === null,
		debugCurrent: document.querySelectorAll('.vue-flow__node [class*="ring-warning"]').length,
		debugNode: document.querySelector('.vue-flow__node [class*="ring-warning"]')?.closest('.vue-flow__node')?.getAttribute('data-id') || '',
		breakpoints: document.querySelectorAll('[data-testid="node-breakpoint"][aria-pressed="true"]').length,
		currentGraph: document.querySelector('[data-testid="workflow-canvas"]')?.getAttribute('data-graph-id') || '',
		graphCalls: document.querySelectorAll('[data-testid="workflow-graph-call"]').length,
		annotations: document.querySelectorAll('[data-testid="workflow-annotation"]').length,
		graphNameInput: Boolean(document.querySelector('[data-testid="workflow-graph-name"] input, input[data-testid="workflow-graph-name"]')),
		callMenuOptions: document.querySelectorAll('[role="menu"] [role="menuitem"]').length,
		reroutes: document.querySelectorAll('[data-testid="workflow-reroute-point"]').length,
		nodeOverlaps,
		nodeGeometry: [...document.querySelectorAll('.vue-flow__node')].map(node => {
			const rect = node.getBoundingClientRect();
			return { id: node.getAttribute('data-id') || '', x: rect.x, y: rect.y, width: rect.width, height: rect.height, transform: node.style.transform };
		}),
		errors: window.__yottaSmokeErrors || []
		};
	})()`, &out)
	return out, err
}

func dispatchConnectionGesture(ctx context.Context, client *browsercdp.WebSocketClient, gesture connectionGesture) error {
	payload, _ := json.Marshal(gesture)
	return eval(ctx, client, fmt.Sprintf(`(async () => {
		const gesture = %s;
		const hit = document.elementFromPoint(gesture.start.x, gesture.start.y);
		const handle = hit?.closest('.vue-flow__handle.source');
		if (!handle) throw new Error('connection gesture hit ' + (hit?.tagName || 'nothing') + '.' + String(hit?.className || ''));
		const mouse = (type, point, buttons) => new MouseEvent(type, {
			bubbles: true, cancelable: true, view: window, button: 0, buttons,
			clientX: point.x, clientY: point.y
		});
		handle.dispatchEvent(mouse('mousedown', gesture.start, 1));
		await new Promise(resolve => setTimeout(resolve, 50));
		for (let step = 1; step <= 8; step++) {
			const ratio = step / 8;
			const point = {
				x: gesture.start.x + (gesture.end.x - gesture.start.x) * ratio,
				y: gesture.start.y + (gesture.end.y - gesture.start.y) * ratio
			};
			document.dispatchEvent(mouse('mousemove', point, 1));
			await new Promise(resolve => setTimeout(resolve, 20));
		}
		document.dispatchEvent(mouse('mouseup', gesture.end, 0));
	})()`, payload))
}

func dispatchMultiSelectClicks(ctx context.Context, client *browsercdp.WebSocketClient, _ []point) error {
	if _, err := client.Call(ctx, "Input.dispatchKeyEvent", map[string]any{
		"type": "keyDown", "modifiers": 2, "key": "Control", "code": "ControlLeft",
		"windowsVirtualKeyCode": 17, "nativeVirtualKeyCode": 17,
	}); err != nil {
		return fmt.Errorf("press Control for multi-selection: %w", err)
	}
	time.Sleep(50 * time.Millisecond)
	gestureErr := eval(ctx, client, `(() => {
		const nodes = [...document.querySelectorAll('.vue-flow__node')].slice(-2);
		if (nodes.length < 2) throw new Error('multi-selection needs two workflow nodes');
		for (const node of nodes) {
			node.dispatchEvent(new MouseEvent('click', {
				bubbles: true, cancelable: true, view: window, button: 0, ctrlKey: true
			}));
		}
	})()`)
	_, releaseErr := client.Call(ctx, "Input.dispatchKeyEvent", map[string]any{
		"type": "keyUp", "key": "Control", "code": "ControlLeft",
		"windowsVirtualKeyCode": 17, "nativeVirtualKeyCode": 17,
	})
	if gestureErr != nil {
		return fmt.Errorf("dispatch workflow node clicks for multi-selection: %w", gestureErr)
	}
	if releaseErr != nil {
		return fmt.Errorf("release Control after multi-selection: %w", releaseErr)
	}
	return nil
}

func eval(ctx context.Context, client *browsercdp.WebSocketClient, expression string) error {
	result, err := client.Call(ctx, "Runtime.evaluate", map[string]any{
		"expression": expression, "awaitPromise": true, "returnByValue": true,
	})
	if err != nil {
		return err
	}
	if details, ok := result["exceptionDetails"].(map[string]any); ok {
		return fmt.Errorf("WebView evaluation failed: %v", details)
	}
	return nil
}

func evalJSON(ctx context.Context, client *browsercdp.WebSocketClient, expression string, out any) error {
	result, err := client.Call(ctx, "Runtime.evaluate", map[string]any{
		"expression":   fmt.Sprintf("JSON.stringify(%s)", expression),
		"awaitPromise": true, "returnByValue": true,
	})
	if err != nil {
		return err
	}
	if details, ok := result["exceptionDetails"].(map[string]any); ok {
		return fmt.Errorf("WebView evaluation failed: %v", details)
	}
	remote, ok := result["result"].(map[string]any)
	if !ok {
		return errors.New("CDP evaluate response omitted result")
	}
	value, ok := remote["value"].(string)
	if !ok {
		return errors.New("CDP evaluate response omitted JSON value")
	}
	return json.Unmarshal([]byte(value), out)
}

func capture(ctx context.Context, client *browsercdp.WebSocketClient, path string) error {
	result, err := client.Call(ctx, "Page.captureScreenshot", map[string]any{
		"format": "png", "fromSurface": true, "captureBeyondViewport": false,
	})
	if err != nil {
		return err
	}
	encoded, ok := result["data"].(string)
	if !ok {
		return errors.New("CDP screenshot response omitted data")
	}
	raw, err := base64.StdEncoding.DecodeString(encoded)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return err
	}
	return os.WriteFile(path, raw, 0o600)
}
