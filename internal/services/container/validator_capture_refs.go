package container

import (
	nodepkg "yotta/internal/node"
)

// validateCaptureRefs 校验节点 config.capture (输出捕获绑定, Spec C) 的两件事:
//
//  1. 变量名合法: config.capture[field] 引用的变量名必须在 Container.Vars (或子图
//     RequiredGlobals 白名单) 里 — 否则 INVALID_VAR_REF (跟 GetVar/SetVar 同规则,
//     scope 固定 auto)。删变量后残留的捕获绑定即在此暴露 (消费者审计)。
//  2. 字段名合法: field 必须是该 kind 的可绑字段 (nodepkg.BindableFields, 即 exec 出口
//     携带的 Data 字段) — 否则 INVALID_PIN (字段不存在 / 节点输出变更后的悬空绑定)。
//
// config.capture 形态 = node.Config["capture"]: map[string]string (字段名 → 变量名)。
func validateCaptureRefs(c *Container, sgs []Subgraph) []ValidationError {
	if c == nil {
		return nil
	}
	declared := make(map[string]bool, len(c.Vars))
	for _, v := range c.Vars {
		declared[v.Name] = true
	}

	var errs []ValidationError
	check := func(g Graph, path []string, extraDeclared map[string]bool) {
		for _, n := range g.Nodes {
			capRaw, ok := n.Config["capture"].(map[string]any)
			if !ok || len(capRaw) == 0 {
				continue
			}
			// 该 kind 的可绑字段集 (节点未注册 → 跳过字段校验, 仅做 var-ref 校验)。
			bindable := map[string]bool{}
			if rn, ok := nodepkg.Get(n.Kind); ok {
				for _, f := range nodepkg.BindableFields(&rn.Spec) {
					bindable[f] = true
				}
			}
			for field, varNameRaw := range capRaw {
				varName, _ := varNameRaw.(string)
				if varName == "" {
					continue // 空绑定 = 未配, 不校验
				}
				// 字段合法性 (仅当 kind 已注册才判 — 否则缺 spec 无从判)。
				if len(bindable) > 0 && !bindable[field] {
					errs = append(errs, ValidationError{
						Severity:  SeverityError,
						Code:      CodeInvalidPin,
						GraphPath: append([]string(nil), path...),
						NodeID:    n.ID,
						Params:    map[string]any{"pin": field, "kind": n.Kind},
					})
				}
				// 变量名合法性 (scope 固定 auto, 同 var-ref)。
				if declared[varName] {
					continue
				}
				if extraDeclared != nil && extraDeclared[varName] {
					continue // 子图 RequiredGlobals 白名单
				}
				errs = append(errs, ValidationError{
					Severity:  SeverityError,
					Code:      CodeInvalidVarRef,
					GraphPath: append([]string(nil), path...),
					NodeID:    n.ID,
					Params:    map[string]any{"varName": varName, "scope": "auto"},
				})
			}
		}
	}
	check(c.Graph, []string{"main"}, nil)
	for _, sg := range sgs {
		sgWhitelist := map[string]bool{}
		for _, name := range sg.RequiredGlobals {
			sgWhitelist[name] = true
		}
		check(sg.Graph, []string{"subgraph", sg.ID}, sgWhitelist)
	}
	return errs
}
