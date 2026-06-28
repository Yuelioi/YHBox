package container

import (
	"fmt"
	"strings"

	"yotta/internal/automation/controller"
	"yotta/internal/automation/target"
	nodepkg "yotta/internal/node"
)

func validateTargetCapabilities(c *Container, sgs []Subgraph) []ValidationError {
	var errs []ValidationError
	errs = append(errs, validateGraphTargetCapabilities(c.Graph, []string{"main"})...)
	for i := range sgs {
		sg := &sgs[i]
		errs = append(errs, validateGraphTargetCapabilities(sg.Graph, []string{"main", fmt.Sprintf("subgraph-%s (%s)", sg.Label, sg.ID)})...)
	}
	return errs
}

func validateGraphTargetCapabilities(g Graph, graphPath []string) []ValidationError {
	var errs []ValidationError
	nodeByID := graphNodeByID(g)
	for i := range g.Nodes {
		n := &g.Nodes[i]
		rn, ok := nodepkg.Get(n.Kind)
		if !ok {
			continue
		}
		requiredCaps := targetCapabilitiesForNode(n, rn.Spec.TargetCapabilities)
		if len(requiredCaps) == 0 {
			continue
		}
		if windowPinWired(g, n.ID) {
			continue
		}
		targetKind, ok := nearestUpstreamTargetKind(g, nodeByID, n.ID)
		if !ok {
			continue
		}
		profile, ok := controller.DefaultProfileForTargetKind(targetKind)
		if !ok {
			continue
		}
		for _, cap := range requiredCaps {
			if profile.Capabilities.Has(controller.Capability(cap)) {
				continue
			}
			errs = append(errs, ValidationError{
				Severity:  SeverityError,
				Code:      CodeUnsupportedTargetCapability,
				GraphPath: graphPath,
				NodeID:    n.ID,
				Params: map[string]any{
					"kind":       n.Kind,
					"targetKind": targetKind,
					"capability": string(cap),
				},
			})
		}
	}
	return errs
}

func targetCapabilitiesForNode(n *GraphNode, base []nodepkg.TargetCapability) []nodepkg.TargetCapability {
	caps := append([]nodepkg.TargetCapability(nil), base...)
	switch n.Kind {
	case "ClickAt", "ClickTemplate":
		if strings.TrimSpace(PinString(n, "Keys")) != "" {
			caps = appendTargetCapability(caps, nodepkg.TargetCapabilityKeyState)
		}
		switch PinString(n, "Button") {
		case "right", "middle":
			caps = appendTargetCapability(caps, nodepkg.TargetCapabilityMouseButton)
		}
	}
	return caps
}

func appendTargetCapability(caps []nodepkg.TargetCapability, cap nodepkg.TargetCapability) []nodepkg.TargetCapability {
	for _, existing := range caps {
		if existing == cap {
			return caps
		}
	}
	return append(caps, cap)
}

func graphNodeByID(g Graph) map[string]*GraphNode {
	out := make(map[string]*GraphNode, len(g.Nodes))
	for i := range g.Nodes {
		out[g.Nodes[i].ID] = &g.Nodes[i]
	}
	return out
}

func nearestUpstreamTargetKind(g Graph, nodeByID map[string]*GraphNode, nodeID string) (string, bool) {
	visited := map[string]bool{nodeID: true}
	frontier := []string{nodeID}
	for len(frontier) > 0 {
		found := map[string]struct{}{}
		var next []string
		for _, current := range frontier {
			for _, e := range g.Edges {
				toID, toPin := splitRef(e.To)
				if toID != current {
					continue
				}
				toNode := nodeByID[toID]
				if !nodeHasExecInPin(toNode, toPin) {
					continue
				}
				fromID, fromPin := splitRef(e.From)
				fromNode := nodeByID[fromID]
				if fromNode == nil || !nodeHasExecOutPin(fromNode, fromPin) {
					continue
				}
				if kind, ok := targetKindForSelectionNode(fromNode.Kind); ok {
					found[kind] = struct{}{}
					continue
				}
				if !visited[fromID] {
					visited[fromID] = true
					next = append(next, fromID)
				}
			}
		}
		if len(found) == 1 {
			for kind := range found {
				return kind, true
			}
		}
		if len(found) > 1 {
			return "", false
		}
		frontier = next
	}
	return "", false
}

func nodeHasExecInPin(n *GraphNode, pin string) bool {
	if n == nil {
		return false
	}
	rn, ok := nodepkg.Get(n.Kind)
	if !ok {
		return false
	}
	for _, in := range rn.Spec.Inputs {
		if in.Name == pin && in.Type == nodepkg.TypeExec {
			return true
		}
	}
	return false
}

func targetKindForSelectionNode(kind string) (string, bool) {
	switch kind {
	case "Win32WindowTarget":
		return target.KindWin32Window, true
	case "AndroidTarget":
		return target.KindAndroidADB, true
	case "BrowserTarget":
		return target.KindBrowserCDP, true
	default:
		return "", false
	}
}
