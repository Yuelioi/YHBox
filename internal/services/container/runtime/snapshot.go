package runtime

import (
	"context"
	"maps"

	"yhbox/internal/services/expr"
)

// TickSnapshot is a frozen view of rt.vars + rt.sys captured at execNode entry.
// All data-pull operations (GetVar / GetSys) within the same exec tick read this snapshot,
// guaranteeing same-tick data consistency (Determinism contract).
//
// SetVar writes go directly to backing store (rt.vars / rt.sys); current tick's snapshot
// is unaffected, but the next exec node's snapshot picks up the new values.
type TickSnapshot struct {
	Vars map[string]expr.Value
	Sys  SysState
}

// NewTickSnapshot returns an empty snapshot (test-only helper).
func NewTickSnapshot() *TickSnapshot {
	return &TickSnapshot{Vars: map[string]expr.Value{}}
}

// CaptureSnapshot performs a shallow copy of vars (map) and a value copy of sys.
// Cost: O(N) where N = len(vars), typically <20. Microsecond-level.
func CaptureSnapshot(vars map[string]expr.Value, sys SysState) *TickSnapshot {
	cp := make(map[string]expr.Value, len(vars))
	maps.Copy(cp, vars)
	return &TickSnapshot{Vars: cp, Sys: sys}
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
// per-goroutine / per-token scope — 同 *ContainerRunner 跑多 goroutine (listener
// subRunner 共享 bundle 后 + Phase 6 Parallel/Race 未来) 时, 各 goroutine 自己
// ctx chain 独立, 不撞 instance 字段.
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
