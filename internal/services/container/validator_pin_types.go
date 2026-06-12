package container

// PinTypeCompat 单一源 — runtime + validator + 任何 cross-cutting 调用方都 import 这一个.
//
// !! Frontend `pinTypeCompat` in nodeRegistry/index.ts 仍是 TS 平行 impl, TestRegistryParity 抓 drift.
//
// 返 (allow=能否连接, warn=是否需 coercion warning).
//   - from==to / any↔any: allow + no warn
//   - number→bool/string, bool→number/string: allow + warn (implicit coercion)
//   - 其他: reject
func PinTypeCompat(from, to string) (allow, warn bool) {
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

// validateDataPinTypes walks main + subgraph data edges, looks up source/target
// pin types via {dataIn,dataOut}PinTypeForKind, and emits:
//   - PIN_TYPE_MISMATCH (error) for incompatible connections
//   - PIN_TYPE_COERCION_WARNING (warning) for implicit conversions
//
// v4 (C1): edges 没 Kind 字段了 — "data 边" 派生为 "fromPin 是 src.kind 的 data-out".
// 非 data-out 的 from-pin (exec-out) 直接跳过本规则 (data-pin 类型校验不适用 exec 边).
//
// Source-type resolution:
//   - GetVar: Container.Vars[varName].Type (declared variable type)
//   - other kinds: dataOutPinTypeForKind(kind, pinName) — static schema lookup
//
// Target-type: dataInPinTypeForNode(node, pinName) — node-aware variant so Expr's
// dynamic inputs[] data-in pins get validated (kind-only lookup would return "").
//
// Empty type ("") on either side → skip (unknown schema; not an error in itself).
func validateDataPinTypes(c *Container, sgs []Subgraph) []ValidationError {
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
			srcID, srcPin := splitRef(e.From)
			tgtID, tgtPin := splitRef(e.To)
			src := nodesByID[srcID]
			tgt := nodesByID[tgtID]
			if src == nil || tgt == nil {
				continue // DANGLING_EDGE reported elsewhere
			}
			// v4 (C1): 只处理 data 边 — 派生自 "fromPin 在 src 的 data-out 集合里".
			// GetVar 等动态 data-out 节点 spec 里登记 "any", DataOutType 也返非空.
			if !IsDataOutPin(src.Kind, srcPin) {
				continue
			}
			// Resolve source type
			var srcType string
			if src.Kind == "GetVar" {
				varName := PinString(src, "VarName")
				srcType = varsTypes[varName]
			} else {
				srcType = dataOutPinTypeForKind(src.Kind, srcPin)
			}
			// Node-aware: lets Expr's config.inputs[] dynamic data-in pins resolve.
			tgtType := dataInPinTypeForNode(tgt, tgtPin)
			if srcType == "" || tgtType == "" {
				continue // schema unknown — skip type check
			}
			allow, warn := PinTypeCompat(srcType, tgtType)
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
	for i := range sgs {
		sg := &sgs[i]
		all = append(all, walk(sg.Graph, []string{"subgraph", sg.ID})...)
	}
	return all
}
