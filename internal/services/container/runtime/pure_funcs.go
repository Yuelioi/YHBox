package runtime

import (
	"fmt"
	"math"
	"strings"

	"yhbox/internal/services/container"
	"yhbox/internal/services/expr"
)

// evalPureFunc (v4) dispatches the 22 pure-function node Kinds (spec §6).
// All are pure data: no exec, no side effects, no edges.next. They form the
// arithmetic/logic plumbing between GetVar/GetSys/Expr and SetVar/If/Switch/etc.
//
// Pin schema (data-in / data-out) lives in the frontend pinSpec.ts; here we just
// pullDataPin by name and compute. Type coercion follows expr.AsNumber/AsBool/FormatValue.
func (r *ContainerRunner) evalPureFunc(n *container.GraphNode) (expr.Value, error) {
	switch n.Kind {
	// --- §6.1 arithmetic (binary except Neg) ---
	case "Add":
		return r.binNum(n, func(a, b float64) float64 { return a + b })
	case "Sub":
		return r.binNum(n, func(a, b float64) float64 { return a - b })
	case "Mul":
		return r.binNum(n, func(a, b float64) float64 { return a * b })
	case "Div":
		return r.binNum(n, func(a, b float64) float64 {
			if b == 0 {
				return math.NaN() // Div by zero — NaN propagates; consumer decides
			}
			return a / b
		})
	case "Mod":
		return r.binNum(n, func(a, b float64) float64 { return math.Mod(a, b) })
	case "Neg":
		v, err := r.pullDataPin(n.ID, "x")
		if err != nil {
			return nil, err
		}
		x, _ := expr.AsNumber(v)
		return -x, nil

	// --- §6.2 comparison ---
	case "Lt":
		return r.binNumBool(n, func(a, b float64) bool { return a < b })
	case "LtEq":
		return r.binNumBool(n, func(a, b float64) bool { return a <= b })
	case "Gt":
		return r.binNumBool(n, func(a, b float64) bool { return a > b })
	case "GtEq":
		return r.binNumBool(n, func(a, b float64) bool { return a >= b })
	case "Eq":
		return r.evalEq(n, false)
	case "NotEq":
		return r.evalEq(n, true)

	// --- §6.3 logic (short-circuit) ---
	case "And":
		return r.binBoolShort(n, true)
	case "Or":
		return r.binBoolShort(n, false)
	case "Not":
		v, _ := r.pullDataPin(n.ID, "x")
		return !expr.AsBool(v), nil

	// --- §6.4 string ---
	case "Concat":
		av, _ := r.pullDataPin(n.ID, "a")
		bv, _ := r.pullDataPin(n.ID, "b")
		return expr.FormatValue(av) + expr.FormatValue(bv), nil
	case "Contains":
		hv, _ := r.pullDataPin(n.ID, "haystack")
		nv, _ := r.pullDataPin(n.ID, "needle")
		return strings.Contains(expr.FormatValue(hv), expr.FormatValue(nv)), nil
	case "Length":
		sv, _ := r.pullDataPin(n.ID, "s")
		return float64(len(expr.FormatValue(sv))), nil

	// --- §6.5 conversion ---
	case "ToString":
		xv, _ := r.pullDataPin(n.ID, "x")
		return expr.FormatValue(xv), nil
	case "ToNumber":
		xv, _ := r.pullDataPin(n.ID, "x")
		f, _ := expr.AsNumber(xv)
		return f, nil
	case "ToBool":
		xv, _ := r.pullDataPin(n.ID, "x")
		return expr.AsBool(xv), nil

	// --- §6.6 selection ---
	case "Select":
		cv, _ := r.pullDataPin(n.ID, "cond")
		if expr.AsBool(cv) {
			return r.pullDataPin(n.ID, "a")
		}
		return r.pullDataPin(n.ID, "b")
	}
	return nil, fmt.Errorf("evalPureFunc: unknown kind %q", n.Kind)
}

func (r *ContainerRunner) binNum(n *container.GraphNode, op func(a, b float64) float64) (expr.Value, error) {
	av, err := r.pullDataPin(n.ID, "a")
	if err != nil {
		return nil, err
	}
	bv, err := r.pullDataPin(n.ID, "b")
	if err != nil {
		return nil, err
	}
	a, _ := expr.AsNumber(av)
	b, _ := expr.AsNumber(bv)
	return op(a, b), nil
}

func (r *ContainerRunner) binNumBool(n *container.GraphNode, op func(a, b float64) bool) (expr.Value, error) {
	av, _ := r.pullDataPin(n.ID, "a")
	bv, _ := r.pullDataPin(n.ID, "b")
	a, _ := expr.AsNumber(av)
	b, _ := expr.AsNumber(bv)
	return op(a, b), nil
}

// evalEq: same-type direct compare; cross-type via expr.FormatValue (ToString-based).
// Cross-type ToString is intentional per spec §6.2 — but produces surprises like
// "1.0" != "1" depending on FormatValue rules. PIN_TYPE_COERCION_WARNING from
// validator nudges users to insert explicit To* nodes when one side is `any`.
func (r *ContainerRunner) evalEq(n *container.GraphNode, negate bool) (expr.Value, error) {
	av, _ := r.pullDataPin(n.ID, "a")
	bv, _ := r.pullDataPin(n.ID, "b")
	eq := false
	if sameType(av, bv) {
		eq = av == bv
	} else {
		eq = expr.FormatValue(av) == expr.FormatValue(bv)
	}
	if negate {
		return !eq, nil
	}
	return eq, nil
}

func sameType(a, b any) bool {
	if a == nil && b == nil {
		return true
	}
	if a == nil || b == nil {
		return false
	}
	return fmt.Sprintf("%T", a) == fmt.Sprintf("%T", b)
}

// binBoolShort: short-circuit And (false bypasses b) / Or (true bypasses b).
// andMode=true → And; false → Or.
func (r *ContainerRunner) binBoolShort(n *container.GraphNode, andMode bool) (expr.Value, error) {
	av, _ := r.pullDataPin(n.ID, "a")
	aBool := expr.AsBool(av)
	if andMode {
		if !aBool {
			return false, nil // short-circuit
		}
	} else {
		if aBool {
			return true, nil // short-circuit
		}
	}
	bv, _ := r.pullDataPin(n.ID, "b")
	return expr.AsBool(bv), nil
}
