// dispatch_v5.go — Phase 5.5 新 dispatch 路径 (atomic #3 ship: execNode 走这里).
//
// 通过 node.RunNode / RunNodeAsRegion 派发节点.
//
// 核心:
//   - execNodeViaFramework: 非 region 节点 → RunNode + routeResult
//   - execNodeAsRegionViaFramework: RegionRunner 节点 → RunNodeAsRegion + body 回调
//   - dispatchInRegion: 统一 router, 自动 route 到 region 或 normal path. execNode 主入口
//   - runRegionBody: region body sub-dispatch loop
//   - makeBodyFor*: per-kind body callback (Loop / Subgraph / CollapsedNode / Try)
package runtime

import (
	"context"
	"fmt"
	"maps"

	nodepkg "yhbox/internal/node"
	"yhbox/internal/nodes/control"
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

	dataWire := r.buildDataWireFor(ctx, node, rn)
	config := r.buildConfigFor(node)
	execData := r.buildExecDataFor(tok)

	result := nodepkg.RunNode(ctx, rn, dataWire, config, execData, r.bundle)
	return r.routeResult(node, tok, result)
}

// buildDataWireFor pulls all non-Exec data-in pins from node Spec.
//
// 每个 pin 走 resolveDataPinV5 — 上游是 IsPureData + 实现 Evaluator → 框架 EvaluatePureData
// (递归 build 上游 dataWire); 否则 (GetVar/GetSys/GetParam/Expr/exec-node data-out/literal/无 edge)
// fallback 老 r.pullDataPin. 这就是 Phase 6+ partial 的 "上游是 pure-data → 框架 evaluate, 否则
// 兜底转换期".
//
// 注意 pin name 大小写: 用 Spec.Inputs[].Name 当 pin name 查 r.dataEdges. 老 JSON edges 用 lowercase,
// 新 Spec 用 PascalCase — Phase 5.6 fishing-v2 redraw 后, 新 JSON 跟 Spec 一致就 work.
func (r *ContainerRunner) buildDataWireFor(ctx context.Context, node *container.GraphNode, rn *nodepkg.RegisteredNode) map[string]any {
	dw := map[string]any{}
	for _, ip := range rn.Spec.Inputs {
		if ip.Type == "Exec" {
			continue
		}
		v, err := r.resolveDataPinV5(ctx, node.ID, ip.Name)
		if err != nil || v == nil {
			continue
		}
		dw[ip.Name] = v
	}
	return dw
}

// resolveDataPinV5 解析单个 data-in pin 值. transition-period dispatch:
//   - 没 data edge → literal / default → 走老 r.pullDataPin
//   - 上游节点不在 framework registry / 不 IsPureData / 没实现 Evaluator → 走老 r.pullDataPin
//   - 上游 IsPureData + 实现 Evaluator → 走 nodepkg.EvaluatePureData (递归 build 上游 dataWire)
//
// Phase 6+ partial: 22 purefunc + Eq/Not/Concat/... 走 framework. GetVar/GetSys/GetParam/Expr
// 依赖 runtime state / dynamic input, 暂走 fallback 老 evalDataSource switch.
func (r *ContainerRunner) resolveDataPinV5(ctx context.Context, nodeID, pinName string) (any, error) {
	srcID, _ := r.dataEdges.Source(nodeID, pinName)
	if srcID == "" {
		// 无 data edge — literal 或 default. 老 pullDataPin 处理 (其内部会读 config["literal"]).
		v, err := r.pullDataPin(nodeID, pinName)
		return v, err
	}
	srcNode, ok := r.nodesByID[srcID]
	if !ok {
		return r.pullDataPin(nodeID, pinName)
	}
	// Editor v2 C: disabled pure-data → nil (跟老 evalDataSource 行为一致).
	if srcNode.Disabled {
		return nil, nil
	}
	srcRn, regOk := nodepkg.Get(srcNode.Kind)
	if !regOk || !srcRn.Spec.IsPureData || srcRn.Evaluate == nil {
		return r.pullDataPin(nodeID, pinName)
	}
	// 上游是 framework-evaluable pure-data → 递归 build 上游 dataWire + 调 EvaluatePureData.
	srcDataWire := r.buildDataWireFor(ctx, srcNode, srcRn)
	srcConfig := r.buildConfigFor(srcNode)
	return nodepkg.EvaluatePureData(ctx, srcRn, srcDataWire, srcConfig, r.bundle)
}

