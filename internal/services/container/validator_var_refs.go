package container

import "fmt"

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
//	AND config.varName ∉ Container.Vars[].Name       → INVALID_VAR_REF
//
// scope=local is skipped here entirely (frame-scoped, declared implicitly via SetVar).
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
	check := func(g Graph, path []string) {
		for _, n := range g.Nodes {
			if !varKinds[n.Kind] {
				continue
			}
			scope, _ := n.Config["scope"].(string)
			if scope == "" {
				scope = "auto"
			}
			if scope == "local" {
				continue // frame-private, skip
			}
			varName, _ := n.Config["varName"].(string)
			if varName == "" {
				continue // missing-varName reported by other validator path
			}
			if !declared[varName] {
				errs = append(errs, ValidationError{
					Severity:  SeverityError,
					Code:      CodeInvalidVarRef,
					GraphPath: append([]string(nil), path...),
					NodeID:    n.ID,
					Message:   fmt.Sprintf("节点 %s 引用未声明的容器变量 %q (scope=%s) — 在容器变量面板添加或改 scope=local", n.ID, varName, scope),
					Params:    map[string]any{"varName": varName, "scope": scope},
				})
			}
		}
	}
	check(c.Graph, []string{"main"})
	for _, sg := range c.Subgraphs {
		check(sg.Graph, []string{"subgraph", sg.ID})
	}
	return errs
}
