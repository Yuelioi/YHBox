package container

// pinTypeCompat mirrors runtime.PinTypeCompat (spec §2.3). Kept here to avoid
// the container → container/runtime import cycle (runtime imports container).
// Both files MUST stay in sync — when the matrix changes, update both.
func pinTypeCompat(from, to string) (allow, warn bool) {
	if from == to || from == "any" || to == "any" {
		return true, false
	}
	switch from {
	case "number":
		switch to {
		case "bool", "string":
			return true, true
		}
	case "bool":
		switch to {
		case "number", "string":
			return true, true
		}
	}
	return false, false
}

// validateDataPinTypes walks main + subgraph data edges (Kind="data"), looks up
// source/target pin types via {dataIn,dataOut}PinTypeForKind, and emits:
//   - PIN_TYPE_MISMATCH (error) for incompatible connections
//   - PIN_TYPE_COERCION_WARNING (warning) for implicit conversions
//
// Source-type resolution:
//   - GetVar: Container.Vars[varName].Type (declared variable type)
//   - other kinds: dataOutPinTypeForKind(kind, pinName) — static schema lookup
//
// Target-type: dataInPinTypeForNode(node, pinName) — node-aware variant so Expr's
// dynamic inputs[] data-in pins get validated (kind-only lookup would return "").
//
// Empty type ("") on either side → skip (unknown schema; not an error in itself).
func validateDataPinTypes(c *Container) []ValidationError {
	if c == nil {
		return nil
	}
	varsTypes := make(map[string]string, len(c.Vars))
	for _, v := range c.Vars {
		varsTypes[v.Name] = v.Type
	}

	walk := func(g Graph, path []string) []ValidationError {
		var errs []ValidationError
		nodesByID := make(map[string]*GraphNode, len(g.Nodes))
		for i := range g.Nodes {
			nodesByID[g.Nodes[i].ID] = &g.Nodes[i]
		}
		for _, e := range g.Edges {
			if e.Kind != "data" {
				continue
			}
			srcID, srcPin := splitRef(e.From)
			tgtID, tgtPin := splitRef(e.To)
			src := nodesByID[srcID]
			tgt := nodesByID[tgtID]
			if src == nil || tgt == nil {
				continue // DANGLING_EDGE reported elsewhere
			}
			// Resolve source type
			var srcType string
			if src.Kind == "GetVar" {
				varName, _ := src.Config["varName"].(string)
				srcType = varsTypes[varName]
			} else {
				srcType = dataOutPinTypeForKind(src.Kind, srcPin)
			}
			// Node-aware: lets Expr's config.inputs[] dynamic data-in pins resolve.
			tgtType := dataInPinTypeForNode(tgt, tgtPin)
			if srcType == "" || tgtType == "" {
				continue // schema unknown — skip type check
			}
			allow, warn := pinTypeCompat(srcType, tgtType)
			switch {
			case !allow:
				errs = append(errs, ValidationError{
					Severity:  SeverityError,
					Code:      CodePinTypeMismatch,
					GraphPath: path,
					NodeID:    tgtID,
					Params: map[string]any{
						"from":   srcType,
						"to":     tgtType,
						"edge":   e.From + "→" + e.To,
						"srcPin": srcPin,
						"tgtPin": tgtPin,
					},
				})
			case warn:
				errs = append(errs, ValidationError{
					Severity:  SeverityWarning,
					Code:      CodePinTypeCoercionWarning,
					GraphPath: path,
					NodeID:    tgtID,
					Params: map[string]any{
						"from": srcType,
						"to":   tgtType,
					},
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
