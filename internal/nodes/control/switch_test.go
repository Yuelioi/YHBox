package control

import (
	"context"
	"testing"

	"github.com/yottaapp/yotta/internal/node"
)

// runSwitch 跑一个 Switch: cases (named-by-value 列表, 传 nil 表示无 cases),
// value (待比较值), hasValue (false 模拟 Required 缺失)。
func runSwitch(t *testing.T, value string, cases []any, hasValue bool) node.RunResult {
	t.Helper()
	registry := node.NewRegistry()
	registry.Register(&Switch{})
	rn, _ := registry.Get("Switch")
	cfg := map[string]any{}
	if cases != nil {
		cfg["cases"] = cases
	}
	if hasValue {
		cfg[swInValue] = value
	}
	return node.RunNode(context.Background(), rn, nil, cfg, nil, node.StubServices(), false)
}

func TestSwitch_CaseHit(t *testing.T) {
	// 命中 → 走同名出口 (case 值即 exit 名).
	r := runSwitch(t, "B", []any{"A", "B", "C"}, true)
	if r.ExitName != "B" {
		t.Errorf("exit = %q, want B (named-by-value)", r.ExitName)
	}
}

func TestSwitch_DefaultOnMiss(t *testing.T) {
	r := runSwitch(t, "X", []any{"A", "B"}, true)
	if r.ExitName != swOutDefault {
		t.Errorf("exit = %q, want %q", r.ExitName, swOutDefault)
	}
}

func TestSwitch_DefaultWhenNoCases(t *testing.T) {
	r := runSwitch(t, "anything", nil, true)
	if r.ExitName != swOutDefault {
		t.Errorf("exit = %q, want %q (no cases)", r.ExitName, swOutDefault)
	}
}

func TestSwitch_FirstMatchWins(t *testing.T) {
	// 重复 case 被 switchCases 规整 (dedup), 仍命中.
	r := runSwitch(t, "A", []any{"A", "A"}, true)
	if r.ExitName != "A" {
		t.Errorf("exit = %q, want A", r.ExitName)
	}
}

func TestSwitch_CJKAndEmoji(t *testing.T) {
	r := runSwitch(t, "钓鱼", []any{"待机", "钓鱼", "恢复"}, true)
	if r.ExitName != "钓鱼" {
		t.Errorf("exit = %q, want 钓鱼", r.ExitName)
	}
	r2 := runSwitch(t, "🎣", []any{"🎣", "⚔️"}, true)
	if r2.ExitName != "🎣" {
		t.Errorf("exit = %q, want 🎣", r2.ExitName)
	}
}

func TestSwitch_StateMachineStyle(t *testing.T) {
	// 模拟 fishing-v2 swState 9 态 — 每个都能命中同名出口.
	states := []any{"IDLE", "SETUP", "WAITING", "FISHING", "RESULT",
		"RECOVERING", "SHOPSELL", "BUYBAIT", "CHANGEBAIT"}
	for _, s := range states {
		r := runSwitch(t, s.(string), states, true)
		if r.ExitName != s.(string) {
			t.Errorf("state %q exit %q, want %q", s, r.ExitName, s)
		}
	}
	r := runSwitch(t, "UNKNOWN", states, true)
	if r.ExitName != swOutDefault {
		t.Errorf("unknown state exit %q, want %q", r.ExitName, swOutDefault)
	}
}

func TestSwitch_EmptyAndNonStringCasesSkipped(t *testing.T) {
	// 空串 / 非字符串项被 switchCases 跳过; value="" 不命中空 case → default.
	r := runSwitch(t, "", []any{"", "A"}, true)
	if r.ExitName != swOutDefault {
		t.Errorf("exit = %q, want %q (空 case 不参与匹配)", r.ExitName, swOutDefault)
	}
}

func TestSwitch_RequiredValueMissing(t *testing.T) {
	r := runSwitch(t, "", []any{"A"}, false)
	if len(r.Validation) == 0 {
		t.Errorf("expected REQUIRED_FIELD_MISSING for Value, got %+v", r)
	}
}

func TestSwitch_Spec_DynamicOutputs(t *testing.T) {
	sp := Switch{}.Spec()
	if !sp.DynamicOutputs {
		t.Error("Switch.Spec.DynamicOutputs should be true (named-by-value 出口)")
	}
	// 唯一静态出口 = default 兜底
	if len(sp.Outputs) != 1 || sp.Outputs[0].Name != swOutDefault {
		t.Errorf("Outputs = %+v, want single %q", sp.Outputs, swOutDefault)
	}
	// 不再有 CaseNValue 输入 — 只 In + Value
	for _, in := range sp.Inputs {
		if in.Name != swInExec && in.Name != swInValue {
			t.Errorf("unexpected input %q (named-by-value 不应有 CaseNValue)", in.Name)
		}
	}
}
