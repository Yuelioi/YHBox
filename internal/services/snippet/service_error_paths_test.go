package snippet

import (
	"testing"

	"github.com/yottaapp/yotta/internal/apperr"
)

func TestSnippetUnavailablePaths(t *testing.T) {
	service := NewService(nil)
	for _, call := range []func() error{
		func() error { _, err := service.Get("snippet"); return err },
		func() error { _, err := service.Save(nil); return err },
		func() error { return service.Delete("snippet") },
		func() error { _, err := service.MarkUsed("snippet"); return err },
	} {
		if got := apperr.From(call()); got.ID != "snippet.store.unavailable" && got.ID != "snippet.invalid" {
			t.Fatalf("problem = %#v", got)
		}
	}
}
