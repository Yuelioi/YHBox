package main

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/yottaapp/yotta/internal/automation/browsercdp"
)

func exerciseMultigraph(ctx context.Context, client *browsercdp.WebSocketClient, subgraphScreenshot string) error {
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
		return current.CurrentGraph != "" && current.CurrentGraph != "main" && current.CanvasNodes == 0 && current.GraphBoundaries == 1
	}); err != nil {
		return fmt.Errorf("enter new subgraph: %w", err)
	}
	if err := addNodeViaQuickAdd(
		ctx,
		client,
		"delay",
		"https://schemas.yotta.dev/nodes/control/delay",
	); err != nil {
		return fmt.Errorf("author subgraph node: %w", err)
	}
	if err := eval(ctx, client, `(() => {
		const pane = document.querySelector('.vue-flow__pane');
		if (!pane) throw new Error('subgraph canvas pane not found');
		pane.dispatchEvent(new MouseEvent('click', { bubbles: true, cancelable: true, view: window }));
	})()`); err != nil {
		return err
	}
	if err := waitUntil(ctx, client, func(current pageState) bool { return current.GraphInterface }); err != nil {
		return fmt.Errorf("open subgraph interface panel: %w", err)
	}
	if err := eval(ctx, client, `(() => {
		const panel = document.querySelector('[data-testid="workflow-graph-interface"]');
		const infer = document.querySelector('[data-testid="workflow-graph-infer-interface"]');
		if (!panel || !infer || !panel.contains(infer)) {
			throw new Error('interface inference action is not owned by the subgraph interface panel');
		}
	})()`); err != nil {
		return err
	}
	if err := clickRequired(ctx, client, "workflow-graph-infer-interface"); err != nil {
		return err
	}
	if err := waitUntil(ctx, client, func(current pageState) bool {
		return current.ConfirmDialog
	}); err != nil {
		return fmt.Errorf("preview inferred subgraph interface: %w", err)
	}
	if err := clickRequired(ctx, client, "confirm-accept"); err != nil {
		return fmt.Errorf("confirm inferred subgraph interface: %w", err)
	}
	if err := waitUntil(ctx, client, func(current pageState) bool {
		return current.GraphBoundaries >= 2 && current.GraphInterface && current.BoundaryClipped == 0 && current.BoundaryObscured == 0
	}); err != nil {
		return fmt.Errorf("project inferred subgraph interface: %w", err)
	}
	if subgraphScreenshot != "" {
		if err := eval(ctx, client, `new Promise(resolve => requestAnimationFrame(() => requestAnimationFrame(() => setTimeout(resolve, 300))))`); err != nil {
			return err
		}
		if err := capture(ctx, client, subgraphScreenshot); err != nil {
			return fmt.Errorf("capture subgraph authoring surface: %w", err)
		}
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
	if err := clickRequired(ctx, client, "workflow-graph-insert-call"); err != nil {
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
	if err := eval(ctx, client, `(async () => {
		const canvas = document.querySelector('[data-testid="workflow-canvas"]');
		const annotation = document.querySelector('[data-testid="workflow-annotation"]');
		const textarea = annotation?.querySelector('textarea');
		if (!canvas || !annotation || !textarea) throw new Error('annotation editing surface unavailable');
		const canvasRect = canvas.getBoundingClientRect();
		const noteRect = annotation.getBoundingClientRect();
		const expectedX = canvasRect.left + canvasRect.width / 2;
		const expectedY = canvasRect.top + canvasRect.height * 0.38;
		const deltaX = noteRect.left + noteRect.width / 2 - expectedX;
		const deltaY = noteRect.top + noteRect.height / 2 - expectedY;
		if (Math.abs(deltaX) > 32 || Math.abs(deltaY) > 32) {
			throw new Error('toolbar-created annotation missed the upper canvas center by (' + deltaX.toFixed(1) + ', ' + deltaY.toFixed(1) + ')');
		}
		const setter = Object.getOwnPropertyDescriptor(HTMLTextAreaElement.prototype, 'value')?.set;
		setter?.call(textarea, '## Smoke comment\n\nThis is **important**.\n\n- First item\n- Second item\n\nLong content must stay inside the annotation rather than expanding the canvas node.\n\nMore content verifies that the body owns scrolling while the header and resize boundary remain fixed.');
		textarea.dispatchEvent(new Event('input', { bubbles: true }));
		textarea.dispatchEvent(new FocusEvent('blur', { bubbles: true }));
		await new Promise(resolve => requestAnimationFrame(() => requestAnimationFrame(resolve)));
		const content = annotation.querySelector('[data-testid="workflow-annotation-content"]');
		if (!content?.querySelector('h2') || !content.querySelector('strong') || content.querySelectorAll('li').length !== 2) {
			throw new Error('annotation Markdown preview did not render basic structure');
		}
		const articleRect = annotation.getBoundingClientRect();
		const contentRect = content.getBoundingClientRect();
		if (contentRect.bottom > articleRect.bottom + 1 || content.scrollTop !== 0) {
			throw new Error('annotation content escaped its internal top-aligned scroll region');
		}
		if (document.body.textContent?.includes('编辑被拒绝')) throw new Error('annotation edit was rejected');
	})()`); err != nil {
		return fmt.Errorf("edit graph annotation: %w", err)
	}
	return nil
}

func exerciseDebugger(ctx context.Context, client *browsercdp.WebSocketClient) error {
	if err := eval(ctx, client, `(() => {
		const button = document.querySelector('.vue-flow__node[data-id="run-started"] [data-testid="node-breakpoint"]');
		if (!button) throw new Error('RunStarted breakpoint button not found');
		if (Number(getComputedStyle(button).opacity) > 0.01) throw new Error('inactive breakpoint control is visible outside debug context');
		if (document.querySelector('[data-testid="workflow-debug-start"]')) throw new Error('debug start button is visible outside the Tools menu');
	})()`); err != nil {
		return err
	}
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
		return current.DebugPaused && !current.DebugBusy && current.DebugCurrent == 1 && current.DebugNode != "run-started"
	}); err != nil {
		return fmt.Errorf("step from Run start to next visible node: %w", err)
	}
	if err := clickRequired(ctx, client, "workflow-debug-step"); err != nil {
		return err
	}
	if err := waitUntil(ctx, client, func(current pageState) bool {
		return current.DebugCompleted && current.DebugCurrent == 0
	}); err != nil {
		return fmt.Errorf("step next visible node to completion: %w", err)
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
	if err := waitUntil(ctx, client, func(current pageState) bool {
		return current.DebugCompleted
	}); err != nil {
		return fmt.Errorf("stop debug Run: %w", err)
	}
	return nil
}

