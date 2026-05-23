package container

import (
	"testing"
)

// TestValidate_ColorBarTrack_EmptyRois: rois 空数组 → 报 INVALID_COLORBAR_ROIS.
func TestValidate_ColorBarTrack_EmptyRois(t *testing.T) {
	c := &Container{
		Graph: Graph{
			Nodes: []GraphNode{
				{ID: "cbt1", Kind: "ColorBarTrack", Config: map[string]any{"rois": []any{}}},
			},
		},
	}
	errs := validateColorBarTrack(c)
	if len(errs) != 1 || errs[0].Code != CodeInvalidColorBarROIs {
		t.Errorf("expected 1 INVALID_COLORBAR_ROIS, got %v", errs)
	}
}

// TestValidate_ColorBarTrack_MissingResolution: rois 项缺 resolution → 报 INVALID_COLORBAR_ROIS.
func TestValidate_ColorBarTrack_MissingResolution(t *testing.T) {
	c := &Container{
		Graph: Graph{
			Nodes: []GraphNode{
				{ID: "cbt1", Kind: "ColorBarTrack", Config: map[string]any{
					"rois": []any{
						map[string]any{"x": float64(100), "y": float64(100), "w": float64(50), "h": float64(50)},
					},
				}},
			},
		},
	}
	errs := validateColorBarTrack(c)
	if len(errs) != 1 || errs[0].Code != CodeInvalidColorBarROIs {
		t.Errorf("expected 1 INVALID_COLORBAR_ROIS, got %v", errs)
	}
}

// TestValidate_ColorBarTrack_ROIOutOfBounds: ROI 越界 resolution → 报 INVALID_COLORBAR_ROIS.
func TestValidate_ColorBarTrack_ROIOutOfBounds(t *testing.T) {
	c := &Container{
		Graph: Graph{
			Nodes: []GraphNode{
				{ID: "cbt1", Kind: "ColorBarTrack", Config: map[string]any{
					"rois": []any{
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
	errs := validateColorBarTrack(c)
	if len(errs) != 1 || errs[0].Code != CodeInvalidColorBarROIs {
		t.Errorf("expected 1 INVALID_COLORBAR_ROIS, got %v", errs)
	}
}

// TestValidate_ColorBarTrack_DuplicateResolution: 同 resolution 两条 → 报 DUPLICATE_COLORBAR_ROI warning.
func TestValidate_ColorBarTrack_DuplicateResolution(t *testing.T) {
	c := &Container{
		Graph: Graph{
			Nodes: []GraphNode{
				{ID: "cbt1", Kind: "ColorBarTrack", Config: map[string]any{
					"rois": []any{
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
	errs := validateColorBarTrack(c)
	hasDup := false
	for _, e := range errs {
		if e.Code == CodeDuplicateColorBarROI && e.Severity == SeverityWarning {
			hasDup = true
		}
	}
	if !hasDup {
		t.Errorf("expected DUPLICATE_COLORBAR_ROI warning, got %v", errs)
	}
}
