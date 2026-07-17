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

	"github.com/yottaapp/yotta/internal/services/browsercdp"
)

type pageState struct {
	Href               string   `json:"href"`
	Catalog            int      `json:"catalog"`
	CanvasNodes        int      `json:"canvasNodes"`
	AIReview           bool     `json:"aiReview"`
	WorkflowState      bool     `json:"workflowState"`
	RunStarted         bool     `json:"runStarted"`
	AssetsView         bool     `json:"assetsView"`
	AssetsRecording    bool     `json:"assetsRecording"`
	CreateInput        bool     `json:"createInput"`
	GraphChromeDark    bool     `json:"graphChromeDark"`
	HandleOverlaps     int      `json:"handleOverlaps"`
	NativeConfirmCalls int      `json:"nativeConfirmCalls"`
	ConfirmDialog      bool     `json:"confirmDialog"`
	Dirty              bool     `json:"dirty"`
	SaveInlineFeedback bool     `json:"saveInlineFeedback"`
	SaveToast          bool     `json:"saveToast"`
	Errors             []string `json:"errors"`
}

func main() {
	endpoint := flag.String("endpoint", "http://127.0.0.1:9227", "WebView2 CDP endpoint")
	screenshot := flag.String("screenshot", ".task/workflow-editor-smoke.png", "PNG output path")
	assetsScreenshot := flag.String("assets-screenshot", ".task/assets-smoke.png", "asset library PNG output path")
	flag.Parse()

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	if err := run(ctx, *endpoint, *screenshot, *assetsScreenshot); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run(ctx context.Context, endpoint, screenshot, assetsScreenshot string) error {
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
	if err := waitUntil(ctx, client, func(current pageState) bool { return current.CreateInput }); err != nil {
		return fmt.Errorf("wait for workflow list hydration: %w", err)
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
	if err := waitUntil(ctx, client, func(current pageState) bool {
		return !current.Dirty && (current.SaveInlineFeedback || current.SaveToast)
	}); err != nil {
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
		"assetsScreenshot": assetsScreenshot,
	}, "", "  ")
	fmt.Println(string(result))
	return nil
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
			return ctx.Err()
		case <-time.After(100 * time.Millisecond):
		}
	}
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
		return {
		href: location.href,
		catalog: document.querySelectorAll('[data-testid="node-catalog-item"]').length,
		canvasNodes: document.querySelectorAll('.vue-flow__node').length,
		aiReview: Boolean(document.querySelector('[data-testid="ai-workflow-review-panel"]')),
		workflowState: Boolean(document.querySelector('[data-testid="workflow-state-panel"]')),
		runStarted: Boolean(document.querySelector('.vue-flow__node[data-id="run-started"]')),
		assetsView: Boolean(document.querySelector('[data-testid="assets-view"]')),
		assetsRecording: Boolean(document.querySelector('[data-testid="assets-recording-controls"]')),
		createInput: Boolean(document.querySelector('input[data-testid="workflow-create-name"], [data-testid="workflow-create-name"] input')),
		graphChromeDark: darkBackground(controls) && controlButtons.length > 0 && controlButtons.every(darkBackground) && darkBackground(minimap),
		handleOverlaps,
		nativeConfirmCalls: window.__yottaNativeConfirmCalls || 0,
		confirmDialog: Boolean(document.querySelector('[data-testid="confirm-dialog"]')),
		dirty: Boolean(document.querySelector('[data-testid="workflow-unsaved"]')),
		saveInlineFeedback: saveButtonText.includes('已保存') || saveButtonText.includes('Saved'),
		saveToast: bodyText.includes('工作流已保存') || bodyText.includes('Workflow saved'),
		errors: window.__yottaSmokeErrors || []
		};
	})()`, &out)
	return out, err
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
