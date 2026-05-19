package runtime

import "yhbox/internal/services/container"

// passthroughDisabled routes a disabled node's token through a kind-specific exit pin.
// Spec: editor-v2-quick-actions-design.md §6.2.
//
// Mapping (matches spec table):
//
//	Loop / Switch / Race / Parallel → .complete (skip body / aggregate)
//	If                              → .then
//	Try                             → .done
//	Subgraph / CollapsedNode        → first OutputPins exec-out
//	Linear nodes (Sleep, KeyPress, SetVar, etc.) → .out
//	Throw / Stop / terminals        → noop (return nil; runner naturally terminates this token's path)
//	Start / WindowTarget / MouseCalibration / OnEvent → validator should have errored (INVALID_DISABLED_TERMINAL).
func (r *ContainerRunner) passthroughDisabled(node *container.GraphNode, tok ExecToken) ([]ExecToken, error) {
	switch node.Kind {
	case "Loop", "Switch", "Race", "Parallel":
		// Note for Loop: we never enter the loop body when disabled — no LoopStack push to undo.
		return r.edges.next(node.ID+".complete", tok.LoopStack), nil
	case "If":
		return r.edges.next(node.ID+".then", tok.LoopStack), nil
	case "Try":
		return r.edges.next(node.ID+".done", tok.LoopStack), nil
	case "Subgraph", "CollapsedNode":
		// Pick first OutputPins exec-out from backing subgraph.
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
		// Linear nodes (Sleep, KeyPress, SetVar, IncVar, ClickAt, Log, Toast, etc.): send token through .out.
		return r.edges.next(node.ID+".out", tok.LoopStack), nil
	}
}
