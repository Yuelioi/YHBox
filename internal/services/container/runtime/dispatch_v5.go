// dispatch_v5.go — dispatch 路径: execNode 走这里.
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
	"errors"
	"fmt"
	"maps"

	nodepkg "yhbox/internal/node"
	"yhbox/internal/nodes/control"
	"yhbox/internal/nodes/system"
	"yhbox/internal/services/container"
)

// execNodeViaFramework dispatch 单节点 via framework. 返下游 token 或 error.
// 不处理 RegionRunner — 那些走 r.execNodeAsRegionViaFramework.
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
// fallback r.pullDataPin.
//
// 注意 pin name 大小写: 用 Spec.Inputs[].Name 当 pin name 查 r.dataEdges, 两侧都是 PascalCase.
func (r *ContainerRunner) buildDataWireFor(ctx context.Context, node *container.GraphNode, rn *nodepkg.RegisteredNode) map[string]any {
	dw := map[string]any{}
	for _, ip := range rn.Spec.Inputs {
		if ip.Type == nodepkg.TypeExec {
			continue
		}
		v, err := r.resolveDataPinV5(ctx, node.ID, ip.Name)
		if err != nil || v == nil {
			continue
		}
		dw[ip.Name] = coerceToType(v, ip.Type)
	}
	// Expr 节点 dynamic data-in pins 在 Spec 里登记不到 — 走 config.Inputs[] 声明.
	// 必须额外 pull 一轮把声明的 dynamic name 喂进 dataWire, Expr.Evaluate 再从
	// in.Keys() 遍历 (跳过 Expression 静态 pin) 构造 expr.InputEnv.
	if node.Kind == "Expr" {
		cfg, _ := container.ParseExprConfig(node)
		for _, in := range cfg.Inputs {
			if in.Name == "" || in.Name == "Expression" {
				continue
			}
			if _, exists := dw[in.Name]; exists {
				continue
			}
			v, err := r.resolveDataPinV5(ctx, node.ID, in.Name)
			if err != nil || v == nil {
				continue
			}
			dw[in.Name] = v
		}
	}
	return dw
}

// resolveDataPinV5 解析单个 data-in pin 值:
//   - 没 data edge → literal / default → 走 r.pullDataPin
//   - 上游节点不在 framework registry / 不 IsPureData / 没实现 Evaluator → 走 r.pullDataPin
//   - 上游 IsPureData + 实现 Evaluator → 走 nodepkg.EvaluatePureData (递归 build 上游 dataWire)
//
// 22 purefunc + Expr 走 framework. GetVar/GetSys/GetParam 依赖 runtime state (frame /
// per-tick snapshot), 走 fallback evalDataSource switch.
func (r *ContainerRunner) resolveDataPinV5(ctx context.Context, nodeID, pinName string) (any, error) {
	srcID, _ := r.dataEdges.Source(nodeID, pinName)
	if srcID == "" {
		// 无 data edge — literal 或 default. pullDataPin 处理 (其内部会读 config["literal"]).
		v, err := r.pullDataPin(ctx, nodeID, pinName)
		return v, err
	}
	srcNode, ok := r.nodesByID[srcID]
	if !ok {
		return r.pullDataPin(ctx, nodeID, pinName)
	}
	// disabled pure-data → nil.
	if srcNode.Disabled {
		return nil, nil
	}
	srcRn, regOk := nodepkg.Get(srcNode.Kind)
	if !regOk || !srcRn.Spec.IsPureData || srcRn.Evaluate == nil {
		return r.pullDataPin(ctx, nodeID, pinName)
	}
	// 上游是 framework-evaluable pure-data → 递归 build 上游 dataWire + 调 EvaluatePureData.
	srcDataWire := r.buildDataWireFor(ctx, srcNode, srcRn)
	srcConfig := r.buildConfigFor(srcNode)
	return nodepkg.EvaluatePureData(ctx, srcRn, srcDataWire, srcConfig, r.bundle)
}

// buildConfigFor 复制 node.Config 当 framework config map. 扣 "literal" 内部字段
// (pullDataPin 已经消费它做 inline data-edge literal source).
//
// Config key 跟 Spec.Inputs[].Name 严格 case-sensitive 比 (两侧都是 PascalCase).
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

// buildExecDataFor 读 tok.ExecData (上游 routeResult 通过 edges.nextWithData 挂上).
// exec-data 语义: 上游 ctx.Out("exit").Set("k", v).Fire() → 下游 in.X("k") 读到 v.
// nil safe (源节点无 data 时 tok.ExecData == nil).
func (r *ContainerRunner) buildExecDataFor(tok ExecToken) map[string]any {
	if tok.ExecData == nil {
		return map[string]any{}
	}
	return tok.ExecData
}

