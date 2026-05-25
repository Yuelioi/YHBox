package variable

import (
	"context"
	"errors"
	"testing"

	"yhbox/internal/node"
)

// 3 个 pure-data 节点 Run 都返 node.ErrPureDataMustEvaluate.
// Phase 5 加 pull-eval 后 Run 永不调; 现在 stub 行为 = sentinel.

func TestGetVar_PureDataStub(t *testing.T) {
	node.ResetRegistryForTest()
	node.Register(&GetVar{})
	rn, _ := node.Get("GetVar")
	r := node.RunNode(context.Background(), rn, nil,
		map[string]any{gvInVarName: "x"},
		nil, node.StubServices())
	if !errors.Is(r.Error, node.ErrPureDataMustEvaluate) {
		t.Errorf("GetVar error = %v, want node.ErrPureDataMustEvaluate", r.Error)
	}
}

func TestGetSys_PureDataStub(t *testing.T) {
	node.ResetRegistryForTest()
	node.Register(&GetSys{})
	rn, _ := node.Get("GetSys")
	r := node.RunNode(context.Background(), rn, nil,
		map[string]any{gsInPath: "now_ms"},
		nil, node.StubServices())
	if !errors.Is(r.Error, node.ErrPureDataMustEvaluate) {
		t.Errorf("GetSys error = %v, want node.ErrPureDataMustEvaluate", r.Error)
	}
}

func TestGetParam_PureDataStub(t *testing.T) {
	node.ResetRegistryForTest()
	node.Register(&GetParam{})
	rn, _ := node.Get("GetParam")
	r := node.RunNode(context.Background(), rn, nil,
		map[string]any{gpInParamName: "key"},
		nil, node.StubServices())
	if !errors.Is(r.Error, node.ErrPureDataMustEvaluate) {
		t.Errorf("GetParam error = %v, want node.ErrPureDataMustEvaluate", r.Error)
	}
}

// IsPureData spec flag 在
func TestPureData_Flag(t *testing.T) {
	for _, n := range []node.Node{&GetVar{}, &GetSys{}, &GetParam{}} {
		s := n.Spec()
		if !s.IsPureData {
			t.Errorf("%s.IsPureData = false, want true", s.Kind)
		}
	}
}