func exerciseRunState(ctx context.Context, client *browsercdp.WebSocketClient) error {
	if err := eval(ctx, client, `(async () => {
		const waitFor = async (probe, label) => {
			const deadline = performance.now() + 10000;
			while (performance.now() < deadline) {
				const value = probe();
				if (value) return value;
				await new Promise(resolve => requestAnimationFrame(resolve));
			}
			throw new Error('timed out waiting for ' + label);
		};
		const type = await waitFor(
			() => document.querySelector('[data-testid="workflow-state-new-type"]'),
			'Run state type selector',
		);
		type.click();
		const option = await waitFor(
			() => [...document.querySelectorAll('[role="option"]')]
				.find(candidate => /文件元数据|File metadata/.test(candidate.textContent || '')),
			'File metadata state type option',
		);
		option.click();
		await waitFor(
			() => document.querySelector('[data-testid="workflow-state-panel"] textarea')?.value.includes('"mediaType"'),
			'File metadata initial value',
		);
		const name = await waitFor(
			() => document.querySelector(
				'input[data-testid="workflow-state-new-name"], [data-testid="workflow-state-new-name"] input',
			),
			'Run state name input',
		);
		const setter = Object.getOwnPropertyDescriptor(HTMLInputElement.prototype, 'value')?.set;
		if (!setter) throw new Error('native input value setter is unavailable');
		setter.call(name, 'smoke_metadata');
		name.dispatchEvent(new Event('input', { bubbles: true }));
		const add = await waitFor(
			() => document.querySelector(
				'button[data-testid="workflow-state-add"], [data-testid="workflow-state-add"] button',
			),
			'Run state add button',
		);
		await waitFor(() => !add.disabled, 'enabled Run state add button');
		add.click();
		await waitFor(
			() => document.querySelector('[data-testid="workflow-state-variable-smoke_metadata"]'),
			'persisted File metadata state row',
		);
		if (document.querySelector('[data-testid="workflow-state-invalid-json"]')) {
			throw new Error('File metadata state default is reported as invalid JSON');
		}
	})()`); err != nil {
		return fmt.Errorf("exercise File metadata Run state: %w", err)
	}
	return nil
}

