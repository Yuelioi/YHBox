package input

import (
	"context"
	"errors"
	"testing"

	"yhbox/internal/node"
)

func TestOnEvent_StubReturnsPhase5Error(t *testing.T) {
	node.ResetRegistryForTest()
	node.Register(&OnEvent{})
	rn, _ := node.Get("OnEvent")

	r := node.RunNode(context.Background(), rn, nil,
		map[string]any{
			oeInKind:     "template_appeared",
			oeInTemplate: "fishing.hook_icon",
		},
		nil, node.StubServices())

	if r.Error == nil {
		t.Fatal("expected Phase 5 stub error")
	}
	if !errors.Is(r.Error, errOnEventPhase5) {
		t.Errorf("error = %v, want errOnEventPhase5", r.Error)
	}
}

func TestOnEvent_SpecNoExecIn(t *testing.T) {
	// listener 节点没 exec-in pin (ExecIn: nil).
	spec := (OnEvent{}).Spec()
	for _, p := range spec.Inputs {
		if p.Type == "Exec" {
			t.Errorf("OnEvent should have no Exec-type input, found %q", p.Name)
		}
	}
	if len(spec.Outputs) != 1 || spec.Outputs[0].Name != oeOutOut {
		t.Errorf("Outputs = %+v, want single %q", spec.Outputs, oeOutOut)
	}
}

func TestOnEvent_Dependencies(t *testing.T) {
	node.ResetRegistryForTest()
	node.Register(&OnEvent{})
	rn, _ := node.Get("OnEvent")
	if rn.Dependencies == nil {
		t.Fatal("OnEvent should implement Dependencies")
	}
	// 间接验证 via RunNode (Dependencies 跟 Run 用同一份 Inputs 构造)
	r := node.RunNode(context.Background(), rn, nil,
		map[string]any{oeInKind: "template_appeared", oeInTemplate: "fishing.hook_icon"},
		nil, node.StubServices())
	_ = r // Validation 0, Error 是 stub
}
