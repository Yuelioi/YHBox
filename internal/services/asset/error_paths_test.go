package asset

import (
	"testing"

	"github.com/yottaapp/yotta/internal/apperr"
	"github.com/yottaapp/yotta/internal/blob"
)

func TestAssetServiceStableEarlyFailures(t *testing.T) {
	store, _ := newTestStore(t)
	service := NewService(store, nil, nil)
	tests := []struct {
		name string
		err  func() error
		id   string
	}{
		{"get missing", func() error { _, err := service.Get("missing"); return err }, "asset.not_found"},
		{"remove missing", func() error { _, err := service.RemoveVariant("missing", 1, 1); return err }, "asset.not_found"},
		{"update missing", func() error { return service.UpdateMeta("missing", "x", "", "", nil) }, "asset.not_found"},
		{"pick missing", func() error { _, err := service.PickVariant("missing", 100, 100); return err }, "asset.variant.not_found"},
		{"preview media", func() error {
			_, err := service.PreviewBlob(blob.BlobRef{MediaType: "text/plain", Size: 1})
			return err
		}, "asset.preview.invalid"},
		{"preview size", func() error {
			_, err := service.PreviewBlob(blob.BlobRef{MediaType: "image/png", Size: -1})
			return err
		}, "asset.preview.invalid"},
		{"capture unavailable", func() error { _, err := service.Capture("target"); return err }, "asset.capture.unavailable"},
		{"resolution unavailable", func() error { _, err := service.CurrentResolution("target"); return err }, "asset.capture.unavailable"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := apperr.From(test.err()); got.ID != test.id || got.OperationID == "" {
				t.Fatalf("problem = %#v, want %s", got, test.id)
			}
		})
	}
}

func TestAssetPresentationHelpers(t *testing.T) {
	if w, h := previewDimensions(100, 80); w != 100 || h != 80 {
		t.Fatalf("small preview = %dx%d", w, h)
	}
	if w, h := previewDimensions(100, 400); w != 64 || h != 256 {
		t.Fatalf("portrait preview = %dx%d", w, h)
	}
	tags := cleanTags([]string{" One ", "one", "", "Two"})
	if len(tags) != 2 || tags[0] != "One" || tags[1] != "Two" {
		t.Fatalf("tags = %v", tags)
	}
}
