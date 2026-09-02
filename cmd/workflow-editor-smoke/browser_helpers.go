package main

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/yottaapp/yotta/internal/automation/browsercdp"
)

func siblingScreenshot(path, name string) string {
	if path == "" {
		return ""
	}
	return filepath.Join(filepath.Dir(path), name)
}

func dispatchMouseWheel(ctx context.Context, client *browsercdp.WebSocketClient, x, y, deltaY float64) error {
	_, err := client.Call(ctx, "Input.dispatchMouseEvent", map[string]any{
		"type": "mouseWheel", "x": x, "y": y, "deltaX": 0, "deltaY": deltaY,
	})
	return err
}

func dispatchMouseClick(ctx context.Context, client *browsercdp.WebSocketClient, x, y float64) error {
	return dispatchModifiedMouseClick(ctx, client, x, y, 0)
}

func dispatchModifiedMouseClick(ctx context.Context, client *browsercdp.WebSocketClient, x, y float64, modifiers int) error {
	if _, err := client.Call(ctx, "Input.dispatchMouseEvent", map[string]any{
		"type": "mousePressed", "x": x, "y": y,
		"button": "left", "buttons": 1, "clickCount": 1, "modifiers": modifiers,
	}); err != nil {
		return err
	}
	_, err := client.Call(ctx, "Input.dispatchMouseEvent", map[string]any{
		"type": "mouseReleased", "x": x, "y": y,
		"button": "left", "buttons": 0, "clickCount": 1, "modifiers": modifiers,
	})
	return err
}

func dispatchControlClicks(ctx context.Context, client *browsercdp.WebSocketClient, points []point) error {
	const controlModifier = 2
	if _, err := client.Call(ctx, "Input.dispatchKeyEvent", map[string]any{
		"type": "rawKeyDown", "key": "Control", "code": "ControlLeft",
		"windowsVirtualKeyCode": 17, "nativeVirtualKeyCode": 17, "modifiers": controlModifier,
	}); err != nil {
		return err
	}
	var clickErr error
	for _, point := range points {
		if clickErr = dispatchModifiedMouseClick(ctx, client, point.X, point.Y, controlModifier); clickErr != nil {
			break
		}
	}
	_, releaseErr := client.Call(ctx, "Input.dispatchKeyEvent", map[string]any{
		"type": "keyUp", "key": "Control", "code": "ControlLeft",
		"windowsVirtualKeyCode": 17, "nativeVirtualKeyCode": 17,
	})
	if clickErr != nil {
		return clickErr
	}
	return releaseErr
}

func readCanvasNodeErgonomics(ctx context.Context, client *browsercdp.WebSocketClient) (canvasNodeErgonomics, error) {
	var probe canvasNodeErgonomics
	err := evalJSON(ctx, client, `(() => {
		const marker = 'Analyze Color node ergonomics probe';
		const node = document.querySelector('.workflow-node[data-node-type-id="https://schemas.yotta.dev/nodes/vision/analyze-color"]');
		const viewport = document.querySelector('.vue-flow__transformationpane');
		const canvas = document.querySelector('[data-testid="workflow-canvas"]');
		if (!node || !viewport || !canvas) throw new Error(marker + ' unavailable');
		const rect = node.getBoundingClientRect();
		const canvasRect = canvas.getBoundingClientRect();
		let blank = null;
		for (let row = 1; row < 9 && !blank; row++) {
			for (let column = 1; column < 9; column++) {
				const x = canvasRect.left + canvasRect.width * column / 9;
				const y = canvasRect.top + canvasRect.height * row / 9;
				const hit = document.elementFromPoint(x, y);
				if (hit && canvas.contains(hit) && !hit.closest('.vue-flow__node, .vue-flow__controls, .vue-flow__minimap')) {
					blank = { x, y };
					break;
				}
			}
		}
		if (!blank) throw new Error(marker + ' blank point unavailable');
		const transform = getComputedStyle(viewport).transform;
		const zoom = transform && transform !== 'none' ? new DOMMatrixReadOnly(transform).a : 1;
		const compositeInlineEditors = node.querySelectorAll('[data-inline-adapter="color-range"], [data-inline-adapter="point"], [data-inline-adapter="region"], [data-inline-adapter="json"]').length;
		return { centerX: rect.left + rect.width / 2, centerY: rect.top + rect.height / 2, blankX: blank.x, blankY: blank.y, width: node.offsetWidth, height: node.offsetHeight, zoom, selected: Boolean(node.closest('.vue-flow__node')?.classList.contains('selected')), compositeInlineEditors };
	})()`, &probe)
	if err != nil {
		return canvasNodeErgonomics{}, fmt.Errorf("inspect Analyze Color node ergonomics: %w", err)
	}
	return probe, nil
}

