// internal/node/snapshot_test.go
package node

import "testing"

// fakeLiveVarStore — minimal in-memory VarStore for testing snapshot wrap.
// 支持 local scope (frame-style map) + global (单独 map) + auto.
type fakeLiveVarStore struct {
	global map[string]any
	local  map[string]any // 模拟 frame.LocalVars
}

func newFakeLiveVarStore() *fakeLiveVarStore {
	return &fakeLiveVarStore{global: map[string]any{}, local: map[string]any{}}
}

func (f *fakeLiveVarStore) Get(name string) (any, bool) {
	v, ok := f.global[name]
	return v, ok
}
func (f *fakeLiveVarStore) Set(name string, value any)             { f.global[name] = value }
func (f *fakeLiveVarStore) Inc(name string, delta float64) float64 { return 0 }
func (f *fakeLiveVarStore) GetScoped(name, scope string) (any, bool) {
	switch scope {
	case "local":
		v, ok := f.local[name]
		return v, ok
	case "auto":
		if v, ok := f.local[name]; ok {
			return v, true
		}
		v, ok := f.global[name]
		return v, ok
	default:
		v, ok := f.global[name]
		return v, ok
	}
}
func (f *fakeLiveVarStore) SetScoped(name, scope string, value any) {
	if scope == "local" {
		f.local[name] = value
	} else {
		f.global[name] = value
	}
}
func (f *fakeLiveVarStore) IncScoped(name, scope string, delta float64) float64 { return 0 }
func (f *fakeLiveVarStore) LastChange(name string) int64                        { return 0 }

func TestSnapshotVarStore_GlobalReturnsFrozen(t *testing.T) {
	live := newFakeLiveVarStore()
	live.global["x"] = 1
	snap := Snapshot{Vars: map[string]any{"x": 1}}
	ss := newSnapshotVarStore(live, snap)

	live.global["x"] = 99 // mutate live after snapshot
	v, ok := ss.GetScoped("x", "global")
	if !ok || v != 1 {
		t.Errorf("expected frozen value 1, got %v ok=%v", v, ok)
	}
}

func TestSnapshotVarStore_LocalPassesThroughLive(t *testing.T) {
	live := newFakeLiveVarStore()
	live.local["fp"] = "live"
	snap := Snapshot{Vars: map[string]any{}}
	ss := newSnapshotVarStore(live, snap)

	v, ok := ss.GetScoped("fp", "local")
	if !ok || v != "live" {
		t.Errorf("local scope must pass through live; got %v ok=%v", v, ok)
	}
}

func TestSnapshotVarStore_AutoLocalFirstSnapshotFallback(t *testing.T) {
	live := newFakeLiveVarStore()
	live.local["onlyLocal"] = "L"
	live.global["onlyGlobal"] = "live-G" // live; snapshot 没这值
	snap := Snapshot{Vars: map[string]any{"onlyGlobal": "snap-G", "bothInSnap": "snap"}}
	ss := newSnapshotVarStore(live, snap)

	// local 优先
	if v, _ := ss.GetScoped("onlyLocal", "auto"); v != "L" {
		t.Errorf("auto: local first, got %v", v)
	}
	// 没 local 走 snapshot, 不走 live
	if v, _ := ss.GetScoped("onlyGlobal", "auto"); v != "snap-G" {
		t.Errorf("auto: fallback to snapshot, got %v (expected snap-G not live-G)", v)
	}
}

func TestSnapshotVarStore_SetPanics(t *testing.T) {
	ss := newSnapshotVarStore(newFakeLiveVarStore(), Snapshot{Vars: map[string]any{}})
	cases := []struct {
		name string
		fn   func()
	}{
		{"Set", func() { ss.Set("x", 1) }},
		{"Inc", func() { ss.Inc("x", 1) }},
		{"SetScoped-global", func() { ss.SetScoped("x", "global", 1) }},
		{"SetScoped-local", func() { ss.SetScoped("x", "local", 1) }},
		{"IncScoped", func() { ss.IncScoped("x", "global", 1) }},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			defer func() {
				if r := recover(); r == nil {
					t.Errorf("expected panic on %s, got none", c.name)
				}
			}()
			c.fn()
		})
	}
}
