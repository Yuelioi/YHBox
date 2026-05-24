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
