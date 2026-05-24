// internal/node/engine_test.go
package node

import (
	"context"
	"errors"
	"testing"
)

type happyNode struct{}

func (happyNode) Spec() Spec {
	return Spec{
		Kind:    "Happy",
		Inputs:  []InputSpec{{Name: "X", Type: "String", Default: "default-x"}},
		Outputs: []OutputSpec{{Name: "out", Type: "Exec", Data: []DataField{{Name: "echo", Type: "String"}}}},
	}
}

func (happyNode) Run(ctx Ctx, in Inputs) (Outputs, error) {
	return ctx.Out("out").Set("echo", in.String("X")).Fire(), nil
}

func TestRunNode_HappyPath(t *testing.T) {
	ResetRegistryForTest()
	Register(happyNode{})
	rn, _ := Get("Happy")

	r := RunNode(context.Background(), rn, nil, nil, nil, StubServices())
	if r.Error != nil {
		t.Fatalf("error: %v", r.Error)
	}
	if r.Panic != nil {
		t.Fatalf("panic: %v", r.Panic)
	}
	if r.ExitName != "out" {
		t.Errorf("exit = %q, want out", r.ExitName)
	}
	if r.OutputData["echo"] != "default-x" {
		t.Errorf("echo = %v, want default-x", r.OutputData["echo"])
	}
}

type requiredNode struct{}

func (requiredNode) Spec() Spec {
	return Spec{
		Kind:    "Req",
		Inputs:  []InputSpec{{Name: "X", Type: "String", Required: true}},
		Outputs: []OutputSpec{{Name: "out", Type: "Exec"}},
	}
}

func (requiredNode) Run(ctx Ctx, in Inputs) (Outputs, error) {
	return ctx.Out("out").Fire(), nil
}

func TestRunNode_RequiredMissing_ValidationError(t *testing.T) {
	ResetRegistryForTest()
	Register(requiredNode{})
	rn, _ := Get("Req")

	r := RunNode(context.Background(), rn, nil, nil, nil, StubServices())
	if len(r.Validation) != 1 || r.Validation[0].Code != "REQUIRED_FIELD_MISSING" {
		t.Errorf("validation = %v, want 1 REQUIRED_FIELD_MISSING", r.Validation)
	}
	if r.Panic != nil {
		t.Errorf("Required missing should NOT panic (GPT r4 #8), got panic: %v", r.Panic)
	}
}

type errorNode struct{}

func (errorNode) Spec() Spec {
	return Spec{Kind: "Err", Outputs: []OutputSpec{{Name: "out", Type: "Exec"}}}
}

func (errorNode) Run(ctx Ctx, in Inputs) (Outputs, error) {
	return nil, errors.New("boom")
}

func TestRunNode_RuntimeError(t *testing.T) {
	ResetRegistryForTest()
	Register(errorNode{})
	rn, _ := Get("Err")

	r := RunNode(context.Background(), rn, nil, nil, nil, StubServices())
	if r.Error == nil || r.Error.Error() != "boom" {
		t.Errorf("error = %v, want boom", r.Error)
	}
}

type panicNode struct{}

func (panicNode) Spec() Spec {
	return Spec{Kind: "Panic", Outputs: []OutputSpec{{Name: "out", Type: "Exec"}}}
}

func (panicNode) Run(ctx Ctx, in Inputs) (Outputs, error) {
	panic("framework invariant broken")
}

func TestRunNode_Panic_Recovered(t *testing.T) {
	ResetRegistryForTest()
	Register(panicNode{})
	rn, _ := Get("Panic")

	r := RunNode(context.Background(), rn, nil, nil, nil, StubServices())
	if r.Panic == nil {
		t.Error("expected panic recovered")
	}
	if r.PanicStack == "" {
		t.Error("PanicStack should be captured")
	}
}

type doubleFireNode struct{}

func (doubleFireNode) Spec() Spec {
	return Spec{Kind: "DF", Outputs: []OutputSpec{{Name: "out", Type: "Exec"}}}
}

func (doubleFireNode) Run(ctx Ctx, in Inputs) (Outputs, error) {
	ctx.Out("out").Fire()
	return ctx.Out("out").Fire(), nil
}

func TestRunNode_DoubleFire_Panics(t *testing.T) {
	ResetRegistryForTest()
	Register(doubleFireNode{})
	rn, _ := Get("DF")

	r := RunNode(context.Background(), rn, nil, nil, nil, StubServices())
	if r.Panic == nil {
		t.Error("double Fire should panic")
	}
}

