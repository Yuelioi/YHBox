package runtime

import (
	nodepkg "yotta/internal/node"
	"yotta/internal/services/container"
)

// passthroughDisabled routes a disabled node's token through a kind-specific exit pin.
//
// Mapping:
//
//	Loop / Race / Parallel  → .done (skip body — 不是 first exec out, 语义专选)
//	Switch                  → .Default (无配置 case 走 default — 不是 Case1)
//	If                      → .True (默认走 true 分支)
//	Subgraph / CollapsedNode → OutputPins[0].ID (decl ID 动态出口)
//	Linear nodes (Sleep, KeyPress, SetVar, etc.) → 走 nodepkg.Spec 第一个 Type=Exec 出口
//	Throw / Stop / terminals → noop (return nil; runner naturally terminates this token's path)
//	Start / MouseCalibration / EventTick → validator should have errored (INVALID_DISABLED_TERMINAL).
//	Win32WindowTarget — 普通 exec 节点: 未禁用走 execNodeViaFramework→Run; 禁用走 default→firstExecOutPin("Done").
func (r *ContainerRunner) passthroughDisabled(node *container.GraphNode, tok ExecToken) ([]ExecToken, error) {
	switch node.Kind {
	case "Loop", "Race", "Parallel":
		return tryExits(r, node, tok, "done"), nil
	case "Switch":
		return tryExits(r, node, tok, "default"), nil
	case "If":
		return tryExits(r, node, tok, "True"), nil
	case "Subgraph", "CollapsedNode":
		// 出口 = callee OutputPins decl ID (dynamic outputs); 禁用直通走第一个 decl.
		sg, err := ResolveSubgraphCall(r.rt.Subgraphs, node)
		if err != nil || sg == nil || len(sg.OutputPins) == 0 {
			return nil, nil // No exit available — silent terminate
		}
		return r.edges.next(node.ID+"."+sg.OutputPins[0].ID, tok.LoopStack), nil
	case "Throw", "Stop":
		// Terminal — passthrough = noop, runner ends this path naturally.
		return nil, nil
	case "Start", "MouseCalibration", "EventTick":
		// Container-level — validator should have caught this. Defensive: noop.
		return nil, nil
	default:
		// Linear nodes: 走 Spec, 拿第一个 Type=Exec 出口名.
		exit := firstExecOutPin(node.Kind)
		if exit == "" {
			return nil, nil
		}
		return tryExits(r, node, tok, exit), nil
	}
}

// firstExecOutPin 返 nodepkg.Get(kind).Spec.Outputs 中 Type=="Exec" 的第一个出口名.
// kind 未注册 / 无 exec 出口 → 空字符串 (caller 视为无出口, 路径自然终止).
func firstExecOutPin(kind string) string {
	rn, ok := nodepkg.Get(kind)
	if !ok {
		return ""
	}
	for _, o := range rn.Spec.Outputs {
		if o.Type == nodepkg.TypeExec {
			return o.Name
		}
	}
	return ""
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
