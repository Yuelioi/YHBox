package noderuntime

import (
	"context"
	"errors"

	"github.com/yottaapp/yotta/internal/nodeadapter"
	"github.com/yottaapp/yotta/internal/resource"
	"github.com/yottaapp/yotta/internal/targetruntime"
)

func openConfiguredTarget(ctx context.Context, invocation nodeadapter.Invocation, kind string, operations []string) (resource.Handle, error) {
	if invocation.Targets == nil {
		return resource.Handle{}, errors.New("configured target runtime is unavailable")
	}
	slot, ok := invocation.Config["slot"].(string)
	if !ok || slot == "" {
		return resource.Handle{}, errors.New("configured target slot is missing")
	}
	return invocation.Targets.Open(ctx, targetruntime.OpenRequest{
		Slot: slot, Kind: kind, Operations: append([]string(nil), operations...), Config: []byte(`{}`),
	})
}
