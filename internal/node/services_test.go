// internal/node/services_test.go
// stub-service sanity tests. 验:
//   1. Stub constructor 不返 nil interface (不会让 ctx.X() return nil 触发 panic at call site).
//   2. 每个 stub 方法跑通不 panic, 返 zero/no-op.
//   3. VarStore / SysStore / StopwatchStore 的 set/get 双向行为正确.
package node

import (
	"context"
	"testing"
	"time"
)

// ---- InputService stubs ----

func TestStubInputService_AllMethodsNoOp(t *testing.T) {
	in := StubInputService()
	if in == nil {
		t.Fatal("StubInputService() returned nil")
	}
	checks := []func() error{
		func() error { return in.KeyPress("A", 50) },
		func() error { return in.KeyDown("A") },
		func() error { return in.KeyUp("A") },
		func() error { return in.Click(0.5, 0.5, "left", 50) },
		func() error { return in.MouseMoveRel(10, 20, 100) },
		func() error { return in.Scroll(0.5, 0.5, 3) },
		func() error { return in.MouseDown(0.5, 0.5, "left") },
		func() error { return in.MouseUp("left") },
	}
	for i, fn := range checks {
		if err := fn(); err != nil {
			t.Errorf("stub method #%d returned err: %v (expected nil)", i, err)
		}
	}
}

// ---- VarStore stub ----

func TestStubVarStore_SetGetInc(t *testing.T) {
	vs := NewStubVarStore()
	if vs == nil {
		t.Fatal("NewStubVarStore() returned nil")
	}

	// Get on empty
	if _, ok := vs.Get("x"); ok {
		t.Error("Get(x) on empty should return ok=false")
	}

	// Set + Get
	vs.Set("x", 42)
	v, ok := vs.Get("x")
	if !ok || v != 42 {
		t.Errorf("after Set(x,42): Get=%v ok=%v, want 42 true", v, ok)
	}

	// Inc on missing → cur=0, return delta
	n := vs.Inc("y", 1.5)
	if n != 1.5 {
		t.Errorf("Inc(y,1.5) on missing = %v, want 1.5", n)
	}
	v2, _ := vs.Get("y")
	if v2 != 1.5 {
		t.Errorf("after Inc(y,1.5): Get(y)=%v, want 1.5", v2)
	}

	// Inc on existing number
	n2 := vs.Inc("y", 2.5)
	if n2 != 4.0 {
		t.Errorf("Inc(y,2.5) after 1.5 = %v, want 4.0", n2)
	}

	// Inc on int value coerces
	vs.Set("z", 10)
	n3 := vs.Inc("z", 5)
	if n3 != 15.0 {
		t.Errorf("Inc(z,5) after int 10 = %v, want 15", n3)
	}
}

// ---- SysStore stub ----

func TestStubSysStore_GetSetForTest(t *testing.T) {
	s := NewStubSysStore()
	if s == nil {
		t.Fatal("NewStubSysStore() returned nil")
	}

	// Get on empty
	if _, ok := s.Get("now_ms"); ok {
		t.Error("Get(now_ms) on empty should return ok=false")
	}

	// SetForTest + Get
	s.SetForTest("now_ms", int64(123456))
	v, ok := s.Get("now_ms")
	if !ok || v != int64(123456) {
		t.Errorf("Get(now_ms) = %v ok=%v, want 123456 true", v, ok)
	}

	// 跟 SysStore interface 也能用
	var iface SysStore = s
	if _, ok := iface.Get("none"); ok {
		t.Error("interface Get on missing should ok=false")
	}
}

// ---- WindowService stub ----

func TestStubWindowService_ZeroValues(t *testing.T) {
	w := StubWindowService()
	if w == nil {
		t.Fatal("StubWindowService() returned nil")
	}
	if err := w.BringForeground(); err != nil {
		t.Errorf("BringForeground returned err: %v", err)
	}
	if h := w.HWND(); h != 0 {
		t.Errorf("HWND() = %v, want 0", h)
	}
	cw, ch, err := w.ClientSize()
	if err != nil || cw != 0 || ch != 0 {
		t.Errorf("ClientSize = %d,%d,%v want 0,0,nil", cw, ch, err)
	}
}

