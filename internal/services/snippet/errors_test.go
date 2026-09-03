package snippet

import (
	"errors"
	"testing"

	"github.com/yottaapp/yotta/internal/apperr"
)

func TestSnippetProblemProjection(t *testing.T) {
	got := apperr.From(problem("snippet.test", apperr.CategoryDomain, map[string]any{"id": "snippet"}, false, errors.New("private")))
	if got.ID != "snippet.test" || got.Category != apperr.CategoryDomain || got.Retryable {
		t.Fatalf("problem = %#v", got)
	}
}
