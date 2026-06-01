package runtime

import (
	"testing"

	"yhbox/internal/services/container"
	"yhbox/internal/services/execution"
)

// TestListener_SubRunnerStateIsolated 验证 B10 修复: listener subRunner 持独立 bundle,
// Vars(local) / Params 落到 sub.state 自己的 frame 栈, 不 stomp 主 runner.state;
// global var 仍走 rt 跨 flow 共享.
func TestListener_SubRunnerStateIsolated(t *testing.T) {
	c := &container.Container{
		SchemaVersion: 1,
		ID:            "test-listener-iso",
		Name:          "test-listener-iso",
		Graph: container.Graph{
			Nodes: []container.GraphNode{{ID: "start", Kind: "Start"}},
		},
	}
	rt := NewRuntimeContext(c, execution.NewInputBus(), NoopMatcher{}, nil, nil, nil, 0)
	r := NewContainerRunner(rt)
	l := &EventListener{
		runner:        r,
		homeEdges:     r.compiled.Main.Edges,
		homeNodesByID: r.compiled.Main.NodesByID,
	}
	sub := l.makeSubRunner()

	if sub.state == r.state {
		t.Fatal("sub.state 跟主 r.state 是同一实例 — 没隔离")
	}

	// local var: 写 sub → 落 sub.state, 主 r.state 看不到
	sub.bundle.Vars.SetScoped("x", "local", "from-sub")
	if v, ok := sub.bundle.Vars.GetScoped("x", "local"); !ok || v != "from-sub" {
		t.Errorf("sub local x = %v,%v, want from-sub,true", v, ok)
	}
	if v, ok := r.bundle.Vars.GetScoped("x", "local"); ok {
		t.Errorf("主 runner local x 泄漏 = %v (应隔离, want 查无)", v)
	}

	// global var: 写 sub → 走 rt.Vars, 主 runner 也看得到 (容器级共享)
	sub.bundle.Vars.SetScoped("g", "global", "shared")
	if v, ok := r.bundle.Vars.GetScoped("g", "global"); !ok || v != "shared" {
		t.Errorf("主 runner global g = %v,%v, want shared,true (global 应跨 flow 共享)", v, ok)
	}

	// param: 各自 frame 的 LocalParams 互不串
	sub.state.CurrentFrame.LocalParams["p"] = "sub-param"
	r.state.CurrentFrame.LocalParams["p"] = "main-param"
	if v, ok := sub.bundle.Params.Get("p"); !ok || v != "sub-param" {
		t.Errorf("sub param p = %v,%v, want sub-param,true", v, ok)
	}
	if v, ok := r.bundle.Params.Get("p"); !ok || v != "main-param" {
		t.Errorf("主 runner param p = %v,%v, want main-param,true (应隔离)", v, ok)
	}
}