// routeResult turns RunResult into next ExecToken batch + emit lifecycle events.
//
// Priority order:
//  1. Panic   → emit container:node-panic + return error (framework invariant broken)
//  2. Validation → emit container:node-validation + return error (graph 写错, 中断 run)
//  3. Error   → emit container:node-error + return error (Run 返 runtime fail, caller 决定 stop /
//     冒泡 / 走 Try; Stop sentinel / Try internal catch 走早返不 emit)
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
		return nil, fmt.Errorf("node %q (%s) validation: %s %v", node.ID, node.Kind, result.Validation[0].Code, result.Validation[0].Params)
	}

	if result.Error != nil {
		// Translate framework Stop sentinel → runtime errStopRun so ContainerRunner.Run
		// errors.Is(err, errStopRun) graceful halt path 仍 work.
		if control.IsStopRequested(result.Error) {
			return nil, errStopRun
		}
		// ctx 取消 (graph stop / 上层 cancel) 不是节点失败 — 透传, 不 emit node-error 高亮.
		// 跟主 dispatch loop 顶部 ctx.Err() 退出语义一致.
		if errors.Is(result.Error, context.Canceled) {
			return nil, result.Error
		}
		// 注意 Break/Continue/Throw sentinel 错误必须透传 — 外层 Loop.RunRegion / Try.RunRegion
		// 截获. 顶层 leak 防御在 ContainerRunner.Run / runSubFlow 主 loop.
		// Try 的 error path 写 SysState.LastTry.ErrorMsg — state_FISHING 等子图通过
		// GetSys path=lastTry.errorMsg 读. 注意 Try.RunRegion 内部已 catch error 走 catch
		// 出口, 这里 result.Error 漏到 routeResult 表示真失败 (validator drift / framework
		// bug), 仍存一份 ErrorMsg 给诊断.
		if node.Kind == "Try" {
			msg := result.Error.Error()
			r.rt.UpdateSys(func(s *SysState) { s.LastTry.ErrorMsg = msg })
		}
		// 跟 Panic / Validation 对齐 — emit container:node-error 让前端高亮失败节点.
		if r.rt.Emit != nil {
			r.rt.Emit("container:node-error", map[string]any{
				"containerId": r.rt.Container.ID,
				"nodeId":      node.ID,
				"nodeKind":    node.Kind,
				"message":     result.Error.Error(),
			})
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

	// 节点 OutputData 摘录写到 SysState. Spec SysStore 是 read-only, 写责任落到 dispatch
	// 层 (类似 VisionAdapter 在 vision call 里写). 只对真有 GetSys 路径消费的节点做.
	r.writeSysStateFromOutput(node, result)

	if result.ExitName == "" {
		return nil, nil
	}
	tokens := r.edges.nextWithData(node.ID+"."+result.ExitName, tok.LoopStack, result.OutputData)
	return tokens, nil
}

// writeSysStateFromOutput 按 node.Kind 摘录 result.OutputData 字段写 SysState.
// Spec 不暴露写口, 责任转到 dispatch.
//
// 当前只覆盖 Screenshot (LastScreenshot.Path). Match/WaitMatch/DetectColor/HSV/ROIScan/
// BarTrack 在 VisionAdapter 内已写, 不重复. Try.LastTry.ErrorMsg 走 error 路径单独写.
func (r *ContainerRunner) writeSysStateFromOutput(node *container.GraphNode, result nodepkg.RunResult) {
	if result.OutputData == nil {
		return
	}
	switch node.Kind {
	case "Screenshot":
		if path, _ := result.OutputData["Path"].(string); path != "" {
			r.rt.UpdateSys(func(s *SysState) { s.LastScreenshot.Path = path })
		}
	}
}

// ============================================================================
// Region runner sub-dispatch.
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

	body, err := r.makeBodyFor(ctx, node, tok)
	if err != nil {
		return nil, err
	}

	dataWire := r.buildDataWireFor(ctx, node, rn)
	config := r.buildConfigFor(node)
	execData := r.buildExecDataFor(tok)

	result := nodepkg.RunNodeAsRegion(ctx, rn, dataWire, config, execData, r.bundle, body)
	return r.routeResult(node, tok, result)
}

