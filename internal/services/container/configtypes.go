package container

// SwitchConfig 是 Switch 节点的 typed config.
// validator / runtime / pin schema 必须走 ParseSwitchConfig 入口, 不许各自 cast map[string]any.
//
// named-by-value 模型: config.cases[] 是 case 值列表, 每个 case 值即一个同名 exec 出口 pin;
// 待比较的 Value 是 data-in pin (走 literal/边), 不是 config 字段.
type SwitchConfig struct {
	Cases []string // config.cases 原样 (仅 string 转换, 不过滤空/重复 — 留给 validateSwitchConfig 报错)
}

// ParseSwitchConfig 解析 Switch 节点 config 成 typed struct.
// 容错: nil node / nil config / cases 非数组 → 返零值. 严格验证 (空 / 重复 / 含 '.' / 'default') 由
// validateSwitchConfig 负责 — 故此处**不**过滤空/重复, 原样返 (非 string 项跳过).
func ParseSwitchConfig(n *GraphNode) (SwitchConfig, error) {
	var c SwitchConfig
	if n == nil || n.Config == nil {
		return c, nil
	}
	arr, ok := n.Config["cases"].([]any)
	if !ok {
		return c, nil
	}
	for _, item := range arr {
		if s, ok := item.(string); ok {
			c.Cases = append(c.Cases, s)
		}
	}
	return c, nil
}

// --- Expr node typed config ---

// ExprConfig 是 Expr 节点 typed config.
// inputs[] 在画布上动态加 — 每个 input 对应一个 data-in pin (name + type).
// expr 字符串只能引用 inputs 中声明的 name (bare identifier), 不能直接访问 $vars/$sys/$params.
type ExprConfig struct {
	Expr    string          // 用户写的表达式 (e.g. "i + 1", "s == 'FISHING' && hp > 0.5")
	OutType string          // "auto" (默认, 静态推断) 或显式 number/bool/string/point/any
	Inputs  []ExprInputDecl // {Name, Type} — 每个对应 data-in pin
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
	c.Inputs = ParseDynamicInputDecls(n)
	return c, nil
}

// ParseDynamicInputDecls 解析 config.Inputs[] 动态 data-in pin 声明 — 凡 Spec.DynamicInputs
// 节点 (Expr / Script) 通用. nil-safe; 空 Name 项跳过. 重名不在这里去重 — 由 validator 报错.
func ParseDynamicInputDecls(n *GraphNode) []ExprInputDecl {
	if n == nil || n.Config == nil {
		return nil
	}
	raw, ok := n.Config["Inputs"].([]any)
	if !ok {
		return nil
	}
	var decls []ExprInputDecl
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
		decls = append(decls, ExprInputDecl{Name: name, Type: typ})
	}
	return decls
}

// ParseDynamicOutputDecls 解析 config.Outputs[] 动态 Data 字段声明 — Spec.DynamicDataFields
// 节点 (AI) 通用。结构同 Inputs ({Name,Type}); 这些字段挂在 Done 出口, 可绑变量, 也可经
// exec-data 直连紧邻下游(同 Fail.Code 机制)。nil-safe; 空 Name 跳过。
func ParseDynamicOutputDecls(n *GraphNode) []ExprInputDecl {
	if n == nil || n.Config == nil {
		return nil
	}
	raw, ok := n.Config["Outputs"].([]any)
	if !ok {
		return nil
	}
	var decls []ExprInputDecl
	for _, it := range raw {
		m, ok := it.(map[string]any)
		if !ok {
			continue
		}
		name, _ := m["Name"].(string)
		typ, _ := m["Type"].(string)
		if name == "" {
			continue
		}
		decls = append(decls, ExprInputDecl{Name: name, Type: typ})
	}
	return decls
}
