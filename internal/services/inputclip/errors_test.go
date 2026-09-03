package inputclip

import (
	"errors"
	"testing"

	"github.com/yottaapp/yotta/internal/apperr"
)

func TestInputClipProblemProjection(t *testing.T) {
	got := apperr.From(problem("input_clip.test", apperr.CategoryInfrastructure, map[string]any{"id": "clip"}, true, errors.New("private")))
	if got.ID != "input_clip.test" || got.Category != apperr.CategoryInfrastructure || !got.Retryable {
		t.Fatalf("problem = %#v", got)
	}
}
