package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"time"

	"github.com/yottaapp/yotta/internal/automation/browsercdp"
	"github.com/yottaapp/yotta/internal/automation/controller"
	"github.com/yottaapp/yotta/internal/automation/installed"
	"github.com/yottaapp/yotta/internal/automation/target"
)

func main() {
	endpoint := flag.String("endpoint", browsercdp.DefaultEndpoint, "literal loopback Chrome DevTools HTTP origin")
	flag.Parse()
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	if err := run(ctx, *endpoint); err != nil {
		fmt.Fprintln(os.Stderr, "browser CDP smoke failed:", err)
		os.Exit(1)
	}
}

func run(ctx context.Context, endpoint string) error {
	service := browsercdp.NewService(endpoint)
	targets, err := service.ListTargets(ctx, endpoint)
	if err != nil {
		return err
	}
	if len(targets) == 0 {
		return fmt.Errorf("no debuggable page target found")
	}
	info := targets[0]
	profile, err := installed.SealProfile(installed.ProfileDraft{
		TargetKind: installed.TargetKindBrowserCDP, AdapterKind: installed.AdapterKindBrowserCDP,
		BrowserEndpoint: endpoint, BrowserTargetID: info.ID, BrowserWebSocketURL: info.WebSocketDebuggerURL,
		BrowserTitle: info.Title, BrowserURL: info.URL, ResolveTimeoutMilliseconds: 5000,
	})
	if err != nil {
		return err
	}
	probe, err := installed.NewBrowserHealthProbe(profile)
	if err != nil {
		return err
	}
	resolved, err := probe.Resolve(ctx)
	if err != nil {
		return err
	}
	client, err := browsercdp.DialWebSocketClient(ctx, info.WebSocketDebuggerURL)
	if err != nil {
		return err
	}
	defer client.Close()
	control, err := controller.NewBrowserCDPController(resolved, controller.BrowserCDPDeps{Client: client})
	if err != nil {
		return err
	}
	if _, err := client.Call(ctx, "Runtime.evaluate", map[string]any{
		"expression": `document.body.innerHTML='<input id="yotta-cdp-smoke">';document.querySelector('#yotta-cdp-smoke').focus()`,
	}); err != nil {
		return err
	}
	if err := control.Text(ctx, controller.TextRequest{Text: "yotta-cdp-smoke"}); err != nil {
		return err
	}
	value, err := client.Call(ctx, "Runtime.evaluate", map[string]any{
		"expression": "document.querySelector('#yotta-cdp-smoke').value", "returnByValue": true,
	})
	if err != nil {
		return err
	}
	if !runtimeValueEquals(value, "yotta-cdp-smoke") {
		return fmt.Errorf("text input did not round-trip through CDP: %#v", value)
	}
	if err := control.Move(ctx, controller.MoveRequest{Point: target.NewNormalizedPoint(0.5, 0.5)}); err != nil {
		return err
	}
	frame, err := control.Screenshot(ctx, controller.ScreenshotRequest{Space: target.SpaceBrowserView})
	if err != nil {
		return err
	}
	result := map[string]any{
		"endpoint": profile.Machine().BrowserEndpoint, "pageId": info.ID, "title": info.Title,
		"viewport": []int{resolved.Resolution.W, resolved.Resolution.H},
		"capture":  []int{frame.Size.W, frame.Size.H},
	}
	encoded, _ := json.Marshal(result)
	fmt.Println(string(encoded))
	return nil
}

func runtimeValueEquals(response map[string]any, expected string) bool {
	result, ok := response["result"].(map[string]any)
	if !ok {
		return false
	}
	value, ok := result["value"].(string)
	return ok && value == expected
}
