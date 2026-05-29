// internal/services/container/pin_value.go
// PinValue — data-in pin 字面量的唯一读取入口 (validator + runtime setup 共用)。
//
// 正源 = config.literal[<Spec.Input 名>] (input-editing-unification spec)。读取优先级
// 镜像 runtime node.newInputs (literal > 顶层 config), 让静态 validator 看到的值跟运行时一致。
//   - 编辑器一律写 config.literal[name] (PascalCase Spec.Input 名)。
//   - 顶层 config[name] fallback 只为兼容尚未跑一次性迁移脚本的旧数据; 迁移后该分支休眠。
//   - name 必须是规范 PascalCase Spec.Input 名 — 旧 validator 散落的小写 key (roi/hsv/key/
//     counts360/...) 是 case-drift bug, 统一改走本入口 + 规范名修掉。
package container

import "encoding/json"

// PinValue 读 node 某 input pin 当前值: config.literal[name] 优先, 顶层 config[name] fallback。
// 未设返回 nil。
func PinValue(n *GraphNode, name string) any {
	if n == nil || n.Config == nil {
		return nil
	}
	if lit, ok := n.Config["literal"].(map[string]any); ok {
		if v, ok := lit[name]; ok {
			return v
		}
	}
	return n.Config[name]
}

// PinString 读 string 型 pin。非 string / 未设 → ""。
func PinString(n *GraphNode, name string) string {
	s, _ := PinValue(n, name).(string)
	return s
}

// PinFloat 读 number 型 pin。返 (值, 是否拿到可用数值)。
func PinFloat(n *GraphNode, name string) (float64, bool) {
	switch v := PinValue(n, name).(type) {
	case float64:
		return v, true
	case int:
		return float64(v), true
	case int64:
		return float64(v), true
	case json.Number:
		if f, err := v.Float64(); err == nil {
			return f, true
		}
	}
	return 0, false
}

// PinInt 读 number 型 pin 当 int。返 (值, 是否拿到可用数值)。
func PinInt(n *GraphNode, name string) (int, bool) {
	if f, ok := PinFloat(n, name); ok {
		return int(f), true
	}
	return 0, false
}

// PinMap 读 JSON 型 pin 当 map。非 map / 未设 → nil。
func PinMap(n *GraphNode, name string) map[string]any {
	m, _ := PinValue(n, name).(map[string]any)
	return m
}

// SetPinValue 写 node 某 input pin 值到 config.literal[name] (唯一规范写入口)。
// 任何写 "== Spec.Input 名" 的 key 都走这, 禁止再写顶层 config (input-editing-unification guardrail)。
func SetPinValue(n *GraphNode, name string, value any) {
	if n == nil {
		return
	}
	if n.Config == nil {
		n.Config = map[string]any{}
	}
	lit, _ := n.Config["literal"].(map[string]any)
	if lit == nil {
		lit = map[string]any{}
		n.Config["literal"] = lit
	}
	lit[name] = value
}