// dispatchInRegion 统一 dispatch entry for nodes inside region body. 这是 execNode 主入口.
// runRegionBody 调它派发 child 节点 — 自动 route 到 region (RunNodeAsRegion) 或 normal (RunNode).
//
// per-exec-tick snapshot 只在这里抓 (单一抓点) — runner.go::Run / nodes.go::runSubFlow /
// runRegionBody 都不重复抓. consumers (GetVar.Evaluate / GetSys.Evaluate 经 framework
// snapshot wrap) 只在节点 data pull 阶段读, 跟 dispatchInRegion → execNode(AsRegion)ViaFramework
// → buildDataWireFor 同周期, 入口抓一次足够.
func (r *ContainerRunner) dispatchInRegion(ctx context.Context, n *container.GraphNode, tok ExecToken) ([]ExecToken, error) {
	rn, ok := nodepkg.Get(n.Kind)
	if !ok {
		return nil, fmt.Errorf("dispatchInRegion: kind %q not registered", n.Kind)
	}
	// per-tick snapshot 走 ctx (tickCtxKey) — per-goroutine/per-token scope, 让共享 bundle
	// 的 listener subRunner / 并发 runner 都安全, 不撞 instance 字段.
	ctx = withTickSnapshot(ctx, CaptureSnapshot(r.rt.Vars(), r.rt.Sys()))
	if rn.RunRegion != nil {
		return r.execNodeAsRegionViaFramework(ctx, n, tok)
	}
	return r.execNodeViaFramework(ctx, n, tok)
}

// checkSentinelLeak detect Break/Continue/Throw sentinel 漏到顶层 dispatch loop.
// 正常路径里 Loop.RunRegion 截获 Break/Continue, Try.RunRegion 截获 Throw — 漏到
// 顶层主 loop 说明 validator 漏报或子图 misplace. 返非空 leakCode + 包装 err 让主
// loop emit container:node-validation 高亮失败节点; 不属 sentinel 返空 + 原 err.
func (r *ContainerRunner) checkSentinelLeak(node *container.GraphNode, err error) (leakCode string, wrapped error) {
	switch {
	case control.IsBreakRequested(err):
		leakCode = "BREAK_OUTSIDE_LOOP"
	case control.IsContinueRequested(err):
		leakCode = "CONTINUE_OUTSIDE_LOOP"
	case system.IsThrowRequested(err):
		leakCode = "THROW_OUTSIDE_TRY"
	default:
		return "", err
	}
	if r.rt.Emit != nil {
		r.rt.Emit("container:node-validation", map[string]any{
			"containerId": r.rt.Container.ID,
			"nodeId":      node.ID,
			"nodeKind":    node.Kind,
			"errors": []map[string]any{
				{"code": leakCode, "message": err.Error()},
			},
		})
	}
	return leakCode, fmt.Errorf("node %q (%s) %s: %w", node.ID, node.Kind, leakCode, err)
}

// runRegionBody region body sub-dispatch loop. 从 seeds 出发跑到队列空 / error 返.
//
// SubgraphInput / SubgraphOutput 特殊处理:
//   - SubgraphInput: framework 直通 .Done (其 Run 是 stub sentinel, 不该走 dispatchInRegion).
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
		// entry/output 是 virtual marker (不在 nodesByID), 走 metadata 路由.
		if r.currentSG != nil {
			if tok.NodeID == r.currentSG.EntryNodeID {
				queue = append(queue, r.edges.next(tok.NodeID+".Done", tok.LoopStack)...)
				continue
			}
			if _, isOutput := r.currentSG.OutputDeclsByID[tok.NodeID]; isOutput {
				return nil
			}
		}
		n, ok := r.nodesByID[tok.NodeID]
		if !ok {
			return fmt.Errorf("runRegionBody: unknown node %q", tok.NodeID)
		}
		// 节点级事件 — 跟 runner.go::Run 主 loop 同 emit, 让 GUI 高亮子图 / Loop body 内跑的节点.
		if r.rt.Emit != nil {
			r.rt.Emit("container:node-enter", map[string]any{
				"containerId": r.rt.Container.ID,
				"nodeId":      n.ID,
				"nodeKind":    n.Kind,
			})
		}
		// per-exec-tick snapshot 由 dispatchInRegion 入口统一抓 (单一抓点), 这里不重复.
		out, err := r.dispatchInRegion(ctx, n, tok)
		if err != nil {
			return err
		}
		queue = append(queue, out...)
	}
	return nil
}

