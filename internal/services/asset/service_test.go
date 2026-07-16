package asset

import (
	"bytes"
	"context"
	"encoding/base64"
	"errors"
	"image"
	"image/color"
	"image/png"
	"testing"

	"github.com/yottaapp/yotta/internal/automation/target"
)

// stubCaptureAdapter 测试用: ResolveWindow 返预设值/错误; Capture 不被这些测试触及.
type stubCaptureAdapter struct {
	res [2]int
	err error
}

func (s stubCaptureAdapter) CapturePNG(context.Context, string) ([]byte, error) { return nil, nil }
func (s stubCaptureAdapter) ResolveWindow(context.Context, string) (target.WindowHandle, error) {
	return target.WindowHandle{ClientW: s.res[0], ClientH: s.res[1]}, s.err
}

type recordingCaptureAdapter struct {
	targetSlot string
}

func (r *recordingCaptureAdapter) CapturePNG(_ context.Context, targetSlot string) ([]byte, error) {
	r.targetSlot = targetSlot
	return []byte("png"), nil
}

func (r *recordingCaptureAdapter) ResolveWindow(context.Context, string) (target.WindowHandle, error) {
	return target.WindowHandle{}, nil
}

func pngDataURL(t *testing.T, w, h int) string {
	t.Helper()
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			img.Set(x, y, color.RGBA{R: 10, G: 20, B: 30, A: 255})
		}
	}
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		t.Fatal(err)
	}
	return "data:image/png;base64," + base64.StdEncoding.EncodeToString(buf.Bytes())
}

func TestService_SaveTemplateCapture_ListGet(t *testing.T) {
	s, _ := newTestStore(t)
	svc := NewService(s, nil)

	guid, err := svc.SaveTemplateCapture(pngDataURL(t, 32, 16), "登录按钮", "登录", []string{"按钮"}, [2]int{1920, 1080}, [4]float32{0.1, 0.2, 0.3, 0.4})
	if err != nil {
		t.Fatal(err)
	}
	if guid == "" {
		t.Fatal("empty guid")
	}
	rec, err := svc.Get(guid)
	if err != nil {
		t.Fatal(err)
	}
	if rec.Kind != KindTemplate || rec.Name != "登录按钮" || rec.Category != "登录" || len(rec.Variants) != 1 {
		t.Fatalf("bad record: %+v", rec)
	}
	// bbox 换算: x1 = 0.1*1920=192, y1=0.2*1080=216, x2=(0.1+0.3)*1920=768, y2=(0.2+0.4)*1080=648
	v := rec.Variants[0]
	if v.BBox != [4]int{192, 216, 768, 648} {
		t.Errorf("bbox = %v, want [192 216 768 648]", v.BBox)
	}

	list := svc.List()
	if len(list) != 1 || list[0].GUID != guid || list[0].VariantCount != 1 || len(list[0].Variants) != 1 || list[0].Variants[0].Blob != v.Blob {
		t.Fatalf("List = %+v", list)
	}
}

func TestService_SaveTemplateCapture_PersistsCategory(t *testing.T) {
	s, _ := newTestStore(t)
	svc := NewService(s, nil)

	guid, err := svc.SaveTemplateCapture(pngDataURL(t, 8, 8), "确认按钮", "战斗", []string{"按钮"}, [2]int{1280, 720}, [4]float32{0, 0, 1, 1})
	if err != nil {
		t.Fatal(err)
	}
	rec, err := svc.Get(guid)
	if err != nil {
		t.Fatal(err)
	}
	if rec.Category != "战斗" {
		t.Fatalf("category = %q, want %q", rec.Category, "战斗")
	}
}

func TestService_RenameDelete(t *testing.T) {
	s, _ := newTestStore(t)
	svc := NewService(s, nil)
	guid, _ := svc.SaveTemplateCapture(pngDataURL(t, 8, 8), "旧名", "", nil, [2]int{1280, 720}, [4]float32{0, 0, 1, 1})

	if err := svc.UpdateMeta(guid, "新名", "测试描述", "采集", []string{"物品", "背包"}); err != nil {
		t.Fatal(err)
	}
	rec, _ := svc.Get(guid)
	if rec.Name != "新名" {
		t.Errorf("rename failed: %q", rec.Name)
	}
	if rec.Description != "测试描述" || rec.Category != "采集" {
		t.Errorf("desc/category update failed: %q / %q", rec.Description, rec.Category)
	}
	if len(rec.Tags) != 2 || rec.Tags[0] != "物品" {
		t.Errorf("tags update failed: %v", rec.Tags)
	}

	if err := svc.Delete(guid); err != nil {
		t.Fatal(err)
	}
	if _, err := svc.Get(guid); err == nil {
		t.Error("asset should be deleted")
	}
}

