package runtime

import (
	"context"
	"testing"

	"github.com/yottaapp/yotta/internal/services/container"
)

// Switch dispatch 集成 smoke: 验 named-by-value 模型经真实 execNode 路径
// (buildConfigFor → in.Raw("cases") + dynamic output ctx.Out(caseValue) 不 panic) 跑通无 error。
// 命中哪个 pin 的精确路由由 internal/nodes/control/switch_test.go (RunNode 直查 ExitName) 覆盖。

func TestExecSwitch_MatchedCase(t *testing.T) {
	_, r := newTestRunner(t)
	node := &container.GraphNode{
		ID:   "sw1",
		Kind: "Switch",
		Config: map[string]any{
			"Value": "B",
			"cases": []any{"A", "B", "C"},
		},
	}
	if _, err := r.execNode(context.Background(), node, ExecToken{InPin: "in"}); err != nil {
		t.Fatalf("err: %v", err)
	}
}

func TestExecSwitch_DefaultOnNoMatch(t *testing.T) {
	_, r := newTestRunner(t)
	node := &container.GraphNode{
		ID:   "sw2",
		Kind: "Switch",
		Config: map[string]any{
			"Value": "X",
			"cases": []any{"A", "B"},
		},
	}
	if _, err := r.execNode(context.Background(), node, ExecToken{InPin: "in"}); err != nil {
		t.Fatalf("无匹配应走 default 不报错, got: %v", err)
	}
}

func TestExecSwitch_DefaultOnEmptyValue(t *testing.T) {
	_, r := newTestRunner(t)
	node := &container.GraphNode{
		ID:   "sw3",
		Kind: "Switch",
		Config: map[string]any{
			"Value": "",
			"cases": []any{"A"},
		},
	}
	if _, err := r.execNode(context.Background(), node, ExecToken{InPin: "in"}); err != nil {
		t.Fatalf("空 value 应走 default 不报错, got: %v", err)
	}
}

func TestExecSwitch_CJKCaseDispatch(t *testing.T) {
	_, r := newTestRunner(t)
	node := &container.GraphNode{
		ID:   "sw5",
		Kind: "Switch",
		Config: map[string]any{
			"Value": "钓鱼",
			"cases": []any{"待机", "钓鱼", "恢复"},
		},
	}
	if _, err := r.execNode(context.Background(), node, ExecToken{InPin: "in"}); err != nil {
		t.Fatalf("err: %v", err)
	}
}

func TestExecSwitch_EmojiCaseDispatch(t *testing.T) {
	_, r := newTestRunner(t)
	node := &container.GraphNode{
		ID:   "sw6",
		Kind: "Switch",
		Config: map[string]any{
			"Value": "🎣",
			"cases": []any{"🎣", "⚔️"},
		},
	}
	if _, err := r.execNode(context.Background(), node, ExecToken{InPin: "in"}); err != nil {
		t.Fatalf("err: %v", err)
	}
}
