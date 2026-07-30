package noderuntime_test

import (
	"context"
	"testing"

	"github.com/yottaapp/yotta/internal/resource"
	"github.com/yottaapp/yotta/internal/targetruntime"
)

func configuredTargetRun(t *testing.T, slot, targetID string, provider resource.Provider) *targetruntime.Run {
	t.Helper()
	snapshot, err := targetruntime.NewSnapshot([]targetruntime.Installation{{
		Slot: slot, TargetID: targetID, Provider: provider,
	}})
	if err != nil {
		t.Fatal(err)
	}
	runtime, err := snapshot.NewRun()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := runtime.Close(context.Background()); err != nil {
			t.Errorf("close configured targets: %v", err)
		}
	})
	return runtime
}
