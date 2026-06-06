package system

import (
	"context"
	"errors"
	"testing"

	"yotta/internal/node"
)

func TestTry_RunRegion_BodySucceedsThenNormal(t *testing.T) {
	node.ResetRegistryForTest()
	node.Register(&Try{})
	rn, _ := node.Get("Try")

	calls := 0
	body := func(_ node.Ctx) error {
		calls++
		return nil
	}

	// SubgraphID 是 Required input — 塞 dummy 让 framework Required check 过.
	// 实际 sub-dispatch 由 runner 端 makeBodyForTry 走, body callback 这里 mock.
	cfg := map[string]any{tryInSubgraphID: "dummy"}
	r := node.RunNodeAsRegion(context.Background(), rn, nil, cfg, nil,
		node.StubServices(), false, body)

	if r.Error != nil {
		t.Fatalf("error = %v", r.Error)
	}
	if r.Panic != nil {
		t.Fatalf("panic: %v\n%s", r.Panic, r.PanicStack)
	}
	if calls != 1 {
		t.Errorf("body calls = %d, want 1", calls)
	}
	if r.ExitName != tryOutNormal {
		t.Errorf("exit = %q, want %q", r.ExitName, tryOutNormal)
	}
}

func TestTry_RunRegion_BodyErrorRoutesToCatchWithMessage(t *testing.T) {
	node.ResetRegistryForTest()
	node.Register(&Try{})
	rn, _ := node.Get("Try")

	body := func(_ node.Ctx) error { return errors.New("kaboom") }

	// SubgraphID 是 Required input — 塞 dummy 让 framework Required check 过.
	// 实际 sub-dispatch 由 runner 端 makeBodyForTry 走, body callback 这里 mock.
	cfg := map[string]any{tryInSubgraphID: "dummy"}
	r := node.RunNodeAsRegion(context.Background(), rn, nil, cfg, nil,
		node.StubServices(), false, body)

	if r.Error != nil {
		t.Fatalf("error = %v, want nil (catch should swallow)", r.Error)
	}
	if r.ExitName != tryOutCatch {
		t.Errorf("exit = %q, want %q", r.ExitName, tryOutCatch)
	}
	got, _ := r.OutputData[tryDataError].(string)
	if got != "kaboom" {
		t.Errorf("catch.error = %q, want 'kaboom'", got)
	}
}

func TestTry_RunRegion_ThrowErrorCaughtMessageStripped(t *testing.T) {
	// Throw 返 *ThrowError, Error() = "throw: <msg>". Try 用 err.Error() 字符串.
	node.ResetRegistryForTest()
	node.Register(&Try{})
	rn, _ := node.Get("Try")

	body := func(_ node.Ctx) error { return &ThrowError{Message: "fish escaped"} }

	// SubgraphID 是 Required input — 塞 dummy 让 framework Required check 过.
	// 实际 sub-dispatch 由 runner 端 makeBodyForTry 走, body callback 这里 mock.
	cfg := map[string]any{tryInSubgraphID: "dummy"}
	r := node.RunNodeAsRegion(context.Background(), rn, nil, cfg, nil,
		node.StubServices(), false, body)

	if r.Error != nil {
		t.Fatalf("error = %v, want nil (catch should swallow)", r.Error)
	}
	if r.ExitName != tryOutCatch {
		t.Errorf("exit = %q, want %q", r.ExitName, tryOutCatch)
	}
	got, _ := r.OutputData[tryDataError].(string)
	if got != "throw: fish escaped" {
		t.Errorf("catch.error = %q, want 'throw: fish escaped'", got)
	}
}

// recVars 记录式 VarStore — 验证捕获. 本包独立 stub (不跨包).
type recVars struct{ m map[string]any }

func newRecVars() *recVars { return &recVars{m: map[string]any{}} }

func (r *recVars) Get(name string) (any, bool)               { v, ok := r.m[name]; return v, ok }
func (r *recVars) Set(name string, v any)                    { r.m[name] = v }
func (r *recVars) Inc(string, float64) float64               { return 0 }
func (r *recVars) GetScoped(name, _ string) (any, bool)      { v, ok := r.m[name]; return v, ok }
func (r *recVars) SetScoped(name, _ string, v any)           { r.m[name] = v }
func (r *recVars) IncScoped(string, string, float64) float64 { return 0 }
func (r *recVars) LastChange(string) int64                   { return 0 }

func TestTry_Capture_ErrorOnCatch(t *testing.T) {
	node.ResetRegistryForTest()
	node.Register(&Try{})
	rn, _ := node.Get("Try")

	vars := newRecVars()
	services := node.StubServices()
	services.Vars = vars

	body := func(_ node.Ctx) error { return errors.New("boom") }
	cfg := map[string]any{tryInSubgraphID: "dummy", tryCapError: "e"}
	r := node.RunNodeAsRegion(context.Background(), rn, nil, cfg, nil, services, false, body)

	if r.Error != nil {
		t.Fatalf("error = %v, want nil", r.Error)
	}
	if r.ExitName != tryOutCatch {
		t.Fatalf("exit = %q, want %q", r.ExitName, tryOutCatch)
	}
	got, ok := vars.Get("e")
	if !ok || got != "boom" {
		t.Errorf("capture e = %v (ok=%v), want 'boom'", got, ok)
	}
}

func TestTry_Capture_EmptyOnNormal(t *testing.T) {
	node.ResetRegistryForTest()
	node.Register(&Try{})
	rn, _ := node.Get("Try")

	vars := newRecVars()
	services := node.StubServices()
	services.Vars = vars

	body := func(_ node.Ctx) error { return nil }
	cfg := map[string]any{tryInSubgraphID: "dummy", tryCapError: "e"}
	r := node.RunNodeAsRegion(context.Background(), rn, nil, cfg, nil, services, false, body)

	if r.Error != nil {
		t.Fatalf("error = %v, want nil", r.Error)
	}
	if r.ExitName != tryOutNormal {
		t.Fatalf("exit = %q, want %q", r.ExitName, tryOutNormal)
	}
	got, ok := vars.Get("e")
	if !ok || got != "" {
		t.Errorf("capture e = %v (ok=%v), want '' (empty)", got, ok)
	}
}

func TestTry_Spec_CatchHasErrorDataField(t *testing.T) {
	sp := Try{}.Spec()
	var catch *node.OutputSpec
	for i := range sp.Outputs {
		if sp.Outputs[i].Name == tryOutCatch {
			catch = &sp.Outputs[i]
			break
		}
	}
	if catch == nil {
		t.Fatalf("no %q output in spec", tryOutCatch)
	}
	if len(catch.Data) != 1 || catch.Data[0].Name != tryDataError {
		t.Errorf("catch.Data = %+v, want [{error}]", catch.Data)
	}
}
