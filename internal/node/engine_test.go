// internal/node/engine_test.go
package node

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"testing"
)

type requiresAI struct{ ran bool }

func (n *requiresAI) Spec() Spec {
	return Spec{
		Kind:                "requiresAI",
		Outputs:             []OutputSpec{{Name: "Done", Type: TypeExec}},
		RuntimeCapabilities: []RuntimeCapability{RuntimeCapabilityAI},
	}
}

func (n *requiresAI) Run(ctx Ctx, _ Inputs) (Outputs, error) {
	n.ran = true
	return ctx.Out("Done").Fire(), nil
}

func TestRunNode_MissingRuntimeCapabilityReturnsAssemblyError(t *testing.T) {
	ResetRegistryForTest()
	n := &requiresAI{}
	Register(n)
	rn, _ := Get("requiresAI")
	result := RunNode(context.Background(), rn, nil, nil, nil, ServiceBundle{}, false)
	var assemblyErr *AssemblyError
	if !errors.As(result.Error, &assemblyErr) {
		t.Fatalf("error = %v, want AssemblyError", result.Error)
	}
	if assemblyErr.Capability != RuntimeCapabilityAI || n.ran {
		t.Fatalf("assembly error = %+v, ran=%v", assemblyErr, n.ran)
	}
}

type requiresAIEvaluator struct{}

func (requiresAIEvaluator) Spec() Spec {
	return Spec{
		Kind:                "requiresAIEvaluator",
		IsPureData:          true,
		Outputs:             []OutputSpec{{Name: "Value", Type: "String"}},
		RuntimeCapabilities: []RuntimeCapability{RuntimeCapabilityAI},
	}
}

func (requiresAIEvaluator) Evaluate(Ctx, Inputs) (any, error) { return "unexpected", nil }

func TestEvaluatePureData_PreservesAssemblyError(t *testing.T) {
	ResetRegistryForTest()
	Register(requiresAIEvaluator{})
	rn, _ := Get("requiresAIEvaluator")
	_, err := EvaluatePureData(context.Background(), rn, nil, nil, ServiceBundle{})
	var assemblyErr *AssemblyError
	if !errors.As(err, &assemblyErr) || assemblyErr.Capability != RuntimeCapabilityAI {
		t.Fatalf("error = %v, want AI AssemblyError", err)
	}
}

type requiresVarsEvaluator struct{}

func (requiresVarsEvaluator) Spec() Spec {
	return Spec{
		Kind:                "requiresVarsEvaluator",
		IsPureData:          true,
		Outputs:             []OutputSpec{{Name: "Value", Type: "String"}},
		RuntimeCapabilities: []RuntimeCapability{RuntimeCapabilityVars},
	}
}

func (requiresVarsEvaluator) Evaluate(ctx Ctx, _ Inputs) (any, error) {
	value, _ := ctx.Services().Vars.Get("value")
	return value, nil
}

func TestEvaluatePureData_SnapshotDoesNotMaskMissingVars(t *testing.T) {
	ResetRegistryForTest()
	Register(requiresVarsEvaluator{})
	rn, _ := Get("requiresVarsEvaluator")
	services := ServiceBundle{Snapshot: func(context.Context) Snapshot { return Snapshot{} }}
	_, err := EvaluatePureData(context.Background(), rn, nil, nil, services)
	var assemblyErr *AssemblyError
	if !errors.As(err, &assemblyErr) || assemblyErr.Capability != RuntimeCapabilityVars {
		t.Fatalf("error = %v, want Vars AssemblyError", err)
	}
}

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

	r := RunNode(context.Background(), rn, nil, nil, nil, StubServices(), false)
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

	r := RunNode(context.Background(), rn, nil, nil, nil, StubServices(), false)
	if len(r.Validation) != 1 || r.Validation[0].Code != "REQUIRED_FIELD_MISSING" {
		t.Errorf("validation = %v, want 1 REQUIRED_FIELD_MISSING", r.Validation)
	}
	if r.Panic != nil {
		t.Errorf("Required missing should NOT panic, got panic: %v", r.Panic)
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

	r := RunNode(context.Background(), rn, nil, nil, nil, StubServices(), false)
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

	r := RunNode(context.Background(), rn, nil, nil, nil, StubServices(), false)
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

	r := RunNode(context.Background(), rn, nil, nil, nil, StubServices(), false)
	if r.Panic == nil {
		t.Error("double Fire should panic")
	}
}

// ============================================================================
// EvaluatePureData
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
func (pureAdd) Evaluate(_ Ctx, in Inputs) (any, error) { return in.Float64("a") + in.Float64("b"), nil }

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

// (TestEvaluatePureData_NoEvaluator_Errors 已删 — Register strict invariant 让
// IsPureData=true + missing Evaluator 在 Register 时 panic, EvaluatePureData 入口
// 永远拿不到这种节点. Register strict 守护见 registry_test.go.)

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

// ============================================================================
// engine dispatch + panic hygiene 守护
// ============================================================================
// engine dispatch invariant 验证. 跟 Register strict 解耦 —
// 这组只验 RunNode/RunNodeAsRegion/EvaluatePureData 对错 capability 节点返 error.

type runnableOnlyNode struct{}

func (runnableOnlyNode) Spec() Spec                           { return Spec{Kind: "RunnableOnly"} }
func (runnableOnlyNode) Run(_ Ctx, _ Inputs) (Outputs, error) { return nil, nil }

