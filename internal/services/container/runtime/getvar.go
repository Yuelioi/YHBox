package runtime

import (
	"fmt"

	"yhbox/internal/services/container"
	"yhbox/internal/services/expr"
)

// evalGetVar reads a variable based on scope (spec §3.1).
//
// Pure data node: no ExecToken, no edges.next; called from pullDataPin / evalDataSource.
// Reads from the per-exec-tick snapshot (r.currentTick) for "global"/"auto" scope —
// guarantees same-tick consistency (spec §10.3).
//
// scope (default "local"):
//   - "local"  → current frame.LocalVars only (no fallback). Unset → nil.
//   - "global" → snapshot of rt.vars only (skips frame chain). Unset → nil.
//   - "auto"   → frame chain → snapshot of rt.vars (v3-style lookup).
func (r *ContainerRunner) evalGetVar(n *container.GraphNode) (expr.Value, error) {
	name := configString(n, "varName")
	if name == "" {
		return nil, fmt.Errorf("GetVar %s: missing varName", n.ID)
	}
	scope := configString(n, "scope")
	if scope == "" {
		scope = "local"
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
