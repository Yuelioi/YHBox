package container

import (
	"yhbox/internal/services/expr"
)

// validateExprNodes scans main + subgraph graphs for Expr kind nodes, validates:
//   - EXPR_DUPLICATE_INPUT — inputs[] has same name twice
//   - EXPR_PARSE_ERROR — expr string fails to parse
//   - EXPR_UNKNOWN_INPUT — expr references identifier not declared in inputs[]
//
// EXPR_TYPE_MISMATCH (explicit outType vs static inference) is deferred — implementing
// proper type inference is non-trivial and gated on Phase B+ pure-func type schemas.
func validateExprNodes(c *Container) []ValidationError {
	if c == nil {
		return nil
	}
	walk := func(g Graph, path []string) []ValidationError {
		var errs []ValidationError
		for _, n := range g.Nodes {
			if n.Kind != "Expr" {
				continue
			}
			cfg, _ := ParseExprConfig(&n)

			// EXPR_DUPLICATE_INPUT
			seen := make(map[string]bool, len(cfg.Inputs))
			for _, in := range cfg.Inputs {
				if seen[in.Name] {
					errs = append(errs, ValidationError{
						Severity: SeverityError, Code: CodeExprDuplicateInput,
						GraphPath: path, NodeID: n.ID,
						Params: map[string]any{"name": in.Name},
					})
					continue
				}
				seen[in.Name] = true
			}

			// EXPR_PARSE_ERROR
			if cfg.Expr == "" {
				continue // empty draft — no parse / unknown checks
			}
			ast, err := expr.Parse(cfg.Expr)
			if err != nil {
				errs = append(errs, ValidationError{
					Severity: SeverityError, Code: CodeExprParseError,
					GraphPath: path, NodeID: n.ID,
					Params: map[string]any{"error": err.Error()},
				})
				continue
			}

			// EXPR_UNKNOWN_INPUT — walk AST, collect bare identifiers, compare to inputs[].
			reported := map[string]bool{}
			for _, ref := range expr.IdentRefs(ast) {
				if seen[ref] || reported[ref] {
					continue
				}
				reported[ref] = true
				errs = append(errs, ValidationError{
					Severity: SeverityError, Code: CodeExprUnknownInput,
					GraphPath: path, NodeID: n.ID,
					Params: map[string]any{"name": ref},
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
