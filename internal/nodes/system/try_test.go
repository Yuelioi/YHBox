package system

import (
	"context"
	"errors"
	"testing"

	"yhbox/internal/node"
)

func TestTry_Run_ReturnsMustUseRegionSentinel(t *testing.T) {
	node.ResetRegistryForTest()
	node.Register(&Try{})
	rn, _ := node.Get("Try")

	r := node.RunNode(context.Background(), rn, nil, nil, nil, node.StubServices())
	if !errors.Is(r.Error, errTryMustUseRegion) {
		t.Errorf("Run error = %v, want errTryMustUseRegion", r.Error)
	}
}

func TestTry_RunRegion_BodySucceedsThenNormal(t *testing.T) {
	node.ResetRegistryForTest()
	node.Register(&Try{})
	rn, _ := node.Get("Try")

	calls := 0
	body := func(_ node.Ctx) error {
		calls++
		return nil
	}

	r := node.RunNodeAsRegion(context.Background(), rn, nil, nil, nil,
		node.StubServices(), body)

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

	r := node.RunNodeAsRegion(context.Background(), rn, nil, nil, nil,
		node.StubServices(), body)

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

	r := node.RunNodeAsRegion(context.Background(), rn, nil, nil, nil,
		node.StubServices(), body)

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
