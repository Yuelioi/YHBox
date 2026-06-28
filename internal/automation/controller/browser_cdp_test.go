package controller

import (
	"context"
	"encoding/base64"
	"reflect"
	"testing"

	"yotta/internal/automation/target"
	automationtrace "yotta/internal/automation/trace"
)

type cdpCall struct {
	method string
	params map[string]any
}

type fakeCDPClient struct {
	calls []cdpCall
	res   map[string]any
	err   error
}

func (f *fakeCDPClient) Call(_ context.Context, method string, params map[string]any) (map[string]any, error) {
	cp := map[string]any{}
	for k, v := range params {
		cp[k] = v
	}
	f.calls = append(f.calls, cdpCall{method: method, params: cp})
	return f.res, f.err
}

func TestBrowserCDPControllerTargetAndCapabilities(t *testing.T) {
	ctrl, err := NewBrowserCDPController(browserTarget(), BrowserCDPDeps{Client: &fakeCDPClient{}})
	if err != nil {
		t.Fatalf("NewBrowserCDPController() error = %v", err)
	}
	if ctrl.Target().ID != "browser:tab-1" {
		t.Fatalf("target id = %q", ctrl.Target().ID)
	}
	caps := ctrl.Capabilities(context.Background())
	if !caps.Screenshot || !caps.Click || !caps.KeyState || caps.StartApp {
		t.Fatalf("unexpected caps: %#v", caps)
	}
}

func TestBrowserCDPControllerRejectsNonBrowserTarget(t *testing.T) {
	_, err := NewBrowserCDPController(target.Target{
		ID:   "android:emulator",
		Kind: target.KindAndroidADB,
		Ref:  target.TargetRef{ADBSerial: "emulator"},
	}, BrowserCDPDeps{})
	if err == nil {
		t.Fatalf("expected error for non-browser target")
	}
}

func TestBrowserCDPControllerClickDispatchesMouseAndTrace(t *testing.T) {
	client := &fakeCDPClient{}
	rec := automationtrace.NewMemoryRecorder()
	ctrl, err := NewBrowserCDPController(browserTarget(), BrowserCDPDeps{Client: client, Trace: rec})
	if err != nil {
		t.Fatalf("NewBrowserCDPController() error = %v", err)
	}
	if err := ctrl.Click(context.Background(), ClickRequest{Point: target.NewNormalizedPoint(0.5, 0.25)}); err != nil {
		t.Fatalf("Click() error = %v", err)
	}
	want := []cdpCall{
		{method: "Input.dispatchMouseEvent", params: map[string]any{"type": "mousePressed", "x": 640, "y": 180, "button": "left", "clickCount": 1}},
		{method: "Input.dispatchMouseEvent", params: map[string]any{"type": "mouseReleased", "x": 640, "y": 180, "button": "left", "clickCount": 1}},
	}
	if !reflect.DeepEqual(client.calls, want) {
		t.Fatalf("cdp calls = %#v, want %#v", client.calls, want)
	}
	records := rec.Records()
	if len(records) != 1 || records[0].Action != "click" || records[0].Backend != string(BackendBrowserCDP) {
		t.Fatalf("trace records = %#v", records)
	}
	if len(records[0].CoordinateSteps) != 1 || records[0].CoordinateSteps[0].To != target.SpaceBrowserView {
		t.Fatalf("coordinate steps = %#v", records[0].CoordinateSteps)
	}
}

func TestBrowserCDPControllerDragDispatchesMouseSequence(t *testing.T) {
	client := &fakeCDPClient{}
	ctrl, err := NewBrowserCDPController(browserTarget(), BrowserCDPDeps{Client: client})
	if err != nil {
		t.Fatalf("NewBrowserCDPController() error = %v", err)
	}
	err = ctrl.Drag(context.Background(), DragRequest{
		From:   target.NewNormalizedPoint(0.1, 0.2),
		To:     target.NewNormalizedPoint(0.8, 0.9),
		Button: "right",
	})
	if err != nil {
		t.Fatalf("Drag() error = %v", err)
	}
	if len(client.calls) != 3 {
		t.Fatalf("cdp calls len = %d, want 3: %#v", len(client.calls), client.calls)
	}
	if client.calls[0].params["type"] != "mousePressed" || client.calls[1].params["type"] != "mouseMoved" || client.calls[2].params["type"] != "mouseReleased" {
		t.Fatalf("unexpected drag calls: %#v", client.calls)
	}
	if client.calls[0].params["button"] != "right" || client.calls[2].params["button"] != "right" {
		t.Fatalf("drag button not preserved: %#v", client.calls)
	}
}

