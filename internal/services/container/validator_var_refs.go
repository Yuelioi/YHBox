package container

// validateVarRefs checks that GetVar/SetVar/IncVar nodes reference declared Container.Vars,
// EXCEPT for scope=local references which are frame-private (runtime check only).
//
// Spec: editor-v2-vars-panel-design.md §4.3.2 (GPT review #3 — delete var leaves
// invalid references instead of cascading deletes).
//
// Rule:
//
//	kind ∈ {GetVar, SetVar, IncVar}
//	AND config.scope ∈ {auto, global, undefined}     ← undefined treated as auto (backend default)
//	AND config.varName ∉ Container.Vars[].Name
//	AND (subgraph 内时) config.varName ∉ sg.RequiredGlobals[].Name → INVALID_VAR_REF
//
// scope=local is skipped here entirely (frame-scoped, declared implicitly via SetVar).
//
// B11: subgraph 内 var ref 若在 sg.RequiredGlobals 白名单 — 视作 caller 会 provide, 不报错.
// (Library import / 跨容器拖时, caller 容器可能缺 var; RequiredGlobals 自我声明依赖.)
func validateVarRefs(c *Container) []ValidationError {
	if c == nil {
		return nil
	}
	declared := make(map[string]bool, len(c.Vars))
	for _, v := range c.Vars {
		declared[v.Name] = true
	}
	varKinds := map[string]bool{"GetVar": true, "SetVar": true, "IncVar": true}

	var errs []ValidationError
	check := func(g Graph, path []string, extraDeclared map[string]bool) {
		for _, n := range g.Nodes {
			if !varKinds[n.Kind] {
				continue
			}
			scope, _ := n.Config["Scope"].(string)
			if scope == "" {
				scope = "auto"
			}
			if scope == "local" {
				continue // frame-private, skip
			}
			varName, _ := n.Config["VarName"].(string)
			if varName == "" {
				continue // missing-varName reported by other validator path
			}
			if declared[varName] {
				continue
			}
			if extraDeclared != nil && extraDeclared[varName] {
				continue // B11: subgraph RequiredGlobals 白名单
			}
			errs = append(errs, ValidationError{
				Severity:  SeverityError,
				Code:      CodeInvalidVarRef,
				GraphPath: append([]string(nil), path...),
				NodeID:    n.ID,
				Params:    map[string]any{"varName": varName, "scope": scope},
			})
		}
	}
	check(c.Graph, []string{"main"}, nil)
	for _, sg := range c.Subgraphs {
		sgWhitelist := map[string]bool{}
		for _, rg := range sg.RequiredGlobals {
			sgWhitelist[rg.Name] = true
		}
		check(sg.Graph, []string{"subgraph", sg.ID}, sgWhitelist)
	}
	return errs
}
