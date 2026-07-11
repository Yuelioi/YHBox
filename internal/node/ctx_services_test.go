package node

import (
	"context"
	"testing"
)

func TestCtxServicesReturnsBundleCopyWithoutFrameworkSnapshot(t *testing.T) {
	services := StubServices()
	services.Snapshot = func(context.Context) Snapshot { return Snapshot{} }
	ctx := newCtx(context.Background(), services, &Spec{Kind: "test"}, nil)
	got := ctx.Services()
	if got.Snapshot != nil {
		t.Fatal("node-facing Services must hide the framework-internal Snapshot hook")
	}
	if got.Vars != services.Vars || got.Input != services.Input {
		t.Fatal("node-facing Services did not preserve runtime service ports")
	}
	got.Vars = nil
	if ctx.Services().Vars == nil {
		t.Fatal("mutating the returned bundle must not mutate the context bundle")
	}
}
