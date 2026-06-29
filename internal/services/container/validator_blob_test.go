package container

import "testing"

func TestValidateDetectColorBlobs_BadSort(t *testing.T) {
	n := &GraphNode{ID: "n1", Kind: "DetectColorBlobs",
		Config: map[string]any{"literal": map[string]any{"Sort": "biggest"}}}
	errs := validateDetectColorBlobs(n)
	found := false
	for _, e := range errs {
		if e.Code == CodeInvalidSortMode {
			found = true
		}
	}
	if !found {
		t.Errorf("want INVALID_SORT_MODE, got %+v", errs)
	}
}

func TestValidateDetectColorBlobs_NegMinArea(t *testing.T) {
	n := &GraphNode{ID: "n1", Kind: "DetectColorBlobs",
		Config: map[string]any{"literal": map[string]any{"MinArea": float64(-5)}}}
	errs := validateDetectColorBlobs(n)
	found := false
	for _, e := range errs {
		if e.Code == CodeInvalidBlobParam {
			found = true
		}
	}
	if !found {
		t.Errorf("want INVALID_BLOB_PARAM, got %+v", errs)
	}
}

func TestValidateDetectColorBlobs_Clean(t *testing.T) {
	n := &GraphNode{ID: "n1", Kind: "DetectColorBlobs",
		Config: map[string]any{"literal": map[string]any{"Sort": "area_desc", "Mode": "hsv"}}}
	if errs := validateDetectColorBlobs(n); len(errs) != 0 {
		t.Errorf("want no errors, got %+v", errs)
	}
}
