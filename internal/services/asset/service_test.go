package asset

import (
	"bytes"
	"encoding/base64"
	"image"
	"image/color"
	"image/png"
	"testing"
)

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

	guid, err := svc.SaveTemplateCapture(pngDataURL(t, 32, 16), "登录按钮", [2]int{1920, 1080}, [4]float32{0.1, 0.2, 0.3, 0.4})
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
	guid, _ := svc.SaveTemplateCapture(pngDataURL(t, 8, 8), "旧名", [2]int{1280, 720}, [4]float32{0, 0, 1, 1})

	if err := svc.Rename(guid, "新名"); err != nil {
		t.Fatal(err)
	}
	rec, _ := svc.Get(guid)
	if rec.Name != "新名" {
		t.Errorf("rename failed: %q", rec.Name)
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
	guid, _ := svc.SaveTemplateCapture(pngDataURL(t, 8, 8), "x", [2]int{800, 600}, [4]float32{0, 0, 1, 1})
	rec, _ := svc.Get(guid)
	url, err := svc.ReadBlobDataURL(rec.Variants[0].Blob)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.HasPrefix([]byte(url), []byte("data:image/png;base64,")) {
		t.Errorf("bad data url prefix: %.40s", url)
	}
}
