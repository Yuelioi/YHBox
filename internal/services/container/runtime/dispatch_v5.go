// dispatch_v5.go — Phase 5.5 新 dispatch 路径.
//
// 通过 node.RunNode / RunNodeAsRegion 派发节点, 取代老 nodes.go 大 switch.
//
// Phase 5.5a (ship): execNodeViaFramework 处理非 region 节点 + routeResult 映射 RunResult → ExecToken.
// Phase 5.5b (ship): execNodeAsRegionViaFramework + runRegionBody + makeBodyFor (Loop / Subgraph). Try 待用户拍板 Spec 加 SubgraphID 后实现.
// Phase 5.5c (TODO): cutover — execNode entry 改成走 dispatchInRegion + 拆老 switch.
//
// 当前阶段不挂接 execNode entry — 老 nodes.go switch 仍是 production. fishing-v2 零影响.
package runtime

import (
	"context"
	"fmt"

	nodepkg "yhbox/internal/node"
	"yhbox/internal/services/container"
)

// execNodeViaFramework dispatch 单节点 via 新 framework. 返下游 token 或 error.
// 不处理 RegionRunner — 那些走 r.execNodeAsRegionViaFramework (Phase 5.5b).
func (r *ContainerRunner) execNodeViaFramework(ctx context.Context, node *container.GraphNode, tok ExecToken) ([]ExecToken, error) {
	rn, ok := nodepkg.Get(node.Kind)
	if !ok {
		return nil, fmt.Errorf("execNodeViaFramework: kind %q not registered in node package", node.Kind)
	}
	if rn.RunRegion != nil {
		return nil, fmt.Errorf("execNodeViaFramework: kind %q is RegionRunner, must route via region path (Phase 5.5b)", node.Kind)
	}

	dataWire := r.buildDataWireFor(node, rn)
	config := r.buildConfigFor(node)
	execData := r.buildExecDataFor(tok)

	result := nodepkg.RunNode(ctx, rn, dataWire, config, execData, r.bundle)
	return r.routeResult(node, tok, result)
}

// buildDataWireFor pulls all non-Exec data-in pins from node Spec, evaluating via r.pullDataPin.
// Skip pin → pin not in map → framework Inputs.Has = false → falls back to config / default.
//
// 注意 pin name 大小写: r.pullDataPin 用 Spec.Inputs[].Name 当 pin name 查 r.dataEdges, 而
// edges 里的 To 字段是 JSON 序列化的 (老 JSON 的 lowercase pin name). 所以新框架节点 (PascalCase Spec)
// 跟老 JSON edges 对不上 — Phase 5.6 fishing-v2 redraw 后, 新 JSON pin name 跟 Spec 一致就 work.
func (r *ContainerRunner) buildDataWireFor(node *container.GraphNode, rn *nodepkg.RegisteredNode) map[string]any {
	dw := map[string]any{}
	for _, ip := range rn.Spec.Inputs {
		if ip.Type == "Exec" {
			continue
		}
		v, err := r.pullDataPin(node.ID, ip.Name)
		if err != nil || v == nil {
			continue
		}
		dw[ip.Name] = v
	}
	return dw
}

// buildConfigFor 复制 node.Config 当 framework config map. 扣 "literal" 内部字段
// (pullDataPin 已经消费它做 inline data-edge literal source).
//
// Phase 5.5a 不做 pin name normalization — config key 跟 Spec.Inputs[].Name 严格 case-sensitive 比.
// 老 fishing-v2 JSON 用 lowercase, 新 Spec 用 PascalCase, 不会自动对齐. Phase 5.6 redraw 时按新名写.
func (r *ContainerRunner) buildConfigFor(node *container.GraphNode) map[string]any {
	out := make(map[string]any, len(node.Config))
	for k, v := range node.Config {
		if k == "literal" {
			continue
		}
		out[k] = v
	}
	return out
}

// buildExecDataFor 当前返空 map. Phase 5.5b 加 exec-data carry — 上游节点 RunResult.OutputData
// 的字段在 routeResult 时 stash 到 ExecToken (新加字段 Token.ExecData), 下游 build 时 pull 出.
// 这步要先扩 ExecToken struct + 改 edges.next 携带 data + 改 buildExecDataFor 读. 留后续.
func (r *ContainerRunner) buildExecDataFor(_ ExecToken) map[string]any {
	return map[string]any{}
}

