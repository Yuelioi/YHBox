package script

import (
	"context"
	"errors"
	"testing"
	"time"

	"yotta/internal/node"
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

// $hp live getter: Run 起跑时按已知变量注入, 访问时实时读 (脚本中途 set 后 $hp 是新值)。
func TestScript_DollarVarGetter(t *testing.T) {
	rn := setup(t)
	svcs := node.StubServices()
	svcs.Vars.Set("hp", 37.0)
	cfg := map[string]any{"Code": `vars.set("hp", 40); return $hp + 1`}
	res := node.RunNode(context.Background(), rn, nil, cfg, nil, svcs, false)
	if res.Error != nil || res.OutputData["Result"] != float64(41) {
		t.Fatalf("res=%+v", res)
	}
}

func TestScript_VarsRoundtrip(t *testing.T) {
	rn := setup(t)
	svcs := node.StubServices()
	cfg := map[string]any{"Code": `vars.set("k", 5); return vars.get("k") + 1`}
	res := node.RunNode(context.Background(), rn, nil, cfg, nil, svcs, false)
	if res.Error != nil || res.OutputData["Result"] != float64(6) {
		t.Fatalf("res=%+v", res)
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