// ============================================================================
// EvaluatePureData (Phase 6+ partial)
// ============================================================================

// pureAdd minimal Evaluator-implementing pure-data node for engine test (避免依赖 purefunc 包).
type pureAdd struct{}

func (pureAdd) Spec() Spec {
	return Spec{
		Kind: "PureAdd",
		Inputs: []InputSpec{
			{Name: "a", Type: "Number", Default: 0.0},
			{Name: "b", Type: "Number", Default: 0.0},
		},
		Outputs:    []OutputSpec{{Name: "result", Type: "Number"}},
		IsPureData: true,
	}
}
func (pureAdd) Run(_ Ctx, _ Inputs) (Outputs, error)         { return nil, errors.New("Run should not be called") }
func (pureAdd) Evaluate(_ Ctx, in Inputs) (any, error)       { return in.Float64("a") + in.Float64("b"), nil }

func TestEvaluatePureData_Happy(t *testing.T) {
	ResetRegistryForTest()
	Register(pureAdd{})
	rn, _ := Get("PureAdd")

	got, err := EvaluatePureData(context.Background(), rn,
		map[string]any{"a": 3.0, "b": 4.0}, nil, StubServices())
	if err != nil {
		t.Fatalf("EvaluatePureData: %v", err)
	}
	if got != 7.0 {
		t.Errorf("result = %v, want 7", got)
	}
}

// pureMissingReq Required field missing → EvaluatePureData 返 error (没 ValidationError slot).
type pureMissingReq struct{}

func (pureMissingReq) Spec() Spec {
	return Spec{
		Kind: "PureReq",
		Inputs: []InputSpec{
			{Name: "x", Type: "Number", Required: true},
		},
		Outputs:    []OutputSpec{{Name: "result", Type: "Number"}},
		IsPureData: true,
	}
}
func (pureMissingReq) Run(_ Ctx, _ Inputs) (Outputs, error)   { return nil, nil }
func (pureMissingReq) Evaluate(_ Ctx, in Inputs) (any, error) { return in.Float64("x"), nil }

func TestEvaluatePureData_RequiredMissing_Errors(t *testing.T) {
	ResetRegistryForTest()
	Register(pureMissingReq{})
	rn, _ := Get("PureReq")

	_, err := EvaluatePureData(context.Background(), rn, nil, nil, StubServices())
	if err == nil {
		t.Fatal("expected validation error for missing Required, got nil")
	}
}

// pureNonEvaluator IsPureData but no Evaluate method → EvaluatePureData 返 error.
type pureNonEvaluator struct{}

func (pureNonEvaluator) Spec() Spec {
	return Spec{Kind: "PureNoEval", Outputs: []OutputSpec{{Name: "result", Type: "*"}}, IsPureData: true}
}
func (pureNonEvaluator) Run(_ Ctx, _ Inputs) (Outputs, error) { return nil, nil }

func TestEvaluatePureData_NoEvaluator_Errors(t *testing.T) {
	ResetRegistryForTest()
	Register(pureNonEvaluator{})
	rn, _ := Get("PureNoEval")

	_, err := EvaluatePureData(context.Background(), rn, nil, nil, StubServices())
	if err == nil {
		t.Fatal("expected error for node without Evaluator")
	}
}

// Non-pure-data node → EvaluatePureData rejects.
func TestEvaluatePureData_NotIsPureData_Errors(t *testing.T) {
	ResetRegistryForTest()
	Register(happyNode{})
	rn, _ := Get("Happy")

	_, err := EvaluatePureData(context.Background(), rn, nil, nil, StubServices())
	if err == nil {
		t.Fatal("expected error for non-IsPureData node")
	}
}

// purePanic Evaluate panics → recover, return error.
type purePanic struct{}

func (purePanic) Spec() Spec {
	return Spec{Kind: "PurePanic", Outputs: []OutputSpec{{Name: "result", Type: "*"}}, IsPureData: true}
}
func (purePanic) Run(_ Ctx, _ Inputs) (Outputs, error)   { return nil, nil }
func (purePanic) Evaluate(_ Ctx, _ Inputs) (any, error) { panic("oops") }

func TestEvaluatePureData_PanicRecovered(t *testing.T) {
	ResetRegistryForTest()
	Register(purePanic{})
	rn, _ := Get("PurePanic")

	_, err := EvaluatePureData(context.Background(), rn, nil, nil, StubServices())
	if err == nil {
		t.Fatal("expected error from panic recovery, got nil")
	}
}
