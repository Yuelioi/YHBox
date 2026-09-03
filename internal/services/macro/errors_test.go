package macro

import (
	"errors"
	"testing"

	"github.com/yottaapp/yotta/internal/apperr"
)

func TestMacroProblemProjection(t *testing.T) {
	got := apperr.From(problem("macro.test", apperr.CategoryValidation, map[string]any{"field": "label"}, false, errors.New("private")))
	if got.ID != "macro.test" || got.Category != apperr.CategoryValidation || got.Retryable {
		t.Fatalf("problem = %#v", got)
	}
}
