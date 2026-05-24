package control

import (
	"context"
	"testing"
	"time"

	"yhbox/internal/node"
)

func TestSleep_HappyPath(t *testing.T) {
	node.ResetRegistryForTest()
	node.Register(&Sleep{})
	rn, _ := node.Get("Sleep")

	start := time.Now()
	r := node.RunNode(context.Background(), rn, nil,
		map[string]any{sleepInDuration: 50 * time.Millisecond},
		nil, node.StubVisionService(), node.DefaultLogService())
	elapsed := time.Since(start)

	if r.Error != nil {
		t.Fatal(r.Error)
	}
	if r.ExitName != sleepOutDone {
		t.Errorf("exit = %q, want Done", r.ExitName)
	}
	if elapsed < 40*time.Millisecond {
		t.Errorf("elapsed %v < 40ms — Sleep 没真等", elapsed)
	}
}

func TestSleep_ZeroDuration_Error(t *testing.T) {
	node.ResetRegistryForTest()
	node.Register(&Sleep{})
	rn, _ := node.Get("Sleep")

	r := node.RunNode(context.Background(), rn, nil,
		map[string]any{sleepInDuration: time.Duration(0)},
		nil, node.StubVisionService(), node.DefaultLogService())

	if r.Error == nil {
		t.Error("expected error on zero Duration")
	}
}

func TestSleep_RequiredMissing(t *testing.T) {
	node.ResetRegistryForTest()
	node.Register(&Sleep{})
	rn, _ := node.Get("Sleep")

	r := node.RunNode(context.Background(), rn, nil, nil, nil,
		node.StubVisionService(), node.DefaultLogService())
	if len(r.Validation) == 0 {
		t.Error("expected REQUIRED_FIELD_MISSING for Duration")
	}
}
