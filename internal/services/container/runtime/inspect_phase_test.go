package runtime

import (
	"context"
	"encoding/json"
	"image"
	"os"
	"path/filepath"
	gort "runtime"
	"testing"
	"time"

	"yotta/internal/services/container"
	"yotta/internal/services/execution"
	"yotta/internal/services/expr"
)

type mockMatcher struct {
	HitTemplates map[string]bool
}

func (m *mockMatcher) Detect(_ context.Context, _ string, _ *image.RGBA, k string, _ float64, _ []float64) (bool, expr.Point, [4]float64, float64, error) {
	if m.HitTemplates == nil {
		return false, expr.Point{}, [4]float64{}, 0, nil
	}
	hit := m.HitTemplates[k]
	conf := 0.0
	if hit {
		conf = 1.0
	}
	return hit, expr.Point{}, [4]float64{}, conf, nil
}

var _ TemplateMatcher = (*mockMatcher)(nil)

func loadInspectPhase(t *testing.T) container.Subgraph {
	t.Helper()
	_, thisFile, _, _ := gort.Caller(0)
	root := filepath.Join(filepath.Dir(thisFile), "..", "..", "..", "..")
	jsonPath := filepath.Join(root, "bin", "data", "containers", "fishing-v2", "subgraphs", "inspect_phase.json")
	data, err := os.ReadFile(jsonPath)
	if err != nil {
		t.Fatalf("read inspect_phase.json: %v", err)
	}
	var sg container.Subgraph
	if err := json.Unmarshal(data, &sg); err != nil {
		t.Fatalf("unmarshal inspect_phase.json: %v", err)
	}
	return sg
}

func runInspectPhase(t *testing.T, hits map[string]bool, frame *image.RGBA) (*RuntimeContext, error) {
	t.Helper()
	sg := loadInspectPhase(t)
	c := &container.Container{
		SchemaVersion: 1,
		ID:            "inspect_phase_test",
		Name:          "inspect_phase_test",
		Vars: []container.VarDecl{
			{Name: "_inspectPhaseResult", Type: "string", Default: "unknown"},
		},
		Subgraphs: []container.Subgraph{sg},
		Graph: container.Graph{
			Nodes: []container.GraphNode{
				{ID: "start", Kind: "Start"},
				{ID: "call", Kind: "Subgraph", Config: map[string]any{
					"SubgraphID": "inspect_phase",
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
	rt.Matcher = &mockMatcher{HitTemplates: hits}
	rt.Capture = &mockCaptureBackend{FrameROIResult: frame}
	r := NewContainerRunner(rt)
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	err := r.Run(ctx)
	return rt, err
}

func TestInspectPhase_HookIconHit(t *testing.T) {
	rt, err := runInspectPhase(t, map[string]bool{"fishing.hook_icon": true}, nil)
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	got, _ := rt.Vars()["_inspectPhaseResult"].(string)
	if got != "ready" {
		t.Fatalf("HookIconHit: want _inspectPhaseResult=ready, got %q", got)
	}
}

func TestInspectPhase_ResultPriority(t *testing.T) {
	rt, err := runInspectPhase(t, map[string]bool{"fishing.result": true, "fishing.warehouse_full": true}, nil)
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	got, _ := rt.Vars()["_inspectPhaseResult"].(string)
	if got != "settle_win" {
		t.Fatalf("ResultPriority: want _inspectPhaseResult=settle_win (优先级最高 short-circuits warehouse_full), got %q", got)
	}
}

func TestInspectPhase_FightingDetected(t *testing.T) {
	img := image.NewRGBA(image.Rect(0, 0, 200, 20))
	paintCursorBar(img, 50)
	paintTargetBar(img, 100, 115)
	rt, err := runInspectPhase(t, map[string]bool{}, img)
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	got, _ := rt.Vars()["_inspectPhaseResult"].(string)
	if got != "fighting" {
		t.Fatalf("FightingDetected: want _inspectPhaseResult=fighting, got %q", got)
	}
}

func TestInspectPhase_AllMissUnknown(t *testing.T) {
	rt, err := runInspectPhase(t, map[string]bool{}, nil)
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	got, _ := rt.Vars()["_inspectPhaseResult"].(string)
	if got != "unknown" {
		t.Fatalf("AllMissUnknown: want _inspectPhaseResult=unknown, got %q", got)
	}
}
