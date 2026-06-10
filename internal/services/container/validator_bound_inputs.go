package container

import (
	"fmt"
	"strings"

	nodepkg "yotta/internal/node"
)

// validateBoundInputs — DynamicInputs 节点 (Expr/Script) 的变量绑定项 (decl.Var 非空) 校验:
//   BOUND_VAR_UNKNOWN       error   绑定的变量未在容器 Vars 声明 (判定源 = Container.Vars)
//   BOUND_VAR_TYPE_MISMATCH warning 声明 Type 与变量类型都非空、非 any 且不同
//
// 绑定项不是 pin (dataInPinTypeForNode 已排除), 连线/字面量类检查天然不碰它们。
func validateBoundInputs(c *Container) []ValidationError {
	if c == nil {
		return nil
	}
	declared := map[string]string{} // name → type
	for _, v := range c.Vars {
		declared[v.Name] = strings.ToLower(v.Type)
	}
	var errs []ValidationError
	check := func(nodes []GraphNode, graphPath []string) {
		for i := range nodes {
			n := &nodes[i]
			rn, ok := nodepkg.Get(n.Kind)
			if !ok || !rn.Spec.DynamicInputs {
				continue
			}
			for _, d := range ParseDynamicInputDecls(n) {
				if d.Var == "" {
					continue
				}
				varType, exists := declared[d.Var]
				if !exists {
					errs = append(errs, ValidationError{
						Severity: SeverityError, Code: CodeBoundVarUnknown,
						GraphPath: graphPath, NodeID: n.ID,
						Params: map[string]any{"input": d.Name, "var": d.Var},
					})
					continue
				}
				declType := strings.ToLower(d.Type)
				if declType != "" && declType != "any" && varType != "" && varType != "any" && declType != varType {
					errs = append(errs, ValidationError{
						Severity: SeverityWarning, Code: CodeBoundVarTypeMismatch,
						GraphPath: graphPath, NodeID: n.ID,
						Params: map[string]any{"input": d.Name, "var": d.Var, "want": declType, "got": varType},
					})
				}
			}
		}
	}
	check(c.Graph.Nodes, []string{"main"})
	for i := range c.Subgraphs {
		sg := &c.Subgraphs[i]
		check(sg.Graph.Nodes, []string{"main", fmt.Sprintf("subgraph-%s (%s)", sg.Label, sg.ID)})
	}
	return errs
}
