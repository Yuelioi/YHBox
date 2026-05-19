package runtime

import (
	"fmt"
	"strings"

	"yhbox/internal/services/container"
	"yhbox/internal/services/container/sys"
	"yhbox/internal/services/expr"
)

// SysPathSchema is now sourced from the leaf container/sys package (E1: dedup).
// Re-exported here for backward compat with any external callers; new code should
// import "yhbox/internal/services/container/sys" directly.
var SysPathSchema = sys.PathSchema

// evalGetParam (v4) returns the value of a subgraph input parameter from the current frame's
// LocalParams. Set when the parent's Subgraph call node was executed (execSubgraph pulls
// data-in pins and writes them to frame.LocalParams before swapping dispatch views).
//
// Pure data — no exec / no edges. Unset paramName → nil (validator catches at design time).
func (r *ContainerRunner) evalGetParam(n *container.GraphNode) (expr.Value, error) {
	name := configString(n, "paramName")
	if name == "" {
		return nil, fmt.Errorf("GetParam %s: missing paramName", n.ID)
	}
	if r.state.CurrentFrame == nil {
		return nil, fmt.Errorf("GetParam %s: no active frame", n.ID)
	}
	if v, ok := r.state.CurrentFrame.LocalParams[name]; ok {
		return v, nil
	}
	return nil, nil
}

// evalGetSys (v4) returns the value of a single sys path from the per-tick snapshot.
// Pure data — no exec / no edges. Reads tick snapshot (NOT live rt.sys) for determinism.
//
// 例外: now_ms + varLastChange.<name> 总是 live 读 (跟 snapshot 解耦, Fishing v2
// watchdog 需要时间戳比较 → snapshot 化反而出错).
func (r *ContainerRunner) evalGetSys(n *container.GraphNode) (expr.Value, error) {
	path := configString(n, "path")
	if path == "" {
		return nil, fmt.Errorf("GetSys %s: missing path", n.ID)
	}
	// Live wildcard: varLastChange.<name> — 直接读 rt.varTimestamps live (绕过 snapshot + schema).
	if name, ok := strings.CutPrefix(path, "varLastChange."); ok {
		return float64(r.rt.VarLastChange(name)), nil
	}
	// Live path: now_ms — 在 schema 里, 但读 live time.Now() 不读 snapshot.
	if path == "now_ms" {
		return float64(nowMillis()), nil
	}
	if _, ok := SysPathSchema[path]; !ok {
		return nil, fmt.Errorf("GetSys %s: unknown sys path %q", n.ID, path)
	}
	if r.currentTick == nil {
		return nil, fmt.Errorf("GetSys %s: no active tick snapshot", n.ID)
	}
	return resolveSysPath(r.currentTick.Sys, path)
}
