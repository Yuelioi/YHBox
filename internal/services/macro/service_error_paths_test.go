package macro

import (
	"testing"

	"github.com/yottaapp/yotta/internal/apperr"
)

func TestMacroUnavailablePaths(t *testing.T) {
	service := NewService(nil)
	for _, call := range []func() error{
		func() error { _, err := service.Save(nil); return err },
		func() error { _, err := service.Get("macro"); return err },
		func() error { return service.Delete("macro") },
	} {
		if got := apperr.From(call()); got.ID != "macro.unavailable" || !got.Retryable {
			t.Fatalf("problem = %#v", got)
		}
	}
}
