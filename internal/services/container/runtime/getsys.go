package runtime

import (
	"fmt"

	"yhbox/internal/services/container"
	"yhbox/internal/services/expr"
)

// SysPathSchema (v4) is the source of truth for valid GetSys.path values.
// Maps path → frontend pin type (string form, matches PinType). Frontend Inspector
// reads this to render the path dropdown + infer data-out pin type. Validator
// uses this to flag unknown paths (CodeGetSysUnknownPath).
//
// Must mirror resolveSysPath in runtime_context.go. When you add a new sys field,
// update BOTH files (resolveSysPath for runtime, SysPathSchema for editor / validator).
var SysPathSchema = map[string]string{
	"runId":                    "number",
	"iter":                     "number",
	"winnerIdx":                "number",
	"lastTemplate.found":       "bool",
	"lastTemplate.point":       "point",
	"lastTemplate.point.x":     "number",
	"lastTemplate.point.y":     "number",
	"lastTemplate.region":      "any", // [4]float64 ratio — no array type yet
	"lastColor.count":          "number",
	"lastColor.cx":             "number",
	"lastColor.cy":             "number",
	"lastColor.center":         "point",
	"lastColor.center.x":       "number",
	"lastColor.center.y":       "number",
	"lastStopwatch.elapsedMs":  "number",
	"lastTry.errorMsg":         "string",
	"lastDetect.pixelCount":    "number",
	"lastDetect.pixelRatio":    "number",
	"lastROIScan.clusterCount": "number",
	"lastROIScan.clusters":     "any",
	"lastScreenshot.path":      "string",
}

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
func (r *ContainerRunner) evalGetSys(n *container.GraphNode) (expr.Value, error) {
	path := configString(n, "path")
	if path == "" {
		return nil, fmt.Errorf("GetSys %s: missing path", n.ID)
	}
	if _, ok := SysPathSchema[path]; !ok {
		return nil, fmt.Errorf("GetSys %s: unknown sys path %q", n.ID, path)
	}
	if r.currentTick == nil {
		return nil, fmt.Errorf("GetSys %s: no active tick snapshot", n.ID)
	}
	return resolveSysPath(r.currentTick.Sys, path)
}
