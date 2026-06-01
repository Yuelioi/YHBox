// SubgraphRequiredGlobals 派生.
//
// Container.Normalize / SaveSubgraph 调 computeRequiredGlobals 重算 sg.RequiredGlobals.
// validator (validator_var_refs.go) 白名单 sg.RequiredGlobals 不报 INVALID_VAR_REF.
// library.ImportToContainer dry-run 用 RequiredGlobals 算 MissingGlobals diff.
package container

// computeRequiredGlobals 扫 sg.Graph.Nodes 找所有 GetVar/SetVar/IncVar (scope!=local) VarName,
// 跟 containerVars merge type+default. 未在 containerVars 声明的 var (e.g. subgraph ref 漏 declare)
// 仍 emit, Type="" Default=nil — caller import 时按 "any" 兜底.
//
// 顺序按 grep 出现顺序, dedupe 同名. 调用幂等.
func computeRequiredGlobals(sg *Subgraph, containerVars []VarDecl) []SubgraphRequiredGlobal {
	if sg == nil {
		return nil
	}
	declared := make(map[string]*VarDecl, len(containerVars))
	for i := range containerVars {
		v := &containerVars[i]
		declared[v.Name] = v
	}
	varKinds := map[string]bool{"GetVar": true, "SetVar": true, "IncVar": true}
	seen := map[string]bool{}
	var out []SubgraphRequiredGlobal
	for _, n := range sg.Graph.Nodes {
		if !varKinds[n.Kind] {
			continue
		}
		scope := PinString(&n, "Scope")
		if scope == "local" {
			continue
		}
		name := PinString(&n, "VarName")
		if name == "" || seen[name] {
			continue
		}
		seen[name] = true
		rg := SubgraphRequiredGlobal{Name: name}
		if v, ok := declared[name]; ok {
			rg.Type = v.Type
			rg.Default = v.Default
		}
		out = append(out, rg)
	}
	return out
}
