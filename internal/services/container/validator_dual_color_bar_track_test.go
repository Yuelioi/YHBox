package container

import (
	"testing"
)

// TestValidate_DualColorBarTrack_EmptyRois: rois 空数组 → 报 INVALID_DUALBAR_ROIS.
func TestValidate_DualColorBarTrack_EmptyRois(t *testing.T) {
	c := &Container{
		Graph: Graph{
			Nodes: []GraphNode{
				{ID: "cbt1", Kind: "DualColorBarTrack", Config: map[string]any{"Rois": []any{}}},
			},
		},
	}
	errs := validateDualColorBarTrack(c)
	if len(errs) != 1 || errs[0].Code != CodeInvalidDualBarROIs {
		t.Errorf("expected 1 INVALID_DUALBAR_ROIS, got %v", errs)
	}
}

// TestValidate_DualColorBarTrack_MissingResolution: rois 项缺 resolution → 报 INVALID_DUALBAR_ROIS.
func TestValidate_DualColorBarTrack_MissingResolution(t *testing.T) {
	c := &Container{
		Graph: Graph{
			Nodes: []GraphNode{
				{ID: "cbt1", Kind: "DualColorBarTrack", Config: map[string]any{
					"Rois": []any{
						map[string]any{"x": float64(100), "y": float64(100), "w": float64(50), "h": float64(50)},
					},
				}},
			},
		},
	}
	errs := validateDualColorBarTrack(c)
	if len(errs) != 1 || errs[0].Code != CodeInvalidDualBarROIs {
		t.Errorf("expected 1 INVALID_DUALBAR_ROIS, got %v", errs)
	}
}

// TestValidate_DualColorBarTrack_ROIOutOfBounds: ROI 越界 resolution → 报 INVALID_DUALBAR_ROIS.
func TestValidate_DualColorBarTrack_ROIOutOfBounds(t *testing.T) {
	c := &Container{
		Graph: Graph{
			Nodes: []GraphNode{
				{ID: "cbt1", Kind: "DualColorBarTrack", Config: map[string]any{
					"Rois": []any{
						map[string]any{
							"resolution": []any{float64(1920), float64(1080)},
							"x":          float64(1900),
							"y":          float64(1070),
							"w":          float64(50),
							"h":          float64(50),
						},
					},
				}},
			},
		},
	}
	errs := validateDualColorBarTrack(c)
	if len(errs) != 1 || errs[0].Code != CodeInvalidDualBarROIs {
		t.Errorf("expected 1 INVALID_DUALBAR_ROIS, got %v", errs)
	}
}

// TestValidate_DualColorBarTrack_DuplicateResolution: 同 resolution 两条 → 报 DUPLICATE_DUALBAR_ROI warning.
func TestValidate_DualColorBarTrack_DuplicateResolution(t *testing.T) {
	c := &Container{
		Graph: Graph{
			Nodes: []GraphNode{
				{ID: "cbt1", Kind: "DualColorBarTrack", Config: map[string]any{
					"Rois": []any{
						map[string]any{
							"resolution": []any{float64(1920), float64(1080)},
							"x":          float64(100),
							"y":          float64(100),
							"w":          float64(50),
							"h":          float64(50),
						},
						map[string]any{
							"resolution": []any{float64(1920), float64(1080)},
							"x":          float64(200),
							"y":          float64(200),
							"w":          float64(50),
							"h":          float64(50),
						},
					},
				}},
			},
		},
	}
	errs := validateDualColorBarTrack(c)
	hasDup := false
	for _, e := range errs {
		if e.Code == CodeDuplicateDualBarROI && e.Severity == SeverityWarning {
			hasDup = true
		}
	}
	if !hasDup {
		t.Errorf("expected DUPLICATE_DUALBAR_ROI warning, got %v", errs)
	}
}
