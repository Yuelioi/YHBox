// dispatch_v5.go — dispatch 路径: execNode 走这里.
//
// 通过 node.RunNode / RunNodeAsRegion 派发节点.
//
// 核心:
//   - execNodeViaFramework: 非 region 节点 → RunNode + routeResult
//   - execNodeAsRegionViaFramework: RegionRunner 节点 → RunNodeAsRegion + body 回调
//   - dispatchInRegion: 统一 router, 自动 route 到 region 或 normal path. execNode 主入口
//   - runRegionBody: region body sub-dispatch loop
//   - makeBodyFor*: per-kind body callback (Loop / Subgraph / CollapsedNode)
package runtime

import (
	"context"
	"errors"
	"fmt"
	"maps"

	nodepkg "yotta/internal/node"
	"yotta/internal/nodes/control"
	"yotta/internal/services/container"
)

// execNodeViaFramework dispatch 单节点 via framework. 返下游 token 或 error.
// 不处理 RegionRunner — 那些走 r.execNodeAsRegionViaFramework.
func (r *ContainerRunner) execNodeViaFramework(ctx context.Context, node *container.GraphNode, tok ExecToken) ([]ExecToken, error) {
	rn, ok := nodepkg.Get(node.Kind)
	if !ok {
		return nil, fmt.Errorf("execNodeViaFramework: kind %q not registered in node package", node.Kind)
	}
	if rn.RunRegion != nil {
		return nil, fmt.Errorf("execNodeViaFramework: kind %q is RegionRunner, must route via region path", node.Kind)
	}

	dataWire := r.buildDataWireFor(ctx, node, rn)
	config := r.buildConfigFor(node)
	execData := r.buildExecDataFor(tok)
	r.applyExecDataEdges(node, rn, execData, dataWire)

	result := nodepkg.RunNode(ctx, rn, dataWire, config, execData, r.bundle, node.LogEnabled)
	return r.routeResult(node, tok, result)
}