func exerciseMinimap(ctx context.Context, client *browsercdp.WebSocketClient) error {
	if err := clickRequired(ctx, client, "workflow-minimap-toggle"); err != nil {
		return err
	}
	if err := waitUntil(ctx, client, func(current pageState) bool {
		return current.MinimapOpen && current.GraphChromeDark
	}); err != nil {
		return fmt.Errorf("open dark workflow minimap: %w", err)
	}
	if err := clickRequired(ctx, client, "workflow-minimap-toggle"); err != nil {
		return err
	}
	if err := waitUntil(ctx, client, func(current pageState) bool { return !current.MinimapOpen }); err != nil {
		return fmt.Errorf("close workflow minimap: %w", err)
	}
	return nil
}

func exerciseCanvasNodeErgonomics(ctx context.Context, client *browsercdp.WebSocketClient, screenshot string) error {
	before, err := state(ctx, client)
	if err != nil {
		return err
	}
	if err := addNodeViaQuickAddAfter(
		ctx,
		client,
		before,
		"analyze-color",
		"https://schemas.yotta.dev/nodes/vision/analyze-color",
	); err != nil {
		return fmt.Errorf("insert Analyze Color node: %w", err)
	}
	probe, err := readCanvasNodeErgonomics(ctx, client)
	if err != nil {
		return err
	}
	if screenshot != "" {
		if err := capture(ctx, client, screenshot); err != nil {
			return fmt.Errorf("capture Analyze Color node: %w", err)
		}
	}
	if err := dispatchMouseClick(ctx, client, probe.BlankX, probe.BlankY); err != nil {
		return err
	}
	time.Sleep(200 * time.Millisecond)
	unselected, err := readCanvasNodeErgonomics(ctx, client)
	if err != nil {
		return err
	}
	var failures []string
	if probe.Width > 320 || probe.Height > 360 {
		failures = append(failures, fmt.Sprintf("oversized: %.0fx%.0f", probe.Width, probe.Height))
	}
	if probe.CompositeInlineEditors != 0 {
		failures = append(failures, fmt.Sprintf("contains %d composite inline editors", probe.CompositeInlineEditors))
	}
	if unselected.Selected {
		failures = append(failures, "blank-canvas click did not clear node selection")
	}
	if err := dispatchMouseWheel(ctx, client, unselected.BlankX, unselected.BlankY, 320); err != nil {
		return err
	}
	time.Sleep(300 * time.Millisecond)
	afterBlankWheel, err := readCanvasNodeErgonomics(ctx, client)
	if err != nil {
		return err
	}
	if afterBlankWheel.Zoom >= unselected.Zoom-0.001 {
		failures = append(failures, fmt.Sprintf("blank-canvas wheel did not zoom: %.3f -> %.3f", unselected.Zoom, afterBlankWheel.Zoom))
	}
	if err := dispatchMouseWheel(ctx, client, afterBlankWheel.CenterX, afterBlankWheel.CenterY, 320); err != nil {
		return err
	}
	time.Sleep(300 * time.Millisecond)
	afterNodeWheel, err := readCanvasNodeErgonomics(ctx, client)
	if err != nil {
		return err
	}
	if afterNodeWheel.Zoom >= afterBlankWheel.Zoom-0.001 {
		failures = append(failures, fmt.Sprintf("unselected-node wheel did not zoom: %.3f -> %.3f", afterBlankWheel.Zoom, afterNodeWheel.Zoom))
	}
	if len(failures) > 0 {
		return fmt.Errorf("analyze color canvas node %s", strings.Join(failures, "; "))
	}
	if err := dispatchMouseClick(ctx, client, afterNodeWheel.CenterX, afterNodeWheel.CenterY); err != nil {
		return err
	}
	if err := dispatchKeyPress(ctx, client, "Delete", "Delete", 46); err != nil {
		return err
	}
	if err := waitUntil(ctx, client, func(current pageState) bool {
		return current.CanvasNodes == before.CanvasNodes
	}); err != nil {
		return fmt.Errorf("remove Analyze Color smoke node: %w", err)
	}
	return nil
}

