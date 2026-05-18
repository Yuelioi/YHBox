package container

// SwitchConfig 是 Switch 节点的 typed config.
// validator / runtime / pin schema 必须走 ParseSwitchConfig 入口, 不许各自 cast map[string]any.
type SwitchConfig struct {
	Value string   // 表达式字符串 (runtime resolve)
	Cases []string // 规整过 (类型转 + 过滤非字符串)
}

// ParseSwitchConfig 解析 Switch 节点 config 成 typed struct.
// 容错: nil node / nil config 返零值. 非字符串 case 元素 silently skip.
// 严格验证 (空字符串 / 重复 / 含 '.' / 'default') 是 validateSwitchConfig 的事, 不在这里.
func ParseSwitchConfig(n *GraphNode) (SwitchConfig, error) {
	var c SwitchConfig
	if n == nil || n.Config == nil {
		return c, nil
	}
	if v, ok := n.Config["value"].(string); ok {
		c.Value = v
	}
	raw, _ := n.Config["cases"].([]any)
	for _, item := range raw {
		if cs, ok := item.(string); ok {
			c.Cases = append(c.Cases, cs)
		}
	}
	return c, nil
}

// ParallelConfig 是 Parallel / Race 节点的 typed config.
type ParallelConfig struct {
	N int // branch 数, 默认 2 (n<=0 视为 2)
}

// ParseParallelConfig 解析 Parallel/Race 节点 config.
// 容错: nil node / nil config / n<=0 全 fallback to N=2.
func ParseParallelConfig(n *GraphNode) (ParallelConfig, error) {
	c := ParallelConfig{N: 2}
	if n == nil || n.Config == nil {
		return c, nil
	}
	if v, ok := n.Config["n"].(float64); ok && int(v) > 0 {
		c.N = int(v)
	}
	return c, nil
}

// --- v4: Expr node (spec §5) ---

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
	if v, ok := n.Config["expr"].(string); ok {
		c.Expr = v
	}
	if t, ok := n.Config["outType"].(string); ok && t != "" {
		c.OutType = t
	}
	if raw, ok := n.Config["inputs"].([]any); ok {
		for _, it := range raw {
			m, ok := it.(map[string]any)
			if !ok {
				continue
			}
			name, _ := m["name"].(string)
			typ, _ := m["type"].(string)
			if name == "" {
				continue // skip unnamed entries silently; validator may want to flag
			}
			c.Inputs = append(c.Inputs, ExprInputDecl{Name: name, Type: typ})
		}
	}
	return c, nil
}
