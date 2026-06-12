package asset

import (
	"bytes"
	"encoding/base64"
	"errors"
	"image"
	"image/color"
	"image/png"
	"testing"
)

// stubCaptureAdapter 测试用: Resolution 返预设值/错误; Capture 不被这些测试触及.
type stubCaptureAdapter struct {
	res [2]int
	err error
}

func (s stubCaptureAdapter) Capture(string) ([]byte, error)         { return nil, nil }
func (s stubCaptureAdapter) Resolution(string) ([2]int, error)      { return s.res, s.err }

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
	s, _ := NewStore(t.TempDir())
	svc := NewService(s, nil)

	guid, err := svc.SaveTemplateCapture(pngDataURL(t, 32, 16), "登录按钮", []string{"按钮"}, [2]int{1920, 1080}, [4]float32{0.1, 0.2, 0.3, 0.4})
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
	if rec.Kind != KindTemplate || rec.Name != "登录按钮" || len(rec.Variants) != 1 {
		t.Fatalf("bad record: %+v", rec)
	}
	// bbox 换算: x1 = 0.1*1920=192, y1=0.2*1080=216, x2=(0.1+0.3)*1920=768, y2=(0.2+0.4)*1080=648
	v := rec.Variants[0]
	if v.BBox != [4]int{192, 216, 768, 648} {
		t.Errorf("bbox = %v, want [192 216 768 648]", v.BBox)
	}

	list := svc.List()
	if len(list) != 1 || list[0].GUID != guid || list[0].VariantCount != 1 || list[0].FirstBlobSha != v.Blob {
		t.Fatalf("List = %+v", list)
	}
}

func TestService_RenameDelete(t *testing.T) {
	s, _ := NewStore(t.TempDir())
	svc := NewService(s, nil)
	guid, _ := svc.SaveTemplateCapture(pngDataURL(t, 8, 8), "旧名", nil, [2]int{1280, 720}, [4]float32{0, 0, 1, 1})

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

	// 注入引用扫描 — Delete 返回引用列表 (不阻断).
	svc.SetReferrerScanner(func(g string) []Referrer {
		return []Referrer{{ContainerID: "c1", NodeID: "n1", NodeKind: "CheckTemplate"}}
	})
	refs, err := svc.Delete(guid)
	if err != nil {
		t.Fatal(err)
	}
	if len(refs) != 1 || refs[0].ContainerID != "c1" {
		t.Errorf("referrers = %+v", refs)
	}
	if _, err := svc.Get(guid); err == nil {
		t.Error("asset should be deleted")
	}
}

func TestService_ReadBlobDataURL(t *testing.T) {
	s, _ := NewStore(t.TempDir())
	svc := NewService(s, nil)
	guid, _ := svc.SaveTemplateCapture(pngDataURL(t, 8, 8), "x", nil, [2]int{800, 600}, [4]float32{0, 0, 1, 1})
	rec, _ := svc.Get(guid)
	url, err := svc.ReadBlobDataURL(rec.Variants[0].Blob)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.HasPrefix([]byte(url), []byte("data:image/png;base64,")) {
		t.Errorf("bad data url prefix: %.40s", url)
	}
}

func TestService_PickVariant(t *testing.T) {
	s, _ := NewStore(t.TempDir())
	svc := NewService(s, nil)
	// 1280×720 建档 (Variants[0]), 再加 1920×1080 (Variants[1], 不同分辨率 → 追加).
	guid, _ := svc.SaveTemplateCapture(pngDataURL(t, 8, 8), "x", nil, [2]int{1280, 720}, [4]float32{0, 0, 1, 1})
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
	s, _ := NewStore(t.TempDir())
	svc := NewService(s, nil)
	guid, _ := svc.SaveTemplateCapture(pngDataURL(t, 8, 8), "x", nil, [2]int{1280, 720}, [4]float32{0, 0, 1, 1})
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
	s, _ := NewStore(t.TempDir())

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
