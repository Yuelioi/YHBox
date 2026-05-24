package system

import (
	"context"
	"testing"

	"yhbox/internal/node"
)

func TestWindowTarget_Passthrough(t *testing.T) {
	node.ResetRegistryForTest()
	node.Register(&WindowTarget{})
	rn, _ := node.Get("WindowTarget")

	r := node.RunNode(context.Background(), rn, nil,
		map[string]any{
			wtInTitle:          "异环",
			wtInTitleMatch:     "exact",
			wtInInputBackend:   "postmessage",
			wtInCaptureBackend: "auto",
		},
		nil, node.StubServices())
	if r.Error != nil {
		t.Fatal(r.Error)
	}
	if r.ExitName != wtOutFire {
		t.Errorf("exit = %q, want %q", r.ExitName, wtOutFire)
	}
}

func TestWindowTarget_DefaultsAllowEmptyConfig(t *testing.T) {
	// 全部字段都有 Default → 无 config 也应走 Fire (declarative no-op).
	node.ResetRegistryForTest()
	node.Register(&WindowTarget{})
	rn, _ := node.Get("WindowTarget")

	r := node.RunNode(context.Background(), rn, nil, nil, nil, node.StubServices())
	if r.Error != nil {
		t.Fatal(r.Error)
	}
	if r.ExitName != wtOutFire {
		t.Errorf("exit = %q, want %q", r.ExitName, wtOutFire)
	}
}
