package container

import (
	"fmt"
	"strings"

	"github.com/yottaapp/yotta/internal/automation/controller"
	"github.com/yottaapp/yotta/internal/automation/target"
	nodepkg "github.com/yottaapp/yotta/internal/node"
)

func validateTargetCapabilities(registry nodepkg.RegistryReader, c *Container, sgs []Subgraph) []ValidationError {
	var errs []ValidationError
	errs = append(errs, validateGraphTargetCapabilities(registry, c.Graph, []string{"main"}, "")...)
	for i := range sgs {
		sg := &sgs[i]
		errs = append(errs, validateGraphTargetCapabilities(registry, sg.Graph, []string{"main", fmt.Sprintf("subgraph-%s (%s)", sg.Label, sg.ID)}, "")...)
	}
	errs = append(errs, validateSubgraphCallTargetCapabilities(registry, c.Graph, []string{"main"}, "", subgraphsByID(sgs), map[string]bool{})...)
	return errs
}

func validateGraphTargetCapabilities(registry nodepkg.RegistryReader, g Graph, graphPath []string, inheritedTargetKind string) []ValidationError {
	var errs []ValidationError
	nodeByID := graphNodeByID(g)
	for i := range g.Nodes {
		n := &g.Nodes[i]
		rn, ok := registry.Get(n.Kind)
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
		targetKind, ok := nearestUpstreamTargetKind(registry, g, nodeByID, n.ID)
		if !ok {
			targetKind = inheritedTargetKind
			ok = targetKind != ""
		}
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

func validateSubgraphCallTargetCapabilities(registry nodepkg.RegistryReader, g Graph, graphPath []string, inheritedTargetKind string, sgs map[string]*Subgraph, seen map[string]bool) []ValidationError {
	var errs []ValidationError
	nodeByID := graphNodeByID(g)
	for i := range g.Nodes {
		n := &g.Nodes[i]
		if !isSubgraphCallKind(n.Kind) {
			continue
		}
		targetKind, ok := nearestUpstreamTargetKind(registry, g, nodeByID, n.ID)
		if !ok {
			targetKind = inheritedTargetKind
			ok = targetKind != ""
		}
		if !ok {
			continue
		}
		sgID := PinString(n, "SubgraphID")
		sg := sgs[sgID]
		if sg == nil {
			continue
		}
		key := sg.ID + "|" + targetKind
		if seen[key] {
			continue
		}
		seen[key] = true
		sgPath := append(append([]string(nil), graphPath...), fmt.Sprintf("subgraph-%s (%s)", sg.Label, sg.ID))
		errs = append(errs, validateGraphTargetCapabilities(registry, sg.Graph, sgPath, targetKind)...)
		errs = append(errs, validateSubgraphCallTargetCapabilities(registry, sg.Graph, sgPath, targetKind, sgs, seen)...)
	}
	return errs
}

func isSubgraphCallKind(kind string) bool {
	return kind == "Subgraph" || kind == "CollapsedNode"
}

func subgraphsByID(sgs []Subgraph) map[string]*Subgraph {
	out := make(map[string]*Subgraph, len(sgs))
	for i := range sgs {
		out[sgs[i].ID] = &sgs[i]
	}
	return out
}

func targetCapabilitiesForNode(n *GraphNode, base []nodepkg.TargetCapability) []nodepkg.TargetCapability {
	caps := append([]nodepkg.TargetCapability(nil), base...)
	switch n.Kind {
	case "ClickTemplate":
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

func nearestUpstreamTargetKind(registry nodepkg.RegistryReader, g Graph, nodeByID map[string]*GraphNode, nodeID string) (string, bool) {
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
				if !nodeHasExecInPin(registry, toNode, toPin) {
					continue
				}
				fromID, fromPin := splitRef(e.From)
				fromNode := nodeByID[fromID]
				if fromNode == nil || !nodeHasExecOutPinWithRegistry(registry, fromNode, fromPin) {
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

func nodeHasExecInPin(registry nodepkg.RegistryReader, n *GraphNode, pin string) bool {
	if n == nil {
		return false
	}
	rn, ok := registry.Get(n.Kind)
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
	default:
		return "", false
	}
}
