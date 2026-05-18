package container

// sysPathSchemaCopy mirrors runtime.SysPathSchema (avoids container→runtime cyclic import).
// Sync rule: when runtime.SysPathSchema or resolveSysPath changes, update this map too.
// Both go-vet would catch divergence via behavior tests (runtime test that GetSys returns
// expected types per validator schema).
var sysPathSchemaCopy = map[string]string{
	"runId":                    "number",
	"iter":                     "number",
	"winnerIdx":                "number",
	"lastTemplate.found":       "bool",
	"lastTemplate.point":       "point",
	"lastTemplate.point.x":     "number",
	"lastTemplate.point.y":     "number",
	"lastTemplate.region":      "any",
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

// validateGetSysNodes scans for GetSys kinds with unknown path config (GETSYS_UNKNOWN_PATH).
func validateGetSysNodes(c *Container) []ValidationError {
	if c == nil {
		return nil
	}
	walk := func(g Graph, path []string) []ValidationError {
		var errs []ValidationError
		for _, n := range g.Nodes {
			if n.Kind != "GetSys" {
				continue
			}
			p, _ := n.Config["path"].(string)
			if p == "" {
				errs = append(errs, ValidationError{
					Severity: SeverityError, Code: CodeGetSysUnknownPath,
					GraphPath: path, NodeID: n.ID,
					Params: map[string]any{"path": ""},
				})
				continue
			}
			if _, ok := sysPathSchemaCopy[p]; !ok {
				errs = append(errs, ValidationError{
					Severity: SeverityError, Code: CodeGetSysUnknownPath,
					GraphPath: path, NodeID: n.ID,
					Params: map[string]any{"path": p},
				})
			}
		}
		return errs
	}
	var all []ValidationError
	all = append(all, walk(c.Graph, []string{"main"})...)
	for i := range c.Subgraphs {
		sg := &c.Subgraphs[i]
		all = append(all, walk(sg.Graph, []string{"subgraph", sg.ID})...)
	}
	return all
}
