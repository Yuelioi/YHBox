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
				if !literalMatchesType(raw, pinType) {
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
	}
	check(c.Graph.Nodes, []string{"main"})
	for _, sg := range c.Subgraphs {
		check(sg.Graph.Nodes, []string{"main", fmt.Sprintf("subgraph-%s (%s)", sg.Label, sg.ID)})
	}
	return errs
}

// literalMatchesType: JSON-decoded value type vs declared PinType.
// Go's encoding/json decodes:
//   - numbers as float64
//   - bool as bool
//   - string as string
//   - point as map[string]any{x:float64, y:float64}
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
	}
	return false
}
