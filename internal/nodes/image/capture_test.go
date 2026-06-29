package image

import (
	"bytes"
	"context"
	imagepkg "image"
	"image/jpeg"
	"image/png"
	"testing"

	"yotta/internal/node"
)

func tinyPNG(t *testing.T) []byte {
	t.Helper()
	img := imagepkg.NewRGBA(imagepkg.Rect(0, 0, 2, 2))
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		t.Fatal(err)
	}
	return buf.Bytes()
}

type fakeCapture struct {
	png []byte
	err error
}

func (f fakeCapture) Capture() ([]byte, error)                    { return f.png, f.err }
func (f fakeCapture) CaptureROI(_ node.Geometry) ([]byte, error) { return f.png, f.err }

func runCapture(t *testing.T, format string, cap node.CaptureService) node.RunResult {
	t.Helper()
	node.ResetRegistryForTest()
	node.Register(&Capture{})
	rn, _ := node.Get("Capture")
	svc := node.StubServices()
	svc.Capture = cap
	return node.RunNode(context.Background(), rn, nil, map[string]any{"Format": format}, nil, svc, false)
}

func TestCapture_PNG(t *testing.T) {
	pngBytes := tinyPNG(t)
	res := runCapture(t, "png", fakeCapture{png: pngBytes})
	if res.Error != nil || res.ExitName != "Done" {
		t.Fatalf("exit=%q err=%v", res.ExitName, res.Error)
	}
	img, ok := res.OutputData["Image"].(node.Image)
	if !ok {
		t.Fatalf("Image value type = %T, want node.Image", res.OutputData["Image"])
	}
	if img.Format != "png" || !bytes.Equal(img.Data, pngBytes) {
		t.Errorf("png: format=%q dataEqual=%v", img.Format, bytes.Equal(img.Data, pngBytes))
	}
}

func TestCapture_JPEG(t *testing.T) {
	res := runCapture(t, "jpeg", fakeCapture{png: tinyPNG(t)})
	if res.Error != nil || res.ExitName != "Done" {
		t.Fatalf("exit=%q err=%v", res.ExitName, res.Error)
	}
	img := res.OutputData["Image"].(node.Image)
	if img.Format != "jpeg" {
		t.Errorf("format = %q, want jpeg", img.Format)
	}
	if _, err := jpeg.Decode(bytes.NewReader(img.Data)); err != nil {
		t.Errorf("jpeg data not decodable: %v", err)
	}
}

func TestCapture_CaptureError_Fails(t *testing.T) {
	res := runCapture(t, "png", fakeCapture{err: context.DeadlineExceeded})
	if res.Error == nil {
		t.Fatal("capture error should fail the node")
	}
}
