package runtime

import (
	"context"
	"maps"

	"github.com/yottaapp/yotta/internal/services/expr"
)

// TickSnapshot is a frozen view of rt.vars captured at execNode entry.
// All data-pull operations (GetVar) within the same exec tick read this snapshot,
// guaranteeing same-tick data consistency (Determinism contract).
//
// SetVar writes go directly to backing store (rt.vars); current tick's snapshot
// is unaffected, but the next exec node's snapshot picks up the new values.
type TickSnapshot struct {
	Vars map[string]expr.Value
}

// NewTickSnapshot returns an empty snapshot (test-only helper).
func NewTickSnapshot() *TickSnapshot {
	return &TickSnapshot{Vars: map[string]expr.Value{}}
}

// CaptureSnapshot performs a shallow copy of vars (map).
// Cost: O(N) where N = len(vars), typically <20. Microsecond-level.
func CaptureSnapshot(vars map[string]expr.Value) *TickSnapshot {
	cp := make(map[string]expr.Value, len(vars))
	maps.Copy(cp, vars)
	return &TickSnapshot{Vars: cp}
}

// GetVar reads a variable from the snapshot (does NOT walk frame chain).
// Returns (nil, false) if the name is unset in this snapshot.
func (s *TickSnapshot) GetVar(name string) (expr.Value, bool) {
	if s == nil {
		return nil, false
	}
	v, ok := s.Vars[name]
	return v, ok
}

// tickCtxKey 是 ctx.Value 的 key, 持 per-tick *TickSnapshot. dispatchInRegion
// 入口 withTickSnapshot 写, ServiceBundle.Snapshot 闭包 tickFromCtx 读.
//
// per-goroutine / per-token scope — 同 *ContainerRunner 跑多 goroutine (如共享 bundle
// 的 listener subRunner / 并发分支) 时, 各 goroutine 自己 ctx chain 独立, 不撞 instance 字段.
type tickCtxKeyT struct{}

var tickCtxKey = tickCtxKeyT{}

// withTickSnapshot 把 snap 挂到 ctx 的 tickCtxKey 上. dispatchInRegion 入口调.
func withTickSnapshot(ctx context.Context, snap *TickSnapshot) context.Context {
	return context.WithValue(ctx, tickCtxKey, snap)
}

// tickFromCtx 从 ctx 读 *TickSnapshot. 没挂 → nil. ServiceBundle.Snapshot 调.
func tickFromCtx(ctx context.Context) *TickSnapshot {
	if ctx == nil {
		return nil
	}
	snap, _ := ctx.Value(tickCtxKey).(*TickSnapshot)
	return snap
}

// evalKey — per-dispatch 求值缓存 key. 结构体 key: 零碰撞、零字符串拼接分配.
type evalKey struct{ nodeID, pin string }

// dispatchEvalCache — 单个 exec 节点 dispatch 作用域内的 pure-data 求值缓存.
// 只缓存 IsNonDeterministic 节点的成功结果 (见 evalPureDataCached), 让随机在同一求值内多路径稳定,
// 把随机纳入框架既有 Determinism contract.
//
// 并发: 一个实例只属一个 dispatch 的单 goroutine — 唯一 wrap 点是 dispatchInRegion 入口
// (dispatch_v5.go), 每节点 dispatch 新建; 进入 listener 子流程的根 ctx 不带 cache, 子流程内
// 每节点 dispatch 仍经 dispatchInRegion 在自己 goroutine 上各自新建 → 单个 cache 永不跨
// goroutine 共享. pure-data 拉取树同步执行.
// 普通 map 无需 mutex; 不变量靠 TestEvalCache_* 守护.
type dispatchEvalCache struct{ m map[evalKey]expr.Value }

func newDispatchEvalCache() *dispatchEvalCache {
	return &dispatchEvalCache{m: map[evalKey]expr.Value{}}
}

type evalCacheKeyT struct{}

var evalCacheKey = evalCacheKeyT{}

// withEvalCache 把 cache 挂到 ctx. dispatchInRegion 入口与 withTickSnapshot 并列调.
func withEvalCache(ctx context.Context, c *dispatchEvalCache) context.Context {
	return context.WithValue(ctx, evalCacheKey, c)
}

// evalCacheFromCtx 读 ctx 上的 cache. 没挂 (cold path) → nil.
func evalCacheFromCtx(ctx context.Context) *dispatchEvalCache {
	if ctx == nil {
		return nil
	}
	c, _ := ctx.Value(evalCacheKey).(*dispatchEvalCache)
	return c
}
