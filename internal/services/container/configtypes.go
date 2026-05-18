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
