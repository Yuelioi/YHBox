package container

// validateCollapsedReferences: two related checks.
//
//  1. COLLAPSED_REFERENCED_BY_SUBGRAPH_CALL: a regular `Subgraph` call node refers to a
//     subgraph that has isAnonymous=true (which is reserved for `CollapsedNode` use).
//     isAnonymous subgraphs MUST only be invoked through a `CollapsedNode` kind in the
//     same graph that owns the CollapsedNode.
//
//  2. COLLAPSED_PIN_BROKEN: a `CollapsedNode` references a subgraphId that doesn't exist,
//     OR the referenced subgraph is NOT isAnonymous (CollapsedNode must wrap an anonymous
//     one — if user toggled isAnonymous=false they should upgrade to Subgraph kind).
//
// Note: COLLAPSED_PIN_BROKEN 还覆盖 "内部 SubgraphInput/Output marker 跟外部 pin 对不上"
// 这种更深层结构错 — 这里 defer 给 validateInvalidPins 走 dangling edge 兜底.
func validateCollapsedReferences(c *Container, sgs []Subgraph) []ValidationError {
	if c == nil {
		return nil
	}
	sgIndex := make(map[string]*Subgraph, len(sgs))
	for i := range sgs {
		sgIndex[sgs[i].ID] = &sgs[i]
	}

	walk := func(g Graph, path []string) []ValidationError {
		var errs []ValidationError
		for _, n := range g.Nodes {
			sgID := PinString(&n, "SubgraphID")

			switch n.Kind {
			case "Subgraph":
				if sgID == "" {
					continue // MISSING_SUBGRAPH reported elsewhere
				}
				sg := sgIndex[sgID]
				if sg == nil {
					continue
				}
				if sg.IsAnonymous {
					errs = append(errs, ValidationError{
						Severity: SeverityError, Code: CodeCollapsedReferencedBySubgraphCall,
						GraphPath: path, NodeID: n.ID,
						Params: map[string]any{"subgraphId": sgID},
					})
				}

			case "CollapsedNode":
				if sgID == "" {
					errs = append(errs, ValidationError{
						Severity: SeverityError, Code: CodeCollapsedPinBroken,
						GraphPath: path, NodeID: n.ID,
						Params: map[string]any{"reason": "missing subgraphId"},
					})
					continue
				}
				sg := sgIndex[sgID]
				if sg == nil {
					errs = append(errs, ValidationError{
						Severity: SeverityError, Code: CodeCollapsedPinBroken,
						GraphPath: path, NodeID: n.ID,
						Params: map[string]any{"reason": "subgraph not found", "subgraphId": sgID},
					})
					continue
				}
				if !sg.IsAnonymous {
					errs = append(errs, ValidationError{
						Severity: SeverityError, Code: CodeCollapsedPinBroken,
						GraphPath: path, NodeID: n.ID,
						Params: map[string]any{"reason": "subgraph is not anonymous — should upgrade kind to Subgraph", "subgraphId": sgID},
					})
				}
			}
		}
		return errs
	}

	var all []ValidationError
	all = append(all, walk(c.Graph, []string{"main"})...)
	for i := range sgs {
		sg := &sgs[i]
		all = append(all, walk(sg.Graph, []string{"subgraph", sg.ID})...)
	}
	return all
}
