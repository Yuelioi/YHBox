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
	Href        string   `json:"href"`
	Catalog     int      `json:"catalog"`
	CanvasNodes int      `json:"canvasNodes"`
	Errors      []string `json:"errors"`
}

func main() {
	endpoint := flag.String("endpoint", "http://127.0.0.1:9227", "WebView2 CDP endpoint")
	screenshot := flag.String("screenshot", ".task/workflow-editor-smoke.png", "PNG output path")
	flag.Parse()

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	if err := run(ctx, *endpoint, *screenshot); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run(ctx context.Context, endpoint, screenshot string) error {
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

	nameJSON, _ := json.Marshal("Agent UI smoke " + time.Now().UTC().Format("20060102T150405Z"))
	if err := eval(ctx, client, fmt.Sprintf(`(() => {
		const holder = document.querySelector('[data-testid="workflow-create-name"]');
		const input = holder?.matches('input') ? holder : holder?.querySelector('input');
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
	if err := eval(ctx, client, `document.querySelector('[data-testid="node-catalog-item"]')?.click()`); err != nil {
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

	final, err := state(ctx, client)
	if err != nil {
		return err
	}
	if len(final.Errors) != 0 {
		return fmt.Errorf("WebView reported errors: %s", strings.Join(final.Errors, " | "))
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
	result, _ := json.MarshalIndent(map[string]any{
		"status": "passed", "href": final.Href, "catalogNodes": final.Catalog,
		"canvasNodes": final.CanvasNodes, "screenshot": screenshot,
	}, "", "  ")
	fmt.Println(string(result))
	return nil
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
	err := evalJSON(ctx, client, `({
		href: location.href,
		catalog: document.querySelectorAll('[data-testid="node-catalog-item"]').length,
		canvasNodes: document.querySelectorAll('.vue-flow__node').length,
		errors: window.__yottaSmokeErrors || []
	})`, &out)
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
		return fmt.Errorf("WebView evaluation failed: %v", details["text"])
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
		return fmt.Errorf("WebView evaluation failed: %v", details["text"])
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
