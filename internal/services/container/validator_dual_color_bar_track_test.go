package container

import (
	"testing"
)

// validateDualColorBarTrack は Geometry 移行後 no-op. 以下は移行後の期待動作を確認するテスト.

// TestValidate_DualColorBarTrack_NoopValidator: validateDualColorBarTrack 常に nil を返す.
func TestValidate_DualColorBarTrack_NoopValidator(t *testing.T) {
	c := &Container{
		Graph: Graph{
			Nodes: []GraphNode{
				{ID: "cbt1", Kind: "DualColorBarTrack", Config: map[string]any{}},
			},
		},
	}
	errs := validateDualColorBarTrack(c)
	if len(errs) != 0 {
		t.Errorf("validateDualColorBarTrack should be no-op, got %v", errs)
	}
}

// TestLiteralMatchesType_Geometry: map 値は geometry として受け入れる.
func TestLiteralMatchesType_Geometry(t *testing.T) {
	geo := map[string]any{
		"pct": map[string]any{"x": 0.0, "y": 0.0, "w": 0.0, "h": 0.0},
		"overrides": []any{
			map[string]any{
				"resolution": map[string]any{"w": float64(1920), "h": float64(1080)},
				"px":         map[string]any{"x": float64(612), "y": float64(69), "w": float64(704), "h": float64(12)},
			},
		},
	}
	if !literalMatchesType(geo, "geometry") {
		t.Error("expected map[string]any to match geometry")
	}
	if literalMatchesType("not-a-map", "geometry") {
		t.Error("expected string to NOT match geometry")
	}
	if literalMatchesType(42.0, "geometry") {
		t.Error("expected float64 to NOT match geometry")
	}
}
