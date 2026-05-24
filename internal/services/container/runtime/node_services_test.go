package runtime

import (
	"testing"
	"time"

	"github.com/rs/zerolog"

	"yhbox/internal/node"
	"yhbox/internal/services/container"
	"yhbox/internal/services/execution"
)

// newAdapterTestRT 构造一个最小 RuntimeContext 供 adapter 测试用.
// vars 预填 — 镜像 Container.Vars 装载流程.
func newAdapterTestRT(t *testing.T, vars []container.VarDecl) *RuntimeContext {
	t.Helper()
	c := &container.Container{
		SchemaVersion: 1,
		ID:            "test-adapters",
		Name:          "test-adapters",
		Vars:          vars,
		Graph: container.Graph{
			Nodes: []container.GraphNode{{ID: "start", Kind: "Start"}},
		},
	}
	return NewRuntimeContext(c, execution.NewInputBus(), NoopMatcher{}, NoopColorDetector{}, nil, nil, nil, nil, 0)
}

// ============================================================================
// VarStoreAdapter
// ============================================================================

func TestVarStoreAdapter_GetUnset(t *testing.T) {
	rt := newAdapterTestRT(t, nil)
	a := NewVarStoreAdapter(rt)
	if _, ok := a.Get("missing"); ok {
		t.Errorf("Get unknown var should be (nil, false), got ok=true")
	}
}

func TestVarStoreAdapter_GetDeclared(t *testing.T) {
	rt := newAdapterTestRT(t, []container.VarDecl{
		{Name: "count", Type: "number", Default: 7.0},
		{Name: "flag", Type: "bool", Default: true},
	})
	a := NewVarStoreAdapter(rt)
	if v, ok := a.Get("count"); !ok || v.(float64) != 7.0 {
		t.Errorf("Get count = %v, %v; want 7.0, true", v, ok)
	}
	if v, ok := a.Get("flag"); !ok || v.(bool) != true {
		t.Errorf("Get flag = %v, %v; want true, true", v, ok)
	}
}

func TestVarStoreAdapter_SetGetRoundTrip(t *testing.T) {
	rt := newAdapterTestRT(t, nil)
	a := NewVarStoreAdapter(rt)
	a.Set("x", "hello")
	a.Set("n", 42.5)
	if v, ok := a.Get("x"); !ok || v.(string) != "hello" {
		t.Errorf("Get x = %v, %v; want \"hello\", true", v, ok)
	}
	if v, ok := a.Get("n"); !ok || v.(float64) != 42.5 {
		t.Errorf("Get n = %v, %v; want 42.5, true", v, ok)
	}
}

func TestVarStoreAdapter_SetUpdatesTimestamp(t *testing.T) {
	rt := newAdapterTestRT(t, nil)
	a := NewVarStoreAdapter(rt)
	before := time.Now().UnixMilli()
	a.Set("x", 1.0)
	ts := rt.VarLastChange("x")
	if ts < before {
		t.Errorf("VarLastChange = %d, want >= %d (live time)", ts, before)
	}
}

func TestVarStoreAdapter_IncFromZero(t *testing.T) {
	rt := newAdapterTestRT(t, nil)
	a := NewVarStoreAdapter(rt)
	got := a.Inc("counter", 3.0)
	if got != 3.0 {
		t.Errorf("Inc from unset = %v, want 3.0", got)
	}
	if v, ok := a.Get("counter"); !ok || v.(float64) != 3.0 {
		t.Errorf("after Inc Get = %v, %v; want 3.0, true", v, ok)
	}
}

func TestVarStoreAdapter_IncTwice(t *testing.T) {
	rt := newAdapterTestRT(t, []container.VarDecl{{Name: "n", Type: "number", Default: 10.0}})
	a := NewVarStoreAdapter(rt)
	if got := a.Inc("n", 5.0); got != 15.0 {
		t.Errorf("first Inc = %v, want 15.0", got)
	}
	if got := a.Inc("n", -3.0); got != 12.0 {
		t.Errorf("second Inc = %v, want 12.0", got)
	}
}

// ============================================================================
// SysStoreAdapter
// ============================================================================

func TestSysStoreAdapter_NowMs(t *testing.T) {
	rt := newAdapterTestRT(t, nil)
	a := NewSysStoreAdapter(rt)
	before := float64(time.Now().UnixMilli())
	v, ok := a.Get("now_ms")
	after := float64(time.Now().UnixMilli())
	if !ok {
		t.Fatal("Get now_ms returned ok=false")
	}
	got := v.(float64)
	if got < before || got > after {
		t.Errorf("now_ms = %v, want in [%v, %v]", got, before, after)
	}
}