type evaluatorOnlyNode struct{}

func (evaluatorOnlyNode) Spec() Spec {
	return Spec{Kind: "EvaluatorOnly", IsPureData: true}
}
func (evaluatorOnlyNode) Evaluate(_ Ctx, _ Inputs) (any, error) { return 42, nil }

func TestRunNode_NonRunnable_Errors(t *testing.T) {
	ResetRegistryForTest()
	Register(evaluatorOnlyNode{})
	rn, _ := Get("EvaluatorOnly")
	r := RunNode(context.Background(), rn, nil, nil, nil, StubServices(), false)
	if r.Error == nil || !strings.Contains(r.Error.Error(), "not Runnable") {
		t.Errorf("expected 'not Runnable' error, got %v", r.Error)
	}
}

func TestRunNodeAsRegion_NonRegionRunner_Errors(t *testing.T) {
	ResetRegistryForTest()
	Register(runnableOnlyNode{})
	rn, _ := Get("RunnableOnly")
	r := RunNodeAsRegion(context.Background(), rn, nil, nil, nil, StubServices(), false, func(Ctx) (string, error) { return "", nil })
	if r.Error == nil || !strings.Contains(r.Error.Error(), "not a RegionRunner") {
		t.Errorf("expected 'not a RegionRunner' error, got %v", r.Error)
	}
}

func TestEvaluatePureData_NonEvaluator_Errors(t *testing.T) {
	ResetRegistryForTest()
	Register(runnableOnlyNode{})
	rn, _ := Get("RunnableOnly")
	_, err := EvaluatePureData(context.Background(), rn, nil, nil, StubServices())
	if err == nil || !strings.Contains(err.Error(), "is not IsPureData") {
		t.Errorf("expected 'is not IsPureData' error, got %v", err)
	}
}

// panic hygiene 验证: Run 内 panic — Set 之后, Fire 之前 panic. fn() 没返回 →
// result.ExitName/OutputData 根本没被赋值 → recover 时已是 zero, clear 块不会改变啥.
// 这是 "panic 路径整体被 recover 干净" 的 smoke.

// runPanicNode — Run 内在 Fire 之前 panic. fn() never returns, result 字段全是 zero.
type runPanicNode struct{}

func (runPanicNode) Spec() Spec {
	return Spec{
		Kind:    "RunPanic",
		Outputs: []OutputSpec{{Name: "Out", Type: "Exec", Data: []DataField{{Name: "X", Type: "Number"}}}},
	}
}
func (runPanicNode) Run(ctx Ctx, _ Inputs) (Outputs, error) {
	o := ctx.Out("Out").Set("X", 42)
	_ = o
	panic("deliberate test panic in Run before Fire")
}

func TestRunWithRecover_RunPanicYieldsRecoveredResult(t *testing.T) {
	ResetRegistryForTest()
	Register(runPanicNode{})
	rn, _ := Get("RunPanic")
	r := RunNode(context.Background(), rn, nil, nil, nil, StubServices(), false)
	if r.Panic == nil {
		t.Fatal("expected Panic to be set, got nil")
	}
	// Run 内 panic 在 Fire 之前 → fn() 没返回 → result 字段从来没被 assign, 本来就是 zero.
	// 这里只是确认 recover 流程整体没把脏数据漏出来.
	if r.ExitName != "" {
		t.Errorf("ExitName should be empty, got %q", r.ExitName)
	}
	if r.OutputData != nil {
		t.Errorf("OutputData should be nil, got %v", r.OutputData)
	}
}

// ============================================================================
// ResolvedInputs — opt-in input snapshot (Task 3)
// ============================================================================

func TestRunNode_ResolvedInputs_OnlyWhenEnabled(t *testing.T) {
	rn := &RegisteredNode{
		Spec: Spec{Kind: "Echo", Inputs: []InputSpec{{Name: "msg", Type: "string"}}},
		Run: func(c Ctx, in Inputs) (Outputs, error) {
			return c.Out("Done").Set("echo", in.String("msg")).Fire(), nil
		},
	}
	cfg := map[string]any{"msg": "hi"}
	r1 := RunNode(context.Background(), rn, nil, cfg, nil, StubServices(), false)
	if r1.ResolvedInputs != nil {
		t.Fatalf("logEnabled=false should not populate ResolvedInputs, got %#v", r1.ResolvedInputs)
	}
	r2 := RunNode(context.Background(), rn, nil, cfg, nil, StubServices(), true)
	if r2.ResolvedInputs["msg"] != "hi" {
		t.Fatalf("ResolvedInputs[msg] = %v, want hi", r2.ResolvedInputs["msg"])
	}
}

func TestRunNode_ResolvedInputs_SurvivesError(t *testing.T) {
	rn := &RegisteredNode{
		Spec: Spec{Kind: "Boom", Inputs: []InputSpec{{Name: "msg", Type: "string"}}},
		Run:  func(c Ctx, in Inputs) (Outputs, error) { return nil, fmt.Errorf("boom") },
	}
	cfg := map[string]any{"msg": "hi"}
	r := RunNode(context.Background(), rn, nil, cfg, nil, StubServices(), true)
	if r.Error == nil {
		t.Fatal("expected error")
	}
	if r.ResolvedInputs["msg"] != "hi" {
		t.Fatalf("ResolvedInputs must survive error, got %#v", r.ResolvedInputs)
	}
}
