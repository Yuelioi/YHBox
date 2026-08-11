package asset

import (
	"bytes"
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"image"
	"image/color"
	"image/png"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/yottaapp/yotta/internal/automation/target"
	"github.com/yottaapp/yotta/internal/blob"
	"github.com/yottaapp/yotta/internal/storage"
	"github.com/yottaapp/yotta/internal/storage/catalog"
)

// stubCaptureAdapter 测试用: ResolveTarget 返预设值/错误; Capture 不被这些测试触及.
type stubCaptureAdapter struct {
	res [2]int
	err error
}

func (s stubCaptureAdapter) CapturePNG(context.Context, string) ([]byte, error) { return nil, nil }
func (s stubCaptureAdapter) ResolveTarget(context.Context, string) (target.Target, error) {
	return target.Target{Resolution: target.Size{W: s.res[0], H: s.res[1]}}, s.err
}

type recordingCaptureAdapter struct {
	targetSlot string
}

func (r *recordingCaptureAdapter) CapturePNG(_ context.Context, targetSlot string) ([]byte, error) {
	r.targetSlot = targetSlot
	return []byte("png"), nil
}

func (r *recordingCaptureAdapter) ResolveTarget(context.Context, string) (target.Target, error) {
	return target.Target{}, nil
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
	svc := NewService(s, nil, nil)

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

	list, err := svc.List()
	if err != nil {
		t.Fatal(err)
	}
	if len(list) != 1 || list[0].GUID != guid || list[0].VariantCount != 1 || len(list[0].Variants) != 1 || list[0].Variants[0].Blob != v.Blob {
		t.Fatalf("List = %+v", list)
	}
}

func TestServiceListPropagatesCatalogFailure(t *testing.T) {
	dir := t.TempDir()
	roots, err := storage.Resolve(filepath.Join(dir, "profile"))
	if err != nil {
		t.Fatal(err)
	}
	foundation, err := catalog.Open(context.Background(), roots)
	if err != nil {
		t.Fatal(err)
	}
	store, err := NewStore(foundation.Assets(), foundation.Objects(), newTestBlobStore(t, dir, foundation.Objects()))
	if err != nil {
		t.Fatal(err)
	}
	if err := foundation.Close(); err != nil {
		t.Fatal(err)
	}
	if _, err := NewService(store, nil, nil).List(); err == nil {
		t.Fatal("asset List hid the closed Catalog as an empty library")
	}
}

func TestServicePreviewBlobReturnsBoundedPNG(t *testing.T) {
	s, _ := newTestStore(t)
	svc := NewService(s, nil, nil)
	guid, err := svc.SaveTemplateCapture(pngDataURL(t, 800, 400), "preview", "", nil, [2]int{800, 400}, [4]float32{0, 0, 1, 1})
	if err != nil {
		t.Fatal(err)
	}
	record, err := svc.Get(guid)
	if err != nil {
		t.Fatal(err)
	}
	preview, err := svc.PreviewBlob(record.Variants[0].Blob)
	if err != nil {
		t.Fatal(err)
	}
	if preview.MediaType != "image/png" || preview.Width != 256 || preview.Height != 128 {
		t.Fatalf("preview = %+v", preview)
	}
	decoded, err := base64.StdEncoding.DecodeString(preview.Base64)
	if err != nil {
		t.Fatal(err)
	}
	config, err := png.DecodeConfig(bytes.NewReader(decoded))
	if err != nil || config.Width != 256 || config.Height != 128 || len(decoded) > previewMaxOutputBytes {
		t.Fatalf("decoded preview config=%+v bytes=%d err=%v", config, len(decoded), err)
	}
}

func TestServicePreviewBlobRejectsUntrustedShapeBeforeRead(t *testing.T) {
	s, _ := newTestStore(t)
	svc := NewService(s, nil, nil)
	for _, ref := range []blob.BlobRef{
		{MediaType: "application/octet-stream", Digest: "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa", Size: 1},
		{MediaType: "image/png", Digest: "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa", Size: previewMaxSourceBytes + 1},
	} {
		if _, err := svc.PreviewBlob(ref); err == nil {
			t.Fatalf("PreviewBlob(%+v) succeeded", ref)
		}
	}
}

func TestService_SaveTemplateCapture_PersistsCategory(t *testing.T) {
	s, _ := newTestStore(t)
	svc := NewService(s, nil, nil)

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
	svc := NewService(s, nil, nil)
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
	svc := NewService(s, capture, nil)

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
	svc := NewService(s, nil, nil)
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
	svc := NewService(s, nil, nil)
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

	svc := NewService(s, stubCaptureAdapter{res: [2]int{1600, 900}}, nil)
	if r, err := svc.CurrentResolution("c1"); err != nil || r != [2]int{1600, 900} {
		t.Errorf("CurrentResolution = %v, err %v; want [1600 900]", r, err)
	}

	// 窗口没开 → 透传 error.
	svcErr := NewService(s, stubCaptureAdapter{err: errors.New("窗口没开")}, nil)
	if _, err := svcErr.CurrentResolution("c1"); err == nil {
		t.Error("CurrentResolution should propagate adapter error")
	}

	// 未注入 adapter → error (不 panic).
	svcNil := NewService(s, nil, nil)
	if _, err := svcNil.CurrentResolution("c1"); err == nil {
		t.Error("CurrentResolution with nil adapter should error")
	}
}

