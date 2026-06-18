// pkg/vision/qr_test.go
package vision

import (
	"image"
	"image/color"
	"testing"

	"github.com/makiuchi-d/gozxing"
	"github.com/makiuchi-d/gozxing/qrcode"
)

// newWhiteRGBA 返一张全白 RGBA 图, 用于测试"空白图解码" 路径.
func newWhiteRGBA(w, h int) *image.RGBA {
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	white := color.RGBA{255, 255, 255, 255}
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			img.SetRGBA(x, y, white)
		}
	}
	return img
}

// TestDecodeQRFromImage_RoundTrip 用 gozxing 自带 encoder 生成 QR 图再解回, 验证
// DecodeQRFromImage 能正确还原文本并回报定位点。
func TestDecodeQRFromImage_RoundTrip(t *testing.T) {
	const text = "YHFish-QR-roundtrip"

	// 编码: qrcode.NewQRCodeWriter().Encode → *gozxing.BitMatrix (同时实现 image.Image)
	writer := qrcode.NewQRCodeWriter()
	matrix, err := writer.Encode(text, gozxing.BarcodeFormat_QR_CODE, 200, 200, nil)
	if err != nil {
		t.Fatalf("encode QR: %v", err)
	}

	// BitMatrix 实现 image.Image (go_image_bit_matrix.go), 直接传给 DecodeQRFromImage.
	hits, err := DecodeQRFromImage(matrix)
	if err != nil {
		t.Fatalf("DecodeQRFromImage: %v", err)
	}
	if len(hits) == 0 {
		t.Fatal("expected at least one QRHit, got none")
	}
	if hits[0].Text != text {
		t.Errorf("decoded text = %q, want %q", hits[0].Text, text)
	}
	if len(hits[0].Points) == 0 {
		t.Error("expected at least one result point, got none")
	}
	t.Logf("decoded text=%q, points=%v", hits[0].Text, hits[0].Points)
}

// TestDecodeQRFromImage_NotFound 传一张空白图, 期望返回空 slice + nil error.
func TestDecodeQRFromImage_NotFound(t *testing.T) {
	// 全白 BitMatrix (1bit 黑白矩阵, 所有位为 false → 像素为白)
	// 用 encoder 生成任意图再把内容故意损坏: 直接用一个小白色 RGBA 图。
	img := newWhiteRGBA(50, 50)
	hits, err := DecodeQRFromImage(img)
	if err != nil {
		t.Fatalf("expected nil error on blank image, got: %v", err)
	}
	if len(hits) != 0 {
		t.Errorf("expected empty hits on blank image, got %d", len(hits))
	}
}