func readConnectionMenuWheelProbe(ctx context.Context, client *browsercdp.WebSocketClient) (wheelOwnershipProbe, error) {
	var probe wheelOwnershipProbe
	err := evalJSON(ctx, client, `(() => {
		const menu = document.querySelector('[data-testid="workflow-connection-menu"]');
		const viewport = document.querySelector('.vue-flow__transformationpane');
		if (!menu || !viewport) throw new Error('connection menu wheel probe unavailable');
		const rect = menu.getBoundingClientRect();
		const transform = getComputedStyle(viewport).transform;
		const zoom = transform && transform !== 'none' ? new DOMMatrixReadOnly(transform).a : 1;
		return { x: rect.left + rect.width / 2, y: rect.top + rect.height / 2, zoom };
	})()`, &probe)
	if err != nil {
		return wheelOwnershipProbe{}, fmt.Errorf("inspect connection menu wheel ownership: %w", err)
	}
	return probe, nil
}

func clickRequired(ctx context.Context, client *browsercdp.WebSocketClient, testID string) error {
	testIDJSON, _ := json.Marshal(testID)
	return eval(ctx, client, fmt.Sprintf(`(async () => {
		const selector = '[data-testid=' + %s + ']';
		let button = document.querySelector(selector);
		if (!button) {
			const toolbarItems = new Set([
				'workflow-state-open',
				'ai-workflow-review-open',
				'workflow-inspector-toggle',
				'workflow-check',
				'workflow-diagnostics-open',
				'workflow-timeline-open',
				'workflow-debug-start',
				'workflow-settings',
				'workflow-reload',
			]);
			const triggerID = toolbarItems.has(%s)
				? 'workflow-editor-tools'
				: '';
			const trigger = triggerID
				? document.querySelector('[data-testid="' + triggerID + '"]')
				: null;
			if (trigger) {
				trigger.click();
				await new Promise(resolve => setTimeout(resolve, 100));
				button = document.querySelector(selector);
			}
		}
		if (!button) throw new Error(%s + ' button not found');
		if (button.disabled) throw new Error(%s + ' button is disabled');
		button.click();
	})()`, testIDJSON, testIDJSON, testIDJSON, testIDJSON))
}

