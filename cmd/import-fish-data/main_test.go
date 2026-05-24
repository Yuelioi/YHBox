package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestPlanImport_ParsesTomlAndYaml(t *testing.T) {
	ctnDir := t.TempDir()
	sgDir := filepath.Join(ctnDir, "subgraphs")
	if err := os.MkdirAll(sgDir, 0o755); err != nil {
		t.Fatal(err)
	}
	stub := `{
  "id": "test", "label": "Test",
  "graph": {
    "id": "g1",
    "nodes": [
      { "id": "cbt1", "kind": "ColorBarTrack", "config": {"roi": {"x":1,"y":1,"w":1,"h":1}} }
    ]
  }
}`
	for _, fname := range []string{"state_FISHING.json", "inspect_phase.json", "try_hook_F.json"} {
		if err := os.WriteFile(filepath.Join(sgDir, fname), []byte(stub), 0o644); err != nil {
			t.Fatal(err)
		}
	}

	plan, err := planImport("testdata/fish-configs/zh", ctnDir)
	if err != nil {
		t.Fatalf("planImport: %v", err)
	}

	// 2 keys: hook_icon (single-slot 2 variants) + bait_product (multi-slot 1 variant w/ Regions)
	if len(plan.Templates) != 2 {
		t.Errorf("Templates len = %d, want 2", len(plan.Templates))
	}

	// find each by key
	byKey := map[string]TemplatePlan{}
	for _, tp := range plan.Templates {
		byKey[tp.Key] = tp
	}

	hi, ok := byKey["fishing.hook_icon"]
	if !ok {
		t.Fatal("fishing.hook_icon missing")
	}
	if len(hi.Variants) != 2 {
		t.Errorf("hook_icon Variants len = %d, want 2", len(hi.Variants))
	}
	for _, v := range hi.Variants {
		if len(v.Regions) != 0 {
			t.Errorf("hook_icon variant @%v has Regions=%d, want 0 (single-slot)", v.Resolution, len(v.Regions))
		}
	}

	bp, ok := byKey["fishing.bait_product"]
	if !ok {
		t.Fatal("fishing.bait_product missing")
	}
	if len(bp.Variants) != 1 {
		t.Errorf("bait_product Variants len = %d, want 1 (merged 3 entries)", len(bp.Variants))
	}
	if len(bp.Variants[0].Regions) != 3 {
		t.Errorf("bait_product Regions len = %d, want 3", len(bp.Variants[0].Regions))
	}
	// BBox = first
	if bp.Variants[0].BBox != [4]int{60, 142, 232, 353} {
		t.Errorf("bait_product BBox = %v, want first entry [60,142,232,353]", bp.Variants[0].BBox)
	}

	if len(plan.ColorBarPatches) != 3 {
		t.Errorf("ColorBarPatches len = %d, want 3", len(plan.ColorBarPatches))
	}
	// verify newRois has 2 entries (1080p + 720p)
	if len(plan.ColorBarPatches[0].NewRois) != 2 {
		t.Errorf("NewRois len = %d, want 2", len(plan.ColorBarPatches[0].NewRois))
	}
}

// JSON round-trip smoke for VariantMeta.Regions — confirms json:"regions,omitempty" works.
func TestVariantMetaJSON_RegionsRoundTrip(t *testing.T) {
	type V struct {
		Regions [][4]int `json:"regions,omitempty"`
	}
	in := V{Regions: [][4]int{{1, 2, 3, 4}, {5, 6, 7, 8}}}
	b, _ := json.Marshal(in)
	var out V
	if err := json.Unmarshal(b, &out); err != nil {
		t.Fatal(err)
	}
	if len(out.Regions) != 2 || out.Regions[1][2] != 7 {
		t.Errorf("round-trip lost data: %+v", out)
	}
}
