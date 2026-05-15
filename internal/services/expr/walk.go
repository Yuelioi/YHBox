package expr

// VarRefs 收集 AST 中所有 $vars./$params./$sys. 路径（去重前保序）。
// Validate 用：把 $vars.X 比对 Container.Vars 声明，确保 typo / 漏声明早暴露。
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
	if n.Kind == nVar {
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
