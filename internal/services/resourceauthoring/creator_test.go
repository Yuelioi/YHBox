package resourceauthoring

import (
	"bytes"
	"context"
	"encoding/base64"
	"image"
	"image/color"
	"image/png"
	"testing"

	"github.com/yottaapp/yotta/internal/blob"
	"github.com/yottaapp/yotta/internal/services/asset"
	"github.com/yottaapp/yotta/internal/services/inputclip"
	"github.com/yottaapp/yotta/internal/services/macro"
	"github.com/yottaapp/yotta/internal/storage"
	"github.com/yottaapp/yotta/internal/storage/catalog"
	"github.com/yottaapp/yotta/internal/workflow/schema"
)

func TestCreatorCreatesPortableImageWithoutAssetIdentity(t *testing.T) {
	creator, blobs, _ := newTestCreator(t)
	resource, err := creator.CreateImage(context.Background(), ImageDraft{
		Metadata: Metadata{
			Name: "  Submit  ", Category: " UI ", Tags: []string{"button", "BUTTON", " "},
		},
		DataURL: pngDataURL(t, 20, 10), Resolution: [2]int{100, 50},
		Region: [4]float32{0.1, 0.2, 0.2, 0.2},
	})
	if err != nil {
		t.Fatal(err)
	}
	if resource.Kind != "image" || resource.Name != "Submit" || resource.Category != "UI" ||
		len(resource.Tags) != 1 || resource.Image == nil || len(resource.Image.Variants) != 1 {
		t.Fatalf("image resource = %#v", resource)
	}
	variant := resource.Image.Variants[0]
	if variant.ID != "default" || variant.Resolution != [2]int{100, 50} ||
		variant.BBox != [4]int{10, 10, 30, 20} {
		t.Fatalf("image variant = %#v", variant)
	}
	if err := blobs.Verify(context.Background(), variant.Blob); err != nil {
		t.Fatalf("verify image blob: %v", err)
	}
}

