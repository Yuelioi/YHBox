package runtime

import (
	"testing"

	nodepkg "yotta/internal/node"
)

// TestAllNeedsWindowNodesHaveWindowInput — 全部注册节点中, NeedsWindow 的都必须有 Window 输入。
// 本包 dispatch_v5_test.go 已 blank-import 全部 node 包, 故 All() 是全集。
func TestAllNeedsWindowNodesHaveWindowInput(t *testing.T) {
	for _, rn := range nodepkg.All() {
		if !rn.Spec.NeedsWindow {
			continue
		}
		has := false
		for _, ip := range rn.Spec.Inputs {
			if ip.Name == "Window" && ip.Type == "Window" {
				has = true
				break
			}
		}
		if !has {
			t.Errorf("%s NeedsWindow 但缺 Window 输入", rn.Spec.Kind)
		}
	}
}