// buildConfigFor 复制 node.Config 当 framework config map. 扣 "literal" 内部字段
// (pullDataPin 已经消费它做 inline data-edge literal source).
//
// Config key 跟 Spec.Inputs[].Name 严格 case-sensitive 比. fishing-v2 JSON 跟所有 test
// fixture 已 redraw 到 PascalCase, 不再需要 lowercase 镜像.
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
		// Translate framework Stop sentinel → runtime errStopRun so ContainerRunner.Run
		// errors.Is(err, errStopRun) graceful halt path 仍 work.
		if control.IsStopRequested(result.Error) {
			return nil, errStopRun
		}
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
	tokens := r.edges.next(node.ID+"."+result.ExitName, tok.LoopStack)
	return tokens, nil
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

	dataWire := r.buildDataWireFor(ctx, node, rn)
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
		// 节点级事件 — 跟 runner.go::Run 主 loop 同 emit, 让 GUI 高亮子图 / Loop body 内
		// 跑的节点 (老 runner 只 emit 顶层一层, 子区域走 runRegionBody 不进 execNode 也没 emit).
		if r.rt.Emit != nil {
			r.rt.Emit("container:node-enter", map[string]any{
				"containerId": r.rt.Container.ID,
				"nodeId":      n.ID,
				"nodeKind":    n.Kind,
			})
		}
		// 每次 sub-dispatch 刷 per-exec-tick snapshot — 老 evalGetVar / evalGetSys 从
		// r.currentTick.Vars 读. 不刷 → Loop body 跨 iter 看 stale snapshot, 影响
		// Break/Continue 条件判断. runner.go::Run 在 execNode entry 抓一次, 但 region
		// body 走 runRegionBody 不经 execNode, 这里得自己 refresh.
		r.currentTick = CaptureSnapshot(r.rt.Vars(), r.rt.Sys())
		out, err := r.dispatchInRegion(ctx, n, tok)
		r.currentTick = nil
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
	// Params 来源 (3 路 union, 优先级 1→3):
	//   1. node.Config["Params"] static JSON map (Phase 5.5 设计).
	//   2. sg.InputParams 声明的入参, 每个 pullDataPin(callerNode, paramName) — 复刻老
	//      execSubgraph 行为, 让 caller 通过 data-in pin 或 literal pin 推 dynamic param.
	//      Phase 6+ pull-eval 让上游 pure-data via framework 走通这条.
	//   3. p.Default fallback if pull 出 nil.
	//
	// 转换期里 #1 + #2 同时认 — atomic #5 拆老后只留 #2 (规范 dynamic Params 走 InputParams + data-in).
	staticParams, _ := node.Config["Params"].(map[string]any)
	type pulledParam struct {
		name string
		val  any
	}
	var pulled []pulledParam
	for _, p := range sg.InputParams {
		v, perr := r.pullDataPin(node.ID, p.Name)
		if perr != nil {
			return nil, fmt.Errorf("Subgraph %s: pull input %q: %w", node.ID, p.Name, perr)
		}
		if v == nil && p.Default != nil {
			v = toExprValue(p.Default)
		}
		pulled = append(pulled, pulledParam{name: p.Name, val: v})
	}
	return func(c nodepkg.Ctx) error {
		// Push frame + seed LocalParams (#1 static, then #2 pulled covers same name).
		r.state.PushFrame(container.MainGraphRef(), sg, node.ID)
		maps.Copy(r.state.CurrentFrame.LocalParams, staticParams)
		for _, p := range pulled {
			// Only override static Params if pulled value is non-nil — caller might use
			// Config["Params"] static literal (no data-in pin), pulled nil would clobber.
			if p.val != nil {
				r.state.CurrentFrame.LocalParams[p.name] = p.val
			}
		}
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