func TestService_CaptureUsesExactTargetSlot(t *testing.T) {
	s, _ := newTestStore(t)
	capture := &recordingCaptureAdapter{}
	svc := NewService(s, capture)

	dataURL, err := svc.Capture("editor")
	if err != nil {
		t.Fatal(err)
	}
	if dataURL != "data:image/png;base64,cG5n" {
		t.Fatalf("Capture dataURL = %q, want png data URL", dataURL)
	}
	if capture.targetSlot != "editor" {
		t.Fatalf("Capture target slot = %q, want editor", capture.targetSlot)
	}
}

func TestService_PickVariant(t *testing.T) {
	s, _ := newTestStore(t)
	svc := NewService(s, nil)
	// 1280×720 建档 (Variants[0]), 再加 1920×1080 (Variants[1], 不同分辨率 → 追加).
	guid, _ := svc.SaveTemplateCapture(pngDataURL(t, 8, 8), "x", "", nil, [2]int{1280, 720}, [4]float32{0, 0, 1, 1})
	if _, err := svc.AddTemplateVariant(guid, pngDataURL(t, 8, 8), [2]int{1920, 1080}, [4]float32{0, 0, 1, 1}); err != nil {
		t.Fatal(err)
	}

	// 精确命中各档 → 对应下标 + exact=true.
	if p, err := svc.PickVariant(guid, 1280, 720); err != nil || p.Index != 0 || !p.Exact {
		t.Errorf("pick 1280x720 = %+v, err %v; want {0 true}", p, err)
	}
	if p, err := svc.PickVariant(guid, 1920, 1080); err != nil || p.Index != 1 || !p.Exact {
		t.Errorf("pick 1920x1080 = %+v, err %v; want {1 true}", p, err)
	}
	// 无精确档 2560×1440 → 长边比最近 (1920 更近) → 下标 1, exact=false.
	if p, err := svc.PickVariant(guid, 2560, 1440); err != nil || p.Index != 1 || p.Exact {
		t.Errorf("pick 2560x1440 = %+v, err %v; want {1 false}", p, err)
	}
	// guid 不存在 → error.
	if _, err := svc.PickVariant("missing", 1920, 1080); err == nil {
		t.Error("pick missing guid should error")
	}
}

func TestService_RemoveVariant(t *testing.T) {
	s, _ := newTestStore(t)
	svc := NewService(s, nil)
	guid, _ := svc.SaveTemplateCapture(pngDataURL(t, 8, 8), "x", "", nil, [2]int{1280, 720}, [4]float32{0, 0, 1, 1})
	if _, err := svc.AddTemplateVariant(guid, pngDataURL(t, 8, 8), [2]int{1920, 1080}, [4]float32{0, 0, 1, 1}); err != nil {
		t.Fatal(err)
	}

	// 删 1280×720 一档 → 剩 1920×1080.
	if g, err := svc.RemoveVariant(guid, 1280, 720); err != nil || g != guid {
		t.Fatalf("RemoveVariant = %q, err %v", g, err)
	}
	rec, _ := svc.Get(guid)
	if len(rec.Variants) != 1 || rec.Variants[0].Resolution != [2]int{1920, 1080} {
		t.Fatalf("after remove, variants = %+v", rec.Variants)
	}

	// 仅剩 1 档 → 拒删 (走整删).
	if _, err := svc.RemoveVariant(guid, 1920, 1080); err == nil {
		t.Error("RemoveVariant on last variant should error")
	}

	// guid 不存在 → error.
	if _, err := svc.RemoveVariant("missing", 1, 1); err == nil {
		t.Error("RemoveVariant missing guid should error")
	}
}

func TestService_CurrentResolution(t *testing.T) {
	s, _ := newTestStore(t)

	svc := NewService(s, stubCaptureAdapter{res: [2]int{1600, 900}})
	if r, err := svc.CurrentResolution("c1"); err != nil || r != [2]int{1600, 900} {
		t.Errorf("CurrentResolution = %v, err %v; want [1600 900]", r, err)
	}

	// 窗口没开 → 透传 error.
	svcErr := NewService(s, stubCaptureAdapter{err: errors.New("窗口没开")})
	if _, err := svcErr.CurrentResolution("c1"); err == nil {
		t.Error("CurrentResolution should propagate adapter error")
	}

	// 未注入 adapter → error (不 panic).
	svcNil := NewService(s, nil)
	if _, err := svcNil.CurrentResolution("c1"); err == nil {
		t.Error("CurrentResolution with nil adapter should error")
	}
}