func workflowEditorUIFailures(visualState, confirmState, saveState pageState) []string {
	var failures []string
	if !visualState.GraphChromeDark {
		failures = append(failures, "Vue Flow controls or minimap use a light background")
	}
	if !visualState.MinimapToggle {
		failures = append(failures, "workflow canvas omitted the minimap toggle")
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
	return waitUntilFor(ctx, client, 15*time.Second, predicate)
}

func waitUntilJS(
	ctx context.Context,
	client *browsercdp.WebSocketClient,
	timeout time.Duration,
	expression string,
) error {
	waitCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	for {
		var ready bool
		if err := evalJSON(waitCtx, client, expression, &ready); err != nil {
			return err
		}
		if ready {
			return nil
		}
		select {
		case <-waitCtx.Done():
			return waitCtx.Err()
		case <-time.After(50 * time.Millisecond):
		}
	}
}

func waitUntilFor(
	ctx context.Context,
	client *browsercdp.WebSocketClient,
	timeout time.Duration,
	predicate func(pageState) bool,
) error {
	waitCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	for {
		current, err := state(waitCtx, client)
		if err != nil {
			return err
		}
		if predicate(current) {
			return nil
		}
		select {
		case <-waitCtx.Done():
			details, _ := json.Marshal(current)
			return fmt.Errorf("%w; last page state: %s", waitCtx.Err(), details)
		case <-time.After(100 * time.Millisecond):
		}
	}
}

func waitForStableNodeGeometry(ctx context.Context, client *browsercdp.WebSocketClient) error {
	return eval(ctx, client, `(async () => {
		const deadline = performance.now() + 5000;
		const stableWindow = 250;
		let previous = '';
		let stableSince = performance.now();
		while (performance.now() < deadline) {
			const geometry = [...document.querySelectorAll(
				'.vue-flow__node:not(.vue-flow__node-graph-boundary)'
			)].map(node => {
				const rect = node.getBoundingClientRect();
				return [
					node.getAttribute('data-id') || '',
					rect.left.toFixed(2),
					rect.top.toFixed(2),
					rect.right.toFixed(2),
					rect.bottom.toFixed(2),
				].join(':');
			}).join('|');
			const now = performance.now();
			if (geometry && geometry === previous) {
				if (now - stableSince >= stableWindow) return;
			} else {
				previous = geometry;
				stableSince = now;
			}
			await new Promise(resolve => setTimeout(resolve, 50));
		}
		throw new Error('workflow node geometry did not remain stable for 250ms');
	})()`)
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
	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

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
			var lastObservation struct {
				ClassName string `json:"className"`
				Issue     string `json:"issue"`
				Status    string `json:"status"`
			}
			var lastObservationErr error
			for {
				var succeeded bool
				if err := evalJSON(ctx, launcherClient, `Boolean(document.querySelector('.launcher-command--success'))`, &succeeded); err == nil && succeeded {
					break
				}
				lastObservationErr = evalJSON(ctx, launcherClient, `(() => {
					const command = document.querySelector('.launcher-command');
					return {
						className: command?.className || '',
						issue: document.querySelector('.launcher-health--error')?.textContent?.trim() || '',
						status: command?.querySelector('.launcher-command__status')?.textContent?.trim() || '',
					};
				})()`, &lastObservation)
				select {
				case <-ctx.Done():
					launcherClient.Close()
					return fmt.Errorf(
						"wait for floating launcher workflow success: %w; state=%+v; inspect=%v",
						ctx.Err(), lastObservation, lastObservationErr,
					)
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
		const resourceDock = document.querySelector('[data-testid="workflow-resource-dock"]');
		const resourceScopeButtons = [...document.querySelectorAll('[data-testid^="workflow-resource-scope-"]')];
		const activeResourceScope = resourceScopeButtons.find(button => button.getAttribute('data-active') === 'true');
		const inactiveResourceScope = resourceScopeButtons.find(button => button.getAttribute('data-active') !== 'true');
		const activeResourceScopeStyle = activeResourceScope ? getComputedStyle(activeResourceScope) : null;
		const inactiveResourceScopeStyle = inactiveResourceScope ? getComputedStyle(inactiveResourceScope) : null;
		const resourceFilterRow = document.querySelector('[data-testid="workflow-resource-filter-row"]');
		const resourceFilterControls = [
			document.querySelector('[data-testid="workflow-resource-filter-category"]'),
			document.querySelector('[data-testid="workflow-resource-filter-sort"]')
		].filter(Boolean);
		const resourceFilterRect = resourceFilterRow?.getBoundingClientRect();
		const resourceFilterWidths = resourceFilterControls.map(control => control.getBoundingClientRect().width);
		const nodeRects = [...document.querySelectorAll('.vue-flow__node')].map(node => node.getBoundingClientRect());
		const canvasRect = document.querySelector('[data-testid="workflow-canvas"]')?.getBoundingClientRect();
		const boundaryRects = [...document.querySelectorAll('[data-testid="workflow-graph-boundary"]')].map(node => node.closest('.vue-flow__node')?.getBoundingClientRect()).filter(Boolean);
		const chromeRects = [controls, minimap].filter(Boolean).map(element => element.getBoundingClientRect());
		const intersects = (left, right) => left.left < right.right && left.right > right.left && left.top < right.bottom && left.bottom > right.top;
		let nodeOverlaps = 0;
		for (let left = 0; left < nodeRects.length; left++) {
			for (let right = left + 1; right < nodeRects.length; right++) {
				const a = nodeRects[left], b = nodeRects[right];
				if (a.left < b.right && a.right > b.left && a.top < b.bottom && a.bottom > b.top) nodeOverlaps++;
			}
		}
		return {
		href: location.href,
		nodeAddTrigger: Boolean(document.querySelector('[data-testid="workflow-canvas-add-node"]')),
		workspaceTools: document.querySelectorAll('nav button[data-testid^="workflow-workspace-"]').length,
		graphManager: Boolean(document.querySelector('[data-testid="workflow-graph-manager"]')),
		canvasNodes: document.querySelectorAll('.vue-flow__node:not(.vue-flow__node-graph-boundary)').length,
		canvasEdges: document.querySelectorAll('.vue-flow__edge').length,
		aiReview: Boolean(document.querySelector('[data-testid="ai-workflow-review-panel"]')),
		workflowState: Boolean(document.querySelector('[data-testid="workflow-state-panel"]')),
		resourceDock: Boolean(resourceDock),
		resourceKind: resourceDock?.getAttribute('data-resource-kind') || '',
		resourceCreate: Boolean(document.querySelector('[data-testid="workflow-resource-create"]')),
		resourceScope: resourceDock?.getAttribute('data-resource-scope') || '',
		resourceScopeActive: resourceScopeButtons.filter(button => button.getAttribute('data-active') === 'true').length,
		resourceScopeContrast: Boolean(
			activeResourceScopeStyle && inactiveResourceScopeStyle &&
			(activeResourceScopeStyle.backgroundColor !== inactiveResourceScopeStyle.backgroundColor ||
				activeResourceScopeStyle.color !== inactiveResourceScopeStyle.color ||
				activeResourceScopeStyle.boxShadow !== inactiveResourceScopeStyle.boxShadow)
		),
		resourceModeControls: document.querySelectorAll('[data-testid^="workflow-resource-mode-"]').length,
		resourceFiltersFill: Boolean(
			resourceFilterRect && resourceFilterWidths.length === 2 &&
			Math.abs(resourceFilterWidths[0] - resourceFilterWidths[1]) <= 2 &&
			resourceFilterWidths[0] + resourceFilterWidths[1] >= resourceFilterRect.width - 10
		),
		resourceLoading: Boolean(document.querySelector('[data-testid="workflow-resource-loading"]')),
		recipeItems: document.querySelectorAll('[data-testid="workflow-recipe-item"]').length,
		snippetDock: Boolean(document.querySelector('[data-testid="workflow-snippet-dock"]')),
		snippetItems: document.querySelectorAll('[data-testid="workflow-snippet-item"]').length,
		snippetModal: Boolean(document.querySelector('[data-testid="workflow-snippet-name"]')),
		nodeContextMenu: Boolean(document.querySelector('[data-testid="workflow-node-context-menu"]')),
		templateMenuActions: document.querySelectorAll('[data-testid="workflow-node-menu-choose-template"], [data-testid="workflow-node-menu-capture-template"]').length,
		runStarted: Boolean(document.querySelector('.vue-flow__node[data-id="run-started"]')),
		assetsView: Boolean(document.querySelector('[data-testid="assets-view"]')),
		assetsRecording: Boolean(document.querySelector('[data-testid="assets-recording-start"], [data-testid="assets-recording-controls"]')),
		assetBrowse: Boolean(document.querySelector('[data-testid="asset-library"][data-mode="browse"]')),
		assetManageButton: Boolean(document.querySelector('[data-testid="asset-manage-button"]')),
		assetManagement: Boolean(document.querySelector('[data-testid="asset-library"][data-mode="manage"]')),
		schedulesView: Boolean(document.querySelector('[data-testid="schedules-view"]')),
		scheduleBrowse: Boolean(document.querySelector('[data-testid="schedule-library"][data-mode="browse"]')),
		scheduleManageButton: Boolean(document.querySelector('[data-testid="schedule-manage-button"]')),
		scheduleManagement: Boolean(document.querySelector('[data-testid="schedule-library"][data-mode="manage"]')),
		scheduleEditor: Boolean(document.querySelector('[data-testid="schedule-editor"]')),
		scheduleAdvanced: Boolean(document.querySelector('[data-testid="schedule-advanced"]')),
		scheduleAdvancedToggle: Boolean(document.querySelector('[data-testid="schedule-advanced-toggle"]')),
		scheduleTargetInterval: Boolean(document.querySelector('[data-testid="schedule-target-interval"]')),
		scheduleRows: document.querySelectorAll('[data-testid="schedule-row"]').length,
		scheduleRowTargets: [...document.querySelectorAll('[data-testid="schedule-row"]')]
			.flatMap(row => (row.getAttribute('data-target-ids') || '').split(',').filter(Boolean)),
		scheduleRowStatuses: [...document.querySelectorAll('[data-testid="schedule-row"]')]
			.map(row => row.getAttribute('data-last-status') || ''),
		scheduleEditTargets: [...document.querySelectorAll('[data-testid="schedule-target"]')]
			.map(target => target.getAttribute('data-workflow-id') || '').filter(Boolean),
		settingsView: Boolean(document.querySelector('[data-testid="settings-view"]')),
		settingsGroups: document.querySelectorAll('[data-testid^="settings-group-"]').length,
		appContextTitle: document.querySelector('[data-testid="app-context-title"]')?.textContent?.trim() || '',
		createInput: Boolean(document.querySelector('input[data-testid="workflow-create-name"], [data-testid="workflow-create-name"] input')),
		recoveryPanel: Boolean(document.querySelector('[data-testid="workflow-recovery-panel"]')),
		workflowBrowse: Boolean(document.querySelector('[data-testid="workflow-library"][data-mode="browse"]')),
		workflowManageButton: Boolean(document.querySelector('[data-testid="workflow-manage-button"]')),
		workflowManagement: Boolean(document.querySelector('[data-testid="workflow-library"][data-mode="manage"]')),
		workflowRows: document.querySelectorAll('[data-testid="workflow-library-row"]').length,
		workflowTotal: Number(document.querySelector('[data-testid="workflow-library"]')?.getAttribute('data-total') || 0),
		launcherButton: Boolean(document.querySelector('[data-testid="open-launcher"]')),
		graphChromeDark: darkBackground(controls) && controlButtons.length > 0 && controlButtons.every(darkBackground) && (!minimap || darkBackground(minimap)),
		handleOverlaps,
		nativeConfirmCalls: window.__yottaNativeConfirmCalls || 0,
		confirmDialog: Boolean(document.querySelector('[data-testid="confirm-dialog"]')),
		dirty: Boolean(document.querySelector('[data-testid="workflow-unsaved"]')),
		saveInlineFeedback: saveButtonText.includes('已保存') || saveButtonText.includes('Saved'),
		saveError: document.querySelector('[data-testid="workflow-save-error"]')?.textContent?.trim() || '',
		saveToast: bodyText.includes('工作流已保存') || bodyText.includes('Workflow saved'),
		diagnostics: Boolean(document.querySelector('[data-testid="workflow-diagnostics"]')),
		missingInputWarnings: [...document.querySelectorAll('[data-testid="workflow-diagnostics"] span')]
			.filter(node => node.textContent?.trim() === 'MISSING_INPUT_BINDING').length,
		selectedNodes: document.querySelectorAll('.vue-flow__node.selected').length,
		selectionToolbar: Boolean(document.querySelector('[data-testid="workflow-selection-toolbar"]')),
		connectionMenu: Boolean(document.querySelector('[data-testid="workflow-connection-menu"]')),
		connectionCandidates: document.querySelectorAll('[data-testid="workflow-connection-candidate"]').length,
		connectionError: document.querySelector('[data-testid="workflow-connection-error"]')?.textContent?.trim() || '',
		debugger: Boolean(document.querySelector('[data-testid="workflow-debugger"]')),
		debugStart: Boolean(document.querySelector('[data-testid="workflow-debug-start"]')),
		debugPaused: Boolean(document.querySelector('[data-testid="workflow-debugger"]')) && document.querySelector('[data-testid="workflow-debug-step"]') !== null,
		debugBusy: Boolean(document.querySelector('[data-testid="workflow-debug-step"]')?.disabled),
		debugCompleted: Boolean(document.querySelector('[data-testid="workflow-debugger"]')) && document.querySelector('[data-testid="workflow-debug-stop"]') === null,
		debugCurrent: document.querySelectorAll('.vue-flow__node [data-testid="node-debug-current"]').length,
		debugNode: document.querySelector('.vue-flow__node [data-testid="node-debug-current"]')?.closest('.vue-flow__node')?.getAttribute('data-id') || '',
		breakpoints: document.querySelectorAll('[data-testid="node-breakpoint"][aria-pressed="true"]').length,
		currentGraph: document.querySelector('[data-testid="workflow-canvas"]')?.getAttribute('data-graph-id') || '',
		graphCalls: document.querySelectorAll('[data-testid="workflow-graph-call"]').length,
		graphBoundaries: document.querySelectorAll('[data-testid="workflow-graph-boundary"]').length,
		graphInterface: Boolean(document.querySelector('[data-testid="workflow-graph-interface"]')),
		boundaryClipped: canvasRect ? boundaryRects.filter(rect => rect.left < canvasRect.left || rect.right > canvasRect.right || rect.top < canvasRect.top || rect.bottom > canvasRect.bottom).length : 0,
		boundaryObscured: boundaryRects.filter(rect => chromeRects.some(chrome => intersects(rect, chrome))).length,
		minimapToggle: Boolean(document.querySelector('[data-testid="workflow-minimap-toggle"]')),
		minimapOpen: Boolean(minimap),
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
		const handle = document.querySelector('.vue-flow__node[data-id="run-started"] .vue-flow__handle-right, .vue-flow__node[data-id="run-started"] .vue-flow__handle');
		if (!handle) throw new Error('RunStarted connection handle not found');
		const handleRect = handle.getBoundingClientRect();
		const start = { x: handleRect.left + handleRect.width / 2, y: handleRect.top + handleRect.height / 2 };
		const mouse = (type, point, buttons) => new MouseEvent(type, {
			bubbles: true, cancelable: true, view: window, button: 0, buttons,
			clientX: point.x, clientY: point.y
		});
		handle.dispatchEvent(mouse('mousedown', start, 1));
		await new Promise(resolve => setTimeout(resolve, 50));
		for (let step = 1; step <= 8; step++) {
			const ratio = step / 8;
			const point = {
				x: start.x + (gesture.end.x - start.x) * ratio,
				y: start.y + (gesture.end.y - start.y) * ratio
			};
			document.dispatchEvent(mouse('mousemove', point, 1));
			await new Promise(resolve => setTimeout(resolve, 20));
		}
		document.dispatchEvent(mouse('mouseup', gesture.end, 0));
	})()`, payload))
}

func dispatchKeyPress(ctx context.Context, client *browsercdp.WebSocketClient, key, code string, virtualKey int) error {
	return dispatchModifiedKeyPress(ctx, client, key, code, virtualKey, 0)
}

func dispatchModifiedKeyPress(ctx context.Context, client *browsercdp.WebSocketClient, key, code string, virtualKey, modifiers int) error {
	params := map[string]any{
		"key": key, "code": code, "windowsVirtualKeyCode": virtualKey, "nativeVirtualKeyCode": virtualKey,
		"modifiers": modifiers,
	}
	params["type"] = "keyDown"
	if _, err := client.Call(ctx, "Input.dispatchKeyEvent", params); err != nil {
		return err
	}
	params["type"] = "keyUp"
	_, err := client.Call(ctx, "Input.dispatchKeyEvent", params)
	return err
}

func dispatchMarqueeGesture(ctx context.Context, client *browsercdp.WebSocketClient, gesture connectionGesture) error {
	var gestureErr error
	if _, gestureErr = client.Call(ctx, "Input.dispatchMouseEvent", map[string]any{
		"type": "mouseMoved", "x": gesture.Start.X, "y": gesture.Start.Y,
	}); gestureErr == nil {
		_, gestureErr = client.Call(ctx, "Input.dispatchMouseEvent", map[string]any{
			"type": "mousePressed", "x": gesture.Start.X, "y": gesture.Start.Y,
			"button": "left", "buttons": 1, "clickCount": 1,
		})
	}
	for step := 1; gestureErr == nil && step <= 8; step++ {
		ratio := float64(step) / 8
		_, gestureErr = client.Call(ctx, "Input.dispatchMouseEvent", map[string]any{
			"type":   "mouseMoved",
			"x":      gesture.Start.X + (gesture.End.X-gesture.Start.X)*ratio,
			"y":      gesture.Start.Y + (gesture.End.Y-gesture.Start.Y)*ratio,
			"button": "left", "buttons": 1,
		})
	}
	if gestureErr == nil {
		_, gestureErr = client.Call(ctx, "Input.dispatchMouseEvent", map[string]any{
			"type": "mouseReleased", "x": gesture.End.X, "y": gesture.End.Y,
			"button": "left", "buttons": 0, "clickCount": 1,
		})
	}
	return gestureErr
}

func beginMarqueeTrace(ctx context.Context, client *browsercdp.WebSocketClient) error {
	return eval(ctx, client, `(() => {
		const trace = [];
		const types = ['keydown', 'keyup', 'pointerdown', 'pointermove', 'pointerup'];
		const handler = event => {
			const target = event.target instanceof Element ? event.target : null;
			trace.push({
				type: event.type,
				key: event.key || '',
				shiftKey: Boolean(event.shiftKey),
				target: target?.className?.toString?.() || target?.tagName || '',
				paneSelecting: Boolean(document.querySelector('.vue-flow__pane.selection')),
				selectedNodes: document.querySelectorAll('.vue-flow__node.selected').length
			});
		};
		for (const type of types) window.addEventListener(type, handler, true);
		window.__yottaMarqueeTrace = { trace, types, handler };
	})()`)
}

func finishMarqueeTrace(ctx context.Context, client *browsercdp.WebSocketClient) string {
	var trace map[string]any
	err := evalJSON(ctx, client, `(() => {
		const state = window.__yottaMarqueeTrace;
		if (!state) return { missing: true };
		for (const type of state.types) window.removeEventListener(type, state.handler, true);
		delete window.__yottaMarqueeTrace;
		return {
			events: state.trace,
			paneClass: document.querySelector('.vue-flow__pane')?.className || '',
			selectedNodes: document.querySelectorAll('.vue-flow__node.selected').length
		};
	})()`, &trace)
	if err != nil {
		return err.Error()
	}
	raw, _ := json.Marshal(trace)
	return string(raw)
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
