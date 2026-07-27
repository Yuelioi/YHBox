package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/yottaapp/yotta/internal/automation/browsercdp"
)

func captureCurrent(ctx context.Context, endpoint, urlContains, screenshot string) error {
	if strings.TrimSpace(screenshot) == "" {
		return errors.New("screenshot output path is required")
	}
	targets, err := browsercdp.NewService(endpoint).ListTargets(ctx, endpoint)
	if err != nil {
		return fmt.Errorf("discover Wails WebView: %w", err)
	}
	var selected *browsercdp.TargetInfo
	for index := range targets {
		if urlContains != "" && !strings.Contains(targets[index].URL, urlContains) {
			continue
		}
		if selected != nil {
			return fmt.Errorf("multiple WebView pages matched %q; use -url-contains to select one", urlContains)
		}
		selected = &targets[index]
	}
	if selected == nil {
		return fmt.Errorf("no WebView page matched %q", urlContains)
	}
	client, err := browsercdp.DialWebSocketClient(ctx, selected.WebSocketDebuggerURL)
	if err != nil {
		return fmt.Errorf("connect Wails WebView: %w", err)
	}
	defer client.Close()
	for _, call := range []struct {
		method string
		params map[string]any
	}{
		{method: "Runtime.enable"},
		{method: "Page.enable"},
		{method: "Page.bringToFront"},
		{method: "Emulation.setFocusEmulationEnabled", params: map[string]any{"enabled": true}},
	} {
		if _, err := client.Call(ctx, call.method, call.params); err != nil {
			return err
		}
	}
	if err := eval(ctx, client, `new Promise(resolve => requestAnimationFrame(() => requestAnimationFrame(() => setTimeout(resolve, 300))))`); err != nil {
		return err
	}
	if err := capture(ctx, client, screenshot); err != nil {
		return err
	}
	result, _ := json.MarshalIndent(map[string]any{
		"status": "captured", "title": selected.Title, "url": selected.URL, "screenshot": screenshot,
	}, "", "  ")
	fmt.Println(string(result))
	return nil
}

