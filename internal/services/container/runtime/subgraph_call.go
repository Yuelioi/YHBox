// subgraph_call.go — 子图调用共享核心.
// graph 层 (Subgraph / CollapsedNode 的 region body) 与脚本绑定 (Subgraph() 函数,
// 经 node.SubgraphCaller) 都走 runSubgraphCall: push frame + seed params + 切 dispatch
// 表到 callee + 从 entry 播种跑 body + 恢复, 返回到达的出口 decl.
package runtime

import (
	"context"
	"fmt"
	"maps"

	nodepkg "yotta/internal/node"
	"yotta/internal/services/container"
)

// maxSubgraphCallDepth 子图调用嵌套上限. 图层有静态防环 (validateCyclicSubgraphs),
// 但脚本动态调用 (子图里 Script 再调回自己) 拦不住, 运行时兜底.
const maxSubgraphCallDepth = 32

func frameDepth(s *ExecState) int {
	d := 0
	for f := s.CurrentFrame; f != nil; f = f.Parent {
		d++
	}
	return d
}

// runSubgraphCall 同步跑一次 sg 子图. params 直接 seed 进 callee frame 的 LocalParams
// (caller 负责 default 补全). 出口裁决: 到达 marker → 该 decl; 跑干单出口 → 唯一 decl
// (未接 marker 的分支也让父图继续, 录制自动折叠图依赖此语义); 跑干多出口 → 出口歧义,
// Coded 错误可被 Fail 接.
func (r *ContainerRunner) runSubgraphCall(ctx context.Context, sg *container.Subgraph, callNodeID string, params map[string]any, loopStack []*LoopFrame) (*container.SubgraphOutputDecl, error) {
	if frameDepth(r.state) >= maxSubgraphCallDepth {
		return nil, nodepkg.Failf(nodepkg.CodeSubgraphRecursion, nil,
			"子图调用嵌套超过 %d 层 (疑似递归), 调用 %q 被拒", maxSubgraphCallDepth, sg.ID)
	}
	sgc, ok := r.compiled.Subgraphs[sg.ID]
	if !ok {
		return nil, fmt.Errorf("子图 %q 没有预编译产物 (compile bug)", sg.ID)
	}
	entryID := sg.Entry.NodeID
	if entryID == "" {
		return nil, fmt.Errorf("子图 %q 缺 Entry marker (Normalize 漏?)", sg.ID)
	}

	r.state.PushFrame(container.MainGraphRef(), sg, callNodeID)
	maps.Copy(r.state.CurrentFrame.LocalParams, params)
	defer r.state.PopFrame()

	// 切 dispatch 表到 callee (读预编译产物, 不 hot rebuild). currentSG 供 runRegionBody
	// 识别 entry/output marker.
	savedEdges, savedDataEdges, savedNodes, savedSG := r.edges, r.dataEdges, r.nodesByID, r.currentSG
	r.edges, r.dataEdges, r.nodesByID, r.currentSG = sgc.Edges, sgc.DataEdges, sgc.NodesByID, sgc
	defer func() {
		r.edges, r.dataEdges, r.nodesByID, r.currentSG = savedEdges, savedDataEdges, savedNodes, savedSG
	}()

	seeds := r.edges.next(entryID+".Done", loopStack)
	reached, err := r.runRegionBody(ctx, seeds)
	if err != nil {
		return nil, err
	}
	if reached != nil {
		return reached, nil
	}
	if len(sg.OutputPins) == 1 {
		return &sg.OutputPins[0], nil
	}
	return nil, nodepkg.Failf(nodepkg.CodeSubgraphNoExit, nil,
		"子图 %q 跑完没有到达任何出口 (共 %d 个出口, 无法判定该走哪个)", sg.ID, len(sg.OutputPins))
}

// CallSubgraph 实现 node.SubgraphCaller — 脚本绑定层 Subgraph({SubgraphID, ...params}) 入口.
// 返回到达出口的 decl Name (人读名; 父图边用的 decl ID 是图层概念, 脚本不可见).
func (r *ContainerRunner) CallSubgraph(ctx context.Context, sgID string, params map[string]any) (string, error) {
	var sg *container.Subgraph
	for i := range r.rt.Subgraphs {
		if r.rt.Subgraphs[i].ID == sgID {
			sg = &r.rt.Subgraphs[i]
			break
		}
	}
	if sg == nil {
		return "", nodepkg.Failf(nodepkg.CodeNotFound, nil, "子图 %q 不存在", sgID)
	}
	merged := map[string]any{}
	maps.Copy(merged, params)
	for _, p := range sg.InputParams {
		if v, ok := merged[p.Name]; (!ok || v == nil) && p.Default != nil {
			merged[p.Name] = toExprValue(p.Default)
		}
	}
	decl, err := r.runSubgraphCall(ctx, sg, "script-call", merged, nil)
	if err != nil {
		return "", err
	}
	return decl.Name, nil
}