func TestServiceQueryAndBatchManagement(t *testing.T) {
	store, _ := newTestStore(t)
	service := NewService(store, nil, nil)
	alpha, err := service.SaveTemplateCapture(pngDataURL(t, 8, 8), "Alpha", "UI", []string{"button", "common"}, [2]int{1280, 720}, [4]float32{0, 0, 1, 1})
	if err != nil {
		t.Fatal(err)
	}
	beta, err := service.SaveTemplateCapture(pngDataURL(t, 8, 8), "Beta", "Game", []string{"common"}, [2]int{1920, 1080}, [4]float32{0, 0, 1, 1})
	if err != nil {
		t.Fatal(err)
	}
	page, err := service.QueryAssets(AssetQuery{Kind: KindTemplate, Search: "a", Tags: []string{"common"}, Sort: "name_desc", Page: 1, PageSize: 1})
	if err != nil || page.Total != 2 || len(page.Items) != 1 || page.Items[0].Name != "Beta" {
		t.Fatalf("QueryAssets() = %#v, %v", page, err)
	}
	if len(page.Categories) != 2 || page.Categories[0].Value != "Game" || page.Categories[0].Count != 1 ||
		len(page.Tags) != 2 || page.Tags[1].Value != "common" || page.Tags[1].Count != 2 {
		t.Fatalf("QueryAssets facets = categories %#v, tags %#v", page.Categories, page.Tags)
	}
	if _, err := service.QueryAssets(AssetQuery{
		CreatedSince: "not-a-date", Page: 1, PageSize: 20,
	}); err == nil {
		t.Fatal("QueryAssets accepted an invalid createdSince filter")
	}
	updated := service.BatchUpdateMeta([]BatchMetaRequest{
		{GUID: alpha, Category: "Shared", Tags: []string{"updated", "Updated"}},
		{GUID: beta, Category: "Shared", Tags: []string{"updated"}},
	})
	if len(updated) != 2 || !updated[0].Updated || !updated[1].Updated {
		t.Fatalf("BatchUpdateMeta() = %#v", updated)
	}
	shared, err := service.QueryAssets(AssetQuery{Category: "shared", Tags: []string{"UPDATED"}, Page: 1, PageSize: 20})
	if err != nil || shared.Total != 2 || len(shared.Items[0].Tags) != 1 {
		t.Fatalf("updated QueryAssets() = %#v, %v", shared, err)
	}
	deleted := service.BatchDelete([]string{alpha, "missing", beta})
	if len(deleted) != 3 || !deleted[0].Deleted || deleted[1].Error == "" || !deleted[2].Deleted {
		t.Fatalf("BatchDelete() = %#v", deleted)
	}
}

