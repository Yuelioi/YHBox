package runtime

import (
	"testing"
	"time"

	"yhbox/internal/services/container"
	"yhbox/internal/services/expr"
)

func TestGetSys_LastTemplateFound(t *testing.T) {
	_, r := newTestRunner(t)
	r.currentTick = CaptureSnapshot(map[string]expr.Value{}, SysState{LastFound: true})

	n := &container.GraphNode{
		ID: "gs1", Kind: "GetSys",
		Config: map[string]any{"Path": "lastTemplate.found"},
	}
	v, err := r.evalGetSys(n)
	if err != nil {
		t.Fatal(err)
	}
	if v != true {
		t.Fatalf("lastTemplate.found: want true, got %v", v)
	}
}

func TestGetSys_Iter(t *testing.T) {
	_, r := newTestRunner(t)
	r.currentTick = CaptureSnapshot(map[string]expr.Value{}, SysState{Iter: 7})

	n := &container.GraphNode{
		ID: "gs1", Kind: "GetSys",
		Config: map[string]any{"Path": "iter"},
	}
	v, _ := r.evalGetSys(n)
	got, _ := expr.AsNumber(v)
	if got != 7.0 {
		t.Fatalf("iter: want 7, got %v", v)
	}
}

func TestGetSys_UnknownPath(t *testing.T) {
	_, r := newTestRunner(t)
	r.currentTick = CaptureSnapshot(map[string]expr.Value{}, SysState{})

	n := &container.GraphNode{
		ID: "gs1", Kind: "GetSys",
		Config: map[string]any{"Path": "bogus.path"},
	}
	_, err := r.evalGetSys(n)
	if err == nil {
		t.Fatal("want err for unknown sys path")
	}
}

// Ensure SysPathSchema (runtime) and validator's sysPathSchemaCopy stay in sync.
// (Mirror check; runs in runtime package since SysPathSchema lives there.)
func TestSysPathSchema_AllPathsResolve(t *testing.T) {
	_, r := newTestRunner(t)
	r.currentTick = CaptureSnapshot(map[string]expr.Value{}, SysState{})

	for path := range SysPathSchema {
		n := &container.GraphNode{
			ID: "gs1", Kind: "GetSys",
			Config: map[string]any{"Path": path},
		}
		_, err := r.evalGetSys(n)
		if err != nil {
			t.Errorf("SysPathSchema lists %q but resolveSysPath rejects: %v", path, err)
		}
	}
}

// TestGetSys_NowMs_LiveTimestamp: now_ms 应总是返 live 当前时间, 不读 snapshot.
// Fishing v2 watchdog 依赖此行为做时间差比较.
func TestGetSys_NowMs_LiveTimestamp(t *testing.T) {
	_, r := newTestRunner(t)
	r.currentTick = CaptureSnapshot(map[string]expr.Value{}, SysState{})

	n := &container.GraphNode{
		ID: "gs1", Kind: "GetSys",
		Config: map[string]any{"Path": "now_ms"},
	}
	before := time.Now().UnixMilli()
	v, err := r.evalGetSys(n)
	after := time.Now().UnixMilli()
	if err != nil {
		t.Fatal(err)
	}
	got, ok := expr.AsNumber(v)
	if !ok {
		t.Fatalf("now_ms 应返 number, got %T", v)
	}
	if int64(got) < before || int64(got) > after {
		t.Errorf("now_ms 应在 [%d, %d], got %d", before, after, int64(got))
	}
}

// TestGetSys_VarLastChange_LivesAfterSetVar: SetVar 后, varLastChange.<name>
// 应反映出新时间戳. Fishing v2 watchdog 依赖.
func TestGetSys_VarLastChange_LivesAfterSetVar(t *testing.T) {
	rt, r := newTestRunner(t)
	r.currentTick = CaptureSnapshot(map[string]expr.Value{}, SysState{})

	// 未 set 前应返 0
	n := &container.GraphNode{
		ID: "gs1", Kind: "GetSys",
		Config: map[string]any{"Path": "varLastChange.state"},
	}
	v, err := r.evalGetSys(n)
	if err != nil {
		t.Fatal(err)
	}
	if got, _ := expr.AsNumber(v); got != 0 {
		t.Errorf("未 set 应返 0, got %v", got)
	}

	// SetVar 后, 时间戳应在 SetVar 前后区间.
	before := time.Now().UnixMilli()
	rt.SetVar("state", "WAITING")
	after := time.Now().UnixMilli()

	v, err = r.evalGetSys(n)
	if err != nil {
		t.Fatal(err)
	}
	got, _ := expr.AsNumber(v)
	if int64(got) < before || int64(got) > after {
		t.Errorf("varLastChange.state 应在 [%d, %d], got %d", before, after, int64(got))
	}
}
