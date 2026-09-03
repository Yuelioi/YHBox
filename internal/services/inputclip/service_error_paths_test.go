package inputclip

import (
	"testing"

	"github.com/yottaapp/yotta/internal/apperr"
)

func TestInputClipUnavailablePaths(t *testing.T) {
	service := NewService(nil)
	tests := []func() error{
		func() error { return service.Save(nil) },
		func() error { _, err := service.Get("clip"); return err },
		func() error { _, err := service.List(); return err },
		func() error { return service.Delete("clip") },
		func() error { return service.Update("clip", "", "", "", nil) },
	}
	for _, call := range tests {
		if got := apperr.From(call()); got.ID != "input_clip.store_unavailable" || !got.Retryable {
			t.Fatalf("problem = %#v", got)
		}
	}
}
