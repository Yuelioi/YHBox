package control

import (
	"context"
	"testing"

	"yhbox/internal/node"
)

func TestSwitch_Case1Hit(t *testing.T) {
	node.ResetRegistryForTest()
	node.Register(&Switch{})
	rn, _ := node.Get("Switch")

	r := node.RunNode(context.Background(), rn, nil,
		map[string]any{
			swInValue: "A",
			swInCase1: "A",
			swInCase2: "B",
		},
		nil, node.StubVisionService(), node.DefaultLogService())
	if r.ExitName != swOutCase1 {
		t.Errorf("exit = %q, want Case1", r.ExitName)
	}
}

func TestSwitch_Default(t *testing.T) {
	node.ResetRegistryForTest()
	node.Register(&Switch{})
	rn, _ := node.Get("Switch")

	r := node.RunNode(context.Background(), rn, nil,
		map[string]any{
			swInValue: "X",
			swInCase1: "A",
			swInCase2: "B",
			swInCase3: "C",
		},
		nil, node.StubVisionService(), node.DefaultLogService())
	if r.ExitName != swOutDefault {
		t.Errorf("exit = %q, want Default", r.ExitName)
	}
}

func TestSwitch_FirstMatchWins(t *testing.T) {
	node.ResetRegistryForTest()
	node.Register(&Switch{})
	rn, _ := node.Get("Switch")

	// 两个 case 同 value, Case1 应该先匹配
	r := node.RunNode(context.Background(), rn, nil,
		map[string]any{
			swInValue: "A",
			swInCase1: "A",
			swInCase2: "A",
		},
		nil, node.StubVisionService(), node.DefaultLogService())
	if r.ExitName != swOutCase1 {
		t.Errorf("exit = %q, want Case1 (first match)", r.ExitName)
	}
}
