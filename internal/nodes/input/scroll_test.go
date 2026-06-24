package input

import (
	"context"
	"errors"
	"testing"

	"yotta/internal/node"
)

func TestScroll_HappyPath(t *testing.T) {
	node.ResetRegistryForTest()
	node.Register(&Scroll{})
	rn, _ := node.Get("Scroll")

	rec := &recordingInput{}
	r := node.RunNode(context.Background(), rn, nil,
		map[string]any{scInXRatio: 0.4, scInYRatio: 0.6, scInDelta: -2},
		nil, withInput(rec), false)

	if r.Error != nil {
		t.Fatal(r.Error)
	}
	if r.ExitName != scOutDone {
		t.Errorf("exit = %q, want Done", r.ExitName)
	}
	if len(rec.calls) != 1 || rec.calls[0] != "Scroll:0.400:0.600:-2:false" {
		t.Errorf("calls = %v, want [Scroll:0.400:0.600:-2:false]", rec.calls)
	}
}

func TestScroll_BackendError_Propagates(t *testing.T) {
	node.ResetRegistryForTest()
	node.Register(&Scroll{})
	rn, _ := node.Get("Scroll")

	rec := &recordingInput{err: errors.New("not focused")}
	r := node.RunNode(context.Background(), rn, nil,
		map[string]any{scInXRatio: 0.5, scInYRatio: 0.5, scInDelta: 3},
		nil, withInput(rec), false)

	if r.Error == nil {
		t.Fatal("expected backend error to propagate")
	}
}

// TestScroll_Axis_Vertical: 默认 Axis=vertical → horizontal=false (零回归)
func TestScroll_Axis_Vertical(t *testing.T) {
	node.ResetRegistryForTest()
	node.Register(&Scroll{})
	rn, _ := node.Get("Scroll")

	rec := &recordingInput{}
	r := node.RunNode(context.Background(), rn, nil,
		map[string]any{scInXRatio: 0.5, scInYRatio: 0.5, scInDelta: 3, scInAxis: "vertical"},
		nil, withInput(rec), false)

	if r.Error != nil {
		t.Fatal(r.Error)
	}
	if len(rec.calls) != 1 || rec.calls[0] != "Scroll:0.500:0.500:3:false" {
		t.Errorf("calls = %v, want [Scroll:0.500:0.500:3:false]", rec.calls)
	}
}

// TestScroll_Axis_Horizontal: Axis=horizontal → horizontal=true
func TestScroll_Axis_Horizontal(t *testing.T) {
	node.ResetRegistryForTest()
	node.Register(&Scroll{})
	rn, _ := node.Get("Scroll")

	rec := &recordingInput{}
	r := node.RunNode(context.Background(), rn, nil,
		map[string]any{scInXRatio: 0.5, scInYRatio: 0.5, scInDelta: 2, scInAxis: "horizontal"},
		nil, withInput(rec), false)

	if r.Error != nil {
		t.Fatal(r.Error)
	}
	if len(rec.calls) != 1 || rec.calls[0] != "Scroll:0.500:0.500:2:true" {
		t.Errorf("calls = %v, want [Scroll:0.500:0.500:2:true]", rec.calls)
	}
}