// routeResult turns RunResult into next ExecToken batch + emit lifecycle events.
//
// Priority order:
//  1. Panic   → emit container:node-panic + return error (framework invariant broken)
//  2. Validation → emit container:node-validation + return error (graph 写错, Phase 5.5a 中断 run;
//     后续可改 emit + continue 让前端高亮但不停)
//  3. Error   → return error (Run 返 runtime fail, caller 决定 stop / 冒泡 / 走 Try)
//  4. Display → emit container:node-log (Display() 非空才 emit)
//  5. ExitName 空 → return nil tokens (no-op 节点, queue 继续)
//  6. ExitName 非空 → r.edges.next(nodeID.exitName)
func (r *ContainerRunner) routeResult(node *container.GraphNode, tok ExecToken, result nodepkg.RunResult) ([]ExecToken, error) {
	if result.Panic != nil {
		if r.rt.Emit != nil {
			r.rt.Emit("container:node-panic", map[string]any{
				"containerId": r.rt.Container.ID,
				"nodeId":      node.ID,
				"nodeKind":    node.Kind,
				"panic":       fmt.Sprintf("%v", result.Panic),
				"stack":       result.PanicStack,
			})
		}
		return nil, fmt.Errorf("node %q (%s) panic: %v", node.ID, node.Kind, result.Panic)
	}

	if len(result.Validation) > 0 {
		if r.rt.Emit != nil {
			r.rt.Emit("container:node-validation", map[string]any{
				"containerId": r.rt.Container.ID,
				"nodeId":      node.ID,
				"nodeKind":    node.Kind,
				"errors":      result.Validation,
			})
		}
		return nil, fmt.Errorf("node %q (%s) validation: %s", node.ID, node.Kind, result.Validation[0].Message)
	}

	if result.Error != nil {
		return nil, result.Error
	}

	if result.DisplayText != "" && r.rt.Emit != nil {
		r.rt.Emit("container:node-log", map[string]any{
			"containerId": r.rt.Container.ID,
			"nodeId":      node.ID,
			"nodeKind":    node.Kind,
			"message":     result.DisplayText,
		})
	}

	if result.ExitName == "" {
		return nil, nil
	}
	return r.edges.next(node.ID+"."+result.ExitName, tok.LoopStack), nil
}

// ============================================================================
// Phase 5.5b — Region runner sub-dispatch.
// ============================================================================

// execNodeAsRegionViaFramework dispatch RegionRunner 节点 (Loop / Subgraph / 未来 Try).
// 构造 body callback (per kind), 调 nodepkg.RunNodeAsRegion, 走 routeResult.
func (r *ContainerRunner) execNodeAsRegionViaFramework(ctx context.Context, node *container.GraphNode, tok ExecToken) ([]ExecToken, error) {
	rn, ok := nodepkg.Get(node.Kind)
	if !ok {
		return nil, fmt.Errorf("execNodeAsRegionViaFramework: kind %q not registered", node.Kind)
	}
	if rn.RunRegion == nil {
		return nil, fmt.Errorf("execNodeAsRegionViaFramework: kind %q is not a RegionRunner", node.Kind)
	}

	body, err := r.makeBodyFor(node, tok)
	if err != nil {
		return nil, err
	}

	dataWire := r.buildDataWireFor(node, rn)
	config := r.buildConfigFor(node)
	execData := r.buildExecDataFor(tok)

	result := nodepkg.RunNodeAsRegion(ctx, rn, dataWire, config, execData, r.bundle, body)
	return r.routeResult(node, tok, result)
}

// dispatchInRegion 统一 dispatch entry for nodes inside region body.
// runRegionBody 调它派发 child 节点 — 自动 route 到 region (RunNodeAsRegion) 或 normal (RunNode).
//
// Phase 5.5c cutover 后这是 execNode 主入口 (替换老 switch).
func (r *ContainerRunner) dispatchInRegion(ctx context.Context, n *container.GraphNode, tok ExecToken) ([]ExecToken, error) {
	rn, ok := nodepkg.Get(n.Kind)
	if !ok {
		return nil, fmt.Errorf("dispatchInRegion: kind %q not registered", n.Kind)
	}
	if rn.RunRegion != nil {
		return r.execNodeAsRegionViaFramework(ctx, n, tok)
	}
	return r.execNodeViaFramework(ctx, n, tok)
}

// runRegionBody region body sub-dispatch loop. 从 seeds 出发跑到队列空 / error 返.
//
// SubgraphInput / SubgraphOutput 特殊处理:
//   - SubgraphInput: framework 直通 .out (其 Run 是 stub sentinel, 不该走 dispatchInRegion).
//   - SubgraphOutput: body 终点, runRegionBody 直接 return nil (framework 不调 Run).
//
// 节点 Run 返 error sentinel (errBreakRequested / errContinueRequested / errThrow) 透传给 body caller.
// 调用方 RunRegion 用 errors.Is 截获.
func (r *ContainerRunner) runRegionBody(ctx context.Context, seeds []ExecToken) error {
	queue := append([]ExecToken{}, seeds...)
	for len(queue) > 0 {
		if err := ctx.Err(); err != nil {
			return err
		}
		tok := queue[0]
		queue = queue[1:]
		n, ok := r.nodesByID[tok.NodeID]
		if !ok {
			return fmt.Errorf("runRegionBody: unknown node %q", tok.NodeID)
		}
		// SubgraphInput / Output markers — sub-runner 不走 Run.
		if n.Kind == "SubgraphInput" {
			queue = append(queue, r.edges.next(n.ID+".out", tok.LoopStack)...)
			continue
		}
		if n.Kind == "SubgraphOutput" {
			return nil
		}
		out, err := r.dispatchInRegion(ctx, n, tok)
		if err != nil {
			return err
		}
		queue = append(queue, out...)
	}
	return nil
}

