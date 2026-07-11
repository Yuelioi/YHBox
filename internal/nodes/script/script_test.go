package script

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/yottaapp/yotta/internal/node"
)

func setup(t *testing.T, extra ...node.Node) *node.RegisteredNode {
	t.Helper()
	node.ResetRegistryForTest()
	node.Register(&Script{})
	for _, n := range extra {
		node.Register(n)
	}
	rn, _ := node.Get("Script")
	return rn
}

// fakeAct — 测试用 Runnable: 回显输入 A 到 Done.Echo; A<0 时返 timeout 错.
type fakeAct struct{}

func (fakeAct) Spec() node.Spec {
	return node.Spec{Kind: "FakeAct", Category: "Control",
		Inputs: []node.InputSpec{
			{Name: "In", Type: node.TypeExec},
			{Name: "A", Type: "Number", Required: true},
		},
		Outputs: []node.OutputSpec{
			{Name: "Done", Type: node.TypeExec, Data: []node.DataField{{Name: "Echo", Type: "Number"}}},
			{Name: "Fail", Type: node.TypeExec, Semantic: "error"},
		}}
}
func (fakeAct) Run(ctx node.Ctx, in node.Inputs) (node.Outputs, error) {
	a := in.Float64("A")
	if a < 0 {
		return nil, node.Failf(node.CodeTimeout, nil, "fake timeout")
	}
	return ctx.Out("Done").Set("Echo", a).Fire(), nil
}

func runScript(t *testing.T, rn *node.RegisteredNode, code string, dataWire map[string]any) node.RunResult {
	t.Helper()
	cfg := map[string]any{"Code": code}
	return node.RunNode(context.Background(), rn, dataWire, cfg, nil, node.StubServices(), false)
}

func TestScript_ReturnValue(t *testing.T) {
	rn := setup(t)
	res := runScript(t, rn, "return 1 + 2", nil)
	if res.Error != nil || res.ExitName != "Done" || res.OutputData["Result"] != float64(3) {
		t.Fatalf("res=%+v", res)
	}
}

func TestScript_CallNode_ExitAndData(t *testing.T) {
	rn := setup(t, fakeAct{})
	res := runScript(t, rn, `let r = FakeAct({A: 7}); return r.exit + ":" + r.Echo`, nil)
	if res.Error != nil || res.OutputData["Result"] != "Done:7" {
		t.Fatalf("res=%+v", res)
	}
}

func TestScript_NodeFail_Catchable(t *testing.T) {
	rn := setup(t, fakeAct{})
	res := runScript(t, rn, `try { FakeAct({A: -1}) } catch (e) { return e.code }`, nil)
	if res.Error != nil || res.OutputData["Result"] != "timeout" {
		t.Fatalf("res=%+v", res)
	}
}

func TestScript_NodeFail_Uncaught_CodePassthrough(t *testing.T) {
	rn := setup(t, fakeAct{})
	res := runScript(t, rn, `FakeAct({A: -1})`, nil)
	var coded node.Coded
	if res.Error == nil || !errors.As(res.Error, &coded) || coded.ErrCode() != node.CodeTimeout {
		t.Fatalf("want coded timeout, res=%+v", res)
	}
}

func TestScript_Throw_CodeThrown(t *testing.T) {
	rn := setup(t)
	res := runScript(t, rn, `throw new Error("boom")`, nil)
	var coded node.Coded
	if res.Error == nil || !errors.As(res.Error, &coded) || coded.ErrCode() != node.CodeThrown {
		t.Fatalf("res=%+v", res)
	}
}

func TestScript_SyntaxError_PlainError(t *testing.T) {
	rn := setup(t)
	res := runScript(t, rn, `let = ;`, nil)
	var coded node.Coded
	if res.Error == nil || errors.As(res.Error, &coded) {
		t.Fatalf("config 错应是裸 error (冒泡, 不走 Fail), res=%+v", res)
	}
}

func TestScript_DynamicInput(t *testing.T) {
	rn := setup(t)
	cfg := map[string]any{"Code": "return hp * 2",
		"Inputs": []any{map[string]any{"Name": "hp", "Type": "number"}}}
	res := node.RunNode(context.Background(), rn, map[string]any{"hp": 21.0}, cfg, nil, node.StubServices(), false)
	if res.Error != nil || res.OutputData["Result"] != float64(42) {
		t.Fatalf("res=%+v", res)
	}
}

// fakeSetVar — 最小变量写替身 (真实 SetVar 在 nodes/variable, 不跨包陪绑)。
type fakeSetVar struct{}

func (fakeSetVar) Spec() node.Spec {
	return node.Spec{Kind: "FakeSetVar", Category: "Test",
		Inputs: []node.InputSpec{
			{Name: "In", Type: node.TypeExec},
			{Name: "Name", Type: "String", Required: true},
			{Name: "Value", Type: "Number", Required: true},
		},
		Outputs: []node.OutputSpec{{Name: "Done", Type: node.TypeExec}}}
}

func (fakeSetVar) Run(ctx node.Ctx, in node.Inputs) (node.Outputs, error) {
	ctx.Services().Vars.SetScoped(in.String("Name"), "auto", in.Float64("Value"))
	return ctx.Out("Done").Fire(), nil
}

// $hp live getter: Run 起跑时按已知变量注入, 访问时实时读 — 脚本中途经节点函数
// 写入后 $hp 读到新值 (写路径只剩节点函数, vars.* 糖已删)。
func TestScript_DollarVarGetter(t *testing.T) {
	rn := setup(t, fakeSetVar{})
	svcs := node.StubServices()
	svcs.Vars.Set("hp", 37.0)
	cfg := map[string]any{"Code": `FakeSetVar({Name: "hp", Value: 40}); return $hp + 1`}
	res := node.RunNode(context.Background(), rn, nil, cfg, nil, svcs, false)
	if res.Error != nil || res.OutputData["Result"] != float64(41) {
		t.Fatalf("res=%+v", res)
	}
}

// vars.* 糖已删干净: 引用 vars 必须是 ReferenceError → thrown。
func TestScript_VarsSugarGone(t *testing.T) {
	rn := setup(t)
	res := runScript(t, rn, `vars.set("k", 5)`, nil)
	var coded node.Coded
	if res.Error == nil || !errors.As(res.Error, &coded) || coded.ErrCode() != node.CodeThrown {
		t.Fatalf("want thrown ReferenceError, res=%+v", res)
	}
}

func TestScript_CancelInterruptsLoop(t *testing.T) {
	rn := setup(t)
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan node.RunResult, 1)
	go func() {
		done <- node.RunNode(ctx, rn, nil, map[string]any{"Code": "while(true){}"}, nil, node.StubServices(), false)
	}()
	time.Sleep(50 * time.Millisecond)
	cancel()
	select {
	case res := <-done:
		if !errors.Is(res.Error, context.Canceled) {
			t.Fatalf("want context.Canceled, got %+v", res)
		}
	case <-time.After(3 * time.Second):
		t.Fatal("script not interrupted")
	}
}