func TestServiceAssetPickerQueryScalesAndResolvesExactVariant(t *testing.T) {
	store, _ := newTestStore(t)
	for index := 0; index < 1_000; index++ {
		guid := fmt.Sprintf("asset-%04d", index)
		first := testBlobRef(guid + "-720")
		second := testBlobRef(guid + "-1080")
		observeTestBlob(t, store, first)
		observeTestBlob(t, store, second)
		if err := store.putRecord(AssetRecord{
			SchemaVersion: RecordSchemaVersion, GUID: guid, Kind: KindTemplate,
			Name: fmt.Sprintf("Asset %04d", index), Category: "fixture", Tags: []string{"common"},
			Origin: Origin{Kind: "user"}, CreatedAt: time.Unix(int64(index), 0).UTC(),
			Variants: []Variant{
				{Resolution: [2]int{1280, 720}, Blob: first},
				{Resolution: [2]int{1920, 1080}, Blob: second},
			},
		}); err != nil {
			t.Fatal(err)
		}
	}

	service := NewService(store, nil, nil)
	page, err := service.QueryAssets(AssetQuery{
		Kind: KindTemplate, Category: "fixture", Tags: []string{"COMMON"}, Sort: "recent_desc",
		Page: 1, PageSize: 20, ThumbnailBudget: 4, RecentGUIDs: []string{"asset-0900", "asset-0100"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if page.Total != 1_000 || len(page.Items) != 20 || page.Items[0].GUID != "asset-0900" || page.Items[1].GUID != "asset-0100" || page.Revision == 0 {
		t.Fatalf("picker page = %#v", page)
	}
	if len(page.Categories) != 1 || page.Categories[0].Value != "fixture" || page.Categories[0].Count != 1_000 ||
		len(page.Tags) != 1 || page.Tags[0].Value != "common" || page.Tags[0].Count != 1_000 {
		t.Fatalf("picker facets = categories %#v, tags %#v", page.Categories, page.Tags)
	}
	for index, item := range page.Items {
		if (index < 4) != (item.Thumbnail != nil) {
			t.Fatalf("thumbnail budget at item %d = %#v", index, item.Thumbnail)
		}
	}
	located, err := service.QueryAssets(AssetQuery{
		Kind: KindTemplate, Search: "asset-0900", Sort: "name_asc",
		Page: 1, PageSize: 20, ThumbnailBudget: 1,
	})
	if err != nil || located.Total != 1 || len(located.Items) != 1 || located.Items[0].GUID != "asset-0900" {
		t.Fatalf("GUID lookup = %#v, %v", located, err)
	}
	binding, err := service.ResolveBinding(page.Items[0].Variants[1].Blob)
	if err != nil || !binding.Found || binding.GUID != "asset-0900" || binding.Resolution != [2]int{1920, 1080} || binding.MatchCount != 1 {
		t.Fatalf("ResolveBinding() = %#v, %v", binding, err)
	}
	if err := store.DeleteRecord(binding.GUID); err != nil {
		t.Fatal(err)
	}
	stale, err := service.ResolveBinding(binding.Blob)
	if err != nil || stale.Found {
		t.Fatalf("stale ResolveBinding() = %#v, %v", stale, err)
	}
}

func TestServiceEmitsOneRevisionedAssetInvalidationPerMutation(t *testing.T) {
	store, _ := newTestStore(t)
	revisions := make([]uint64, 0)
	service := NewService(store, nil, nil, func(name string, data any) {
		if name != "asset:changed" {
			t.Fatalf("event name = %q", name)
		}
		payload, ok := data.(map[string]any)
		if !ok {
			t.Fatalf("event payload = %#v", data)
		}
		revisions = append(revisions, payload["revision"].(uint64))
	})
	guid, err := service.SaveTemplateCapture(pngDataURL(t, 8, 8), "Asset", "", nil, [2]int{8, 8}, [4]float32{0, 0, 1, 1})
	if err != nil {
		t.Fatal(err)
	}
	if err := service.UpdateMeta(guid, "Renamed", "", "", nil); err != nil {
		t.Fatal(err)
	}
	if err := service.Delete(guid); err != nil {
		t.Fatal(err)
	}
	if len(revisions) != 3 || revisions[0] >= revisions[1] || revisions[1] >= revisions[2] {
		t.Fatalf("asset revisions = %v", revisions)
	}
}

type testDurableReferences struct{ refs []blob.BlobRef }

func (s *testDurableReferences) WithDurableBlobReferences(_ context.Context, visit func([]blob.BlobRef) error) error {
	return visit(append([]blob.BlobRef(nil), s.refs...))
}

func TestServiceCleanupProtectsAllRootsAndRejectsStalePreview(t *testing.T) {
	ctx := context.Background()
	store, _ := newTestStore(t)
	assetID, err := NewService(store, nil, nil).SaveTemplateCapture(pngDataURL(t, 8, 8), "live asset", "", nil, [2]int{8, 8}, [4]float32{0, 0, 1, 1})
	if err != nil {
		t.Fatal(err)
	}
	assetRecord, _ := store.Get(assetID)
	external, err := store.blobs.Put(ctx, "application/octet-stream", strings.NewReader("workflow root"))
	if err != nil {
		t.Fatal(err)
	}
	orphan, err := store.blobs.Put(ctx, "application/octet-stream", strings.NewReader("orphan one"))
	if err != nil {
		t.Fatal(err)
	}
	references := &testDurableReferences{refs: []blob.BlobRef{external}}
	service := NewService(store, nil, references)
	preview, err := service.PreviewCleanup()
	if err != nil || preview.CandidateCount != 1 || preview.LiveCount != 2 {
		t.Fatalf("PreviewCleanup() = %#v, %v", preview, err)
	}
	secondOrphan, err := store.blobs.Put(ctx, "application/octet-stream", strings.NewReader("orphan two"))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := service.CommitCleanup(preview.Token); !errors.Is(err, ErrCleanupPreviewStale) {
		t.Fatalf("stale CommitCleanup() error = %v", err)
	}
	preview, err = service.PreviewCleanup()
	if err != nil || preview.CandidateCount != 2 {
		t.Fatalf("second PreviewCleanup() = %#v, %v", preview, err)
	}
	result, err := service.CommitCleanup(preview.Token)
	if err != nil || result.Reclaimed != 2 {
		t.Fatalf("CommitCleanup() = %#v, %v", result, err)
	}
	for _, live := range []blob.BlobRef{assetRecord.Variants[0].Blob, external} {
		if err := store.blobs.Verify(ctx, live); err != nil {
			t.Fatalf("live blob %s was reclaimed: %v", live.Digest, err)
		}
	}
	for _, dead := range []blob.BlobRef{orphan, secondOrphan} {
		if err := store.blobs.Verify(ctx, dead); err == nil {
			t.Fatalf("orphan blob %s was retained", dead.Digest)
		}
	}
}
