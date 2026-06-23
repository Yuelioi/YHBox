package mcpserver

import (
	"testing"

	"yotta/internal/node"
)

func TestIsRunnable_GatesByNeedsWindowNotPureData(t *testing.T) {
	clickAt, ok := node.Get("ClickAt")
	if !ok {
		t.Skip("ClickAt 未注册")
	}
	if !isRunnable(clickAt.Spec) {
		t.Error("ClickAt (NeedsWindow 动作节点) 应可跑")
	}
	if execInPin(clickAt.Spec) == "" {
		t.Error("ClickAt 应有 exec 输入 pin")
	}
	loop, ok := node.Get("Loop")
	if ok && isRunnable(loop.Spec) {
		t.Error("Loop (结构节点) 不该可跑")
	}
	getVar, ok := node.Get("GetVar")
	if ok && isRunnable(getVar.Spec) {
		t.Error("GetVar (IsPureData) 不该可跑")
	}
}
