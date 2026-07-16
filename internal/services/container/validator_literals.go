package container

import (
	"fmt"

	nodepkg "github.com/yottaapp/yotta/internal/node"
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
// Coverage: static data-in pins and descriptor-driven dynamic inputs. Using
// the same cfg-aware resolver as edge validation keeps editor, validator and
// runtime materialization aligned.
func validateLiteralTypes(registry nodepkg.RegistryReader, c *Container, sgs []Subgraph) []ValidationError {
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
			if !knownKind(registry, n.Kind) {
				continue
			}
			for pinName, raw := range lit {
				pinType := dataInPinTypeForNode(registry, n, pinName)
				if pinType == "" {
					continue // UNKNOWN_LITERAL_PIN handles unknown stored values
				}
				if literalMatchesType(raw, pinType) {
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
	for _, sg := range sgs {
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
