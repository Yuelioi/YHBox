package container

import "yhbox/internal/services/container/sys"

// E1: sysPathSchemaCopy duplicate map removed. Single source of truth now lives
// in the leaf container/sys package, imported by both runtime and validator.

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
			p := PinString(&n, "Path")
			if p == "" {
				errs = append(errs, ValidationError{
					Severity: SeverityError, Code: CodeGetSysUnknownPath,
					GraphPath: path, NodeID: n.ID,
					Params: map[string]any{"path": ""},
				})
				continue
			}
			// Live wildcard paths (e.g. varLastChange.<name>) — 接受任何 suffix.
			if sys.IsLiveWildcardPath(p) {
				continue
			}
			if _, ok := sys.PathSchema[p]; !ok {
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
