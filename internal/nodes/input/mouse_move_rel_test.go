package input

import (
	"context"
	"errors"
	"testing"

	"yotta/internal/node"
)

func TestMouseMoveRel_HappyPath(t *testing.T) {
	node.ResetRegistryForTest()
	node.Register(&MouseMoveRel{})
	rn, _ := node.Get("MouseMoveRel")

	rec := &recordingInput{}
	r := node.RunNode(context.Background(), rn, nil,
		map[string]any{mmrInDx: 10, mmrInDy: -20, mmrInDurationMs: 150},
		nil, withInput(rec), false)

	if r.Error != nil {
		t.Fatal(r.Error)
	}
	if r.ExitName != mmrOutDone {
		t.Errorf("exit = %q, want Done", r.ExitName)
	}
	if len(rec.calls) != 1 || rec.calls[0] != "MouseMoveRel:10:-20:150" {
		t.Errorf("calls = %v, want [MouseMoveRel:10:-20:150]", rec.calls)
	}
}

func TestMouseMoveRel_BackendError_Propagates(t *testing.T) {
	node.ResetRegistryForTest()
	node.Register(&MouseMoveRel{})
	rn, _ := node.Get("MouseMoveRel")

	rec := &recordingInput{err: errors.New("input rejected")}
	r := node.RunNode(context.Background(), rn, nil,
		map[string]any{mmrInDx: 5, mmrInDy: 5, mmrInDurationMs: 100},
		nil, withInput(rec), false)

	if r.Error == nil {
		t.Fatal("expected backend error to propagate")
	}
}
