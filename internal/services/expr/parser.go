package expr

import "fmt"

// Node 是 AST 节点。eval.go 按 kind 求值。
type Node struct {
	Kind nodeKind

	// 字面量
	Num float64
	Str string
	Bool bool
	Null bool

	// 变量路径
	VarPath string

	// 运算符
	Op nodeKind // unary/binary 时存子算符 kind 自身或冗余；这里用 Kind 已够

	// 子节点
	Left, Right, Cond, Then, Else *Node

	// 函数调用
	FuncName string
	Args     []*Node

	// 错误定位
	Pos int
}

type nodeKind int

const (
	nNumber nodeKind = iota
	nString
	nBool
	nNull
	// nVar removed in v4 (was: $vars.X / $sys.Y / $params.Z). Slot kept to preserve const ordering.
	nVarReservedDoNotUse
	nIdent // v4: bare identifier (e.g. `i`, `state`) — looked up via Env.Get(name) with no $ prefix
	nNeg     // unary -
	nNot     // unary !
	nAdd
	nSub
	nMul
	nDiv
	nMod
	nEq
	nNe
	nLt
	nLe
	nGt
	nGe
	nAnd
	nOr
	nTernary
	nCall
)

// Parse 把表达式串 parse 成 AST root。失败返 typed error 含 col 信息。
func Parse(src string) (*Node, error) {
	toks, err := newLexer(src).tokens()
	if err != nil {
		return nil, err
	}
	p := &parser{toks: toks}
	n, err := p.parseExpr(0)
	if err != nil {
		return nil, err
	}
	if p.peek().kind != tkEOF {
		t := p.peek()
		return nil, fmt.Errorf("expr: unexpected token %q at col %d", t.val, t.pos)
	}
	return n, nil
}

// Pratt parser ----------

type parser struct {
	toks []token
	i    int
}

func (p *parser) peek() token { return p.toks[p.i] }
func (p *parser) advance() token {
	t := p.toks[p.i]
	p.i++
	return t
}

// Operator precedence (higher = binds tighter).
// 跟 Go / JS 一致：unary > * / % > + - > comparison > == != > && > || > ternary.
const (
	prTernary  = 1
	prOr       = 2
	prAnd      = 3
	prEq       = 4
	prCompare  = 5
	prAddSub   = 6
	prMulDiv   = 7
	prUnary    = 8
)

func (p *parser) parseExpr(minPrec int) (*Node, error) {
	left, err := p.parsePrefix()
	if err != nil {
		return nil, err
	}
	for {
		op := p.peek()
		prec, kind, ok := infixInfo(op.kind)
		if !ok || prec < minPrec {
			break
		}
		// 三元单独处理：cond ? a : b 右结合
		if op.kind == tkQuestion {
			p.advance()
			thenN, err := p.parseExpr(0)
			if err != nil {
				return nil, err
			}
			if p.peek().kind != tkColon {
				return nil, fmt.Errorf("expr: ternary missing ':' at col %d", p.peek().pos)
			}
			p.advance()
			elseN, err := p.parseExpr(prTernary)
			if err != nil {
				return nil, err
			}
			left = &Node{Kind: nTernary, Cond: left, Then: thenN, Else: elseN, Pos: op.pos}
			continue
		}
		p.advance()
		right, err := p.parseExpr(prec + 1) // left-assoc
		if err != nil {
			return nil, err
		}
		left = &Node{Kind: kind, Left: left, Right: right, Pos: op.pos}
	}
	return left, nil
}

func infixInfo(k tokKind) (int, nodeKind, bool) {
	switch k {
	case tkQuestion:
		return prTernary, nTernary, true
	case tkOr:
		return prOr, nOr, true
	case tkAnd:
		return prAnd, nAnd, true
	case tkEq:
		return prEq, nEq, true
	case tkNe:
		return prEq, nNe, true
	case tkLt:
		return prCompare, nLt, true
	case tkLe:
		return prCompare, nLe, true
	case tkGt:
		return prCompare, nGt, true
	case tkGe:
		return prCompare, nGe, true
	case tkPlus:
		return prAddSub, nAdd, true
	case tkMinus:
		return prAddSub, nSub, true
	case tkStar:
		return prMulDiv, nMul, true
	case tkSlash:
		return prMulDiv, nDiv, true
	case tkPercent:
		return prMulDiv, nMod, true
	}
	return 0, 0, false
}

func (p *parser) parsePrefix() (*Node, error) {
	t := p.peek()
	switch t.kind {
	case tkNumber:
		p.advance()
		return &Node{Kind: nNumber, Num: t.num, Pos: t.pos}, nil
	case tkString:
		p.advance()
		return &Node{Kind: nString, Str: t.val, Pos: t.pos}, nil
	// v4: tkVarPath / nVar removed. Lexer rejects '$' so we never receive that token.
	case tkIdent:
		p.advance()
		switch t.val {
		case "true":
			return &Node{Kind: nBool, Bool: true, Pos: t.pos}, nil
		case "false":
			return &Node{Kind: nBool, Bool: false, Pos: t.pos}, nil
		case "null":
			return &Node{Kind: nNull, Null: true, Pos: t.pos}, nil
		}
		// v4: 没跟 '(' → bare identifier (Env.Get(name) 查 InputEnv inputs map).
		if p.peek().kind != tkLParen {
			return &Node{Kind: nIdent, VarPath: t.val, Pos: t.pos}, nil
		}
		p.advance()
		var args []*Node
		if p.peek().kind != tkRParen {
			for {
				a, err := p.parseExpr(0)
				if err != nil {
					return nil, err
				}
				args = append(args, a)
				if p.peek().kind == tkComma {
					p.advance()
					continue
				}
				break
			}
		}
		if p.peek().kind != tkRParen {
			return nil, fmt.Errorf("expr: missing ')' at col %d", p.peek().pos)
		}
		p.advance()
		return &Node{Kind: nCall, FuncName: t.val, Args: args, Pos: t.pos}, nil
	case tkLParen:
		p.advance()
		inner, err := p.parseExpr(0)
		if err != nil {
			return nil, err
		}
		if p.peek().kind != tkRParen {
			return nil, fmt.Errorf("expr: missing ')' at col %d", p.peek().pos)
		}
		p.advance()
		return inner, nil
	case tkMinus:
		p.advance()
		right, err := p.parseExpr(prUnary)
		if err != nil {
			return nil, err
		}
		return &Node{Kind: nNeg, Right: right, Pos: t.pos}, nil
	case tkNot:
		p.advance()
		right, err := p.parseExpr(prUnary)
		if err != nil {
			return nil, err
		}
		return &Node{Kind: nNot, Right: right, Pos: t.pos}, nil
	}
	return nil, fmt.Errorf("expr: unexpected token %q at col %d", t.val, t.pos)
}
