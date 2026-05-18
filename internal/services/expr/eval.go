package expr

import (
	"fmt"
	"math"
	"time"
)

// Eval 求值。env 提供 $vars/$params/$sys 解析。
func Eval(n *Node, env Env) (Value, error) {
	if n == nil {
		return nil, fmt.Errorf("expr: nil node")
	}
	switch n.Kind {
	case nNumber:
		return n.Num, nil
	case nString:
		return n.Str, nil
	case nBool:
		return n.Bool, nil
	case nNull:
		return nil, nil
	// v4: nVar removed — only nIdent (bare identifier) accesses Env.
	case nIdent:
		// v4: bare identifier (no $ prefix). Env.Get receives the plain name.
		// InputEnv looks it up in the inputs map; old envs that only know $-paths return nil.
		v, err := env.Get(n.VarPath)
		if err != nil {
			return nil, fmt.Errorf("expr: env.Get(%q): %w", n.VarPath, err)
		}
		return v, nil
	case nNeg:
		r, err := Eval(n.Right, env)
		if err != nil {
			return nil, err
		}
		x, ok := AsNumber(r)
		if !ok {
			return nil, fmt.Errorf("expr: unary - on non-number at col %d", n.Pos)
		}
		return -x, nil
	case nNot:
		r, err := Eval(n.Right, env)
		if err != nil {
			return nil, err
		}
		return !AsBool(r), nil
	case nAdd, nSub, nMul, nDiv, nMod:
		return evalArith(n, env)
	case nEq, nNe:
		return evalEq(n, env)
	case nLt, nLe, nGt, nGe:
		return evalCompare(n, env)
	case nAnd:
		l, err := Eval(n.Left, env)
		if err != nil {
			return nil, err
		}
		if !AsBool(l) {
			return false, nil
		}
		r, err := Eval(n.Right, env)
		if err != nil {
			return nil, err
		}
		return AsBool(r), nil
	case nOr:
		l, err := Eval(n.Left, env)
		if err != nil {
			return nil, err
		}
		if AsBool(l) {
			return true, nil
		}
		r, err := Eval(n.Right, env)
		if err != nil {
			return nil, err
		}
		return AsBool(r), nil
	case nTernary:
		c, err := Eval(n.Cond, env)
		if err != nil {
			return nil, err
		}
		if AsBool(c) {
			return Eval(n.Then, env)
		}
		return Eval(n.Else, env)
	case nCall:
		return evalCall(n, env)
	}
	return nil, fmt.Errorf("expr: unknown node kind %d at col %d", n.Kind, n.Pos)
}

func evalArith(n *Node, env Env) (Value, error) {
	l, err := Eval(n.Left, env)
	if err != nil {
		return nil, err
	}
	r, err := Eval(n.Right, env)
	if err != nil {
		return nil, err
	}
	// 字符串拼接：+ 且任一边是 string
	if n.Kind == nAdd {
		if ls, ok := l.(string); ok {
			return ls + FormatValue(r), nil
		}
		if rs, ok := r.(string); ok {
			return FormatValue(l) + rs, nil
		}
	}
	lx, lok := AsNumber(l)
	rx, rok := AsNumber(r)
	if !lok || !rok {
		return nil, fmt.Errorf("expr: arith on non-number at col %d", n.Pos)
	}
	switch n.Kind {
	case nAdd:
		return lx + rx, nil
	case nSub:
		return lx - rx, nil
	case nMul:
		return lx * rx, nil
	case nDiv:
		if rx == 0 {
			return nil, fmt.Errorf("expr: divide by zero at col %d", n.Pos)
		}
		return lx / rx, nil
	case nMod:
		if rx == 0 {
			return nil, fmt.Errorf("expr: modulo by zero at col %d", n.Pos)
		}
		return math.Mod(lx, rx), nil
	}
	return nil, fmt.Errorf("expr: unknown arith op")
}

func evalEq(n *Node, env Env) (Value, error) {
	l, err := Eval(n.Left, env)
	if err != nil {
		return nil, err
	}
	r, err := Eval(n.Right, env)
	if err != nil {
		return nil, err
	}
	eq := valueEqual(l, r)
	if n.Kind == nNe {
		return !eq, nil
	}
	return eq, nil
}

func valueEqual(l, r Value) bool {
	if l == nil || r == nil {
		return l == nil && r == nil
	}
	// number / bool 互转后比较
	if ln, lok := AsNumber(l); lok {
		if rn, rok := AsNumber(r); rok {
			return ln == rn
		}
	}
	if ls, ok := l.(string); ok {
		if rs, ok := r.(string); ok {
			return ls == rs
		}
	}
	if lp, ok := l.(Point); ok {
		if rp, ok := r.(Point); ok {
			return lp.X == rp.X && lp.Y == rp.Y
		}
	}
	return false
}

func evalCompare(n *Node, env Env) (Value, error) {
	l, err := Eval(n.Left, env)
	if err != nil {
		return nil, err
	}
	r, err := Eval(n.Right, env)
	if err != nil {
		return nil, err
	}
	lx, lok := AsNumber(l)
	rx, rok := AsNumber(r)
	if !lok || !rok {
		return nil, fmt.Errorf("expr: comparison on non-number at col %d", n.Pos)
	}
	switch n.Kind {
	case nLt:
		return lx < rx, nil
	case nLe:
		return lx <= rx, nil
	case nGt:
		return lx > rx, nil
	case nGe:
		return lx >= rx, nil
	}
	return nil, fmt.Errorf("expr: unknown compare op")
}

func evalCall(n *Node, env Env) (Value, error) {
	args := make([]Value, len(n.Args))
	for i, a := range n.Args {
		v, err := Eval(a, env)
		if err != nil {
			return nil, err
		}
		args[i] = v
	}
	switch n.FuncName {
	case "abs":
		if len(args) != 1 {
			return nil, fmt.Errorf("expr: abs() expects 1 arg at col %d", n.Pos)
		}
		x, ok := AsNumber(args[0])
		if !ok {
			return nil, fmt.Errorf("expr: abs() needs number at col %d", n.Pos)
		}
		return math.Abs(x), nil
	case "min":
		if len(args) != 2 {
			return nil, fmt.Errorf("expr: min() expects 2 args at col %d", n.Pos)
		}
		a, aok := AsNumber(args[0])
		b, bok := AsNumber(args[1])
		if !aok || !bok {
			return nil, fmt.Errorf("expr: min() needs numbers at col %d", n.Pos)
		}
		return math.Min(a, b), nil
	case "max":
		if len(args) != 2 {
			return nil, fmt.Errorf("expr: max() expects 2 args at col %d", n.Pos)
		}
		a, aok := AsNumber(args[0])
		b, bok := AsNumber(args[1])
		if !aok || !bok {
			return nil, fmt.Errorf("expr: max() needs numbers at col %d", n.Pos)
		}
		return math.Max(a, b), nil
	case "now":
		if len(args) != 0 {
			return nil, fmt.Errorf("expr: now() expects 0 args at col %d", n.Pos)
		}
		return float64(time.Now().UnixMilli()), nil
	}
	return nil, fmt.Errorf("expr: unknown function %q at col %d", n.FuncName, n.Pos)
}
