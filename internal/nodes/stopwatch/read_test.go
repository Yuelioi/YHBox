package stopwatch

import (
	"context"
	"testing"
	"time"

	"yotta/internal/node"
)

func TestRead_RunningReturnsPositive(t *testing.T) {
	node.ResetRegistryForTest()
	node.Register(&Read{})
	rn, _ := node.Get("StopwatchRead")

	services := node.StubServices()
	services.Stopwatches.Start("k")
	time.Sleep(10 * time.Millisecond)

	r := node.RunNode(context.Background(), rn, nil,
		map[string]any{swReadInKey: "k"}, nil, services, false)
	if r.Error != nil {
		t.Fatal(r.Error)
	}
	if r.ExitName != swReadOutOut {
		t.Errorf("exit = %q, want Out", r.ExitName)
	}
	elapsed, _ := r.OutputData[swReadDataElapsedMs].(int64)
	if elapsed <= 0 {
		t.Errorf("ElapsedMs = %d, want > 0 after 10ms sleep", elapsed)
	}
}

func TestRead_MissingKeyReturnsZero(t *testing.T) {
	node.ResetRegistryForTest()
	node.Register(&Read{})
	rn, _ := node.Get("StopwatchRead")

	services := node.StubServices()
	r := node.RunNode(context.Background(), rn, nil,
		map[string]any{swReadInKey: "never_started"}, nil, services, false)
	if r.Error != nil {
		t.Fatal(r.Error)
	}
	elapsed, _ := r.OutputData[swReadDataElapsedMs].(int64)
	if elapsed != 0 {
		t.Errorf("ElapsedMs = %d, want 0 for missing key", elapsed)
	}
}

// recVars 记录式 VarStore — 验证捕获. 本包独立 stub (不跨包).
type recVars struct{ m map[string]any }

func newRecVars() *recVars { return &recVars{m: map[string]any{}} }

func (r *recVars) Get(name string) (any, bool)             { v, ok := r.m[name]; return v, ok }
func (r *recVars) Set(name string, v any)                  { r.m[name] = v }
func (r *recVars) Inc(string, float64) float64             { return 0 }
func (r *recVars) GetScoped(name, _ string) (any, bool)    { v, ok := r.m[name]; return v, ok }
func (r *recVars) SetScoped(name, _ string, v any)         { r.m[name] = v }
func (r *recVars) IncScoped(string, string, float64) float64 { return 0 }
func (r *recVars) LastChange(string) int64                 { return 0 }

func TestRead_Capture_ElapsedMs(t *testing.T) {
	node.ResetRegistryForTest()
	node.Register(&Read{})
	rn, _ := node.Get("StopwatchRead")

	services := node.StubServices()
	vars := newRecVars()
	services.Vars = vars
	services.Stopwatches.Start("default")
	time.Sleep(10 * time.Millisecond)

	r := node.RunNode(context.Background(), rn, nil,
		map[string]any{swReadInKey: "default", swReadCapElapsed: "e"}, nil, services, false)
	if r.Error != nil {
		t.Fatal(r.Error)
	}
	want, _ := r.OutputData[swReadDataElapsedMs].(int64)
	got, ok := vars.Get("e")
	if !ok {
		t.Fatal("capture e not written")
	}
	if got != want {
		t.Errorf("capture e = %v, want %v (= ElapsedMs out)", got, want)
	}
}

func TestRead_StoppedReturnsFrozen(t *testing.T) {
	node.ResetRegistryForTest()
	node.Register(&Read{})
	rn, _ := node.Get("StopwatchRead")

	services := node.StubServices()
	services.Stopwatches.Start("k")
	time.Sleep(10 * time.Millisecond)
	services.Stopwatches.Stop("k")

	r1 := node.RunNode(context.Background(), rn, nil,
		map[string]any{swReadInKey: "k"}, nil, services, false)
	first, _ := r1.OutputData[swReadDataElapsedMs].(int64)

	time.Sleep(15 * time.Millisecond)

	r2 := node.RunNode(context.Background(), rn, nil,
		map[string]any{swReadInKey: "k"}, nil, services, false)
	second, _ := r2.OutputData[swReadDataElapsedMs].(int64)

	if first != second {
		t.Errorf("stopped Read drift first=%d second=%d (should be frozen)", first, second)
	}
}
