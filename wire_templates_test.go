package main

import (
	"bytes"
	"context"
	"image"
	"image/color"
	"image/png"
	"reflect"
	"testing"

	"yotta/internal/services/container"
)

type templateCaptureADBCall struct {
	serial string
	args   []string
}

type templateCaptureFakeADBRunner struct {
	calls []templateCaptureADBCall
	out   []byte
	err   error
}

func (f *templateCaptureFakeADBRunner) Run(_ context.Context, serial string, args ...string) ([]byte, error) {
	f.calls = append(f.calls, templateCaptureADBCall{serial: serial, args: append([]string(nil), args...)})
	return f.out, f.err
}

func TestTemplateCaptureAdapter_CaptureAndroidTargetUsesADBScreenshot(t *testing.T) {
	store, err := container.NewStore(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	c := container.Container{
		ID:             "c1",
		Name:           "android",
		InputBackend:   "postmessage",
		CaptureBackend: "auto",
		Graph: container.Graph{
			ID:      "g1",
			Version: container.GraphSchemaVersion,
			Nodes: []container.GraphNode{
				{ID: "start", Kind: "Start"},
				{ID: "at", Kind: "AndroidTarget", Config: map[string]any{"literal": map[string]any{
					"Serial": "127.0.0.1:7555",
					"Name":   "MuMu",
					"Width":  1280,
					"Height": 720,
				}}},
				{ID: "click", Kind: "ClickAt"},
				{ID: "stop", Kind: "Stop"},
			},
			Edges: []container.GraphEdge{
				{From: "start.Done", To: "at.In"},
				{From: "at.Done", To: "click.In"},
				{From: "click.Done", To: "stop.In"},
			},
		},
	}
	if err := store.Save(&c); err != nil {
		t.Fatal(err)
	}
	adb := &templateCaptureFakeADBRunner{out: templateCaptureTinyPNG(t)}
	adapter := &templateCaptureAdapter{containers: container.NewService(store), adbRunner: adb}

	got, err := adapter.Capture("c1", "click")
	if err != nil {
		t.Fatal(err)
	}
	img, err := png.Decode(bytes.NewReader(got))
	if err != nil {
		t.Fatalf("decode captured png: %v", err)
	}
	if b := img.Bounds(); b.Dx() != 2 || b.Dy() != 1 {
		t.Fatalf("captured bounds = %v", b)
	}
	wantArgs := []string{"exec-out", "screencap", "-p"}
	if len(adb.calls) != 1 || adb.calls[0].serial != "127.0.0.1:7555" || !reflect.DeepEqual(adb.calls[0].args, wantArgs) {
		t.Fatalf("adb calls = %#v, want serial 127.0.0.1:7555 args %#v", adb.calls, wantArgs)
	}
}

func templateCaptureTinyPNG(t *testing.T) []byte {
	t.Helper()
	img := image.NewRGBA(image.Rect(0, 0, 2, 1))
	img.Set(0, 0, color.RGBA{R: 12, G: 34, B: 56, A: 255})
	img.Set(1, 0, color.RGBA{R: 78, G: 90, B: 123, A: 255})
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		t.Fatalf("encode png: %v", err)
	}
	return buf.Bytes()
}
