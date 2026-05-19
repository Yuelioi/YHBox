package container

import (
	"testing"
)

// TestValidateColorBarTrack_MissingROI: 缺 roi config → 报 INVALID_ROI.
func TestValidateColorBarTrack_MissingROI(t *testing.T) {
	errs := validateColorBarTrack(&GraphNode{
		ID:     "cbt1",
		Kind:   "ColorBarTrack",
		Config: map[string]any{},
	})
	if len(errs) != 1 {
		t.Fatalf("应报 1 错 (missing roi), got %d: %+v", len(errs), errs)
	}
	if errs[0].Code != CodeInvalidROI {
		t.Errorf("应是 INVALID_ROI, got: %s", errs[0].Code)
	}
}

// TestValidateColorBarTrack_InvalidROI_ZeroSize: roi w/h=0 → 报 INVALID_ROI.
func TestValidateColorBarTrack_InvalidROI_ZeroSize(t *testing.T) {
	errs := validateColorBarTrack(&GraphNode{
		ID:   "cbt2",
		Kind: "ColorBarTrack",
		Config: map[string]any{
			"roi": map[string]any{"x": 0.0, "y": 0.0, "w": 0.0, "h": 0.0},
		},
	})
	if len(errs) != 1 {
		t.Fatalf("应报 1 错 (zero roi), got %d: %+v", len(errs), errs)
	}
	if errs[0].Code != CodeInvalidROI {
		t.Errorf("应是 INVALID_ROI, got: %s", errs[0].Code)
	}
}

// TestValidateColorBarTrack_HappyPath: 有效 roi → 无错.
func TestValidateColorBarTrack_HappyPath(t *testing.T) {
	errs := validateColorBarTrack(&GraphNode{
		ID:   "cbt3",
		Kind: "ColorBarTrack",
		Config: map[string]any{
			"roi": map[string]any{"x": 100.0, "y": 200.0, "w": 400.0, "h": 30.0},
		},
	})
	if len(errs) != 0 {
		t.Errorf("合法 roi 不应报错, got: %+v", errs)
	}
}
