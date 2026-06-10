package expr

// IdentRefs (v4) 收集 AST 中所有 bare identifier 引用 (nIdent — 无 $ 前缀).
// Validator 用: 比对 Expr 节点 inputs[] 声明, 报 EXPR_UNKNOWN_INPUT.
// 跟 VarRefs (收集 $vars/$sys/$params) 互补.
func IdentRefs(n *Node) []string {
	if n == nil {
		return nil
	}
	var out []string
	walkIdentRefs(n, &out)
	return out
}

func walkIdentRefs(n *Node, out *[]string) {
	if n == nil {
		return
	}
	if n.Kind == nIdent {
		*out = append(*out, n.VarPath)
	}
	walkIdentRefs(n.Left, out)
	walkIdentRefs(n.Right, out)
	walkIdentRefs(n.Cond, out)
	walkIdentRefs(n.Then, out)
	walkIdentRefs(n.Else, out)
	for _, a := range n.Args {
		walkIdentRefs(a, out)
	}
}

// VarRefs 收集 AST 中所有 $变量引用 (nVarRef), 返名字不含 $ 前缀.
// Validator 用: 比对容器 Vars 声明, 报 EXPR_UNKNOWN_VAR.
func VarRefs(n *Node) []string {
	if n == nil {
		return nil
	}
	var out []string
	walkVarRefs(n, &out)
	return out
}

func walkVarRefs(n *Node, out *[]string) {
	if n == nil {
		return
	}
	if n.Kind == nVarRef {
		*out = append(*out, n.VarPath)
	}
	walkVarRefs(n.Left, out)
	walkVarRefs(n.Right, out)
	walkVarRefs(n.Cond, out)
	walkVarRefs(n.Then, out)
	walkVarRefs(n.Else, out)
	for _, a := range n.Args {
		walkVarRefs(a, out)
	}
}

// CallRef 一次函数调用引用 — validator 编辑期校验用 (对照 Builtins 报
// EXPR_UNKNOWN_FUNCTION / EXPR_FN_ARITY).
type CallRef struct {
	Name string
	ArgN int
	Pos  int
}

// CallRefs 收集 AST 中所有函数调用 (nCall), 含嵌套.
func CallRefs(n *Node) []CallRef {
	if n == nil {
		return nil
	}
	var out []CallRef
	walkCallRefs(n, &out)
	return out
}

func walkCallRefs(n *Node, out *[]CallRef) {
	if n == nil {
		return
	}
	if n.Kind == nCall {
		*out = append(*out, CallRef{Name: n.FuncName, ArgN: len(n.Args), Pos: n.Pos})
	}
	walkCallRefs(n.Left, out)
	walkCallRefs(n.Right, out)
	walkCallRefs(n.Cond, out)
	walkCallRefs(n.Then, out)
	walkCallRefs(n.Else, out)
	for _, a := range n.Args {
		walkCallRefs(a, out)
	}
}
