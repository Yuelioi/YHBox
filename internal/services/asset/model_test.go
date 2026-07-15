// internal/services/asset/model_test.go
package asset

import (
	"encoding/json"
	"testing"
	"time"
)

func TestRoundTrip_AssetRecord(t *testing.T) {
	now := time.Date(2025, 1, 2, 3, 4, 5, 0, time.UTC)
	orig := AssetRecord{
		GUID: "guid-abc",
		Kind: KindTemplate,
		Name: "测试模板",
		Tags: []string{"ui", "button"},
		Origin: Origin{
			Kind:     "user",
			SourceID: "src-1",
		},
		Variants: []Variant{
			{
				Resolution: [2]int{1920, 1080},
				BBox:       [4]int{10, 20, 100, 80},
				Regions:    [][4]int{{10, 20, 50, 80}, {50, 20, 100, 80}},
				Blob:       testBlobRef("deadbeef"),
			},
		},
		CreatedAt: now,
	}

	b, err := json.Marshal(orig)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	var got AssetRecord
	if err := json.Unmarshal(b, &got); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	if got.GUID != orig.GUID {
		t.Errorf("GUID: got %q want %q", got.GUID, orig.GUID)
	}
	if got.Kind != KindTemplate {
		t.Errorf("Kind: got %q want %q", got.Kind, KindTemplate)
	}
	if got.Name != orig.Name {
		t.Errorf("Name: got %q want %q", got.Name, orig.Name)
	}
	if len(got.Tags) != 2 || got.Tags[0] != "ui" || got.Tags[1] != "button" {
		t.Errorf("Tags: got %v", got.Tags)
	}
	if got.Origin.Kind != "user" || got.Origin.SourceID != "src-1" {
		t.Errorf("Origin: got %+v", got.Origin)
	}
	if len(got.Variants) != 1 {
		t.Fatalf("Variants len: got %d want 1", len(got.Variants))
	}
	v := got.Variants[0]
	if v.Resolution != [2]int{1920, 1080} {
		t.Errorf("Variant.Resolution: got %v", v.Resolution)
	}
	if v.BBox != [4]int{10, 20, 100, 80} {
		t.Errorf("Variant.BBox: got %v", v.BBox)
	}
	if len(v.Regions) != 2 {
		t.Errorf("Variant.Regions len: got %d", len(v.Regions))
	}
	if v.Blob != testBlobRef("deadbeef") {
		t.Errorf("Variant.Blob: got %#v", v.Blob)
	}
	if !got.CreatedAt.Equal(now) {
		t.Errorf("CreatedAt: got %v want %v", got.CreatedAt, now)
	}
}

func TestRoundTrip_AssetRecord_Clip(t *testing.T) {
	orig := AssetRecord{
		GUID:      "guid-clip",
		Kind:      KindClip,
		Name:      "测试 clip",
		Origin:    Origin{Kind: "imported"},
		Blob:      blobPtr(testBlobRef("cafebabe")),
		CreatedAt: time.Now().UTC(),
	}

	b, err := json.Marshal(orig)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	var got AssetRecord
	if err := json.Unmarshal(b, &got); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	if got.Kind != KindClip {
		t.Errorf("Kind: got %q", got.Kind)
	}
	if got.Blob == nil || *got.Blob != testBlobRef("cafebabe") {
		t.Errorf("Blob: got %#v", got.Blob)
	}
	// Variants 和 Tags 的 omitempty: clip 无 Variants，序列化后应为 nil
	if got.Variants != nil {
		t.Errorf("Variants should be nil for clip, got %v", got.Variants)
	}
	if got.Tags != nil {
		t.Errorf("Tags should be nil when omitted, got %v", got.Tags)
	}
}

func TestRoundTrip_Variant_NoRegions(t *testing.T) {
	v := Variant{
		Resolution: [2]int{1280, 720},
		BBox:       [4]int{0, 0, 100, 100},
		Blob:       testBlobRef("abc123"),
	}
	b, err := json.Marshal(v)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	var got Variant
	if err := json.Unmarshal(b, &got); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if got.Regions != nil {
		t.Errorf("Regions should be nil when omitempty, got %v", got.Regions)
	}
}
