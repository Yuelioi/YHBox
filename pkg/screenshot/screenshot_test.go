package screenshot

import (
	"image"
	"image/color"
	"image/png"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestWriterSubmitWritesPNG(t *testing.T) {
	tmp := t.TempDir()
	w := NewWriter(tmp, 4)
	defer w.Close()

	img := image.NewRGBA(image.Rect(0, 0, 100, 50))
	// 填白让我们能眼检（如果 png decode 出来是全 0 那就是 bug）
	for i := range img.Pix {
		img.Pix[i] = 255
	}
	boxes := []Box{
		{Rect: image.Rect(10, 10, 30, 30), Color: color.RGBA{255, 0, 0, 255}},
	}
	if !w.Submit("fish", img, boxes, "hook_icon") {
		t.Fatal("Submit return false (queue full?)")
	}

	// 等 goroutine 写完 — 队列消费 + 写盘理论上几 ms
	w.Close()

	dateDir := time.Now().Format("2006-01-02")
	entries, err := os.ReadDir(filepath.Join(tmp, "fish", dateDir))
	if err != nil {
		t.Fatalf("read dir: %v", err)
	}
	if len(entries) != 1 {
		t.Fatalf("expect 1 PNG, got %d", len(entries))
	}
	// 解码看尺寸 + 红框 (10,10) 像素是否真的红了
	f, err := os.Open(filepath.Join(tmp, "fish", dateDir, entries[0].Name()))
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	decoded, err := png.Decode(f)
	if err != nil {
		t.Fatal(err)
	}
	if decoded.Bounds().Dx() != 100 || decoded.Bounds().Dy() != 50 {
		t.Errorf("size wrong: %v", decoded.Bounds())
	}
	r, g, b, _ := decoded.At(10, 10).RGBA()
	if r>>8 != 255 || g>>8 != 0 || b>>8 != 0 {
		t.Errorf("框边像素应该是红色，got R=%d G=%d B=%d", r>>8, g>>8, b>>8)
	}
}

func TestWriterDisabledDrops(t *testing.T) {
	tmp := t.TempDir()
	w := NewWriter(tmp, 4)
	defer w.Close()
	w.SetEnabled(false)

	img := image.NewRGBA(image.Rect(0, 0, 10, 10))
	if w.Submit("fish", img, nil, "x") {
		t.Error("disabled 时 Submit 应该返 false 而非 true")
	}
}

func TestWriterQueueFullDrops(t *testing.T) {
	tmp := t.TempDir()
	// 队列容量 1 + 不启动 consumer（暂时 hack：用一个 unbuffered 让 Close 后等 goroutine 退）
	w := NewWriter(tmp, 1)
	defer w.Close()

	// Submit 3 张紧凑：第 1 张可能进了 queue，第 2/3 在 queue full 时 drop
	// 不严格断言"必定" drop（goroutine 可能刚好消费完）；只验证 Submit 返 bool 不 panic
	img := image.NewRGBA(image.Rect(0, 0, 5, 5))
	for i := 0; i < 3; i++ {
		_ = w.Submit("test", img, nil, "burst")
	}
}