func addNodeViaQuickAdd(
	ctx context.Context,
	client *browsercdp.WebSocketClient,
	query string,
	nodeTypeID string,
) error {
	before, err := state(ctx, client)
	if err != nil {
		return err
	}
	return addNodeViaQuickAddAfter(ctx, client, before, query, nodeTypeID)
}

func addNodeViaQuickAddAfter(
	ctx context.Context,
	client *browsercdp.WebSocketClient,
	before pageState,
	query string,
	nodeTypeID string,
) error {
	queryJSON, _ := json.Marshal(query)
	nodeTypeJSON, _ := json.Marshal(nodeTypeID)
	if err := eval(ctx, client, `(async () => {
		let trigger = document.querySelector('[data-testid="workflow-canvas-add-node"]');
		if (!trigger) {
			document.querySelector('[data-testid="workflow-canvas-assist-compact"]')?.click();
			const deadline = performance.now() + 3000;
			while (!trigger && performance.now() < deadline) {
				await new Promise(resolve => setTimeout(resolve, 25));
				trigger = document.querySelector('[data-testid="workflow-canvas-add-node"]');
			}
		}
		if (!trigger) throw new Error('explicit add node trigger not found');
		trigger.click();
	})()`); err != nil {
		return err
	}
	if err := waitUntilJS(ctx, client, 5*time.Second, `(() => {
		/* quick add search ready */
		return Boolean(document.querySelector(
			'[data-testid="workflow-quick-add-search"] input, input[data-testid="workflow-quick-add-search"]'
		));
	})()`); err != nil {
		return fmt.Errorf("wait for quick-add search: %w", err)
	}
	if err := eval(ctx, client, `(() => {
		const trigger = document.querySelector('[data-testid="workflow-canvas-assist"]');
		const panel = document.querySelector('[data-testid="workflow-quick-add"]');
		if (!trigger || !panel) throw new Error('explicit quick-add geometry unavailable');
		const triggerRect = trigger.getBoundingClientRect();
		const panelRect = panel.getBoundingClientRect();
		if (Math.abs(panelRect.left - triggerRect.right - 4) > 3 ||
			Math.abs(panelRect.top - triggerRect.top) > 3) {
			throw new Error('explicit quick add did not open beside the canvas assist toolbar');
		}
	})()`); err != nil {
		return err
	}
	if err := eval(ctx, client, fmt.Sprintf(`(() => {
		const input = document.querySelector(
			'[data-testid="workflow-quick-add-search"] input, input[data-testid="workflow-quick-add-search"]'
		);
		if (!input) throw new Error('node quick-add search unavailable');
		const setter = Object.getOwnPropertyDescriptor(HTMLInputElement.prototype, 'value').set;
		setter.call(input, %s);
		input.dispatchEvent(new InputEvent('input', { bubbles: true, inputType: 'insertText' }));
	})()`, queryJSON)); err != nil {
		return err
	}
	if err := waitUntilJS(ctx, client, 5*time.Second, fmt.Sprintf(`(() => {
		/* quick add item ready */
		return [...document.querySelectorAll('[data-testid="workflow-quick-add-item"]')]
			.some(candidate => candidate.getAttribute('data-item-id') === %s);
	})()`, nodeTypeJSON)); err != nil {
		return fmt.Errorf("wait for quick-add item %s: %w", nodeTypeID, err)
	}
	if err := eval(ctx, client, fmt.Sprintf(`(() => {
		const item = [...document.querySelectorAll('[data-testid="workflow-quick-add-item"]')]
			.find(candidate => candidate.getAttribute('data-item-id') === %s);
		if (!item) throw new Error('node quick-add item unavailable');
		item.click();
	})()`, nodeTypeJSON)); err != nil {
		return err
	}
	if err := waitUntil(ctx, client, func(current pageState) bool {
		return current.CanvasNodes == before.CanvasNodes+1
	}); err != nil {
		return fmt.Errorf("wait for quick-added node %s: %w", nodeTypeID, err)
	}
	return nil
}

