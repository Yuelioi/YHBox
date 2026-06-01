package container

import (
	"fmt"
)

// validateLiteralTypes scans every node.Config["literal"][pin] entry and
// checks the JSON-decoded value type against the pin's declared PinType.
//
// Why: v4 stores inline pin defaults at config.literal.<pinName>. A user can
// type a string into a number-typed pin and the runtime silently coerces to 0
// (or fallback) — which feels like a bug ("my Sleep node ignored my 500"
// because user wrote "500ms" not 500). LITERAL_TYPE_MISMATCH catches this at
// design time.
//
// Coverage: static data-in pins only. Expr's dynamic inputs are NOT scanned
// here — the Expr Inspector validates its own literals against declared input
// types.
func validateLiteralTypes(c *Container) []ValidationError {
	if c == nil {
		return nil
	}
	var errs []ValidationError
	check := func(nodes []GraphNode, graphPath []string) {
		for i := range nodes {
			n := &nodes[i]
			lit, _ := n.Config["literal"].(map[string]any)
			if lit == nil {
				continue
			}
			if !knownKind(n.Kind) {
				continue
			}
			for pinName, raw := range lit {
				pinType := dataInPinTypeForKind(n.Kind, pinName)
				if pinType == "" {
					continue // pin not in static schema — INVALID_PIN handles unknown pins
				}
				if literalMatchesType(raw, pinType) {
					continue
				}
				// TemplateKey 语义 pin (WaitTemplate/CheckTemplate/ClickTemplate/OnEvent 的 Templates)
				// 实为字符串列表: template-picker 多选, runtime 走 PinStringList 读. 标量
				// literalMatchesType 会把数组误判成 string mismatch, 这里按列表放行 (元素全 string,
				// 或裸 string 单值兜底, 跟 PinStringList 的容忍一致).
				if dataInPinSemanticForKind(n.Kind, pinName) == "TemplateKey" && isStringList(raw) {
					continue
				}
				errs = append(errs, ValidationError{
					Severity:  SeverityError,
					Code:      CodeLiteralTypeMismatch,
					GraphPath: graphPath,
					NodeID:    n.ID,
					Params: map[string]any{
						"nodeID":   n.ID,
						"pin":      pinName,
						"value":    raw,
						"expected": pinType,
					},
				})
			}
		}
	}
	check(c.Graph.Nodes, []string{"main"})
	for _, sg := range c.Subgraphs {
		check(sg.Graph.Nodes, []string{"main", fmt.Sprintf("subgraph-%s (%s)", sg.Label, sg.ID)})
	}
	return errs
}

// isStringList: TemplateKey 列表 literal 的容忍形态 — []any(元素全 string) / []string / 裸 string.
// 跟 container.PinStringList 的读取容忍一致 (裸 string 当一元列表).
func isStringList(v any) bool {
	switch vv := v.(type) {
	case string:
		return true
	case []string:
		return true
	case []any:
		for _, e := range vv {
			if _, ok := e.(string); !ok {
				return false
			}
		}
		return true
	}
	return false
}

// literalMatchesType: JSON-decoded value type vs declared PinType.
// Go's encoding/json decodes:
//   - numbers as float64
//   - bool as bool
//   - string as string
//   - point as map[string]any{x:float64, y:float64}
//   - geometry as map[string]any{pct:{...}, overrides:[...]}
//   - any matches anything (escape hatch)
func literalMatchesType(v any, pinType string) bool {
	if pinType == "any" {
		return true
	}
	switch pinType {
	case "number":
		switch v.(type) {
		case float64, int, int64:
			return true
		}
	case "bool":
		_, ok := v.(bool)
		return ok
	case "string":
		_, ok := v.(string)
		return ok
	case "point":
		m, ok := v.(map[string]any)
		if !ok {
			return false
		}
		_, hasX := m["x"]
		_, hasY := m["y"]
		return hasX && hasY
	case "geometry":
		// Geometry JSON: {pct:{x,y,w,h}, overrides:[...]} or at minimum a map.
		_, ok := v.(map[string]any)
		return ok
	}
	return false
}
