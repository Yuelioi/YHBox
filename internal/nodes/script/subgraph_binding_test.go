package script

import (
	"context"
	"errors"
	"testing"

	"github.com/yottaapp/yotta/internal/node"
)

// fakeSubgraphs — node.SubgraphCaller 测试替身.
type fakeSubgraphs struct {
	gotID     string
	gotParams map[string]any
	exit      string
	err       error
}

func (f *fakeSubgraphs) CallSubgraph(_ context.Context, sgID string, params map[string]any) (string, error) {
	f.gotID = sgID
	f.gotParams = params
	return f.exit, f.err
}

func runScriptWithSubgraphs(t *testing.T, rn *node.RegisteredNode, code string, fake *fakeSubgraphs) node.RunResult {
	t.Helper()
	svcs := node.StubServices()
	if fake != nil {
		svcs.Subgraphs = fake
	}
	return node.RunNode(context.Background(), rn, nil, map[string]any{"Code": code}, nil, svcs, false)
}

func TestScript_Subgraph_CallAndExit(t *testing.T) {
	rn := setup(t)
	fake := &fakeSubgraphs{exit: "failed"}
	res := runScriptWithSubgraphs(t, rn,
		`let r = Subgraph({SubgraphID: "sg-1", msg: "hi", n: 2}); return r.exit`, fake)
	if res.Error != nil || res.OutputData["Result"] != "failed" {
		t.Fatalf("res=%+v", res)
	}
	if fake.gotID != "sg-1" {
		t.Errorf("sgID = %q, want sg-1", fake.gotID)
	}
	if fake.gotParams["msg"] != "hi" || fake.gotParams["n"] != float64(2) {
		t.Errorf("params = %#v, want msg=hi n=2 (NormalizeJS float64)", fake.gotParams)
	}
	if _, has := fake.gotParams["SubgraphID"]; has {
		t.Error("SubgraphID 不该混进 params")
	}
}

func TestScript_Subgraph_MissingID_Throws(t *testing.T) {
	rn := setup(t)
	res := runScriptWithSubgraphs(t, rn,
		`try { Subgraph({}) } catch (e) { return e.code }`, &fakeSubgraphs{exit: "ok"})
	if res.Error != nil || res.OutputData["Result"] != string(node.CodeError) {
		t.Fatalf("res=%+v", res)
	}
}

func TestScript_Subgraph_ErrorCodePassthrough(t *testing.T) {
	rn := setup(t)
	fake := &fakeSubgraphs{err: node.Failf(node.CodeSubgraphNoExit, nil, "no exit")}
	res := runScriptWithSubgraphs(t, rn,
		`try { Subgraph({SubgraphID: "sg-1"}) } catch (e) { return e.code }`, fake)
	if res.Error != nil || res.OutputData["Result"] != string(node.CodeSubgraphNoExit) {
		t.Fatalf("res=%+v", res)
	}
}

func TestScript_Subgraph_NoService_NotDefined(t *testing.T) {
	// 非 runner 语境 (Subgraphs nil) 不装 Subgraph 函数 → ReferenceError → thrown.
	rn := setup(t)
	res := runScriptWithSubgraphs(t, rn, `Subgraph({SubgraphID: "sg-1"})`, nil)
	var coded node.Coded
	if res.Error == nil || !errors.As(res.Error, &coded) || coded.ErrCode() != node.CodeThrown {
		t.Fatalf("want thrown ReferenceError, res=%+v", res)
	}
}