// ---- CaptureService stub ----

func TestStubCaptureService_NilBytes(t *testing.T) {
	c := StubCaptureService()
	if c == nil {
		t.Fatal("StubCaptureService() returned nil")
	}
	b, err := c.Capture()
	if err != nil || b != nil {
		t.Errorf("Capture = %v,%v want nil,nil", b, err)
	}
	b2, err2 := c.CaptureROI(Geometry{})
	if err2 != nil || b2 != nil {
		t.Errorf("CaptureROI = %v,%v want nil,nil", b2, err2)
	}
}

// ---- StopwatchStore stub ----

func TestStubStopwatchStore_StartStopRead(t *testing.T) {
	s := NewStubStopwatchStore()
	if s == nil {
		t.Fatal("NewStubStopwatchStore() returned nil")
	}

	// Read missing → 0
	if e := s.Read("fish"); e != 0 {
		t.Errorf("Read(missing) = %d, want 0", e)
	}

	// Stop missing → no-op (no panic)
	s.Stop("fish")

	// Start + Read while running
	s.Start("fish")
	time.Sleep(20 * time.Millisecond)
	e := s.Read("fish")
	if e < 15 {
		t.Errorf("Read(running) = %d, want >= 15", e)
	}

	// Stop + Read frozen
	s.Stop("fish")
	e1 := s.Read("fish")
	time.Sleep(20 * time.Millisecond)
	e2 := s.Read("fish")
	if e1 != e2 {
		t.Errorf("Read after Stop changed: e1=%d e2=%d, want frozen", e1, e2)
	}

	// Start again resets
	s.Start("fish")
	e3 := s.Read("fish")
	if e3 > 5 {
		t.Errorf("Read after Start (reset) = %d, want < 5", e3)
	}
}

// ---- VisionService stub ----

func TestStubVisionService_AlwaysMiss(t *testing.T) {
	v := StubVisionService()
	if v == nil {
		t.Fatal("StubVisionService() returned nil")
	}
	pt, conf, err := v.Match(context.Background(), "any", 0.5)
	if pt != nil || conf != 0 || err != nil {
		t.Errorf("Match = (%v,%v,%v), want (nil,0,nil)", pt, conf, err)
	}
	pt2, _, err2 := v.WaitMatch(context.Background(), "any", 0.5, 10*time.Millisecond)
	if pt2 != nil || err2 != nil {
		t.Errorf("WaitMatch = (%v,%v), want (nil,nil)", pt2, err2)
	}
	br, err3 := v.DualBarTrack(Rect{X: 0, Y: 0, W: 100, H: 50}, HSVRange{}, HSVRange{}, DualBarOptions{})
	if err3 != nil || br.Found {
		t.Errorf("DualBarTrack = (%+v,%v), want zero/Found=false,nil", br, err3)
	}
}

// ---- LogService default ----

func TestDefaultLogService_NoPanic(t *testing.T) {
	l := DefaultLogService()
	if l == nil {
		t.Fatal("DefaultLogService() returned nil")
	}
	// 只测不 panic. log 写 stderr 默认, 测试时容忍.
	l.Debug("test %d", 1)
	l.Info("test %s", "ok")
	l.Warn("test")
}

// ---- ServiceBundle helper ----

func TestStubServices_AllFieldsNonNil(t *testing.T) {
	b := StubServices()
	if b.Vision == nil {
		t.Error("StubServices().Vision is nil")
	}
	if b.Log == nil {
		t.Error("StubServices().Log is nil")
	}
	if b.Input == nil {
		t.Error("StubServices().Input is nil")
	}
	if b.Vars == nil {
		t.Error("StubServices().Vars is nil")
	}
	if b.Sys == nil {
		t.Error("StubServices().Sys is nil")
	}
	if b.Window == nil {
		t.Error("StubServices().Window is nil")
	}
	if b.Capture == nil {
		t.Error("StubServices().Capture is nil")
	}
	if b.Stopwatches == nil {
		t.Error("StubServices().Stopwatches is nil")
	}
}
