package container

import (
	"fmt"

	nodepkg "github.com/yottaapp/yotta/internal/node"
)

// validateUnknownLiteralPins: config.literal 里写了 Spec 不存在的 data-in pin 名 (LLM 笔误
// Text→Message) → UNKNOWN_LITERAL_PIN warning。现状是静默丢值 (literals.go 跳过未知 pin,
// INVALID_PIN 只查边端点不查 literal key)。
//
// descriptor-driven dynamic inputs are resolved from their declared config
// records. Package-owned literal metadata is accepted explicitly; everything
// else must match a static or materialized data-in pin.
func validateUnknownLiteralPins(c *Container, sgs []Subgraph) []ValidationError {
	return validateUnknownLiteralPinsWithRegistry(nodepkg.DefaultRegistrySnapshot(), c, sgs)
}

func validateUnknownLiteralPinsWithRegistry(registry nodepkg.RegistryReader, c *Container, sgs []Subgraph) []ValidationError {
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
			rn, ok := registry.Get(n.Kind)
			if !ok {
				continue
			}
			valid := map[string]bool{}
			for _, ip := range rn.Spec.Inputs {
				if ip.Type != nodepkg.TypeExec {
					valid[ip.Name] = true
				}
			}
			if dynamic, found := nodepkg.DynamicPortForRole(&rn.Spec, nodepkg.DynamicPortInput); found && dynamic.Shape == nodepkg.DynamicPortNameTypeRecords {
				for _, input := range ParseDynamicPortDecls(n, dynamic.ConfigKey) {
					valid[input.Name] = true
				}
			}
			for pinName := range lit {
				if !valid[pinName] && !isLiteralMetadataKey(n.Kind, pinName) {
					errs = append(errs, ValidationError{
						Severity:  SeverityWarning,
						Code:      CodeUnknownLiteralPin,
						GraphPath: graphPath,
						NodeID:    n.ID,
						Params:    map[string]any{"kind": n.Kind, "pin": pinName},
					})
				}
			}
		}
	}
	check(c.Graph.Nodes, []string{"main"})
	for i := range sgs {
		sg := &sgs[i]
		check(sg.Graph.Nodes, []string{"main", fmt.Sprintf("subgraph-%s (%s)", sg.Label, sg.ID)})
	}
	return errs
}
