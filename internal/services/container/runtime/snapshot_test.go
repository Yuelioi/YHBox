package runtime

import (
	"testing"

	"yhbox/internal/services/expr"
)

func TestSnapshotIsACopy(t *testing.T) {
	vars := map[string]expr.Value{"hp": float64(0.8)}
	snap := CaptureSnapshot(vars, SysState{})
	// Mutate backing store after snapshot — snapshot must NOT see it.
	vars["hp"] = float64(0.2)

	v, ok := snap.GetVar("hp")
	if !ok {
		t.Fatal("snapshot lost hp key")
	}
	if v != expr.Value(0.8) {
		t.Fatalf("snapshot must be isolated from backing store: got %v want 0.8", v)
	}
}

func TestSnapshotCapturesSys(t *testing.T) {
	sys := SysState{LastFound: true, Iter: 5}
	snap := CaptureSnapshot(nil, sys)
	if !snap.Sys.LastFound {
		t.Fatal("snapshot Sys.LastFound lost")
	}
	if snap.Sys.Iter != 5 {
		t.Fatalf("snapshot Sys.Iter: got %d want 5", snap.Sys.Iter)
	}
}

func TestSnapshotGetVarNilSafe(t *testing.T) {
	var snap *TickSnapshot
	v, ok := snap.GetVar("x")
	if ok || v != nil {
		t.Fatalf("nil snapshot: want (nil,false), got (%v,%v)", v, ok)
	}
}

func TestNewTickSnapshot_Empty(t *testing.T) {
	s := NewTickSnapshot()
	if s.Vars == nil {
		t.Fatal("NewTickSnapshot must init Vars map")
	}
	if len(s.Vars) != 0 {
		t.Fatalf("want empty Vars, got %d entries", len(s.Vars))
	}
}