func TestSysStoreAdapter_VarLastChange(t *testing.T) {
	rt := newAdapterTestRT(t, nil)
	a := NewSysStoreAdapter(rt)

	// 未 SetVar 时返 0.
	if v, ok := a.Get("varLastChange.foo"); !ok || v.(float64) != 0 {
		t.Errorf("unset varLastChange = %v, %v; want 0, true", v, ok)
	}
	// SetVar 后跟 rt.VarLastChange 同步.
	rt.SetVar("foo", "bar")
	expected := float64(rt.VarLastChange("foo"))
	if v, ok := a.Get("varLastChange.foo"); !ok || v.(float64) != expected {
		t.Errorf("after SetVar varLastChange = %v, %v; want %v, true", v, ok, expected)
	}
}

func TestSysStoreAdapter_ResolveSysPath(t *testing.T) {
	rt := newAdapterTestRT(t, nil)
	a := NewSysStoreAdapter(rt)
	rt.UpdateSys(func(s *SysState) {
		s.LastFound = true
		s.LastBarTrack.CursorX = 123
	})
	if v, ok := a.Get("lastTemplate.found"); !ok || v.(bool) != true {
		t.Errorf("lastTemplate.found = %v, %v; want true, true", v, ok)
	}
	if v, ok := a.Get("lastBarTrack.cursorX"); !ok || v.(float64) != 123 {
		t.Errorf("lastBarTrack.cursorX = %v, %v; want 123, true", v, ok)
	}
}

func TestSysStoreAdapter_UnknownPath(t *testing.T) {
	rt := newAdapterTestRT(t, nil)
	a := NewSysStoreAdapter(rt)
	if v, ok := a.Get("does.not.exist"); ok {
		t.Errorf("unknown path returned (%v, true), want (_, false)", v)
	}
}

// ============================================================================
// StopwatchAdapter
// ============================================================================

func TestStopwatchAdapter_ReadUnknown(t *testing.T) {
	a := NewStopwatchAdapter(newStopwatchTable())
	if got := a.Read("never-started"); got != 0 {
		t.Errorf("Read unknown = %d, want 0", got)
	}
}

func TestStopwatchAdapter_StartReadRunning(t *testing.T) {
	a := NewStopwatchAdapter(newStopwatchTable())
	a.Start("k")
	time.Sleep(15 * time.Millisecond)
	got := a.Read("k")
	if got < 10 {
		t.Errorf("Read after 15ms = %d, want >= 10", got)
	}
}

func TestStopwatchAdapter_StopFreezesElapsed(t *testing.T) {
	a := NewStopwatchAdapter(newStopwatchTable())
	a.Start("k")
	time.Sleep(15 * time.Millisecond)
	a.Stop("k")
	first := a.Read("k")
	time.Sleep(15 * time.Millisecond)
	second := a.Read("k")
	if first != second {
		t.Errorf("Read after Stop should be stable: first=%d second=%d", first, second)
	}
}

func TestStopwatchAdapter_RestartResets(t *testing.T) {
	a := NewStopwatchAdapter(newStopwatchTable())
	a.Start("k")
	time.Sleep(20 * time.Millisecond)
	a.Start("k") // restart
	got := a.Read("k")
	if got >= 20 {
		t.Errorf("Read after restart = %d, want < 20 (reset)", got)
	}
}

// ============================================================================
// ServiceBundle 构造 smoke
// ============================================================================

func TestNewServiceBundleFor_AllSlotsFilled(t *testing.T) {
	rt := newAdapterTestRT(t, nil)
	bundle := NewServiceBundleFor(rt, newStopwatchTable(), zerolog.Nop())
	if bundle.Vision == nil {
		t.Error("bundle.Vision is nil")
	}
	if bundle.Log == nil {
		t.Error("bundle.Log is nil")
	}
	if bundle.Input == nil {
		t.Error("bundle.Input is nil")
	}
	if bundle.Vars == nil {
		t.Error("bundle.Vars is nil")
	}
	if bundle.Sys == nil {
		t.Error("bundle.Sys is nil")
	}
	if bundle.Window == nil {
		t.Error("bundle.Window is nil")
	}
	if bundle.Capture == nil {
		t.Error("bundle.Capture is nil")
	}
	if bundle.Stopwatches == nil {
		t.Error("bundle.Stopwatches is nil")
	}
}

// ============================================================================
// Adapter interface compliance (compile-time)
// ============================================================================

var (
	_ node.VarStore       = (*varStoreAdapter)(nil)
	_ node.SysStore       = (*sysStoreAdapter)(nil)
	_ node.StopwatchStore = (*stopwatchAdapter)(nil)
	_ node.InputService   = (*inputAdapter)(nil)
	_ node.WindowService  = (*windowAdapter)(nil)
	_ node.CaptureService = (*captureAdapter)(nil)
	_ node.VisionService  = (*visionAdapter)(nil)
	_ node.LogService     = logAdapter{}
)

