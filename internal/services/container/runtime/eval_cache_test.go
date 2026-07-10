package runtime

import (
	"context"
	"errors"
	"sync/atomic"
	"testing"

	"github.com/yottaapp/yotta/internal/node"
	_ "github.com/yottaapp/yotta/internal/nodes/input" // MouseMoveRel (Dx+Dy 两 Number 输入 — C1 regression 用)
	"github.com/yottaapp/yotta/internal/services/container"
	"github.com/yottaapp/yotta/internal/services/expr"
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

// ============================================================================
// C1 regression + 补充测试
// ============================================================================

// testCtrND2 — 第二个独立 ND 计数器 kind, 用于"跨节点隔离"测试 (两实例不共 cache entry).
var testCtrND2 atomic.Int64

type testCounterND2 struct{}

func (testCounterND2) Spec() node.Spec {
	return node.Spec{
		Kind: "TestCounterND2", Category: "PureFunc",
		Outputs:            []node.OutputSpec{{Name: "Result", Type: "Number"}},
		IsPureData:         true,
		IsNonDeterministic: true,
	}
}
func (testCounterND2) Evaluate(_ node.Ctx, _ node.Inputs) (any, error) {
	return float64(testCtrND2.Add(1)), nil
}

// testCounterFail — ND 节点: 第一次调 Evaluate 返 error, 之后返成功值.
// 用于"失败不缓存"测试.
var testFailCtrCalls atomic.Int64

type testCounterFail struct{}

func (testCounterFail) Spec() node.Spec {
	return node.Spec{
		Kind: "TestCounterFail", Category: "PureFunc",
		Outputs:            []node.OutputSpec{{Name: "Result", Type: "Number"}},
		IsPureData:         true,
		IsNonDeterministic: true,
	}
}
func (testCounterFail) Evaluate(_ node.Ctx, _ node.Inputs) (any, error) {
	n := testFailCtrCalls.Add(1)
	if n == 1 {
		return nil, errors.New("first call fails")
	}
	return float64(n), nil
}

func init() {
	node.Register(&testCounterND2{})
	node.Register(&testCounterFail{})
}

// wireCounterTwoPins 建一条 src.Result → consumer.Dx AND src.Result → consumer.Dy 的数据边.
// consumer 是 MouseMoveRel (Dx+Dy 两个 Number 输入), 用于 C1 regression.
func wireCounterTwoPins(t *testing.T, srcKind string) *ContainerRunner {
	t.Helper()
	_, r := newTestRunner(t)
	src := &container.GraphNode{ID: "ctr", Kind: srcKind}
	dst := &container.GraphNode{ID: "consumer", Kind: "MouseMoveRel", Config: map[string]any{}}
	r.nodesByID = map[string]*container.GraphNode{"ctr": src, "consumer": dst}
	r.dataEdges = buildDataEdgeIndex(container.Graph{
		Nodes: []container.GraphNode{*src, *dst},
		Edges: []container.GraphEdge{
			{From: "ctr.Result", To: "consumer.Dx"},
			{From: "ctr.Result", To: "consumer.Dy"},
		},
	})
	return r
}

// TestEvalCache_C1_TwoPinsSameDispatch — C1 回归:
// 同一 ND 源节点 (TestCounterND) wired → consumer.Dx AND consumer.Dy,
// 在同一 dispatch ctx 下通过 resolveDataPinV5 (buildDataWireFor 真实路径) 分别拉两个 pin.
// 期望: 两个 pin 拿到相同值 (memoized), 计数器只自增 1 次.
// Consumer 节点: MouseMoveRel (有 Dx + Dy 两个 Number 输入 pin).
func TestEvalCache_C1_TwoPinsSameDispatch(t *testing.T) {
	r := wireCounterTwoPins(t, "TestCounterND")
	ctx := dispatchCtx()
	before := testCtrN.Load()

	dxVal, errDx := r.resolveDataPinV5(ctx, "consumer", "Dx")
	dyVal, errDy := r.resolveDataPinV5(ctx, "consumer", "Dy")

	if errDx != nil || errDy != nil {
		t.Fatalf("resolveDataPinV5 errors: dx=%v dy=%v", errDx, errDy)
	}
	dxN, _ := expr.AsNumber(dxVal)
	dyN, _ := expr.AsNumber(dyVal)
	if dxN != dyN {
		t.Fatalf("C1: same ND src, same dispatch — want equal values, got Dx=%v Dy=%v", dxN, dyN)
	}
	if delta := testCtrN.Load() - before; delta != 1 {
		t.Fatalf("C1: ND src should be evaluated exactly once per dispatch, got %d evals", delta)
	}
}

// TestEvalCache_FailureNotCached — 失败不缓存:
// ND 节点第一次 Evaluate 失败 → 错误不应被缓存 → 第二次调用重算并成功.
func TestEvalCache_FailureNotCached(t *testing.T) {
	testFailCtrCalls.Store(0)
	_, r := newTestRunner(t)
	src := &container.GraphNode{ID: "failctr", Kind: "TestCounterFail"}
	dst := &container.GraphNode{ID: "sleep", Kind: "Sleep", Config: map[string]any{}}
	r.nodesByID = map[string]*container.GraphNode{"failctr": src, "sleep": dst}
	r.dataEdges = buildDataEdgeIndex(container.Graph{
		Nodes: []container.GraphNode{*src, *dst},
		Edges: []container.GraphEdge{{From: "failctr.Result", To: "sleep.Duration"}},
	})
	ctx := dispatchCtx()

	v1, err1 := r.pullDataPin(ctx, "sleep", "Duration")
	if err1 == nil {
		t.Fatalf("first call should error, got value=%v", v1)
	}
	v2, err2 := r.pullDataPin(ctx, "sleep", "Duration")
	if err2 != nil {
		t.Fatalf("second call should succeed after failure, got err=%v", err2)
	}
	if v2 == nil {
		t.Fatal("second call returned nil — failure was wrongly cached")
	}
}

// TestEvalCache_DistinctNodesIsolated — 跨节点隔离:
// 两个不同 node ID 的 ND 节点, 分别接到两个 pin, 同一 dispatch ctx →
// 各自独立 evaluate 一次 (total delta=2), 值相互独立.
func TestEvalCache_DistinctNodesIsolated(t *testing.T) {
	before1 := testCtrN.Load()
	before2 := testCtrND2.Load()

	_, r := newTestRunner(t)
	src1 := &container.GraphNode{ID: "ctr1", Kind: "TestCounterND"}
	src2 := &container.GraphNode{ID: "ctr2", Kind: "TestCounterND2"}
	dst := &container.GraphNode{ID: "consumer", Kind: "MouseMoveRel", Config: map[string]any{}}
	r.nodesByID = map[string]*container.GraphNode{
		"ctr1": src1, "ctr2": src2, "consumer": dst,
	}
	r.dataEdges = buildDataEdgeIndex(container.Graph{
		Nodes: []container.GraphNode{*src1, *src2, *dst},
		Edges: []container.GraphEdge{
			{From: "ctr1.Result", To: "consumer.Dx"},
			{From: "ctr2.Result", To: "consumer.Dy"},
		},
	})
	ctx := dispatchCtx()

	dxVal, errDx := r.resolveDataPinV5(ctx, "consumer", "Dx")
	dyVal, errDy := r.resolveDataPinV5(ctx, "consumer", "Dy")
	if errDx != nil || errDy != nil {
		t.Fatalf("resolveDataPinV5 errors: dx=%v dy=%v", errDx, errDy)
	}

	delta1 := testCtrN.Load() - before1
	delta2 := testCtrND2.Load() - before2
	if delta1 != 1 || delta2 != 1 {
		t.Fatalf("each distinct ND node should eval once: ctr1 delta=%d ctr2 delta=%d", delta1, delta2)
	}

	dxN, _ := expr.AsNumber(dxVal)
	dyN, _ := expr.AsNumber(dyVal)
	// values come from separate counters — they should differ (both incremented from different bases)
	_ = dxN
	_ = dyN
}