// makeBodyFor build region body callback per node kind.
// Phase 5.5b 支持 Loop / Subgraph / CollapsedNode / Try.
func (r *ContainerRunner) makeBodyFor(node *container.GraphNode, tok ExecToken) (func(nodepkg.Ctx) error, error) {
	switch node.Kind {
	case "Loop":
		return r.makeBodyForLoop(node, tok), nil
	case "Subgraph", "CollapsedNode":
		return r.makeBodyForSubgraph(node, tok)
	case "Try":
		return r.makeBodyForTry(node, tok)
	}
	return nil, fmt.Errorf("makeBodyFor: no body builder for kind %q (region runner not yet supported)", node.Kind)
}

// makeBodyForLoop body callback 每次调跑一轮 Loop body (从 node.body 出口下游 seed 到 queue 空).
// errBreakRequested / errContinueRequested sentinel 透传; Loop.RunRegion 截获.
func (r *ContainerRunner) makeBodyForLoop(node *container.GraphNode, tok ExecToken) func(nodepkg.Ctx) error {
	parentLoopStack := tok.LoopStack
	return func(c nodepkg.Ctx) error {
		seeds := r.edges.next(node.ID+".body", parentLoopStack)
		return r.runRegionBody(c.Context(), seeds)
	}
}

// makeBodyForTry — Try 节点 body 行为跟 Subgraph 几乎完全一致 (解 SubgraphID, push frame,
// 切 dispatch table, sub-dispatch), 但 body 返 error 时不需要在这里特殊处理 — Try.RunRegion
// 内部已经把 error → catch 出口 + error.Error() 字符串挂 catch.error 字段.
//
// 复用 makeBodyForSubgraph 的实现就好. (如果 Try 未来要区别 Throw vs 普通 error 等更细
// 语义, Try.RunRegion 内做, 这里继续透传.)
func (r *ContainerRunner) makeBodyForTry(node *container.GraphNode, tok ExecToken) (func(nodepkg.Ctx) error, error) {
	return r.makeBodyForSubgraph(node, tok)
}

// makeBodyForSubgraph body 调一次 — 解析 SubgraphID + push frame + 切 dispatch table 到 callee +
// SubgraphInput.out 出发 sub-dispatch + SubgraphOutput 终点 return nil + restore frame & tables.
//
// SubgraphID 从 node.Config["SubgraphID"] (PascalCase, 跟新 Spec.Inputs.SubgraphID 对齐) 取.
// 老 runtime ResolveSubgraphCall 用 lowercase "subgraphId" — Phase 5.5c cutover 后老路径删,
// 那个 helper 一起拆.
func (r *ContainerRunner) makeBodyForSubgraph(node *container.GraphNode, tok ExecToken) (func(nodepkg.Ctx) error, error) {
	sgID, _ := node.Config["SubgraphID"].(string)
	if sgID == "" {
		return nil, fmt.Errorf("Subgraph %s: missing SubgraphID in Config", node.ID)
	}
	var sg *container.Subgraph
	for i := range r.rt.Container.Subgraphs {
		if r.rt.Container.Subgraphs[i].ID == sgID {
			sg = &r.rt.Container.Subgraphs[i]
			break
		}
	}
	if sg == nil {
		return nil, fmt.Errorf("Subgraph %s: subgraph %q not found in container %q", node.ID, sgID, r.rt.Container.ID)
	}
	parentLoopStack := tok.LoopStack
	return func(c nodepkg.Ctx) error {
		// Push frame
		r.state.PushFrame(container.MainGraphRef(), sg, node.ID)
		defer r.state.PopFrame()

		// Save dispatch tables, swap to subgraph's
		savedEdges := r.edges
		savedDataEdges := r.dataEdges
		savedNodesByID := r.nodesByID
		r.edges = buildEdgeIndex(sg.Graph)
		r.dataEdges = buildDataEdgeIndex(sg.Graph)
		r.nodesByID = make(map[string]*container.GraphNode, len(sg.Graph.Nodes))
		for i := range sg.Graph.Nodes {
			sgn := &sg.Graph.Nodes[i]
			r.nodesByID[sgn.ID] = sgn
		}
		defer func() {
			r.edges = savedEdges
			r.dataEdges = savedDataEdges
			r.nodesByID = savedNodesByID
		}()

		// Find SubgraphInput entry
		var inputID string
		for i := range sg.Graph.Nodes {
			if sg.Graph.Nodes[i].Kind == "SubgraphInput" {
				inputID = sg.Graph.Nodes[i].ID
				break
			}
		}
		if inputID == "" {
			return fmt.Errorf("Subgraph %s: callee %q missing SubgraphInput", node.ID, sg.ID)
		}

		seeds := r.edges.next(inputID+".out", parentLoopStack)
		return r.runRegionBody(c.Context(), seeds)
	}, nil
}
