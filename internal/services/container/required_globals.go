// SubgraphRequiredGlobals 派生.
//
// SubgraphStore 保存路径调 computeRequiredGlobals 重算 sg.RequiredGlobals (只存名字 —
// Type/Default 由消费方按目标容器即时现算, 不落盘, 防编辑语境污染子图文件).
// validator (validator_var_refs.go) 白名单 sg.RequiredGlobals 不报 INVALID_VAR_REF;
// validateRequiredGlobalsDeclared 检查引用闭包的名字 ⊆ 容器 Vars.
package container

// computeRequiredGlobals 扫 sg.Graph.Nodes 找所有 GetVar/SetVar/IncVar (scope!=local) VarName.
// 顺序按 grep 出现顺序, dedupe 同名. 调用幂等.
func computeRequiredGlobals(sg *Subgraph) []string {
	if sg == nil {
		return nil
	}
	varKinds := map[string]bool{"GetVar": true, "SetVar": true, "IncVar": true}
	seen := map[string]bool{}
	var out []string
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
		out = append(out, name)
	}
	return out
}

// validateRequiredGlobalsDeclared 引用闭包内子图的 RequiredGlobals 名字必须都已在容器
// Vars 声明 (2026-06-12 全局化新增 — 取代 library import 时的 MissingGlobals auto-add:
// 现在插入引用即检, 编辑器对此错误码提供一键补全).
func validateRequiredGlobalsDeclared(c *Container, sgs []Subgraph) []ValidationError {
	declared := map[string]bool{}
	for _, v := range c.Vars {
		declared[v.Name] = true
	}
	var errs []ValidationError
	seen := map[string]bool{} // 同名只报一次 (多个子图要同一个 var 不刷屏)
	for i := range sgs {
		sg := &sgs[i]
		for _, name := range sg.RequiredGlobals {
			if declared[name] || seen[name] {
				continue
			}
			seen[name] = true
			errs = append(errs, ValidationError{
				Severity:  SeverityError,
				Code:      CodeSubgraphVarUndeclared,
				GraphPath: []string{"main"},
				Params:    map[string]any{"name": name, "subgraphId": sg.ID, "subgraphLabel": sg.Label},
			})
		}
	}
	return errs
}
