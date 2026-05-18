package runtime

import (
	"testing"

	"yhbox/internal/services/container"
	"yhbox/internal/services/expr"
)

// TestGetVar_LocalScope: reads from current frame.LocalVars, ignores rt.vars snapshot.
func TestGetVar_LocalScope(t *testing.T) {
	_, r := newTestRunner(t)
	r.currentTick = CaptureSnapshot(map[string]expr.Value{"x": float64(1)}, SysState{})
	r.state.SetLocalVar("y", float64(2))

	n := &container.GraphNode{
		ID: "g1", Kind: "GetVar",
		Config: map[string]any{"varName": "y", "scope": "local"},
	}
	v, err := r.evalGetVar(n)
	if err != nil {
		t.Fatal(err)
	}
	if got, _ := expr.AsNumber(v); got != 2.0 {
		t.Fatalf("local y: want 2, got %v", v)
	}
}

// TestGetVar_LocalIgnoresGlobal: a name only present in global → local read returns nil.
func TestGetVar_LocalIgnoresGlobal(t *testing.T) {
	_, r := newTestRunner(t)
	r.currentTick = CaptureSnapshot(map[string]expr.Value{"hp": float64(0.8)}, SysState{})
	// hp is NOT in local frame.

	n := &container.GraphNode{
		ID: "g1", Kind: "GetVar",
		Config: map[string]any{"varName": "hp", "scope": "local"},
	}
	v, _ := r.evalGetVar(n)
	if v != nil {
		t.Fatalf("local scope must NOT fall back to global; want nil, got %v", v)
	}
}

// TestGetVar_GlobalScope: reads from rt.vars snapshot, ignores local frame.
func TestGetVar_GlobalScope(t *testing.T) {
	_, r := newTestRunner(t)
	r.currentTick = CaptureSnapshot(map[string]expr.Value{"hp": float64(0.8)}, SysState{})
	r.state.SetLocalVar("hp", float64(0.5)) // local has shadowing value

	n := &container.GraphNode{
		ID: "g1", Kind: "GetVar",
		Config: map[string]any{"varName": "hp", "scope": "global"},
	}
	v, _ := r.evalGetVar(n)
	if got, _ := expr.AsNumber(v); got != 0.8 {
		t.Fatalf("global hp must skip local shadow: want 0.8, got %v", v)
	}
}

// TestGetVar_AutoScope_FrameThenGlobal: local first, fallback to global.
func TestGetVar_AutoScope_FrameThenGlobal(t *testing.T) {
	_, r := newTestRunner(t)
	r.currentTick = CaptureSnapshot(map[string]expr.Value{"hp": float64(0.8), "other": float64(99)}, SysState{})
	r.state.SetLocalVar("hp", float64(0.5))

	// hp is in BOTH local and global → auto returns local (0.5)
	n := &container.GraphNode{
		ID: "g1", Kind: "GetVar",
		Config: map[string]any{"varName": "hp", "scope": "auto"},
	}
	v, _ := r.evalGetVar(n)
	if got, _ := expr.AsNumber(v); got != 0.5 {
		t.Fatalf("auto: frame chain first, want 0.5, got %v", v)
	}

	// 'other' only in global → auto falls back
	n.Config["varName"] = "other"
	v, _ = r.evalGetVar(n)
	if got, _ := expr.AsNumber(v); got != 99 {
		t.Fatalf("auto: fallback global for 'other', want 99, got %v", v)
	}
}

// TestGetVar_DefaultScopeIsLocal: scope unset → treated as "local" per spec §3.1.
func TestGetVar_DefaultScopeIsLocal(t *testing.T) {
	_, r := newTestRunner(t)
	r.currentTick = CaptureSnapshot(map[string]expr.Value{"hp": float64(0.8)}, SysState{})
	r.state.SetLocalVar("hp", float64(0.5))

	n := &container.GraphNode{
		ID: "g1", Kind: "GetVar",
		Config: map[string]any{"varName": "hp"}, // no scope field
	}
	v, _ := r.evalGetVar(n)
	if got, _ := expr.AsNumber(v); got != 0.5 {
		t.Fatalf("default scope must be local: want 0.5 (local), got %v", v)
	}
}

// TestGetVar_MissingVarName: errors out.
func TestGetVar_MissingVarName(t *testing.T) {
	_, r := newTestRunner(t)
	r.currentTick = CaptureSnapshot(map[string]expr.Value{}, SysState{})

	n := &container.GraphNode{ID: "g1", Kind: "GetVar", Config: map[string]any{}}
	_, err := r.evalGetVar(n)
	if err == nil {
		t.Fatal("want err for missing varName")
	}
}

// TestGetVar_UnknownScope: errors out.
func TestGetVar_UnknownScope(t *testing.T) {
	_, r := newTestRunner(t)
	r.currentTick = CaptureSnapshot(map[string]expr.Value{}, SysState{})

	n := &container.GraphNode{ID: "g1", Kind: "GetVar", Config: map[string]any{"varName": "x", "scope": "bogus"}}
	_, err := r.evalGetVar(n)
	if err == nil {
		t.Fatal("want err for unknown scope")
	}
}
