package control

import (
	"context"
	"fmt"
	"testing"

	"yhbox/internal/node"
)

func TestSwitch_Case1Hit(t *testing.T) {
	node.ResetRegistryForTest()
	node.Register(&Switch{})
	rn, _ := node.Get("Switch")

	r := node.RunNode(context.Background(), rn, nil,
		map[string]any{
			swInValue:    "A",
			"Case1Value": "A",
			"Case2Value": "B",
		},
		nil, node.StubServices())
	if r.ExitName != "Case1" {
		t.Errorf("exit = %q, want Case1", r.ExitName)
	}
}

func TestSwitch_Case16Hit(t *testing.T) {
	// 最后一个 case 也能命中, 验 16 上限循环覆盖.
	node.ResetRegistryForTest()
	node.Register(&Switch{})
	rn, _ := node.Get("Switch")

	r := node.RunNode(context.Background(), rn, nil,
		map[string]any{
			swInValue:     "Z",
			"Case16Value": "Z",
		},
		nil, node.StubServices())
	if r.ExitName != "Case16" {
		t.Errorf("exit = %q, want Case16", r.ExitName)
	}
}

func TestSwitch_Default(t *testing.T) {
	node.ResetRegistryForTest()
	node.Register(&Switch{})
	rn, _ := node.Get("Switch")

	r := node.RunNode(context.Background(), rn, nil,
		map[string]any{
			swInValue:    "X",
			"Case1Value": "A",
			"Case2Value": "B",
			"Case3Value": "C",
		},
		nil, node.StubServices())
	if r.ExitName != swOutDefault {
		t.Errorf("exit = %q, want Default", r.ExitName)
	}
}

func TestSwitch_DefaultWhenNoCasesSet(t *testing.T) {
	// 仅 Value 没任何 CaseN — 直接走 Default.
	node.ResetRegistryForTest()
	node.Register(&Switch{})
	rn, _ := node.Get("Switch")

	r := node.RunNode(context.Background(), rn, nil,
		map[string]any{swInValue: "anything"},
		nil, node.StubServices())
	if r.ExitName != swOutDefault {
		t.Errorf("exit = %q, want Default (no cases set)", r.ExitName)
	}
}

func TestSwitch_FirstMatchWins(t *testing.T) {
	// 两个 case 同 value, Case1 先匹配.
	node.ResetRegistryForTest()
	node.Register(&Switch{})
	rn, _ := node.Get("Switch")

	r := node.RunNode(context.Background(), rn, nil,
		map[string]any{
			swInValue:    "A",
			"Case1Value": "A",
			"Case2Value": "A",
		},
		nil, node.StubServices())
	if r.ExitName != "Case1" {
		t.Errorf("exit = %q, want Case1 (first match)", r.ExitName)
	}
}

func TestSwitch_EmptyCaseFieldDoesNotMatchEmptyValue(t *testing.T) {
	// 未设的 CaseN 字段 Has=false, 不参与匹配; Value 空跟未设 Case 都不算命中 → Default.
	node.ResetRegistryForTest()
	node.Register(&Switch{})
	rn, _ := node.Get("Switch")

	r := node.RunNode(context.Background(), rn, nil,
		map[string]any{swInValue: ""},
		nil, node.StubServices())
	if r.ExitName != swOutDefault {
		t.Errorf("exit = %q, want Default (empty value + no cases → default)", r.ExitName)
	}
}

func TestSwitch_AllCases_StateMachineStyle(t *testing.T) {
	// 模拟 fishing-v2 main swState — 9 case 字符串 + default. 验中间 case 命中.
	node.ResetRegistryForTest()
	node.Register(&Switch{})
	rn, _ := node.Get("Switch")

	states := []string{
		"IDLE", "SETUP", "WAITING", "FISHING", "RESULT",
		"RECOVERING", "SHOPSELL", "BUYBAIT", "CHANGEBAIT",
	}

	config := map[string]any{}
	for i, state := range states {
		config[fmt.Sprintf("Case%dValue", i+1)] = state
	}

	for i, state := range states {
		config[swInValue] = state
		r := node.RunNode(context.Background(), rn, nil, config, nil, node.StubServices())
		want := fmt.Sprintf("Case%d", i+1)
		if r.ExitName != want {
			t.Errorf("state=%q exit=%q, want %q", state, r.ExitName, want)
		}
	}

	config[swInValue] = "UNKNOWN_STATE"
	r := node.RunNode(context.Background(), rn, nil, config, nil, node.StubServices())
	if r.ExitName != swOutDefault {
		t.Errorf("unknown state exit=%q, want Default", r.ExitName)
	}
}

func TestSwitch_RequiredValueMissing(t *testing.T) {
	node.ResetRegistryForTest()
	node.Register(&Switch{})
	rn, _ := node.Get("Switch")

	r := node.RunNode(context.Background(), rn, nil, nil, nil, node.StubServices())
	if len(r.Validation) == 0 {
		t.Errorf("expected REQUIRED_FIELD_MISSING for Value, got %+v", r)
	}
}

func TestSwitch_Spec_16CasesPlusDefault(t *testing.T) {
	sp := Switch{}.Spec()
	if len(sp.Outputs) != switchMaxCases+1 {
		t.Errorf("Outputs = %d, want %d (16 cases + Default)", len(sp.Outputs), switchMaxCases+1)
	}
	// 最后一个必须是 Default
	if sp.Outputs[len(sp.Outputs)-1].Name != swOutDefault {
		t.Errorf("last output = %q, want %q", sp.Outputs[len(sp.Outputs)-1].Name, swOutDefault)
	}
	// 前 4 case 输入 Advanced=false, 5-16 Advanced=true
	caseInputs := map[string]node.InputSpec{}
	for _, in := range sp.Inputs {
		caseInputs[in.Name] = in
	}
	for i := 1; i <= switchMaxCases; i++ {
		key := fmt.Sprintf("Case%dValue", i)
		in, ok := caseInputs[key]
		if !ok {
			t.Errorf("missing input %q", key)
			continue
		}
		wantAdvanced := i > 4
		if in.Advanced != wantAdvanced {
			t.Errorf("%s.Advanced = %v, want %v", key, in.Advanced, wantAdvanced)
		}
	}
}