// buildDataWireFor pulls all non-Exec data-in pins from node Spec.
//
// 每个 pin 走 resolveDataPinV5 — 上游是 IsPureData + 实现 Evaluator → 框架 EvaluatePureData
// (递归 build 上游 dataWire); 否则 (GetVar/GetParam/Expr/exec-node data-out/literal/无 edge)
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
	// DynamicInputs 节点 (Expr/Script) 的 dynamic data-in pins 在 Spec 里登记不到 —
	// 走 config.Inputs[] 声明. 必须额外 pull 一轮把声明的 dynamic name 喂进 dataWire,
	// 节点 Evaluate/Run 再从 in.Keys() 遍历 (跳过 Spec 静态 pin) 消费.
	if rn.Spec.DynamicInputs {
		static := map[string]bool{}
		for _, ip := range rn.Spec.Inputs {
			static[ip.Name] = true
		}
		for _, in := range container.ParseDynamicInputDecls(node) {
			if in.Name == "" || static[in.Name] {
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

// applyExecDataEdges 回填"源是 exec-output data 字段 (如 Fail.Code)"的 data-in pin.
// 这类 pin 的值不走 pure-data pull (buildDataWireFor/pullDataPin 已跳过为 nil), 而是沿父
// exec 边作为 exec-data 下发到本节点 —— srcPin (字段名, 如 "Code") 即 exec-data 的 key.
// 让"失败出口的 Error/Code 数据线接到下游 data-in (如 Switch.Value)"端到端跑通.
//
// 约束: data 边应跟父 exec 边并行 (源 = 触发本节点的 exec 上游); 否则 execData 无该 key → 跳过.
func (r *ContainerRunner) applyExecDataEdges(node *container.GraphNode, rn *nodepkg.RegisteredNode, execData, dw map[string]any) {
	if len(execData) == 0 {
		return
	}
	for _, ip := range rn.Spec.Inputs {
		if ip.Type == nodepkg.TypeExec {
			continue
		}
		srcID, srcPin := r.dataEdges.Source(node.ID, ip.Name)
		if srcID == "" {
			continue
		}
		src := r.nodesByID[srcID]
		if src == nil || !container.IsExecOutputDataField(src.Kind, srcPin) {
			continue
		}
		if v, ok := execData[srcPin]; ok {
			dw[ip.Name] = coerceToType(v, ip.Type)
		}
	}
}

// resolveDataPinV5 解析单个 data-in pin 值:
//   - 没 data edge → literal / default → 走 r.pullDataPin
//   - 上游节点不在 framework registry / 不 IsPureData / 没实现 Evaluator → 走 r.pullDataPin
//   - 上游 IsPureData + 实现 Evaluator → 走 evalPureDataCached (共享 per-dispatch 缓存 gate,
//     递归 build 上游 dataWire)
//
// 22 purefunc + Expr 走 framework. GetVar/GetParam 依赖 runtime state (frame /
// per-tick snapshot), 走 fallback evalDataSource switch.
func (r *ContainerRunner) resolveDataPinV5(ctx context.Context, nodeID, pinName string) (any, error) {
	srcID, srcPin := r.dataEdges.Source(nodeID, pinName)
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
	// 上游是 framework-evaluable pure-data → 共享 cache gate (evalPureDataCached) 递归求值.
	return r.evalPureDataCached(ctx, srcID, srcPin, srcNode, srcRn)
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
//  1. Panic   → emitDump(err) + emit container:node-panic + return error (framework invariant broken)
//  2. Validation → emit container:node-validation + return error (graph 写错, 中断 run; 节点没真跑, 不 dump)
//  3. Error   → emitDump(err) + emit container:node-error + return error (Run 返 runtime fail, caller 决定 stop /
//     冒泡 / 走 Fail 出口; Stop sentinel 走早返不 emit)
//  4. Success → emitDump(nil) (仅勾选节点; opt-in node dump)
//  5. ExitName 空 → return nil tokens (no-op 节点, queue 继续)
//  6. ExitName 非空 → r.edges.next(nodeID.exitName)
func (r *ContainerRunner) routeResult(node *container.GraphNode, tok ExecToken, result nodepkg.RunResult) ([]ExecToken, error) {
	if result.Panic != nil {
		r.emitDump(node, result, fmt.Errorf("panic: %v", result.Panic))
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
		// Break/Continue 必须漏给 Loop.RunRegion — 永不进失败路由. 顶层 leak 防御在
		// ContainerRunner.Run 主 loop (checkSentinelLeak).
		if control.IsBreakRequested(result.Error) || control.IsContinueRequested(result.Error) {
			return nil, result.Error
		}
		// 失败路由: 仅 Coded 错误 (Failf / Throw) + 本节点 Fail 出口接线 → 走失败分支;
		// 裸 error (配置错) 或没接线 → 照旧冒泡中断.
		var coded nodepkg.Coded
		handled := errors.As(result.Error, &coded) && r.edges.has(node.ID+".Fail")

		// dump 行先于 node-error (同 goroutine → 有序), 让勾选节点失败也吐 in/err.
		r.emitDump(node, result, result.Error)
		// 跟 Panic / Validation 对齐 — emit container:node-error 让前端高亮失败节点.
		// handled=true → 前端柔和标记 (已就地处理); false → 红高亮 (冒泡中断).
		if r.rt.Emit != nil {
			ev := map[string]any{
				"containerId": r.rt.Container.ID,
				"nodeId":      node.ID,
				"nodeKind":    node.Kind,
				"message":     result.Error.Error(),
				"handled":     handled,
			}
			if coded != nil {
				ev["code"] = string(coded.ErrCode())
			}
			r.rt.Emit("container:node-error", ev)
		}
		if handled {
			return r.edges.nextWithData(node.ID+".Fail", tok.LoopStack, map[string]any{
				"Error": result.Error.Error(),
				"Code":  string(coded.ErrCode()),
			}), nil
		}
		return nil, result.Error
	}

	// 成功路径: 勾选节点吐 dump 行 (in/out), 经 merger 合并.
	r.emitDump(node, result, nil)

	if result.ExitName == "" {
		return nil, nil
	}
	tokens := r.edges.nextWithData(node.ID+"."+result.ExitName, tok.LoopStack, result.OutputData)
	return tokens, nil
}

// emitDump 给勾选节点组 dump 行并经 Emit 送中央 merger (app 拦截 container:node-dump).
// runErr != nil → error 行 (in{} err=, out 仅非空才带); 该行由 merger 落定不参与合并.
func (r *ContainerRunner) emitDump(node *container.GraphNode, result nodepkg.RunResult, runErr error) {
	if !node.LogEnabled || r.rt.Emit == nil {
		return
	}
	rn, ok := nodepkg.Get(node.Kind)
	if !ok {
		return
	}
	line, key := FormatDumpLine(&rn.Spec, node.Label, node.ID, result.ResolvedInputs, result.OutputData, runErr)
	r.rt.Emit("container:node-dump", map[string]any{
		"containerId": r.rt.Container.ID,
		"nodeId":      node.ID,
		"nodeKind":    node.Kind,
		"line":        line,
		"lineKey":     key,
		"isError":     runErr != nil,
	})
}

// ============================================================================
// Region runner sub-dispatch.
// ============================================================================

// execNodeAsRegionViaFramework dispatch RegionRunner 节点 (Loop / Subgraph / CollapsedNode).
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
	r.applyExecDataEdges(node, rn, execData, dataWire)

	result := nodepkg.RunNodeAsRegion(ctx, rn, dataWire, config, execData, r.bundle, node.LogEnabled, body)
	return r.routeResult(node, tok, result)
}

// dispatchInRegion 统一 dispatch entry for nodes inside region body. 这是 execNode 主入口.
// runRegionBody 调它派发 child 节点 — 自动 route 到 region (RunNodeAsRegion) 或 normal (RunNode).
//
// per-exec-tick snapshot 只在这里抓 (单一抓点) — runner.go::Run / nodes.go::runSubFlow /
// runRegionBody 都不重复抓. consumers (GetVar.Evaluate 经 framework
// snapshot wrap) 只在节点 data pull 阶段读, 跟 dispatchInRegion → execNode(AsRegion)ViaFramework
// → buildDataWireFor 同周期, 入口抓一次足够.
func (r *ContainerRunner) dispatchInRegion(ctx context.Context, n *container.GraphNode, tok ExecToken) ([]ExecToken, error) {
	rn, ok := nodepkg.Get(n.Kind)
	if !ok {
		return nil, fmt.Errorf("dispatchInRegion: kind %q not registered", n.Kind)
	}
	// per-tick snapshot 走 ctx (tickCtxKey) — per-goroutine/per-token scope, 让共享 bundle
	// 的 listener subRunner / 并发 runner 都安全, 不撞 instance 字段.
	ctx = withTickSnapshot(ctx, CaptureSnapshot(r.rt.Vars()))
	ctx = withEvalCache(ctx, newDispatchEvalCache())
	if rn.RunRegion != nil {
		return r.execNodeAsRegionViaFramework(ctx, n, tok)
	}
	return r.execNodeViaFramework(ctx, n, tok)
}

// checkSentinelLeak detect Break/Continue sentinel 漏到顶层 dispatch loop.
// 正常路径里 Loop.RunRegion 截获 Break/Continue — 漏到顶层主 loop 说明 validator
// 漏报或子图 misplace. 返非空 leakCode + 包装 err 让主 loop emit
// container:node-validation 高亮失败节点; 不属 sentinel 返空 + 原 err.
// (Throw 不再视为 leak — 它是 Coded error, 由 routeResult 失败路由处理.)
func (r *ContainerRunner) checkSentinelLeak(node *container.GraphNode, err error) (leakCode string, wrapped error) {
	switch {
	case control.IsBreakRequested(err):
		leakCode = "BREAK_OUTSIDE_LOOP"
	case control.IsContinueRequested(err):
		leakCode = "CONTINUE_OUTSIDE_LOOP"
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
//   - SubgraphOutput: body 终点 — 返回到达的出口 decl, 调用方据此 fire 对应动态出口.
//     队列跑干没到任何 marker → 返 nil.
//
// 节点 Run 返 error sentinel (errBreakRequested / errContinueRequested / errThrow) 透传给 body caller.
// 调用方 RunRegion 用 errors.Is 截获.
func (r *ContainerRunner) runRegionBody(ctx context.Context, seeds []ExecToken) (*container.SubgraphOutputDecl, error) {
	queue := append([]ExecToken{}, seeds...)
	for len(queue) > 0 {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		tok := queue[0]
		queue = queue[1:]
		// entry/output 是 virtual marker (不在 nodesByID), 走 metadata 路由.
		if r.currentSG != nil {
			if tok.NodeID == r.currentSG.EntryNodeID {
				queue = append(queue, r.edges.next(tok.NodeID+".Done", tok.LoopStack)...)
				continue
			}
			if decl, isOutput := r.currentSG.OutputDeclsByID[tok.NodeID]; isOutput {
				return decl, nil
			}
		}
		n, ok := r.nodesByID[tok.NodeID]
		if !ok {
			return nil, fmt.Errorf("runRegionBody: unknown node %q", tok.NodeID)
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
			return nil, err
		}
		queue = append(queue, out...)
	}
	return nil, nil
}

// makeBodyFor build region body callback per node kind: Loop / Subgraph / CollapsedNode.
//
// ctx 来自 execNodeAsRegionViaFramework — 携带 tickCtxKey, Subgraph eager pullDataPin
// 链下去 bundle.Snapshot 能拿 frozen Vars. Loop body 不 pullDataPin 不需要外部 ctx.
func (r *ContainerRunner) makeBodyFor(ctx context.Context, node *container.GraphNode, tok ExecToken) (func(nodepkg.Ctx) (string, error), error) {
	switch node.Kind {
	case "Loop", "ForEach":
		// ForEach body 与 Loop 完全同构 (seed node.ID+".Body") — 共用 builder.
		return r.makeBodyForLoop(node, tok), nil
	case "Subgraph", "CollapsedNode":
		return r.makeBodyForSubgraph(ctx, node, tok)
	}
	return nil, fmt.Errorf("makeBodyFor: no body builder for kind %q (region runner not yet supported)", node.Kind)
}

// makeBodyForLoop body callback 每次调跑一轮 Loop/ForEach body (从 node.Body 出口下游 seed 到 queue 空).
// errBreakRequested / errContinueRequested sentinel 透传; Loop.RunRegion 截获.
func (r *ContainerRunner) makeBodyForLoop(node *container.GraphNode, tok ExecToken) func(nodepkg.Ctx) (string, error) {
	parentLoopStack := tok.LoopStack
	return func(c nodepkg.Ctx) (string, error) {
		seeds := r.edges.next(node.ID+".Body", parentLoopStack)
		// Loop body 单轮迭代无出口语义 — 即使 body 内命中子图出口 marker 也只结束本轮.
		_, err := r.runRegionBody(c.Context(), seeds)
		return "", err
	}
}

// makeBodyForSubgraph body 调一次 — 解析 SubgraphID + 组 params, 实跑走共享核心
// runSubgraphCall (push frame + 切表 + 跑 body + 出口裁决), 出口以 decl ID fire
// (父图边 pin = decl ID).
//
// SubgraphID 从 node.Config["SubgraphID"] (PascalCase, 跟 Spec.Inputs.SubgraphID 对齐) 取.
func (r *ContainerRunner) makeBodyForSubgraph(ctx context.Context, node *container.GraphNode, tok ExecToken) (func(nodepkg.Ctx) (string, error), error) {
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
	//      data-in pin 或 literal pin 推 dynamic param. pulled nil 不覆盖 static literal.
	//   3. p.Default fallback if pull 出 nil.
	params := map[string]any{}
	maps.Copy(params, container.PinMap(node, "Params"))
	for _, p := range sg.InputParams {
		v, perr := r.pullDataPin(ctx, node.ID, p.Name)
		if perr != nil {
			return nil, fmt.Errorf("Subgraph %s: pull input %q: %w", node.ID, p.Name, perr)
		}
		if v == nil && p.Default != nil {
			v = toExprValue(p.Default)
		}
		if v != nil {
			params[p.Name] = v
		}
	}
	return func(c nodepkg.Ctx) (string, error) {
		decl, err := r.runSubgraphCall(c.Context(), sg, node.ID, params, parentLoopStack)
		if err != nil {
			return "", err
		}
		return decl.ID, nil
	}, nil
}
