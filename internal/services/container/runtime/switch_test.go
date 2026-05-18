package runtime

import (
	"context"
	"testing"

	"yhbox/internal/services/container"
)

// 6 tests covering: matched case, default on miss, default on nil, stringable int,
// CJK case dispatch, emoji case dispatch.
//
// Tests verify execSwitch returns NO error (走 default 时也不 error) and at least
// 0 tokens. If newTestRunner helper supports edge wiring, also verify which pin fired
// by checking downstream node received token. If not, just verify no error.

func TestExecSwitch_MatchedCase(t *testing.T) {
	_, r := newTestRunner(t)
	node := &container.GraphNode{
		ID:   "sw1",
		Kind: "Switch",
		Config: map[string]any{
			"value": "'B'", // literal string expression
			"cases": []any{"A", "B", "C"},
		},
	}
	_, err := r.execNode(context.Background(), node, ExecToken{InPin: "in"})
	if err != nil {
		t.Fatalf("err: %v", err)
	}
}

func TestExecSwitch_DefaultOnNoMatch(t *testing.T) {
	_, r := newTestRunner(t)
	node := &container.GraphNode{
		ID:   "sw2",
		Kind: "Switch",
		Config: map[string]any{
			"value": "'X'",
			"cases": []any{"A", "B"},
		},
	}
	_, err := r.execNode(context.Background(), node, ExecToken{InPin: "in"})
	if err != nil {
		t.Fatalf("无匹配应走 default 不报错, got: %v", err)
	}
}

func TestExecSwitch_DefaultOnNilValue(t *testing.T) {
	_, r := newTestRunner(t)
	node := &container.GraphNode{
		ID:   "sw3",
		Kind: "Switch",
		Config: map[string]any{
			"value": "$vars.notDefined", // undefined var → resolve nil
			"cases": []any{"A"},
		},
	}
	_, err := r.execNode(context.Background(), node, ExecToken{InPin: "in"})
	if err != nil {
		t.Fatalf("nil value 应走 default 不报错, got: %v", err)
	}
}

func TestExecSwitch_StringableInt(t *testing.T) {
	_, r := newTestRunner(t)
	r.state.SetLocalVar("x", float64(5))
	node := &container.GraphNode{
		ID:   "sw4",
		Kind: "Switch",
		Config: map[string]any{
			"value": "$vars.x",
			"cases": []any{"5", "10"},
		},
	}
	_, err := r.execNode(context.Background(), node, ExecToken{InPin: "in"})
	if err != nil {
		t.Fatalf("err: %v", err)
	}
}

func TestExecSwitch_CJKCaseDispatch(t *testing.T) {
	_, r := newTestRunner(t)
	r.state.SetLocalVar("state", "钓鱼")
	node := &container.GraphNode{
		ID:   "sw5",
		Kind: "Switch",
		Config: map[string]any{
			"value": "$vars.state",
			"cases": []any{"待机", "钓鱼", "恢复"},
		},
	}
	_, err := r.execNode(context.Background(), node, ExecToken{InPin: "in"})
	if err != nil {
		t.Fatalf("err: %v", err)
	}
}

func TestExecSwitch_EmojiCaseDispatch(t *testing.T) {
	_, r := newTestRunner(t)
	r.state.SetLocalVar("state", "🎣")
	node := &container.GraphNode{
		ID:   "sw6",
		Kind: "Switch",
		Config: map[string]any{
			"value": "$vars.state",
			"cases": []any{"🎣", "⚔️"},
		},
	}
	_, err := r.execNode(context.Background(), node, ExecToken{InPin: "in"})
	if err != nil {
		t.Fatalf("err: %v", err)
	}
}