func exerciseQuickAdd(ctx context.Context, client *browsercdp.WebSocketClient, screenshot string) error {
	before, err := state(ctx, client)
	if err != nil {
		return err
	}
	if err := eval(ctx, client, `(async () => {
		const wait = async predicate => {
			const deadline = performance.now() + 5000;
			while (performance.now() < deadline) {
				const value = predicate();
				if (value) return value;
				await new Promise(resolve => setTimeout(resolve, 25));
			}
			throw new Error('quick add did not become ready');
		};
		const canvas = document.querySelector('[data-testid="workflow-canvas"]');
		if (!canvas) throw new Error('workflow canvas not found for quick add');
		const rect = canvas.getBoundingClientRect();
		const point = { clientX: rect.left + rect.width * 0.58, clientY: rect.top + rect.height * 0.48 };
		canvas.dispatchEvent(new PointerEvent('pointerenter', { bubbles: true, pointerType: 'mouse', ...point }));
		canvas.dispatchEvent(new PointerEvent('pointermove', { bubbles: true, pointerType: 'mouse', ...point }));
		document.dispatchEvent(new KeyboardEvent('keydown', { key: 'Tab', code: 'Tab', bubbles: true, cancelable: true }));
		await wait(() => document.querySelector('[data-testid="workflow-quick-add-search"] input, input[data-testid="workflow-quick-add-search"]'));
		const input = document.querySelector('[data-testid="workflow-quick-add-search"] input, input[data-testid="workflow-quick-add-search"]');
		const setter = Object.getOwnPropertyDescriptor(HTMLInputElement.prototype, 'value').set;
		setter.call(input, 'click-template');
		input.dispatchEvent(new InputEvent('input', { bubbles: true, inputType: 'insertText' }));
		await wait(() => [...document.querySelectorAll('[data-testid="workflow-quick-add-item"]')]
			.find(item => item.textContent.includes('点击模板') || item.textContent.includes('Click Template')));
	})()`); err != nil {
		return fmt.Errorf("open and search workflow quick add: %w", err)
	}
	if screenshot != "" {
		if err := capture(ctx, client, screenshot); err != nil {
			return fmt.Errorf("capture workflow quick add: %w", err)
		}
	}
	if err := dispatchKeyPress(ctx, client, "Enter", "Enter", 13); err != nil {
		return fmt.Errorf("choose quick-add result: %w", err)
	}
	time.Sleep(500 * time.Millisecond)
	if screenshot != "" {
		if err := capture(ctx, client, siblingScreenshot(screenshot, "quick-add-result.png")); err != nil {
			return fmt.Errorf("capture selected quick-add result: %w", err)
		}
	}
	if err := waitUntil(ctx, client, func(current pageState) bool {
		return current.CanvasNodes == before.CanvasNodes+1
	}); err != nil {
		return fmt.Errorf("insert quick-add node: %w", err)
	}
	if err := eval(ctx, client, `(() => {
		const inserted = document.querySelector('.workflow-node[data-node-type-id="https://schemas.yotta.dev/nodes/automation/click-template"]');
		if (!inserted) throw new Error('quick-added click template node not found');
	})()`); err != nil {
		return err
	}
	var point struct {
		X float64 `json:"x"`
		Y float64 `json:"y"`
	}
	if err := evalJSON(ctx, client, `(() => {
		const inserted = document.querySelector('.workflow-node[data-node-type-id="https://schemas.yotta.dev/nodes/automation/click-template"]');
		if (!inserted) throw new Error('quick-added click template node not found for cleanup');
		const rect = inserted.getBoundingClientRect();
		return { x: rect.left + rect.width / 2, y: rect.top + rect.height / 2 };
	})()`, &point); err != nil {
		return fmt.Errorf("locate quick-added node: %w", err)
	}
	if err := dispatchMouseClick(ctx, client, point.X, point.Y); err != nil {
		return fmt.Errorf("select quick-added node: %w", err)
	}
	if err := waitUntil(ctx, client, func(current pageState) bool {
		return current.SelectedNodes == 1
	}); err != nil {
		return fmt.Errorf("wait for quick-added node selection: %w", err)
	}
	if err := dispatchKeyPress(ctx, client, "Delete", "Delete", 46); err != nil {
		return fmt.Errorf("delete quick-added node: %w", err)
	}
	if err := waitUntil(ctx, client, func(current pageState) bool {
		return current.CanvasNodes == before.CanvasNodes
	}); err != nil {
		return fmt.Errorf("remove quick-added node: %w", err)
	}
	return nil
}

