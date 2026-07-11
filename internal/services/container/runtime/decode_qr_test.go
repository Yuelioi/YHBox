package runtime

import (
	"image"
	"image/draw"
	"testing"

	"github.com/makiuchi-d/gozxing"
	"github.com/makiuchi-d/gozxing/qrcode"

	"github.com/yottaapp/yotta/internal/node"
)

// TestDecodeQR_FoundViaRealAdapter: 用 gozxing encoder 生成一张 QR, 画进 RGBA 帧注入,
// 走真实 visionAdapter.DecodeQR (controller screenshot → cropFrameByGeometry → vision.DecodeQRFromImage)
// 验证解回原文 + 有定位点。
func TestDecodeQR_FoundViaRealAdapter(t *testing.T) {
	const text = "yotta-qr-integ"
	matrix, err := qrcode.NewQRCodeWriter().Encode(text, gozxing.BarcodeFormat_QR_CODE, 200, 200, nil)
	if err != nil {
		t.Fatalf("encode QR: %v", err)
	}
	frame := image.NewRGBA(image.Rect(0, 0, 200, 200))
	draw.Draw(frame, frame.Bounds(), matrix, image.Point{}, draw.Src)

	rt, _ := newTestRunner(t)
	installedTestWin32Capture(rt).(*mockCaptureBackend).FrameROIResult = frame

	va := NewVisionAdapter(rt)
	res, err := va.DecodeQR(node.Geometry{})
	if err != nil {
		t.Fatalf("DecodeQR error: %v", err)
	}
	if len(res) != 1 {
		t.Fatalf("want 1 QR decoded, got %d: %+v", len(res), res)
	}
	if res[0].Text != text {
		t.Errorf("Text=%q, want %q", res[0].Text, text)
	}
	if len(res[0].Points) == 0 {
		t.Error("expected locator points, got none")
	}
}

// TestDecodeQR_NotFoundViaRealAdapter: 全白帧 → 无 QR → 空结果 (节点据此走 NotFound)。
func TestDecodeQR_NotFoundViaRealAdapter(t *testing.T) {
	frame := image.NewRGBA(image.Rect(0, 0, 100, 100))
	draw.Draw(frame, frame.Bounds(), image.NewUniform(image.White), image.Point{}, draw.Src)

	rt, _ := newTestRunner(t)
	installedTestWin32Capture(rt).(*mockCaptureBackend).FrameROIResult = frame

	va := NewVisionAdapter(rt)
	res, err := va.DecodeQR(node.Geometry{})
	if err != nil {
		t.Fatalf("DecodeQR error: %v", err)
	}
	if len(res) != 0 {
		t.Errorf("want 0 QR on blank frame, got %d", len(res))
	}
}

// TestDecodeQR_NodeRegistered: 注册检查。
func TestDecodeQR_NodeRegistered(t *testing.T) {
	rn, ok := node.Get("DecodeQR")
	if !ok {
		t.Fatal("DecodeQR not in node registry")
	}
	if rn.Spec.Category != "Detect" {
		t.Errorf("Category=%q, want Detect", rn.Spec.Category)
	}
}
