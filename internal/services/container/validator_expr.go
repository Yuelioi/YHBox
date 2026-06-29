package container

import (
	"fmt"

	"yotta/internal/services/expr"
)

// validateExprNodes scans main + subgraph graphs for Expr kind nodes, validates:
//   - EXPR_DUPLICATE_INPUT — inputs[] has same name twice
//   - EXPR_PARSE_ERROR — expr string fails to parse
//   - EXPR_UNKNOWN_INPUT — expr references identifier not declared in inputs[]
//   - EXPR_UNKNOWN_VAR — $name 引用未在容器 Vars 声明的变量
//   - EXPR_UNKNOWN_FUNCTION / EXPR_FN_ARITY — call 名/参数个数对照 expr.Builtins()
//
// EXPR_TYPE_MISMATCH (explicit outType vs static inference) is deferred — implementing
// proper type inference is non-trivial and depends on pure-func type schemas.
func validateExprNodes(c *Container, sgs []Subgraph) []ValidationError {
	if c == nil {
		return nil
	}
	declaredVars := make(map[string]bool, len(c.Vars))
	for _, v := range c.Vars {
		declaredVars[v.Name] = true
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

			// EXPR_UNKNOWN_VAR — $name 对照容器 Vars 声明 (拼错不再静默走 nil, v4 教训).
			varReported := map[string]bool{}
			for _, ref := range expr.VarRefs(ast) {
				if declaredVars[ref] || varReported[ref] {
					continue
				}
				varReported[ref] = true
				errs = append(errs, ValidationError{
					Severity: SeverityError, Code: CodeExprUnknownVar,
					GraphPath: path, NodeID: n.ID,
					Params: map[string]any{"name": ref},
				})
			}

			// EXPR_UNKNOWN_FUNCTION / EXPR_FN_ARITY — call 名与参数个数对照 builtin 表.
			// 没这两条时函数 typo / 参数缺漏只能等运行时 evalCall 才炸.
			builtins := expr.Builtins()
			fnReported := map[string]bool{}
			for _, call := range expr.CallRefs(ast) {
				if fnReported[call.Name] {
					continue
				}
				b, ok := builtins[call.Name]
				if !ok {
					fnReported[call.Name] = true
					errs = append(errs, ValidationError{
						Severity: SeverityError, Code: CodeExprUnknownFunction,
						GraphPath: path, NodeID: n.ID,
						Params: map[string]any{"name": call.Name},
					})
					continue
				}
				if call.ArgN < b.MinArgs || call.ArgN > b.MaxArgs {
					fnReported[call.Name] = true
					want := fmt.Sprintf("%d", b.MinArgs)
					if b.MaxArgs != b.MinArgs {
						want = fmt.Sprintf("%d-%d", b.MinArgs, b.MaxArgs)
					}
					errs = append(errs, ValidationError{
						Severity: SeverityError, Code: CodeExprFnArity,
						GraphPath: path, NodeID: n.ID,
						Params: map[string]any{"name": call.Name, "want": want, "got": call.ArgN},
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