func verifyEditorToolsAlignment(ctx context.Context, client *browsercdp.WebSocketClient) error {
	return eval(ctx, client, `(async () => {
		const trigger = document.querySelector('[data-testid="workflow-editor-tools"]');
		if (!trigger) throw new Error('editor tools trigger not found');
		trigger.click();
		const deadline = performance.now() + 3000;
		let item = null;
		while (performance.now() < deadline) {
			item = document.querySelector('[data-testid="workflow-inspector-toggle"]');
			if (item) break;
			await new Promise(resolve => setTimeout(resolve, 25));
		}
		if (!item) throw new Error('editor tools menu did not open');
		if (getComputedStyle(item).textAlign !== 'left') {
			throw new Error('editor tools menu text is not left aligned');
		}
		trigger.click();
	})()`)
}

func verifyEditorToolbarConsolidation(ctx context.Context, client *browsercdp.WebSocketClient) error {
	return eval(ctx, client, `(async () => {
		const toolbar = document.querySelector('[data-testid="workflow-editor-toolbar"]');
		if (!toolbar) throw new Error('editor toolbar not found');
		const requiredSelectors = [
			'[data-testid="workflow-graph-breadcrumb-main"]',
			'[data-testid="workflow-find-node"]',
			'[data-testid="workflow-run-timeline"]',
			'[data-testid="workflow-save"]',
			'[data-testid="workflow-editor-tools"]'
		];
		for (const selector of requiredSelectors) {
			const item = document.querySelector(selector);
			if (!item || !toolbar.contains(item)) {
				throw new Error('editor command is outside the consolidated toolbar: ' + selector);
			}
		}
		const addNode = document.querySelector('[data-testid="workflow-canvas-add-node"]');
		const assist = document.querySelector('[data-testid="workflow-canvas-assist"]');
		if (!addNode || !assist?.contains(addNode) || toolbar.contains(addNode) || !assist.querySelector('[data-testid="workflow-layout-lr"]')) {
			throw new Error('canvas creation actions are not isolated from the editor toolbar');
		}
		const contextActions = document.querySelector('[data-testid="workflow-canvas-context-actions"]');
		const target = document.querySelector('[data-testid="workflow-target-default"]');
		const ai = document.querySelector('[data-testid="workflow-canvas-ai"]');
		const assistVisibility = document.querySelector('[data-testid="workflow-canvas-assist-visibility"]');
		if (!contextActions || !target || !ai || !assistVisibility || !contextActions.contains(target) || !contextActions.contains(ai) || !contextActions.contains(assistVisibility)) {
			throw new Error('AI proposal and target actions are not grouped at the canvas upper-right');
		}
		if (document.querySelector('[data-testid="workflow-graph-infer-interface"]')) {
			throw new Error('subgraph interface inference leaked into the main graph toolbar');
		}
		const editing = document.querySelector('[data-testid="workflow-editor-editing"]');
		const actions = document.querySelector('[data-testid="workflow-editor-actions"]');
		if (!editing || !actions || editing.compareDocumentPosition(actions) !== Node.DOCUMENT_POSITION_FOLLOWING) {
			throw new Error('editing commands are not placed on the left of workflow actions');
		}
		const tools = document.querySelector('[data-testid="workflow-editor-tools"]');
		if (!tools) throw new Error('editor tools action is unavailable');
		const previousStyle = toolbar.getAttribute('style');
		toolbar.style.width = '900px';
		toolbar.style.maxWidth = '900px';
		toolbar.style.flex = '0 0 900px';
		await new Promise(resolve => requestAnimationFrame(() => requestAnimationFrame(resolve)));
		const toolbarRect = toolbar.getBoundingClientRect();
		for (const selector of requiredSelectors.slice(1)) {
			const item = document.querySelector(selector);
			const rect = item?.getBoundingClientRect();
			if (!rect || rect.width <= 0 || rect.height <= 0 ||
				rect.left < toolbarRect.left - 1 || rect.right > toolbarRect.right + 1) {
				throw new Error('editor command left the narrow toolbar viewport: ' + selector);
			}
		}
		if (previousStyle === null) toolbar.removeAttribute('style');
		else toolbar.setAttribute('style', previousStyle);
	})()`)
}

