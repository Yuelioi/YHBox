package serviceproblem

import (
	"errors"
	"testing"

	"github.com/yottaapp/yotta/internal/apperr"
)

func TestWrap(t *testing.T) {
	if Wrap("unused", apperr.CategoryDomain, nil, false, nil) != nil {
		t.Fatal("nil cause was projected")
	}
	typed := apperr.New("existing", nil)
	if Wrap("unused", apperr.CategoryDomain, nil, false, typed) != typed {
		t.Fatal("existing Problem was replaced")
	}
	got := apperr.From(Wrap("test.failed", apperr.CategoryAdapter, map[string]any{"safe": true}, true, errors.New("private")))
	if got.ID != "test.failed" || got.Category != apperr.CategoryAdapter || !got.Retryable {
		t.Fatalf("problem = %#v", got)
	}
	if message := Wrap("test.failed", apperr.CategoryAdapter, nil, false, errors.New("private")).Error(); message == "" {
		t.Fatal("projected error has no diagnostic string")
	}
}
