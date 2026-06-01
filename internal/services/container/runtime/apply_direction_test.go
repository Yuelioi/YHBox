package runtime

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	"github.com/lxn/win"

	"yhbox/internal/services/container"
	"yhbox/internal/services/execution"
	"yhbox/internal/services/expr"
	pkginput "yhbox/pkg/input"
)

type spyInputBackend struct {
	fakeInputBackend
	keyEvents []string
}

func (s *spyInputBackend) KeyDown(_ win.HWND, k string) error {
	s.keyEvents = append(s.keyEvents, "down:"+k)
	return nil
}

func (s *spyInputBackend) KeyUp(_ win.HWND, k string) error {
	s.keyEvents = append(s.keyEvents, "up:"+k)
	return nil
}

var _ pkginput.Backend = (*spyInputBackend)(nil)

func loadApplyDirection(t *testing.T) container.Subgraph {
	t.Helper()
	_, thisFile, _, _ := runtime.Caller(0)
	root := filepath.Join(filepath.Dir(thisFile), "..", "..", "..", "..")
	jsonPath := filepath.Join(root, "bin", "data", "containers", "fishing-v2", "subgraphs", "apply_direction.json")
	data, err := os.ReadFile(jsonPath)
	if err != nil {
		t.Fatalf("read apply_direction.json: %v", err)
	}
	var sg container.Subgraph
	if err := json.Unmarshal(data, &sg); err != nil {
		t.Fatalf("unmarshal apply_direction.json: %v", err)
	}
	return sg
}

func runApplyDirection(t *testing.T, dirInput float64, preControlDir float64) ([]string, float64) {
	t.Helper()
	sg := loadApplyDirection(t)
	c := &container.Container{
		SchemaVersion: 1,
		ID:            "apply_direction_test",
		Name:          "apply_direction_test",
		Vars: []container.VarDecl{
			{Name: "controlDir", Type: "number", Default: 0.0},
		},
		Subgraphs: []container.Subgraph{sg},
		Graph: container.Graph{
			Nodes: []container.GraphNode{
				{ID: "start", Kind: "Start"},
				{ID: "call", Kind: "Subgraph", Config: map[string]any{
					"SubgraphID": "apply_direction",
					"literal":    map[string]any{"dir": dirInput},
				}},
				{ID: "stop", Kind: "Stop"},
			},
			Edges: []container.GraphEdge{
				{From: "start.Done", To: "call.in"},
				{From: "call.Done", To: "stop.in"},
			},
		},
	}
	rt := NewRuntimeContext(c, execution.NewInputBus(), NoopMatcher{}, nil, nil, nil, 0)
	stubRuntimeWindowAndInput(rt)
	spy := &spyInputBackend{}
	rt.Input = spy
	rt.SetVar("controlDir", preControlDir)
	r := NewContainerRunner(rt)
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	if err := r.Run(ctx); err != nil {
		t.Fatalf("Run: %v", err)
	}
	out, _ := expr.AsNumber(rt.Vars()["controlDir"])
	return spy.keyEvents, out
}

func TestApplyDirection_Right(t *testing.T) {
	events, ctrlDir := runApplyDirection(t, 1.0, 0.0)
	want := []string{"up:a", "down:d"}
	if !equalStrings(events, want) {
		t.Fatalf("dir=1: want events %v, got %v", want, events)
	}
	if ctrlDir != 1.0 {
		t.Fatalf("dir=1: want controlDir=1, got %v", ctrlDir)
	}
}

func equalStrings(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

func TestApplyDirection_Left(t *testing.T) {
	events, ctrlDir := runApplyDirection(t, -1.0, 0.0)
	want := []string{"up:d", "down:a"}
	if !equalStrings(events, want) {
		t.Fatalf("dir=-1: want events %v, got %v", want, events)
	}
	if ctrlDir != -1.0 {
		t.Fatalf("dir=-1: want controlDir=-1, got %v", ctrlDir)
	}
}

func TestApplyDirection_Stop(t *testing.T) {
	events, ctrlDir := runApplyDirection(t, 0.0, 1.0)
	want := []string{"up:a", "up:d"}
	if !equalStrings(events, want) {
		t.Fatalf("dir=0: want events %v, got %v", want, events)
	}
	if ctrlDir != 0.0 {
		t.Fatalf("dir=0: want controlDir=0, got %v", ctrlDir)
	}
}

func TestApplyDirection_CacheHit(t *testing.T) {
	events, ctrlDir := runApplyDirection(t, 1.0, 1.0)
	if len(events) != 0 {
		t.Fatalf("dir=1 with controlDir=1 (cache hit): want empty events, got %v", events)
	}
	if ctrlDir != 1.0 {
		t.Fatalf("dir=1 cache hit: want controlDir stays 1, got %v", ctrlDir)
	}
}

func runApplyDirectionTwice(t *testing.T, d1, d2 float64) ([]string, float64) {
	t.Helper()
	sg := loadApplyDirection(t)
	c := &container.Container{
		SchemaVersion: 1,
		ID:            "apply_direction_test_twice",
		Name:          "apply_direction_test_twice",
		Vars: []container.VarDecl{
			{Name: "controlDir", Type: "number", Default: 0.0},
		},
		Subgraphs: []container.Subgraph{sg},
		Graph: container.Graph{
			Nodes: []container.GraphNode{
				{ID: "start", Kind: "Start"},
				{ID: "call1", Kind: "Subgraph", Config: map[string]any{
					"SubgraphID": "apply_direction",
					"literal":    map[string]any{"dir": d1},
				}},
				{ID: "call2", Kind: "Subgraph", Config: map[string]any{
					"SubgraphID": "apply_direction",
					"literal":    map[string]any{"dir": d2},
				}},
				{ID: "stop", Kind: "Stop"},
			},
			Edges: []container.GraphEdge{
				{From: "start.Done", To: "call1.in"},
				{From: "call1.Done", To: "call2.in"},
				{From: "call2.Done", To: "stop.in"},
			},
		},
	}
	rt := NewRuntimeContext(c, execution.NewInputBus(), NoopMatcher{}, nil, nil, nil, 0)
	stubRuntimeWindowAndInput(rt)
	spy := &spyInputBackend{}
	rt.Input = spy
	r := NewContainerRunner(rt)
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	if err := r.Run(ctx); err != nil {
		t.Fatalf("Run: %v", err)
	}
	out, _ := expr.AsNumber(rt.Vars()["controlDir"])
	return spy.keyEvents, out
}

func TestApplyDirection_PersistentCache(t *testing.T) {
	events, ctrlDir := runApplyDirectionTwice(t, 1.0, -1.0)
	want := []string{"up:a", "down:d", "up:d", "down:a"}
	if !equalStrings(events, want) {
		t.Fatalf("dir=1 then dir=-1: want %v, got %v", want, events)
	}
	if ctrlDir != -1.0 {
		t.Fatalf("after persistent calls: want controlDir=-1, got %v", ctrlDir)
	}
}