func TestCreatorCreatesMacroAndInputClipMetadataFromCarrier(t *testing.T) {
	creator, blobs, _ := newTestCreator(t)
	document := macro.Document{
		SchemaVersion: macro.SchemaVersion, BaseResolution: [2]int{1280, 720},
		Actions: []macro.Action{{ID: "wait", Kind: macro.ActionSleep, DurationUs: 25_000}},
	}
	macroResource, err := creator.CreateMacro(context.Background(), MacroDraft{
		Metadata: Metadata{Name: "Wait"}, Document: document,
	})
	if err != nil {
		t.Fatal(err)
	}
	if macroResource.Macro == nil || macroResource.Macro.ActionCount != 1 ||
		macroResource.Macro.DurationUs != 25_000 ||
		macroResource.Macro.BaseResolution != document.BaseResolution {
		t.Fatalf("macro resource = %#v", macroResource)
	}
	if err := blobs.Verify(context.Background(), macroResource.Macro.Blob); err != nil {
		t.Fatalf("verify macro blob: %v", err)
	}

	clipResource, err := creator.CreateInputClip(context.Background(), InputClipDraft{
		Metadata: Metadata{Name: "Turn"},
		Clip: inputclip.InputClip{
			Meta: inputclip.ClipMeta{
				RecordingMode: inputclip.RecordingModePrecise, MouseMode: "relative",
				BaseResolution: [2]int{1920, 1080}, MouseCounts360: 1200, StopHotkeyVK: 0x7B,
			},
			Events: []inputclip.Event{
				{TUs: 0, Type: inputclip.EventTypeRawDelta, B: 2},
				{TUs: 5_000, Seq: 1, Type: inputclip.EventTypeRawDelta, B: -1},
			},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if clipResource.InputClip == nil || clipResource.InputClip.DurationUs != 5_000 ||
		clipResource.InputClip.EventCount != 2 || clipResource.InputClip.RecordingMode != "precise" ||
		clipResource.InputClip.MouseMode != "relative" || clipResource.InputClip.MouseCounts360 != 1200 {
		t.Fatalf("InputClip resource = %#v", clipResource)
	}
	if err := blobs.Verify(context.Background(), clipResource.InputClip.Blob); err != nil {
		t.Fatalf("verify InputClip blob: %v", err)
	}
}

func TestCreatorPromotesIndependentGlobalAssetsWithoutCopyingBlobs(t *testing.T) {
	creator, _, assets := newTestCreator(t)
	imageResource, err := creator.CreateImage(context.Background(), ImageDraft{
		Metadata: Metadata{Name: "Image"}, DataURL: pngDataURL(t, 8, 8),
		Resolution: [2]int{8, 8}, Region: [4]float32{0, 0, 1, 1},
	})
	if err != nil {
		t.Fatal(err)
	}
	macroResource, err := creator.CreateMacro(context.Background(), MacroDraft{
		Metadata: Metadata{Name: "Macro"},
		Document: macro.Document{
			SchemaVersion: macro.SchemaVersion, BaseResolution: [2]int{1280, 720},
			Actions: []macro.Action{{ID: "wait", Kind: macro.ActionSleep, DurationUs: 25_000}},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	clipResource, err := creator.CreateInputClip(context.Background(), InputClipDraft{
		Metadata: Metadata{Name: "Clip"},
		Clip: inputclip.InputClip{
			Meta: inputclip.ClipMeta{
				RecordingMode: inputclip.RecordingModePrecise, MouseMode: "relative",
				BaseResolution: [2]int{1920, 1080}, MouseCounts360: 900,
			},
			Events: []inputclip.Event{{TUs: 0, Type: inputclip.EventTypeRawDelta, B: 1}},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	for _, resource := range []schema.WorkflowResource{imageResource, macroResource, clipResource} {
		first, err := creator.Promote(context.Background(), resource)
		if err != nil {
			t.Fatal(err)
		}
		second, err := creator.Promote(context.Background(), resource)
		if err != nil {
			t.Fatal(err)
		}
		if first.GUID == second.GUID || first.Kind != second.Kind {
			t.Fatalf("promotions = %+v %+v", first, second)
		}
		for _, promotion := range []Promotion{first, second} {
			record, found := assets.Get(promotion.GUID)
			if !found || record.Origin.Kind != "workflow-resource" ||
				record.Origin.SourceID != resource.ID || record.Name != resource.Name {
				t.Fatalf("promoted record = %+v", record)
			}
			switch resource.Kind {
			case schema.ResourceImage:
				if len(record.Variants) != 1 ||
					record.Variants[0].Blob != resource.Image.Variants[0].Blob {
					t.Fatalf("promoted image = %+v", record)
				}
			case schema.ResourceMacro:
				if record.Blob == nil || *record.Blob != resource.Macro.Blob {
					t.Fatalf("promoted Macro = %+v", record)
				}
			case schema.ResourceInputClip:
				if record.Blob == nil || *record.Blob != resource.InputClip.Blob {
					t.Fatalf("promoted InputClip = %+v", record)
				}
			}
		}
	}
	if records := assets.List(); len(records) != 6 {
		t.Fatalf("promoted Global Asset count = %d", len(records))
	}
}

func TestCreatorRejectsPromotionWhenBlobIsMissing(t *testing.T) {
	creator, _, assets := newTestCreator(t)
	resource := schema.WorkflowResource{
		ID: "missing", Kind: schema.ResourceMacro, Name: "Missing",
		Macro: &schema.MacroResource{
			Blob: blob.BlobRef{
				MediaType: macro.MediaType,
				Digest:    "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
				Size:      42,
			},
			BaseResolution: [2]int{1280, 720},
		},
	}
	if _, err := creator.Promote(context.Background(), resource); err == nil {
		t.Fatal("Promote accepted a missing BlobRef")
	}
	if records := assets.List(); len(records) != 0 {
		t.Fatalf("failed promotion published records: %+v", records)
	}
}

func newTestCreator(t *testing.T) (*Creator, *blob.Store, *asset.Store) {
	t.Helper()
	roots, err := storage.Resolve(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	foundation, err := catalog.Open(context.Background(), roots)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = foundation.Close() })
	blobs, err := blob.Open(
		roots.Objects,
		blob.Limits{MaxBlobBytes: 1 << 20, MaxTotalBytes: 4 << 20},
		foundation.Objects(),
	)
	if err != nil {
		t.Fatal(err)
	}
	assets, err := asset.NewStore(foundation.Assets(), foundation.Objects(), blobs)
	if err != nil {
		t.Fatal(err)
	}
	creator, err := NewCreator(blobs, assets)
	if err != nil {
		t.Fatal(err)
	}
	return creator, blobs, assets
}

func pngDataURL(t *testing.T, width, height int) string {
	t.Helper()
	value := image.NewRGBA(image.Rect(0, 0, width, height))
	value.Set(0, 0, color.White)
	var encoded bytes.Buffer
	if err := png.Encode(&encoded, value); err != nil {
		t.Fatal(err)
	}
	return pngDataURLPrefix + base64.StdEncoding.EncodeToString(encoded.Bytes())
}
