package purefunc

import (
	"context"
	"errors"
	"testing"

	"yhbox/internal/node"
)

// 所有 22 个 purefunc + Expr 都是 pure-data stub. Run 返 errPureDataNotEvaluatable.
// Phase 5 加 pull-eval framework 后, Run 永不调; FE inspector 已能渲染 Spec.

func TestAll_RegisterAndPureDataStub(t *testing.T) {
	node.ResetRegistryForTest()

	// 全部 23 节点显式注册一遍 (init 注册的是 ResetRegistryForTest 之前的, 这里重新调)
	all := []node.Node{
		&Add{}, &Sub{}, &Mul{}, &Div{}, &Mod{}, &Neg{},
		&Lt{}, &LtEq{}, &Gt{}, &GtEq{}, &Eq{}, &NotEq{},
		&And{}, &Or{}, &Not{},
		&Concat{}, &Contains{}, &Length{},
		&ToString{}, &ToNumber{}, &ToBool{},
		&Select{},
		&Expr{},
	}
	for _, n := range all {
		node.Register(n)
	}

	if len(all) != 23 {
		t.Fatalf("expected 23 purefunc nodes, got %d", len(all))
	}

	for _, n := range all {
		spec := n.Spec()
		if !spec.IsPureData {
			t.Errorf("%s.IsPureData = false, want true", spec.Kind)
		}
		if len(spec.Outputs) != 1 {
			t.Errorf("%s outputs len = %d, want 1", spec.Kind, len(spec.Outputs))
		}

		rn, _ := node.Get(spec.Kind)
		r := node.RunNode(context.Background(), rn, nil, nil, nil, node.StubServices())
		if !errors.Is(r.Error, errPureDataNotEvaluatable) {
			t.Errorf("%s Run error = %v, want errPureDataNotEvaluatable", spec.Kind, r.Error)
		}
	}
}

func TestSpecBuilder_HasResultOutput(t *testing.T) {
	s := specBuilder("Test", "测试", "", numIn(), "Number")
	if s.Outputs[0].Name != "result" {
		t.Errorf("output name = %q, want result", s.Outputs[0].Name)
	}
	if s.Outputs[0].Type != "Number" {
		t.Errorf("output type = %q, want Number", s.Outputs[0].Type)
	}
}

func TestExpr_RequiredExprField(t *testing.T) {
	node.ResetRegistryForTest()
	node.Register(&Expr{})
	rn, _ := node.Get("Expr")
	// 不传 expr → Required 应触发 ValidationError (在 Run 入口前)
	r := node.RunNode(context.Background(), rn, nil, nil, nil, node.StubServices())
	if len(r.Validation) == 0 && r.Error != errPureDataNotEvaluatable {
		t.Errorf("Expr missing expr should ValidationError or sentinel, got error=%v validation=%v", r.Error, r.Validation)
	}
}
