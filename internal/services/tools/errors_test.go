package tools

import (
	"errors"
	"testing"

	"github.com/yottaapp/yotta/internal/apperr"
)

func TestToolErrorProjection(t *testing.T) {
	if err := toolError("tools.test", apperr.CategoryAdapter, map[string]any{"slot": "safe"}, true, nil); err != nil {
		t.Fatalf("nil cause projected as %v", err)
	}
	typed := apperr.New("existing.problem", nil)
	if got := toolError("unused", apperr.CategoryAdapter, nil, false, typed); got != typed {
		t.Fatal("typed problem was replaced")
	}
	problem := apperr.From(toolError("tools.test", apperr.CategoryAdapter, map[string]any{"slot": "safe"}, true, errors.New("private")))
	if problem.ID != "tools.test" || problem.Category != apperr.CategoryAdapter || !problem.Retryable {
		t.Fatalf("problem = %#v", problem)
	}
}