func exerciseSnippets(ctx context.Context, client *browsercdp.WebSocketClient, nodeMenuScreenshot string) error {
	before, err := state(ctx, client)
	if err != nil {
		return err
	}
	if before.RecipeItems != 0 {
		return fmt.Errorf("node catalog still exposes %d hardcoded recipes", before.RecipeItems)
	}
	if err := eval(ctx, client, `(() => {
		const node = document.querySelector('.workflow-node[data-node-type-id="https://schemas.yotta.dev/nodes/control/delay"]');
		if (!node) throw new Error('configured Delay node not found for snippet save');
		node.dispatchEvent(new MouseEvent('contextmenu', { bubbles: true, cancelable: true, view: window }));
	})()`); err != nil {
		return err
	}
	if err := waitUntil(ctx, client, func(current pageState) bool {
		return current.NodeContextMenu && !current.SnippetModal
	}); err != nil {
		return fmt.Errorf("open node context menu without executing snippet action: %w", err)
	}
	if err := capture(ctx, client, nodeMenuScreenshot); err != nil {
		return err
	}
	if err := eval(ctx, client, `(() => {
		const action = document.querySelector('[data-testid="workflow-node-menu-save-snippet"]');
		if (!action) throw new Error('save snippet context action not found');
		action.click();
	})()`); err != nil {
		return err
	}
	if err := waitUntil(ctx, client, func(current pageState) bool { return current.SnippetModal }); err != nil {
		return fmt.Errorf("open save-as-snippet dialog: %w", err)
	}
	if err := eval(ctx, client, `(() => {
		const capture = document.querySelector('[data-testid="workflow-snippet-shortcut"] button');
		if (!capture) throw new Error('snippet shortcut capture not found');
		capture.click();
	})()`); err != nil {
		return fmt.Errorf("activate current snippet shortcut capture: %w", err)
	}
	if err := eval(ctx, client, `(async () => {
		const deadline = performance.now() + 15000;
		while (performance.now() < deadline) {
			const capture = document.querySelector('[data-testid="workflow-snippet-shortcut"] button');
			if (capture && document.activeElement === capture) return;
			await new Promise(resolve => setTimeout(resolve, 25));
		}
		throw new Error('snippet shortcut capture did not receive focus');
	})()`); err != nil {
		return err
	}
	if err := dispatchModifiedKeyPress(ctx, client, "K", "KeyK", 75, 2|8); err != nil {
		return fmt.Errorf("capture snippet shortcut: %w", err)
	}
	if err := eval(ctx, client, `(async () => {
		const deadline = performance.now() + 5000;
		while (performance.now() < deadline) {
			const capture = document.querySelector('[data-testid="workflow-snippet-shortcut"] button');
			if (capture?.textContent?.includes('Ctrl+Shift+K')) return;
			await new Promise(resolve => setTimeout(resolve, 25));
		}
		throw new Error('snippet shortcut capture did not persist the chord');
	})()`); err != nil {
		return err
	}
	if err := eval(ctx, client, `(() => {
		const input = document.querySelector('[data-testid="workflow-snippet-name"] input, input[data-testid="workflow-snippet-name"]');
		if (!input) throw new Error('snippet name input not found');
		const setter = Object.getOwnPropertyDescriptor(HTMLInputElement.prototype, 'value').set;
		setter.call(input, 'Smoke configured delay');
		input.dispatchEvent(new InputEvent('input', { bubbles: true, inputType: 'insertText' }));
		const save = document.querySelector('[data-testid="workflow-snippet-save"]');
		if (!save || save.disabled) throw new Error('snippet save button unavailable');
		save.click();
	})()`); err != nil {
		return err
	}
	if err := waitUntil(ctx, client, func(current pageState) bool {
		return !current.SnippetModal && current.SnippetDock && current.SnippetItems > 0
	}); err != nil {
		return fmt.Errorf("persist snippet and open snippet dock: %w", err)
	}
	saved, err := state(ctx, client)
	if err != nil {
		return err
	}
	if err := eval(ctx, client, `(() => {
		const canvas = document.querySelector('[data-testid="workflow-canvas"]');
		if (!canvas) throw new Error('workflow canvas not found for snippet shortcut');
		const rect = canvas.getBoundingClientRect();
		const point = { clientX: rect.left + rect.width * 0.55, clientY: rect.top + rect.height * 0.45 };
		canvas.dispatchEvent(new PointerEvent('pointerenter', { bubbles: true, pointerType: 'mouse', ...point }));
		canvas.dispatchEvent(new PointerEvent('pointermove', { bubbles: true, pointerType: 'mouse', ...point }));
	})()`); err != nil {
		return err
	}
	if err := dispatchModifiedKeyPress(ctx, client, "K", "KeyK", 75, 2|8); err != nil {
		return fmt.Errorf("insert snippet with shortcut: %w", err)
	}
	if err := waitUntil(ctx, client, func(current pageState) bool {
		return current.CanvasNodes == saved.CanvasNodes+1 && current.SelectedNodes == 1
	}); err != nil {
		return fmt.Errorf("insert snippet at viewport center: %w", err)
	}
	if err := dispatchKeyPress(ctx, client, "Delete", "Delete", 46); err != nil {
		return err
	}
	if err := waitUntil(ctx, client, func(current pageState) bool {
		return current.CanvasNodes == saved.CanvasNodes
	}); err != nil {
		return fmt.Errorf("remove inserted snippet node: %w", err)
	}
	if err := clickRequired(ctx, client, "workflow-save"); err != nil {
		return err
	}
	if err := waitForSave(ctx, client); err != nil {
		return fmt.Errorf("save workflow after snippet journey: %w", err)
	}
	if err := eval(ctx, client, `(() => {
		const item = [...document.querySelectorAll('[data-testid="workflow-snippet-item"]')]
			.find(candidate => candidate.textContent.includes('Smoke configured delay'));
		const remove = item?.querySelector('[data-testid="workflow-snippet-delete"]');
		if (!remove) throw new Error('saved snippet delete action not found');
		remove.click();
	})()`); err != nil {
		return err
	}
	if err := waitUntil(ctx, client, func(current pageState) bool { return current.ConfirmDialog }); err != nil {
		return fmt.Errorf("confirm snippet deletion: %w", err)
	}
	if err := clickRequired(ctx, client, "confirm-accept"); err != nil {
		return err
	}
	if err := waitUntil(ctx, client, func(current pageState) bool {
		return current.SnippetItems == saved.SnippetItems-1 && !current.ConfirmDialog
	}); err != nil {
		return fmt.Errorf("delete snippet: %w", err)
	}
	return nil
}
