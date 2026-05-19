package container

// validateDataGraphAcyclic (v4, spec §10 + §11) checks that data edges form a DAG.
// Data flow uses pull semantics (consumer pulls source), so cycles would mean a
// node's input depends transitively on its own output — infinite recursion at runtime.
//
// Algorithm: standard DFS with WHITE/GRAY/BLACK coloring on per-graph adjacency
// derived from data edges. Walks main + every subgraph independently (cycles can't
// cross subgraph boundaries — Subgraph call nodes are exec, not data). Returns at
// most one DATA_GRAPH_CYCLE per graph (first cycle found).
//
// v4 (C1): GraphEdge.Kind 已删 — "data 边" 派生自 "fromPin 在 src.kind 的 data-out 集合里".
func validateDataGraphAcyclic(c *Container) []ValidationError {
	if c == nil {
		return nil
	}
	check := func(g Graph, path []string) *ValidationError {
		// Build src→[targets] adjacency over data edges only.
		adj := map[string][]string{}
		nodes := map[string]bool{}
		kindByID := map[string]string{}
		for _, n := range g.Nodes {
			nodes[n.ID] = true
			kindByID[n.ID] = n.Kind
		}
		for _, e := range g.Edges {
			srcID, srcPin := splitRef(e.From)
			tgtID, _ := splitRef(e.To)
			if !nodes[srcID] || !nodes[tgtID] {
				continue // DANGLING_EDGE reported elsewhere
			}
			// 派生 data 边: src.kind 在 fromPin 上有 data-out 才算.
			if !IsDataOutPin(kindByID[srcID], srcPin) {
				continue
			}
			adj[srcID] = append(adj[srcID], tgtID)
		}
		const (
			white = 0
			gray  = 1
			black = 2
		)
		color := map[string]int{}
		var cycle []string
		var visit func(node string) bool
		visit = func(node string) bool {
			color[node] = gray
			for _, next := range adj[node] {
				switch color[next] {
				case gray:
					cycle = append(cycle, node, next)
					return true
				case white:
					if visit(next) {
						cycle = append([]string{node}, cycle...)
						return true
					}
				}
			}
			color[node] = black
			return false
		}
		for _, n := range g.Nodes {
			if color[n.ID] != white {
				continue
			}
			if visit(n.ID) {
				return &ValidationError{
					Severity:  SeverityError,
					Code:      CodeDataGraphCycle,
					GraphPath: path,
					NodeID:    cycle[0],
					Params:    map[string]any{"cycle": cycle},
				}
			}
		}
		return nil
	}

	var errs []ValidationError
	if e := check(c.Graph, []string{"main"}); e != nil {
		errs = append(errs, *e)
	}
	for i := range c.Subgraphs {
		sg := &c.Subgraphs[i]
		if e := check(sg.Graph, []string{"subgraph", sg.ID}); e != nil {
			errs = append(errs, *e)
		}
	}
	return errs
}
