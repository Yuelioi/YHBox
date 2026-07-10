package container

import (
	"fmt"

	nodepkg "github.com/yottaapp/yotta/internal/node"
)

// validateRequiredPins: 每个节点 Spec 里 Required 的 data-in pin, 若无连入 data 边、
// config.literal 无该 key、Spec 又无非 nil Default → MISSING_REQUIRED_PIN (error)。
//
// "无值" = config.literal map 无该 key (不看 ""/0 内容; present 即满足)。深层"非空/合法"
// 归 runtime per-node Validate()。Required+Default 并存 (e.g. KeyPress.VK Default "W")
// 走"有 Default 放过"分支。
func validateRequiredPins(c *Container, sgs []Subgraph) []ValidationError {
	if c == nil {
		return nil
	}
	var errs []ValidationError
	check := func(nodes []GraphNode, edges []GraphEdge, graphPath []string) {
		for i := range nodes {
			n := &nodes[i]
			rn, ok := nodepkg.Get(n.Kind)
			if !ok {
				continue
			}
			lit, _ := n.Config["literal"].(map[string]any)
			for _, ip := range rn.Spec.Inputs {
				if ip.Type == nodepkg.TypeExec || !ip.Required {
					continue
				}
				target := n.ID + "." + ip.Name
				hasEdge := false
				for _, e := range edges {
					if e.To == target {
						hasEdge = true
						break
					}
				}
				if hasEdge {
					continue
				}
				if lit != nil {
					if _, present := lit[ip.Name]; present {
						continue
					}
				}
				if _, present := n.Config[ip.Name]; present { // top-level config (mirror PinValue fallback)
					continue
				}
				if ip.Default != nil {
					continue
				}
				errs = append(errs, ValidationError{
					Severity:  SeverityError,
					Code:      CodeMissingRequiredPin,
					GraphPath: graphPath,
					NodeID:    n.ID,
					Params:    map[string]any{"kind": n.Kind, "pin": ip.Name},
				})
			}
		}
	}
	check(c.Graph.Nodes, c.Graph.Edges, []string{"main"})
	for i := range sgs {
		sg := &sgs[i]
		check(sg.Graph.Nodes, sg.Graph.Edges, []string{"main", fmt.Sprintf("subgraph-%s (%s)", sg.Label, sg.ID)})
	}
	return errs
}
