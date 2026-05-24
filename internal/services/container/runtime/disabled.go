package runtime

import "yhbox/internal/services/container"

// passthroughDisabled routes a disabled node's token through a kind-specific exit pin.
// Spec: editor-v2-quick-actions-design.md §6.2 — Phase 5.6 atomic 后 pin name 改新 spec.
//
// Mapping:
//
//	Loop / Race / Parallel  → .done (skip body)
//	Switch                  → .Default (无配置 case 走 default)
//	If                      → .True (默认走 true 分支)
//	Try                     → .out (正常完成出口)
//	Subgraph / CollapsedNode → .Done (新框架 Subgraph/CollapsedNode 单 Done 出口)
//	Linear nodes (Sleep, KeyPress, SetVar, etc.) → .out / .Done — caller knows下游;
//	                          老节点 lowercase out (SetVar/IncVar/Start), 新框架节点 PascalCase Done (KeyPress/Sleep).
//	                          先试 .out, 没下游再试 .Done.
//	Throw / Stop / terminals → noop (return nil; runner naturally terminates this token's path)
//	Start / WindowTarget / MouseCalibration / OnEvent → validator should have errored (INVALID_DISABLED_TERMINAL).
func (r *ContainerRunner) passthroughDisabled(node *container.GraphNode, tok ExecToken) ([]ExecToken, error) {
	switch node.Kind {
	case "Loop", "Race", "Parallel":
		return tryExits(r, node, tok, "done"), nil
	case "Switch":
		return tryExits(r, node, tok, "Default"), nil
	case "If":
		return tryExits(r, node, tok, "True"), nil
	case "Try":
		return tryExits(r, node, tok, "out"), nil
	case "Subgraph", "CollapsedNode":
		// 新框架 Subgraph/CollapsedNode 都是单 Done 出口 (固定). 老 Subgraph 用 OutputPins[0].
		tokens := r.edges.next(node.ID+".Done", tok.LoopStack)
		if len(tokens) > 0 {
			return tokens, nil
		}
		sg, err := ResolveSubgraphCall(r.rt.Container, node)
		if err != nil || sg == nil || len(sg.OutputPins) == 0 {
			return nil, nil // No exit available — silent terminate
		}
		return r.edges.next(node.ID+"."+sg.OutputPins[0].ID, tok.LoopStack), nil
	case "Throw", "Stop":
		// Terminal — passthrough = noop, runner ends this path naturally.
		return nil, nil
	case "Start", "WindowTarget", "MouseCalibration", "OnEvent":
		// Container-level — validator should have caught this. Defensive: noop.
		return nil, nil
	default:
		// Linear nodes: 老 .out (SetVar/Start/Log/Toast), 新 .Done (Sleep/KeyPress/CheckTemplate.Found etc.).
		// 先 .out 后 .Done — atomic 转换期同时认.
		return tryExits(r, node, tok, "out", "Done"), nil
	}
}

// tryExits 按 fallback 顺序查 exit edge, 第一个有下游的 return.
func tryExits(r *ContainerRunner, node *container.GraphNode, tok ExecToken, exits ...string) []ExecToken {
	for _, e := range exits {
		toks := r.edges.next(node.ID+"."+e, tok.LoopStack)
		if len(toks) > 0 {
			return toks
		}
	}
	return nil
}
