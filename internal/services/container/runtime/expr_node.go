package runtime

import (
	"fmt"
	"sync"

	"yhbox/internal/services/container"
	"yhbox/internal/services/expr"
)

// exprCache caches parsed AST keyed by expr string.
// Editor-scale (few hundred Expr nodes per container) — no eviction needed.
var exprCache = struct {
	sync.RWMutex
	m map[string]*expr.Node
}{m: map[string]*expr.Node{}}

func parseExprCached(src string) (*expr.Node, error) {
	exprCache.RLock()
	if ast, ok := exprCache.m[src]; ok {
		exprCache.RUnlock()
		return ast, nil
	}
	exprCache.RUnlock()
	ast, err := expr.Parse(src)
	if err != nil {
		return nil, err
	}
	exprCache.Lock()
	exprCache.m[src] = ast
	exprCache.Unlock()
	return ast, nil
}

// evalExpr (v4) pulls all declared data-in pins, builds InputEnv, evaluates the expression.
// Pure data — no ExecToken / no edges.next. Returned from pullDataPin via evalDataSource.
//
// Notes:
//   - InputEnv exposes ONLY bare identifiers from inputs[]; $vars/$sys/$params are intentionally
//     unreachable — want explicit data-flow via GetVar/GetSys/GetParam upstream nodes.
//   - outType is metadata for pin schema / coercion warnings; runtime returns the natural type
//     (validator handles EXPR_TYPE_MISMATCH for explicit outType).
func (r *ContainerRunner) evalExpr(n *container.GraphNode) (expr.Value, error) {
	cfg, _ := container.ParseExprConfig(n)
	if cfg.Expr == "" {
		// Empty expr → nil (no error). Useful for "draft" nodes user hasn't filled.
		return nil, nil
	}
	inputs := make(expr.InputEnv, len(cfg.Inputs))
	for _, in := range cfg.Inputs {
		v, err := r.pullDataPin(n.ID, in.Name)
		if err != nil {
			return nil, fmt.Errorf("Expr %s: pull input %q: %w", n.ID, in.Name, err)
		}
		inputs[in.Name] = v
	}
	ast, err := parseExprCached(cfg.Expr)
	if err != nil {
		return nil, fmt.Errorf("Expr %s: parse: %w", n.ID, err)
	}
	return expr.Eval(ast, inputs)
}
