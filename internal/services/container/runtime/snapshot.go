package runtime

import (
	"maps"

	"yhbox/internal/services/expr"
)

// TickSnapshot is a frozen view of rt.vars + rt.sys captured at execNode entry.
// All data-pull operations (GetVar / GetSys) within the same exec tick read this snapshot,
// guaranteeing same-tick data consistency (spec §10.3 Determinism contract).
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
