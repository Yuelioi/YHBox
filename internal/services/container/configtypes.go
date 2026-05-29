package container

import "fmt"

// SwitchConfig 是 Switch 节点的 typed config.
// validator / runtime / pin schema 必须走 ParseSwitchConfig 入口, 不许各自 cast map[string]any.
//
// 待比较的值是 data-in pin (走 r.pullValue), 不是 config 字段; config 只剩 Cases.
type SwitchConfig struct {
	Cases []string // 规整过 (类型转 + 过滤非字符串)
}

// ParseSwitchConfig 解析 Switch 节点 config 成 typed struct.
// 容错: nil node / nil config 返零值. 严格验证 (空 / 重复 / 含 '.' / 'default') 由
// validateSwitchConfig 负责.
//
// Switch config schema (nodepkg Spec): Case1Value..Case16Value=string (单值).
// 空值 ("") 视为未声明该 case, 不进 Cases 列表.
func ParseSwitchConfig(n *GraphNode) (SwitchConfig, error) {
	var c SwitchConfig
	if n == nil || n.Config == nil {
		return c, nil
	}
	for i := 1; i <= 16; i++ {
		key := fmt.Sprintf("Case%dValue", i)
		if cs := PinString(n, key); cs != "" {
			c.Cases = append(c.Cases, cs)
		}
	}
	return c, nil
}

// --- Expr node typed config ---

// ExprConfig 是 Expr 节点 typed config.
// inputs[] 在画布上动态加 — 每个 input 对应一个 data-in pin (name + type).
// expr 字符串只能引用 inputs 中声明的 name (bare identifier), 不能直接访问 $vars/$sys/$params.
type ExprConfig struct {
	Expr    string           // 用户写的表达式 (e.g. "i + 1", "s == 'FISHING' && hp > 0.5")
	OutType string           // "auto" (默认, 静态推断) 或显式 number/bool/string/point/any
	Inputs  []ExprInputDecl  // {Name, Type} — 每个对应 data-in pin
}

type ExprInputDecl struct {
	Name string // identifier in expr (e.g. "i", "state")
	Type string // PinType string ("number" / "bool" / "string" / "point" / "any")
}

// ParseExprConfig 解析 Expr 节点 config. nil-safe.
// OutType 缺省值 "auto" (走静态推断 — runtime/validator 后续展开).
// 重名 inputs 不在这里去重 — 由 validator 报 EXPR_DUPLICATE_INPUT.
func ParseExprConfig(n *GraphNode) (ExprConfig, error) {
	c := ExprConfig{OutType: "auto"}
	if n == nil || n.Config == nil {
		return c, nil
	}
	c.Expr = PinString(n, "Expression")
	if t, ok := n.Config["OutType"].(string); ok && t != "" {
		c.OutType = t
	}
	if raw, ok := n.Config["Inputs"].([]any); ok {
		for _, it := range raw {
			m, ok := it.(map[string]any)
			if !ok {
				continue
			}
			// Inputs[] entry keys 跟节点 InputSpec.Name 风格对齐 — Name (PascalCase 用户面),
			// Type (Pascal type tag). Dynamic input.name 在 graph 内是 data-pin 名, 由用户起.
			name, _ := m["Name"].(string)
			typ, _ := m["Type"].(string)
			if name == "" {
				continue
			}
			c.Inputs = append(c.Inputs, ExprInputDecl{Name: name, Type: typ})
		}
	}
	return c, nil
}