// makeBodyFor build region body callback per node kind: Loop / Subgraph / CollapsedNode / Try.
//
// ctx 来自 execNodeAsRegionViaFramework — 携带 tickCtxKey, Subgraph/Try eager pullDataPin
// 链下去 bundle.Snapshot 能拿 frozen Vars. Loop body 不 pullDataPin 不需要外部 ctx.
func (r *ContainerRunner) makeBodyFor(ctx context.Context, node *container.GraphNode, tok ExecToken) (func(nodepkg.Ctx) error, error) {
	switch node.Kind {
	case "Loop":
		return r.makeBodyForLoop(node, tok), nil
	case "Subgraph", "CollapsedNode":
		return r.makeBodyForSubgraph(ctx, node, tok)
	case "Try":
		return r.makeBodyForTry(ctx, node, tok)
	}
	return nil, fmt.Errorf("makeBodyFor: no body builder for kind %q (region runner not yet supported)", node.Kind)
}

// makeBodyForLoop body callback 每次调跑一轮 Loop body (从 node.body 出口下游 seed 到 queue 空).
// errBreakRequested / errContinueRequested sentinel 透传; Loop.RunRegion 截获.
func (r *ContainerRunner) makeBodyForLoop(node *container.GraphNode, tok ExecToken) func(nodepkg.Ctx) error {
	parentLoopStack := tok.LoopStack
	return func(c nodepkg.Ctx) error {
		seeds := r.edges.next(node.ID+".Body", parentLoopStack)
		return r.runRegionBody(c.Context(), seeds)
	}
}

// makeBodyForTry — Try 节点 body 行为跟 Subgraph 几乎完全一致 (解 SubgraphID, push frame,
// 切 dispatch table, sub-dispatch), 但 body 返 error 时不需要在这里特殊处理 — Try.RunRegion
// 内部已经把 error → catch 出口 + error.Error() 字符串挂 catch.error 字段.
//
// 复用 makeBodyForSubgraph 的实现就好. (如果 Try 未来要区别 Throw vs 普通 error 等更细
// 语义, Try.RunRegion 内做, 这里继续透传.)
func (r *ContainerRunner) makeBodyForTry(ctx context.Context, node *container.GraphNode, tok ExecToken) (func(nodepkg.Ctx) error, error) {
	return r.makeBodyForSubgraph(ctx, node, tok)
}

// makeBodyForSubgraph body 调一次 — 解析 SubgraphID + push frame + 切 dispatch table 到 callee +
// SubgraphInput.Done 出发 sub-dispatch + SubgraphOutput 终点 return nil + restore frame & tables.
//
// SubgraphID 从 node.Config["SubgraphID"] (PascalCase, 跟 Spec.Inputs.SubgraphID 对齐) 取.
func (r *ContainerRunner) makeBodyForSubgraph(ctx context.Context, node *container.GraphNode, tok ExecToken) (func(nodepkg.Ctx) error, error) {
	sgID := container.PinString(node, "SubgraphID")
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
	//   1. node.Config["Params"] static JSON map.
	//   2. sg.InputParams 声明的入参, 每个 pullDataPin(callerNode, paramName), 让 caller 通过
	//      data-in pin 或 literal pin 推 dynamic param.
	//   3. p.Default fallback if pull 出 nil.
	staticParams := container.PinMap(node, "Params")
	type pulledParam struct {
		name string
		val  any
	}
	var pulled []pulledParam
	for _, p := range sg.InputParams {
		v, perr := r.pullDataPin(ctx, node.ID, p.Name)
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

		// Save dispatch tables, swap to subgraph's.
		// 读 r.compiled.Subgraphs 预编译产物 (不 hot rebuild edge index).
		// 同时 swap currentSG, runRegionBody 用来识 entry/output marker.
		savedEdges := r.edges
		savedDataEdges := r.dataEdges
		savedNodesByID := r.nodesByID
		savedCurrentSG := r.currentSG
		sgc, ok := r.compiled.Subgraphs[sg.ID]
		if !ok {
			return fmt.Errorf("Subgraph %s: callee %q not pre-compiled (compile bug)", node.ID, sg.ID)
		}
		r.edges = sgc.Edges
		r.dataEdges = sgc.DataEdges
		r.nodesByID = sgc.NodesByID
		r.currentSG = sgc
		defer func() {
			r.edges = savedEdges
			r.dataEdges = savedDataEdges
			r.nodesByID = savedNodesByID
			r.currentSG = savedCurrentSG
		}()

		// entry NodeID 从 sg.Entry metadata 拿 (不 scan Graph.Nodes).
		entryID := sg.Entry.NodeID
		if entryID == "" {
			return fmt.Errorf("Subgraph %s: callee %q missing Entry (Normalize 漏?)", node.ID, sg.ID)
		}

		seeds := r.edges.next(entryID+".Done", parentLoopStack)
		return r.runRegionBody(c.Context(), seeds)
	}, nil
}