func TestBrowserCDPControllerKeyChordDispatchesDownAndReverseUp(t *testing.T) {
	client := &fakeCDPClient{}
	ctrl, err := NewBrowserCDPController(browserTarget(), BrowserCDPDeps{Client: client})
	if err != nil {
		t.Fatalf("NewBrowserCDPController() error = %v", err)
	}
	if err := ctrl.KeyChord(context.Background(), KeyChordRequest{Keys: []string{"Control", "N"}}); err != nil {
		t.Fatalf("KeyChord() error = %v", err)
	}
	want := []cdpCall{
		{method: "Input.dispatchKeyEvent", params: map[string]any{"type": "keyDown", "key": "Control"}},
		{method: "Input.dispatchKeyEvent", params: map[string]any{"type": "keyDown", "key": "N"}},
		{method: "Input.dispatchKeyEvent", params: map[string]any{"type": "keyUp", "key": "N"}},
		{method: "Input.dispatchKeyEvent", params: map[string]any{"type": "keyUp", "key": "Control"}},
	}
	if !reflect.DeepEqual(client.calls, want) {
		t.Fatalf("cdp calls = %#v, want %#v", client.calls, want)
	}
}

func TestBrowserCDPControllerTextUsesInsertText(t *testing.T) {
	client := &fakeCDPClient{}
	ctrl, err := NewBrowserCDPController(browserTarget(), BrowserCDPDeps{Client: client})
	if err != nil {
		t.Fatalf("NewBrowserCDPController() error = %v", err)
	}
	if err := ctrl.Text(context.Background(), TextRequest{Text: "hello world"}); err != nil {
		t.Fatalf("Text() error = %v", err)
	}
	want := []cdpCall{{method: "Input.insertText", params: map[string]any{"text": "hello world"}}}
	if !reflect.DeepEqual(client.calls, want) {
		t.Fatalf("cdp calls = %#v, want %#v", client.calls, want)
	}
}

func TestBrowserCDPControllerScreenshotDecodesBase64PNGAndTracesResult(t *testing.T) {
	client := &fakeCDPClient{res: map[string]any{"data": base64.StdEncoding.EncodeToString(tinyPNG(t))}}
	rec := automationtrace.NewMemoryRecorder()
	ctrl, err := NewBrowserCDPController(browserTarget(), BrowserCDPDeps{Client: client, Trace: rec})
	if err != nil {
		t.Fatalf("NewBrowserCDPController() error = %v", err)
	}
	frame, err := ctrl.Screenshot(context.Background(), ScreenshotRequest{})
	if err != nil {
		t.Fatalf("Screenshot() error = %v", err)
	}
	want := []cdpCall{{method: "Page.captureScreenshot", params: map[string]any{"format": "png"}}}
	if !reflect.DeepEqual(client.calls, want) {
		t.Fatalf("cdp calls = %#v, want %#v", client.calls, want)
	}
	if frame.Image == nil || frame.Size.W != 2 || frame.Size.H != 1 || frame.Space != target.SpaceBrowserView {
		t.Fatalf("unexpected frame: %#v", frame)
	}
	records := rec.Records()
	if len(records) != 1 || records[0].Action != "screenshot" {
		t.Fatalf("trace records = %#v", records)
	}
	result, _ := records[0].Result.(map[string]any)
	if result["width"] != 2 || result["height"] != 1 {
		t.Fatalf("trace result = %#v", records[0].Result)
	}
}

func browserTarget() target.Target {
	return target.Target{
		ID:         "browser:tab-1",
		Kind:       target.KindBrowserCDP,
		Ref:        target.TargetRef{BrowserID: "tab-1"},
		Resolution: target.Size{W: 1280, H: 720},
	}
}
