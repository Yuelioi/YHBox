package control

import (
	"context"
	"errors"
	"testing"

	"yotta/internal/node"
)

func TestForEach_IteratesAllItems(t *testing.T) {
	node.ResetRegistryForTest()
	node.Register(&ForEach{})
	rn, _ := node.Get("ForEach")

	vars := newRecVars()
	services := node.StubServices()
	services.Vars = vars

	var items, indices []any
	body := func(_ node.Ctx) (string, error) {
		v, _ := vars.Get("item")
		i, _ := vars.Get("idx")
		items = append(items, v)
		indices = append(indices, i)
		return "", nil
	}

	r := node.RunNodeAsRegion(context.Background(), rn,
		map[string]any{feInList: []any{"a", "b", "c"}},
		map[string]any{feCapItem: "item", feCapIndex: "idx"},
		nil, services, false, body)

	if r.Error != nil {
		t.Fatal(r.Error)
	}
	if len(items) != 3 || items[0] != "a" || items[2] != "c" {
		t.Errorf("items = %v, want [a b c]", items)
	}
	if len(indices) != 3 || indices[0] != 0 || indices[2] != 2 {
		t.Errorf("indices = %v, want [0 1 2]", indices)
	}
	if r.ExitName != feOutDone {
		t.Errorf("exit = %q, want %q", r.ExitName, feOutDone)
	}
}

func TestForEach_EmptyOrNonList_ZeroIterationsDone(t *testing.T) {
	node.ResetRegistryForTest()
	node.Register(&ForEach{})
	rn, _ := node.Get("ForEach")

	for _, listVal := range []any{[]any{}, nil, "not a list", 42} {
		iterations := 0
		r := node.RunNodeAsRegion(context.Background(), rn,
			map[string]any{feInList: listVal}, nil, nil,
			node.StubServices(), false, func(_ node.Ctx) (string, error) { iterations++; return "", nil })
		if r.Error != nil {
			t.Fatalf("listVal=%v: %v", listVal, r.Error)
		}
		if iterations != 0 {
			t.Errorf("listVal=%v: iterations = %d, want 0", listVal, iterations)
		}
		if r.ExitName != feOutDone {
			t.Errorf("listVal=%v: exit = %q, want Done", listVal, r.ExitName)
		}
	}
}

func TestForEach_BreakSentinel(t *testing.T) {
	node.ResetRegistryForTest()
	node.Register(&ForEach{})
	rn, _ := node.Get("ForEach")

	iterations := 0
	body := func(_ node.Ctx) (string, error) {
		iterations++
		if iterations == 2 {
			return "", errBreakRequested
		}
		return "", nil
	}
	r := node.RunNodeAsRegion(context.Background(), rn,
		map[string]any{feInList: []any{1, 2, 3, 4}}, nil, nil,
		node.StubServices(), false, body)
	if r.Error != nil {
		t.Fatal(r.Error)
	}
	if iterations != 2 || r.ExitName != feOutDone {
		t.Errorf("iterations=%d exit=%q, want 2/Done", iterations, r.ExitName)
	}
}

func TestForEach_ContinueSentinel(t *testing.T) {
	node.ResetRegistryForTest()
	node.Register(&ForEach{})
	rn, _ := node.Get("ForEach")

	iterations := 0
	r := node.RunNodeAsRegion(context.Background(), rn,
		map[string]any{feInList: []any{1, 2, 3}}, nil, nil,
		node.StubServices(), false, func(_ node.Ctx) (string, error) { iterations++; return "", errContinueRequested })
	if r.Error != nil {
		t.Fatal(r.Error)
	}
	if iterations != 3 || r.ExitName != feOutDone {
		t.Errorf("iterations=%d exit=%q, want 3/Done", iterations, r.ExitName)
	}
}

func TestForEach_BodyErrorPropagates(t *testing.T) {
	node.ResetRegistryForTest()
	node.Register(&ForEach{})
	rn, _ := node.Get("ForEach")

	boom := errors.New("boom")
	r := node.RunNodeAsRegion(context.Background(), rn,
		map[string]any{feInList: []any{1, 2}}, nil, nil,
		node.StubServices(), false, func(_ node.Ctx) (string, error) { return "", boom })
	if !errors.Is(r.Error, boom) {
		t.Errorf("error = %v, want boom", r.Error)
	}
	if r.ExitName != "" {
		t.Errorf("exit = %q, want empty", r.ExitName)
	}
}
