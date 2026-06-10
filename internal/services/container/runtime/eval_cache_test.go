package runtime

import (
	"context"
	"sync/atomic"
	"testing"

	"yotta/internal/node"
	"yotta/internal/services/container"
	"yotta/internal/services/expr"
)

// 测试用计数源节点: 每次 Evaluate 自增. 用于确定性地探测记忆化是否生效.
var testCtrN atomic.Int64
var testCtrDetN atomic.Int64

type testCounter struct{}

func (testCounter) Spec() node.Spec {
	return node.Spec{
		Kind: "TestCounterND", Category: "PureFunc",
		Outputs:            []node.OutputSpec{{Name: "Result", Type: "Number"}},
		IsPureData:         true,
		IsNonDeterministic: true,
	}
}
func (testCounter) Evaluate(_ node.Ctx, _ node.Inputs) (any, error) {
	return float64(testCtrN.Add(1)), nil
}

type testCounterDet struct{}

func (testCounterDet) Spec() node.Spec {
	return node.Spec{
		Kind: "TestCounterDet", Category: "PureFunc",
		Outputs:    []node.OutputSpec{{Name: "Result", Type: "Number"}},
		IsPureData: true, // 注意: 故意 IsNonDeterministic=false
	}
}
func (testCounterDet) Evaluate(_ node.Ctx, _ node.Inputs) (any, error) {
	return float64(testCtrDetN.Add(1)), nil
}

func init() {
	node.Register(&testCounter{})
	node.Register(&testCounterDet{})
}

// wireCounter 建一条 src.Result → sleep.Duration 的数据边.
func wireCounter(t *testing.T, srcKind string) *ContainerRunner {
	t.Helper()
	_, r := newTestRunner(t)
	src := &container.GraphNode{ID: "ctr", Kind: srcKind}
	dst := &container.GraphNode{ID: "sleep", Kind: "Sleep", Config: map[string]any{}}
	r.nodesByID = map[string]*container.GraphNode{"ctr": src, "sleep": dst}
	r.dataEdges = buildDataEdgeIndex(container.Graph{
		Nodes: []container.GraphNode{*src, *dst},
		Edges: []container.GraphEdge{{From: "ctr.Result", To: "sleep.Duration"}},
	})
	return r
}

func dispatchCtx() context.Context {
	ctx := withTickSnapshot(context.Background(), NewTickSnapshot())
	return withEvalCache(ctx, newDispatchEvalCache())
}

// 非确定节点: 同一 dispatch 内多次拉取 → 记忆化同值.
func TestEvalCache_NonDeterministic_MemoizedWithinDispatch(t *testing.T) {
	r := wireCounter(t, "TestCounterND")
	ctx := dispatchCtx()
	v1, _ := r.pullDataPin(ctx, "sleep", "Duration")
	v2, _ := r.pullDataPin(ctx, "sleep", "Duration")
	n1, _ := expr.AsNumber(v1)
	n2, _ := expr.AsNumber(v2)
	if n1 != n2 {
		t.Fatalf("same dispatch: want memoized equal, got %v vs %v", n1, n2)
	}
}

// 非确定节点: 跨 dispatch → 重新求值 (断言重算发生, 不断言"值不等").
func TestEvalCache_NonDeterministic_FreshAcrossDispatch(t *testing.T) {
	r := wireCounter(t, "TestCounterND")
	before := testCtrN.Load()
	v1, _ := r.pullDataPin(dispatchCtx(), "sleep", "Duration")
	v2, _ := r.pullDataPin(dispatchCtx(), "sleep", "Duration")
	_ = v1
	_ = v2
	if got := testCtrN.Load() - before; got != 2 {
		t.Fatalf("two dispatches should re-eval twice, got %d evals", got)
	}
}

// 确定性节点 (IsNonDeterministic=false): 同一 dispatch 内不走缓存 → 每次重算.
func TestEvalCache_Deterministic_NotCached(t *testing.T) {
	r := wireCounter(t, "TestCounterDet")
	ctx := dispatchCtx()
	before := testCtrDetN.Load()
	r.pullDataPin(ctx, "sleep", "Duration")
	r.pullDataPin(ctx, "sleep", "Duration")
	if got := testCtrDetN.Load() - before; got != 2 {
		t.Fatalf("deterministic node must not be cached; want 2 evals, got %d", got)
	}
}
