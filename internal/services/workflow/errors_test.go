package workflow

import (
	"errors"
	"testing"

	"github.com/yottaapp/yotta/internal/apperr"
)

func TestWorkflowErrorProjection(t *testing.T) {
	cause := errors.New("private cause")
	tests := []struct {
		name string
		err  error
		id   string
	}{
		{"source", sourceError("save", cause), "workflow.source.failed"},
		{"bundle", bundleError("import", cause), "workflow.bundle.failed"},
		{"run", runError("start", cause), "workflow.run.failed"},
		{"unavailable", unavailable("bundle"), "workflow.feature.unavailable"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			problem := apperr.From(test.err)
			if problem.ID != test.id || problem.OperationID == "" {
				t.Fatalf("problem = %#v", problem)
			}
		})
	}
	if err := projectError("unused", apperr.CategoryDomain, nil, false, nil); err != nil {
		t.Fatalf("nil cause projected as %v", err)
	}
	typed := apperr.New("existing.problem", nil)
	if got := projectError("unused", apperr.CategoryDomain, nil, false, typed); got != typed {
		t.Fatal("typed problem was replaced")
	}
}