func run(
	ctx context.Context,
	endpoint string,
	screenshot string,
	assetsScreenshot string,
	workflowsScreenshot string,
	launcherScreenshot string,
	schedulesScreenshot string,
	subgraphScreenshot string,
	retentionWorkflowID string,
	firstScreenBudget time.Duration,
) error {
	firstScreenStarted := time.Now()
	nodeMenuScreenshot := siblingScreenshot(screenshot, "node-context-menu.png")
	quickAddScreenshot := siblingScreenshot(screenshot, "quick-add.png")
	runStateScreenshot := siblingScreenshot(screenshot, "run-state.png")
	settingsScreenshot := siblingScreenshot(screenshot, "settings.png")
	assetManagementScreenshot := siblingScreenshot(assetsScreenshot, "asset-management.png")
	workflowManagementScreenshot := siblingScreenshot(workflowsScreenshot, "workflow-management.png")
	manyWorkflowsScreenshot := siblingScreenshot(workflowsScreenshot, "workflows-many.png")
	scheduleBrowseScreenshot := siblingScreenshot(schedulesScreenshot, "schedules-browse.png")
	scheduleManagementScreenshot := siblingScreenshot(schedulesScreenshot, "schedules-management.png")
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
	expectedWorkflowTotal := 0
	if retentionWorkflowID != "" {
		expectedWorkflowTotal = 1
	}
	if firstScreenBudget <= 0 {
		return errors.New("first-screen budget must be positive")
	}
	if err := waitUntilFor(ctx, client, firstScreenBudget, func(current pageState) bool {
		return current.RecoveryPanel && current.WorkflowBrowse &&
			current.WorkflowManageButton && !current.WorkflowManagement && current.LauncherButton &&
			current.WorkflowRows == expectedWorkflowTotal &&
			current.WorkflowTotal == expectedWorkflowTotal
	}); err != nil {
		return fmt.Errorf("wait for workflow list hydration: %w", err)
	}
	firstScreenElapsed := time.Since(firstScreenStarted)
	if workflowsScreenshot != "" {
		if err := capture(ctx, client, workflowsScreenshot); err != nil {
			return fmt.Errorf("capture workflow recovery surface: %w", err)
		}
	}
	if retentionWorkflowID != "" {
		if err := eval(ctx, client, `(() => {
			const rows = document.querySelectorAll('[data-testid="workflow-library-row"]');
			if (rows.length !== 1) throw new Error('golden Workflow row is not unique');
			const button = rows[0].querySelector('[data-testid="workflow-library-open"]');
			if (!button) throw new Error('golden Workflow open button not found');
			button.click();
		})()`); err != nil {
			return err
		}
		if err := waitUntilFor(ctx, client, 15*time.Second, func(current pageState) bool {
			return current.NodeAddTrigger && current.GraphManager &&
				current.CanvasNodes >= 9 && current.GraphCalls >= 6 &&
				current.AppContextTitle == ""
		}); err != nil {
			return fmt.Errorf("open golden Workflow editor: %w", err)
		}
		final, err := state(ctx, client)
		if err != nil {
			return err
		}
		if len(final.Errors) != 0 {
			return fmt.Errorf("golden Workflow editor reported browser errors: %s", strings.Join(final.Errors, " | "))
		}
		if screenshot != "" {
			if err := capture(ctx, client, screenshot); err != nil {
				return fmt.Errorf("capture golden Workflow editor: %w", err)
			}
		}
		result, _ := json.MarshalIndent(map[string]any{
			"status":             "retained",
			"workflowId":         retentionWorkflowID,
			"firstScreenMillis":  firstScreenElapsed.Milliseconds(),
			"mainCanvasNodes":    final.CanvasNodes,
			"mainGraphCalls":     final.GraphCalls,
			"workflowScreenshot": workflowsScreenshot,
			"editorScreenshot":   screenshot,
		}, "", "  ")
		fmt.Println(string(result))
		return nil
	}
	if err := eval(ctx, client, `(() => {
		const button = document.querySelector('[data-testid="workflow-manage-button"]');
		if (!button) throw new Error('workflow manage button not found');
		button.click();
	})()`); err != nil {
		return err
	}
	if err := waitUntil(ctx, client, func(current pageState) bool {
		return current.WorkflowManagement && !current.WorkflowBrowse
	}); err != nil {
		return fmt.Errorf("open workflow management mode: %w", err)
	}
	if workflowManagementScreenshot != "" {
		if err := capture(ctx, client, workflowManagementScreenshot); err != nil {
			return fmt.Errorf("capture workflow management mode: %w", err)
		}
	}
	if err := eval(ctx, client, `(() => {
		const button = document.querySelector('[data-testid="workflow-manage-button"]');
		if (!button) throw new Error('workflow manage done button not found');
		button.click();
	})()`); err != nil {
		return err
	}
	if err := waitUntil(ctx, client, func(current pageState) bool {
		return current.WorkflowBrowse && !current.WorkflowManagement
	}); err != nil {
		return fmt.Errorf("return to workflow browse mode: %w", err)
	}
	if err := eval(ctx, client, `(() => {
		const button = document.querySelector('[data-testid="workflow-new-button"]');
		if (!button) throw new Error('new workflow button not found');
		button.click();
	})()`); err != nil {
		return err
	}
	if err := waitUntil(ctx, client, func(current pageState) bool { return current.CreateInput }); err != nil {
		return fmt.Errorf("wait for workflow creation dialog: %w", err)
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
	if err := waitFor(ctx, client, func(state pageState) bool { return !state.NodeAddTrigger }, func() error {
		return eval(ctx, client, `(() => {
			const button = document.querySelector('[data-testid="workflow-create-submit"]');
			if (!button || button.disabled) throw new Error('create workflow button is unavailable');
			button.click();
		})()`)
	}); err != nil {
		return err
	}
	if err := waitUntil(ctx, client, func(state pageState) bool {
		return state.NodeAddTrigger && state.GraphManager && state.WorkspaceTools == 5 &&
			state.AppContextTitle == ""
	}); err != nil {
		return fmt.Errorf("open focused workflow editor: %w", err)
	}
	if err := verifyEditorToolsAlignment(ctx, client); err != nil {
		return err
	}
	if err := verifyEditorToolbarConsolidation(ctx, client); err != nil {
		return err
	}
	if err := exerciseMinimap(ctx, client); err != nil {
		return err
	}
	if err := exerciseCanvasNodeErgonomics(ctx, client, siblingScreenshot(screenshot, "analyze-color.png")); err != nil {
		return err
	}
	if err := exerciseQuickAdd(ctx, client, quickAddScreenshot); err != nil {
		return err
	}
	if launcherScreenshot != "" {
		if err := configureLauncherWorkflow(ctx, client); err != nil {
			return err
		}
		if err := exerciseLauncher(ctx, endpoint, targets[0].ID, client, launcherScreenshot); err != nil {
			return err
		}
	}
	if err := addNodeViaQuickAdd(
		ctx,
		client,
		"concat",
		"https://schemas.yotta.dev/nodes/text/concat",
	); err != nil {
		return fmt.Errorf("add text concat from the explicit node entry: %w", err)
	}
	if err := addNodeViaQuickAdd(
		ctx,
		client,
		"delay",
		"https://schemas.yotta.dev/nodes/control/delay",
	); err != nil {
		return fmt.Errorf("add delay from the explicit node entry: %w", err)
	}
	if err := clickRequired(ctx, client, "workflow-check"); err != nil {
		return fmt.Errorf("check incomplete workflow: %w", err)
	}
	if err := waitUntil(ctx, client, func(current pageState) bool {
		return current.Dirty && current.Diagnostics && current.MissingInputWarnings > 0
	}); err != nil {
		return fmt.Errorf("show required-input warnings without saving the draft: %w", err)
	}
	if err := clickRequired(ctx, client, "workflow-diagnostics-open"); err != nil {
		return fmt.Errorf("close workflow issues after checking: %w", err)
	}
	if err := waitUntil(ctx, client, func(current pageState) bool { return !current.Diagnostics }); err != nil {
		return fmt.Errorf("close workflow issues: %w", err)
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
	if err := waitForStableNodeGeometry(ctx, client); err != nil {
		return fmt.Errorf("wait for auto-layout geometry to settle: %w", err)
	}
	if err := dispatchKeyPress(ctx, client, "Escape", "Escape", 27); err != nil {
		return err
	}
	if err := waitUntil(ctx, client, func(current pageState) bool {
		return current.SelectedNodes == 0 && !current.SelectionToolbar
	}); err != nil {
		return fmt.Errorf("clear workflow selection before box select: %w", err)
	}

	var selectionGesture connectionGesture
	if err := evalJSON(ctx, client, `(() => {
		const rects = [...document.querySelectorAll('.vue-flow__node:not(.vue-flow__node-graph-boundary)')]
			.map(node => node.getBoundingClientRect());
		if (rects.length < 2) throw new Error('multi-selection needs two workflow nodes');
		const intersects = (left, right) =>
			left.left < right.right && left.right > right.left &&
			left.top < right.bottom && left.bottom > right.top;
		const pane = document.querySelector('.vue-flow__pane');
		const hitsPane = point => document.elementFromPoint(point.x, point.y) === pane;
		const candidates = [];
		for (let left = 0; left < rects.length; left++) {
			for (let right = left + 1; right < rects.length; right++) {
				const box = {
					left: Math.min(rects[left].left, rects[right].left) - 8,
					top: Math.min(rects[left].top, rects[right].top) - 4,
					right: Math.max(rects[left].right, rects[right].right) + 8,
					bottom: Math.max(rects[left].bottom, rects[right].bottom) + 4,
				};
				const covered = rects.filter(rect => intersects(box, rect)).length;
				if (covered === 2) {
					const corners = [
						{ start: { x: box.left, y: box.top }, end: { x: box.right, y: box.bottom } },
						{ start: { x: box.right, y: box.top }, end: { x: box.left, y: box.bottom } },
						{ start: { x: box.right, y: box.bottom }, end: { x: box.left, y: box.top } },
						{ start: { x: box.left, y: box.bottom }, end: { x: box.right, y: box.top } }
					];
					const gesture = corners.find(candidate =>
						hitsPane(candidate.start) && hitsPane(candidate.end));
					if (gesture) {
						candidates.push({
							...gesture,
							area: (box.right - box.left) * (box.bottom - box.top)
						});
					}
				}
			}
		}
		const candidate = candidates.sort((left, right) => left.area - right.area)[0];
		if (!candidate) throw new Error('no isolated two-node marquee region with pane-owned corners found');
		return { start: candidate.start, end: candidate.end };
	})()`, &selectionGesture); err != nil {
		return err
	}
	if err := beginMarqueeTrace(ctx, client); err != nil {
		return err
	}
	if err := dispatchMarqueeGesture(ctx, client, selectionGesture); err != nil {
		return err
	}
	if err := waitUntil(ctx, client, func(current pageState) bool {
		return current.SelectedNodes == 2 && current.SelectionToolbar
	}); err != nil {
		return fmt.Errorf("multi-select workflow nodes: %w; events: %s", err, finishMarqueeTrace(ctx, client))
	}
	_ = finishMarqueeTrace(ctx, client)
	if err := dispatchKeyPress(ctx, client, "Escape", "Escape", 27); err != nil {
		return err
	}
	if err := waitUntil(ctx, client, func(current pageState) bool {
		return current.SelectedNodes == 0 && !current.SelectionToolbar
	}); err != nil {
		return fmt.Errorf("clear marquee selection before targeted batch delete: %w", err)
	}
	var deletePoints []point
	if err := evalJSON(ctx, client, `(() => {
		const nodes = [...document.querySelectorAll('.vue-flow__node:not(.vue-flow__node-graph-boundary)')]
			.filter(node => node.getAttribute('data-id') !== 'run-started')
			.slice(-2);
		if (nodes.length < 2) throw new Error('batch delete needs two non-root workflow nodes');
		return nodes.map(node => {
			const header = node.querySelector('.workflow-node-drag-handle');
			if (!header) throw new Error('batch delete node header not found');
			const rect = header.getBoundingClientRect();
			return { x: rect.left + 32, y: rect.top + rect.height / 2 };
		});
	})()`, &deletePoints); err != nil {
		return err
	}
	if err := dispatchControlClicks(ctx, client, deletePoints); err != nil {
		return err
	}
	if err := waitUntil(ctx, client, func(current pageState) bool {
		return current.SelectedNodes == 2 && current.SelectionToolbar
	}); err != nil {
		return fmt.Errorf("select non-root workflow nodes for batch delete: %w", err)
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
	wheelBefore, err := readConnectionMenuWheelProbe(ctx, client)
	if err != nil {
		return err
	}
	if err := dispatchMouseWheel(ctx, client, wheelBefore.X, wheelBefore.Y, 320); err != nil {
		return err
	}
	time.Sleep(300 * time.Millisecond)
	wheelAfter, err := readConnectionMenuWheelProbe(ctx, client)
	if err != nil {
		return err
	}
	if wheelAfter.Zoom < wheelBefore.Zoom-0.001 || wheelAfter.Zoom > wheelBefore.Zoom+0.001 {
		return fmt.Errorf("connection candidate wheel zoomed canvas: %.3f -> %.3f", wheelBefore.Zoom, wheelAfter.Zoom)
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
	if err := exerciseDebugger(ctx, client); err != nil {
		return err
	}
	if err := clickRequired(ctx, client, "workflow-state-open"); err != nil {
		return err
	}
	if err := waitUntil(ctx, client, func(current pageState) bool { return current.WorkflowState }); err != nil {
		return fmt.Errorf("open workflow state panel: %w", err)
	}
	if err := exerciseRunState(ctx, client); err != nil {
		return err
	}
	if err := capture(ctx, client, runStateScreenshot); err != nil {
		return err
	}
	if err := clickRequired(ctx, client, "workflow-state-open"); err != nil {
		return err
	}
	if err := exerciseMultigraph(ctx, client, subgraphScreenshot); err != nil {
		return err
	}
	if err := clickRequired(ctx, client, "workflow-save"); err != nil {
		return err
	}
	if err := waitForSave(ctx, client); err != nil {
		return fmt.Errorf("save multigraph workflow: %w", err)
	}
	uiFailures := workflowEditorUIFailures(visualState, confirmState, saveState)
	if err := clickRequired(ctx, client, "ai-workflow-review-open"); err != nil {
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
	if err := exerciseSnippets(ctx, client, nodeMenuScreenshot); err != nil {
		return err
	}
	if err := clickRequired(ctx, client, "workflow-workspace-macro"); err != nil {
		return err
	}
	if err := waitUntil(ctx, client, func(current pageState) bool {
		return current.ResourceDock && current.ResourceKind == "macro" &&
			current.ResourceCreate && current.WorkspaceTools == 5
	}); err != nil {
		return fmt.Errorf("open macro workspace tool: %w", err)
	}
	if err := clickRequired(ctx, client, "workflow-workspace-clip"); err != nil {
		return err
	}
	if err := waitUntil(ctx, client, func(current pageState) bool {
		return current.ResourceDock && current.ResourceKind == "clip" && current.ResourceCreate
	}); err != nil {
		return fmt.Errorf("open precise recording workspace tool: %w", err)
	}
	if err := clickRequired(ctx, client, "workflow-workspace-template"); err != nil {
		return err
	}
	if err := waitUntil(ctx, client, func(current pageState) bool {
		return current.ResourceDock && current.ResourceKind == "template" && current.ResourceCreate &&
			current.ResourceScope == "workflow" && current.ResourceScopeActive == 1 &&
			current.ResourceScopeContrast && current.ResourceModeControls == 0 &&
			current.ResourceFiltersFill && !current.ResourceLoading
	}); err != nil {
		return fmt.Errorf("open visual template workspace tool: %w", err)
	}
	if err := capture(ctx, client, siblingScreenshot(screenshot, "resource-tools-workflow.png")); err != nil {
		return err
	}
	if err := clickRequired(ctx, client, "workflow-resource-scope-library"); err != nil {
		return err
	}
	if err := waitUntil(ctx, client, func(current pageState) bool {
		return current.ResourceDock && current.ResourceKind == "template" &&
			current.ResourceScope == "library" && current.ResourceScopeActive == 1 &&
			current.ResourceScopeContrast && current.ResourceModeControls == 0 &&
			current.ResourceFiltersFill && !current.ResourceLoading
	}); err != nil {
		return fmt.Errorf("switch to local resource library: %w", err)
	}
	if err := capture(ctx, client, siblingScreenshot(screenshot, "resource-tools.png")); err != nil {
		return err
	}
	if err := clickRequired(ctx, client, "workflow-workspace-graphs"); err != nil {
		return err
	}
	if err := waitUntil(ctx, client, func(current pageState) bool {
		return current.GraphManager && !current.ResourceDock
	}); err != nil {
		return fmt.Errorf("restore subgraph management workspace tool: %w", err)
	}
	if err := capture(ctx, client, screenshot); err != nil {
		return err
	}
	if err := eval(ctx, client, `location.hash = '#/assets'`); err != nil {
		return err
	}
	if err := waitUntil(ctx, client, func(current pageState) bool {
		return current.AssetsView && current.AssetsRecording && current.AssetBrowse &&
			current.AssetManageButton && !current.AssetManagement
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
	if err := clickRequired(ctx, client, "asset-manage-button"); err != nil {
		return fmt.Errorf("open asset management mode: %w", err)
	}
	if err := waitUntil(ctx, client, func(current pageState) bool {
		return current.AssetManagement && !current.AssetBrowse
	}); err != nil {
		return fmt.Errorf("verify asset management mode: %w", err)
	}
	if err := capture(ctx, client, assetManagementScreenshot); err != nil {
		return fmt.Errorf("capture asset management mode: %w", err)
	}
	if err := clickRequired(ctx, client, "asset-manage-button"); err != nil {
		return fmt.Errorf("close asset management mode: %w", err)
	}
	if err := waitUntil(ctx, client, func(current pageState) bool {
		return current.AssetBrowse && !current.AssetManagement
	}); err != nil {
		return fmt.Errorf("restore asset browse mode: %w", err)
	}
	workflowHash, err := workflowEditorHash(final.Href)
	if err != nil {
		return err
	}
	workflowID, err := workflowIDFromEditorHash(workflowHash)
	if err != nil {
		return err
	}
	workflowHashJSON, _ := json.Marshal(workflowHash)
	if err := eval(ctx, client, fmt.Sprintf(`location.hash = %s`, workflowHashJSON)); err != nil {
		return err
	}
	if err := waitUntil(ctx, client, func(current pageState) bool {
		return current.RunStarted && current.CanvasNodes > 0 && current.CurrentGraph == "main" &&
			current.GraphCalls == 1 && current.Annotations == 1
	}); err != nil {
		return fmt.Errorf("reopen saved workflow: %w", err)
	}
	if schedulesScreenshot != "" {
		if err := eval(ctx, client, `location.hash = '#/schedules'`); err != nil {
			return err
		}
		if err := waitUntil(ctx, client, func(current pageState) bool {
			return current.SchedulesView && current.ScheduleBrowse && current.ScheduleManageButton &&
				!current.ScheduleManagement && !current.ScheduleEditor
		}); err != nil {
			return fmt.Errorf("open schedules view: %w", err)
		}
		if err := capture(ctx, client, scheduleBrowseScreenshot); err != nil {
			return fmt.Errorf("capture schedule browse mode: %w", err)
		}
		if err := clickRequired(ctx, client, "schedule-manage-button"); err != nil {
			return fmt.Errorf("open schedule management mode: %w", err)
		}
		if err := waitUntil(ctx, client, func(current pageState) bool {
			return current.ScheduleManagement && !current.ScheduleBrowse
		}); err != nil {
			return fmt.Errorf("verify schedule management mode: %w", err)
		}
		if err := capture(ctx, client, scheduleManagementScreenshot); err != nil {
			return fmt.Errorf("capture schedule management mode: %w", err)
		}
		if err := clickRequired(ctx, client, "schedule-manage-button"); err != nil {
			return fmt.Errorf("close schedule management mode: %w", err)
		}
		if err := waitUntil(ctx, client, func(current pageState) bool {
			return current.ScheduleBrowse && !current.ScheduleManagement
		}); err != nil {
			return fmt.Errorf("restore schedule browse mode: %w", err)
		}
		if err := eval(ctx, client, `(() => {
			const button = document.querySelector('[data-testid="schedule-create"]');
			if (!button) throw new Error('new schedule button not found');
			button.click();
		})()`); err != nil {
			return err
		}
		if err := waitUntil(ctx, client, func(current pageState) bool { return current.ScheduleEditor }); err != nil {
			return fmt.Errorf("open schedule editor: %w", err)
		}
		if err := clickRequired(ctx, client, "schedule-add-target"); err != nil {
			return fmt.Errorf("add workflow to schedule: %w", err)
		}
		if err := waitUntil(ctx, client, func(current pageState) bool {
			return len(current.ScheduleEditTargets) == 1 && current.ScheduleEditTargets[0] == workflowID
		}); err != nil {
			return fmt.Errorf("bind workflow to schedule: %w", err)
		}
		if err := clickRequired(ctx, client, "schedule-save"); err != nil {
			return fmt.Errorf("save schedule: %w", err)
		}
		if err := waitUntil(ctx, client, func(current pageState) bool {
			return current.SchedulesView && !current.ScheduleEditor && current.ScheduleRows == 1 &&
				len(current.ScheduleRowTargets) == 1 && current.ScheduleRowTargets[0] == workflowID
		}); err != nil {
			return fmt.Errorf("persist schedule workflow reference: %w", err)
		}
		if err := clickRequired(ctx, client, "schedule-run"); err != nil {
			return fmt.Errorf("run saved schedule: %w", err)
		}
		if err := waitUntil(ctx, client, func(current pageState) bool {
			return len(current.ScheduleRowStatuses) == 1 && current.ScheduleRowStatuses[0] == "queued"
		}); err != nil {
			return fmt.Errorf("verify schedule used the Workflow run path: %w", err)
		}
		if err := eval(ctx, client, `new Promise(resolve => requestAnimationFrame(() => requestAnimationFrame(() => setTimeout(resolve, 200))))`); err != nil {
			return err
		}
		if err := capture(ctx, client, scheduleBrowseScreenshot); err != nil {
			return fmt.Errorf("capture populated schedule browse mode: %w", err)
		}
		if err := clickRequired(ctx, client, "schedule-edit"); err != nil {
			return fmt.Errorf("reopen saved schedule: %w", err)
		}
		if err := waitUntil(ctx, client, func(current pageState) bool {
			return current.ScheduleEditor && len(current.ScheduleEditTargets) == 1 &&
				current.ScheduleEditTargets[0] == workflowID
		}); err != nil {
			return fmt.Errorf("verify reopened schedule reference: %w", err)
		}
		if err := eval(ctx, client, `new Promise(resolve => requestAnimationFrame(() => requestAnimationFrame(() => setTimeout(resolve, 300))))`); err != nil {
			return err
		}
		if err := capture(ctx, client, schedulesScreenshot); err != nil {
			return err
		}
	}
	if err := eval(ctx, client, `location.hash = '#/settings'`); err != nil {
		return err
	}
	if err := waitUntil(ctx, client, func(current pageState) bool {
		return current.SettingsView && current.SettingsGroups == 4
	}); err != nil {
		return fmt.Errorf("open grouped settings center: %w", err)
	}
	if err := capture(ctx, client, settingsScreenshot); err != nil {
		return fmt.Errorf("capture grouped settings center: %w", err)
	}
	if err := eval(ctx, client, `(async () => {
		const { workflowTransport } = await import('/src/app/transport/workflow.ts');
		for (let index = 1; index <= 40; index++) {
			await workflowTransport.createSourceWithMetadata({
				name: 'Scale workflow ' + String(index).padStart(2, '0'),
				description: 'Isolated V4 scale journey',
				category: index % 2 ? 'Scale odd' : 'Scale even',
				tags: ['scale']
			});
		}
		location.hash = '#/workflows';
	})()`); err != nil {
		return fmt.Errorf("seed many-workflow journey: %w", err)
	}
	if err := waitUntilFor(ctx, client, 45*time.Second, func(current pageState) bool {
		return current.WorkflowBrowse && current.WorkflowTotal >= 41 && current.WorkflowRows == 20
	}); err != nil {
		return fmt.Errorf("verify bounded 40+ workflow browse page: %w", err)
	}
	if manyWorkflowsScreenshot != "" {
		if err := capture(ctx, client, manyWorkflowsScreenshot); err != nil {
			return fmt.Errorf("capture many-workflow browse page: %w", err)
		}
	}
	result, _ := json.MarshalIndent(map[string]any{
		"status": "passed", "href": final.Href, "workspaceTools": final.WorkspaceTools,
		"canvasNodes": final.CanvasNodes, "aiReview": final.AIReview, "screenshot": screenshot,
		"assetsScreenshot": assetsScreenshot, "assetManagementScreenshot": assetManagementScreenshot,
		"workflowsScreenshot":          workflowsScreenshot,
		"workflowManagementScreenshot": workflowManagementScreenshot,
		"manyWorkflowsScreenshot":      manyWorkflowsScreenshot,
		"schedulesScreenshot":          schedulesScreenshot,
		"scheduleBrowseScreenshot":     scheduleBrowseScreenshot,
		"scheduleManagementScreenshot": scheduleManagementScreenshot,
		"subgraphScreenshot":           subgraphScreenshot,
		"nodeMenuScreenshot":           nodeMenuScreenshot,
		"quickAddScreenshot":           quickAddScreenshot,
		"runStateScreenshot":           runStateScreenshot,
		"settingsScreenshot":           settingsScreenshot,
	}, "", "  ")
	fmt.Println(string(result))
	return nil
}

func workflowEditorHash(href string) (string, error) {
	const marker = "#/workflows/"
	start := strings.Index(href, marker)
	if start < 0 {
		return "", fmt.Errorf("created workflow identity is unavailable in %q", href)
	}
	hash := href[start+1:]
	if !strings.HasSuffix(hash, "/edit") {
		return "", fmt.Errorf("created workflow editor route is invalid: %q", hash)
	}
	parts := strings.Split(strings.Trim(hash, "/"), "/")
	if len(parts) != 3 || parts[0] != "workflows" || strings.TrimSpace(parts[1]) == "" || parts[2] != "edit" {
		return "", fmt.Errorf("created workflow editor route is invalid: %q", hash)
	}
	return "#" + hash, nil
}

func workflowIDFromEditorHash(hash string) (string, error) {
	parts := strings.Split(strings.Trim(strings.TrimPrefix(hash, "#"), "/"), "/")
	if len(parts) != 3 || parts[0] != "workflows" || parts[1] == "" || parts[2] != "edit" {
		return "", fmt.Errorf("workflow editor route is invalid: %q", hash)
	}
	return parts[1], nil
}
