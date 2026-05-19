package runtime

import (
	"fmt"

	"yhbox/internal/services/container"
	"yhbox/internal/services/expr"
)

// evalGetVar reads a variable based on scope.
//
// Pure data node: no ExecToken, no edges.next; called from pullDataPin / evalDataSource.
// Reads from the per-exec-tick snapshot (r.currentTick) for "global"/"auto" scope —
// guarantees same-tick consistency.
//
// scope (default "auto"):
//   - "local"  → current frame.LocalVars only (no fallback). Unset → nil.
//   - "global" → snapshot of rt.vars only (skips frame chain). Unset → nil.
//   - "auto"   → frame chain → snapshot of rt.vars (default; 跟 Container.Vars 面板默认一致).
//
// 2026-05-19 默认从 "local" 改成 "auto" — local 没 UI 入口, 默认指向它会让
// Container.Vars 面板声明的 var 在 GetVar 读不到 (反直觉).
func (r *ContainerRunner) evalGetVar(n *container.GraphNode) (expr.Value, error) {
	name := configString(n, "varName")
	if name == "" {
		return nil, fmt.Errorf("GetVar %s: missing varName", n.ID)
	}
	scope := configString(n, "scope")
	if scope == "" {
		scope = "auto"
	}

	switch scope {
	case "local":
		if v, ok := r.state.GetLocalVarHere(name); ok {
			return v, nil
		}
		return nil, nil
	case "global":
		if r.currentTick == nil {
			return nil, fmt.Errorf("GetVar %s: no active tick snapshot", n.ID)
		}
		if v, ok := r.currentTick.Vars[name]; ok {
			return v, nil
		}
		return nil, nil
	case "auto":
		if v, ok := r.state.GetLocalVarChain(name); ok {
			return v, nil
		}
		if r.currentTick == nil {
			return nil, fmt.Errorf("GetVar %s: no active tick snapshot", n.ID)
		}
		if v, ok := r.currentTick.Vars[name]; ok {
			return v, nil
		}
		return nil, nil
	default:
		return nil, fmt.Errorf("GetVar %s: unknown scope %q", n.ID, scope)
	}
}
