package authoring

import (
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"strings"

	"github.com/yottaapp/yotta/internal/artifact"
	"github.com/yottaapp/yotta/internal/nodecontract"
	"github.com/yottaapp/yotta/internal/workflow/schema"
)

const (
	aiExtractNodeTypeID        = "https://schemas.yotta.dev/nodes/ai/extract"
	aiGenerateNodeTypeID       = "https://schemas.yotta.dev/nodes/ai/generate"
	clickTemplateNodeTypeID    = "https://schemas.yotta.dev/nodes/automation/click-template"
	movePointerNodeTypeID      = "https://schemas.yotta.dev/nodes/automation/move-pointer"
	playInputClipNodeTypeID    = "https://schemas.yotta.dev/nodes/automation/play-input-clip"
	waitTemplateNodeTypeID     = "https://schemas.yotta.dev/nodes/automation/wait-template"
	waitTemplateGoneNodeTypeID = "https://schemas.yotta.dev/nodes/automation/wait-template-gone"
	currentNodeContractV100    = "1.0.0"
	currentNodeContractV110    = "1.1.0"
	legacyNodeContractV100     = "1.0.0"
	movePointerLegacyDuration  = "300"
	movePointerLegacyMotion    = `"linear"`
)

type nodeContractMigrationKind uint8

const (
	nodeContractMigrationShapeCompatible nodeContractMigrationKind = iota
	nodeContractMigrationRetractPlayInputClipScale
	nodeContractMigrationPreserveMovePointerDefaults
)

type nodeContractMigration struct {
	nodeTypeID string
	from       string
	fromDigest string
	to         string
	toDigest   string
	kind       nodeContractMigrationKind
}

// nodeContractMigrations is intentionally an exact, finite registry. A new
// digest or non-adjacent version is unsupported until its migration is added
// with a frozen regression fixture.
var nodeContractMigrations = [...]nodeContractMigration{
	{
		nodeTypeID: clickTemplateNodeTypeID,
		from:       legacyNodeContractV100,
		fromDigest: "sha256:370ee214c1f0e99d149a5f709019e74480f9054442e058ae6349054997f72c1d",
		to:         currentNodeContractV100,
		toDigest:   "sha256:16babb2401a04127a949e0b855179ce03233a89dc79e63e93f8643e416e208b6",
		kind:       nodeContractMigrationShapeCompatible,
	},
	{
		nodeTypeID: waitTemplateNodeTypeID,
		from:       legacyNodeContractV100,
		fromDigest: "sha256:cd4f4177c886c36ba2ede3634b67b87c4d22597fa86e21269cae9bc38cc2066e",
		to:         currentNodeContractV100,
		toDigest:   "sha256:2c28ed5b6ff228dd5d5e083b368f59a73a92a265364df07f8bdc355e9c282435",
		kind:       nodeContractMigrationShapeCompatible,
	},
	{
		nodeTypeID: waitTemplateGoneNodeTypeID,
		from:       legacyNodeContractV100,
		fromDigest: "sha256:28eaccad35ee50f132dce3e2a45a85ad09d2c59d99c48dcdd42f9f7d3fb80253",
		to:         currentNodeContractV100,
		toDigest:   "sha256:a5366ec1e69d656de1b3c714e432420c0a1d65b83c01ff3fd235c50834a3d62b",
		kind:       nodeContractMigrationShapeCompatible,
	},
	{
		nodeTypeID: movePointerNodeTypeID,
		from:       legacyNodeContractV100,
		fromDigest: "sha256:2bf1f8059f1269e407d2aedf4f717cc6c0b860eb46b92abd1794a3aa3bf559af",
		to:         currentNodeContractV110,
		toDigest:   "sha256:0e632e15564076e0292a3da0672c7e7a5cd852d19fd9603acab0c1fd568aec2a",
		kind:       nodeContractMigrationPreserveMovePointerDefaults,
	},
	{
		nodeTypeID: playInputClipNodeTypeID,
		from:       legacyNodeContractV100,
		fromDigest: "sha256:ff7ea9d0b2ca91cb2062cff30dd5ca8575555ec5363b4c76e746925ee6ae027b",
		to:         currentNodeContractV110,
		toDigest:   "sha256:abc200829a50376c1ad914f0ae25b1dea61a874a824a49f850a5ed4839db16a8",
		kind:       nodeContractMigrationRetractPlayInputClipScale,
	},
	{
		nodeTypeID: playInputClipNodeTypeID,
		from:       legacyNodeContractV100,
		fromDigest: "sha256:5c353fb0725ca6a841a7ef5e9adcca12bb10e2d6362fed4d7d38449a58608e02",
		to:         currentNodeContractV110,
		toDigest:   "sha256:abc200829a50376c1ad914f0ae25b1dea61a874a824a49f850a5ed4839db16a8",
		kind:       nodeContractMigrationShapeCompatible,
	},
	{
		nodeTypeID: playInputClipNodeTypeID,
		from:       legacyNodeContractV100,
		fromDigest: "sha256:bab93b5e1f655e3f5e23c254139b92a23b048c93d3d212ff7f32d2dd009e0d75",
		to:         currentNodeContractV110,
		toDigest:   "sha256:abc200829a50376c1ad914f0ae25b1dea61a874a824a49f850a5ed4839db16a8",
		kind:       nodeContractMigrationShapeCompatible,
	},
	{
		nodeTypeID: playInputClipNodeTypeID,
		from:       legacyNodeContractV100,
		fromDigest: "sha256:0d6e75a9c06ef29cc8bdeb79f7f79f420461d9abb05ad4a9196d74324c7d2545",
		to:         currentNodeContractV110,
		toDigest:   "sha256:abc200829a50376c1ad914f0ae25b1dea61a874a824a49f850a5ed4839db16a8",
		kind:       nodeContractMigrationShapeCompatible,
	},
	{
		nodeTypeID: aiExtractNodeTypeID,
		from:       legacyNodeContractV100,
		fromDigest: "sha256:dbcb528cb623272c3a7544c1a2ff6ed2e77c14dba1d795b6fea9511f87d99646",
		to:         currentNodeContractV110,
		toDigest:   "sha256:05a063bb119608c66afac903198c60ac8763dafd732e2c4af550693a998d70af",
		kind:       nodeContractMigrationShapeCompatible,
	},
	{
		nodeTypeID: aiExtractNodeTypeID,
		from:       legacyNodeContractV100,
		fromDigest: "sha256:eee97d21d98e56ffec8d7e1cf4cd6b4ccc667394e5419f0b3ad0eab465d79a85",
		to:         currentNodeContractV110,
		toDigest:   "sha256:05a063bb119608c66afac903198c60ac8763dafd732e2c4af550693a998d70af",
		kind:       nodeContractMigrationShapeCompatible,
	},
	{
		nodeTypeID: aiGenerateNodeTypeID,
		from:       legacyNodeContractV100,
		fromDigest: "sha256:28e1267b308079ea892e23b3e89c0d97c1a6ddd81891cb42eb4157ee9a2af30a",
		to:         currentNodeContractV110,
		toDigest:   "sha256:00f2342d44deca9db66ab5d43c80d2484216a207bc7c8be6edfe088cfead9fc1",
		kind:       nodeContractMigrationShapeCompatible,
	},
}

func admittedNodeContractUpgradePath(from, to nodecontract.NodeRef) ([]nodeContractMigration, bool) {
	return nodeContractUpgradePath(nodeContractMigrations[:], from, to)
}

func nodeContractUpgradePath(
	registry []nodeContractMigration,
	from, to nodecontract.NodeRef,
) ([]nodeContractMigration, bool) {
	if from == to {
		return []nodeContractMigration{}, true
	}
	if from.NodeTypeID == "" || from.NodeTypeID != to.NodeTypeID {
		return nil, false
	}
	path := make([]nodeContractMigration, 0, len(registry))
	seen := make(map[nodecontract.NodeRef]struct{}, len(registry))
	cursor := from
	for cursor != to {
		if _, cycle := seen[cursor]; cycle {
			return nil, false
		}
		seen[cursor] = struct{}{}
		var next nodeContractMigration
		matches := 0
		for _, migration := range registry {
			if cursor == migrationFromRef(migration) {
				next = migration
				matches++
			}
		}
		if matches != 1 {
			return nil, false
		}
		path = append(path, next)
		cursor = migrationToRef(next)
		if len(path) > len(registry) {
			return nil, false
		}
	}
	return path, true
}

func migrationFromRef(migration nodeContractMigration) nodecontract.NodeRef {
	return nodecontract.NodeRef{
		NodeTypeID:     migration.nodeTypeID,
		Version:        migration.from,
		SemanticDigest: nodecontractDigest(migration.fromDigest),
	}
}

func migrationToRef(migration nodeContractMigration) nodecontract.NodeRef {
	return nodecontract.NodeRef{
		NodeTypeID:     migration.nodeTypeID,
		Version:        migration.to,
		SemanticDigest: nodecontractDigest(migration.toDigest),
	}
}

func nodecontractDigest(value string) artifact.Digest {
	return artifact.Digest(value)
}

// ValidateReleasedNodeContracts is the release-gate interface for built-in
// NodeRefs. A released type must still exist; a changed semantic digest must
// use a strictly newer stable node version and have a complete adjacent
// migration path to the current Catalog ref.
func ValidateReleasedNodeContracts(released, current []nodecontract.NodeRef) error {
	currentByType := make(map[string]nodecontract.NodeRef, len(current))
	for _, ref := range current {
		if err := validateReleaseNodeRef(ref); err != nil {
			return fmt.Errorf("current node contract: %w", err)
		}
		if _, duplicate := currentByType[ref.NodeTypeID]; duplicate {
			return fmt.Errorf("current node contract %q is duplicated", ref.NodeTypeID)
		}
		currentByType[ref.NodeTypeID] = ref
	}
	releasedByType := make(map[string]struct{}, len(released))
	for _, previous := range released {
		if err := validateReleaseNodeRef(previous); err != nil {
			return fmt.Errorf("released node contract: %w", err)
		}
		if _, duplicate := releasedByType[previous.NodeTypeID]; duplicate {
			return fmt.Errorf("released node contract %q is duplicated", previous.NodeTypeID)
		}
		releasedByType[previous.NodeTypeID] = struct{}{}
		next, found := currentByType[previous.NodeTypeID]
		if !found {
			return fmt.Errorf("released node contract %q was removed without a supported replacement", previous.NodeTypeID)
		}
		if previous == next {
			continue
		}
		if compareStableNodeVersion(next.Version, previous.Version) <= 0 {
			return fmt.Errorf(
				"released node contract %q changed semantic digest without a newer stable version (%s -> %s)",
				previous.NodeTypeID, previous.Version, next.Version,
			)
		}
		if _, ok := admittedNodeContractUpgradePath(previous, next); !ok {
			return fmt.Errorf(
				"released node contract %q has no migration path from %s/%s to %s/%s",
				previous.NodeTypeID, previous.Version, previous.SemanticDigest, next.Version, next.SemanticDigest,
			)
		}
	}
	return nil
}

func validateReleaseNodeRef(ref nodecontract.NodeRef) error {
	if strings.TrimSpace(ref.NodeTypeID) == "" || !ref.SemanticDigest.Valid() {
		return errors.New("node ref identity is invalid")
	}
	if _, ok := stableNodeVersion(ref.Version); !ok {
		return fmt.Errorf("node %q version %q is not stable numeric SemVer", ref.NodeTypeID, ref.Version)
	}
	return nil
}

func compareStableNodeVersion(left, right string) int {
	a, aOK := stableNodeVersion(left)
	b, bOK := stableNodeVersion(right)
	if !aOK || !bOK {
		return 0
	}
	for index := range a {
		if a[index] < b[index] {
			return -1
		}
		if a[index] > b[index] {
			return 1
		}
	}
	return 0
}

func stableNodeVersion(value string) ([3]uint64, bool) {
	var result [3]uint64
	parts := strings.Split(value, ".")
	if len(parts) != len(result) {
		return result, false
	}
	for index, part := range parts {
		if part == "" || len(part) > 1 && part[0] == '0' {
			return result, false
		}
		parsed, err := strconv.ParseUint(part, 10, 64)
		if err != nil {
			return result, false
		}
		result[index] = parsed
	}
	return result, true
}

func prepareAdmittedNodeContractUpgrade(migration nodeContractMigration, graph *schema.Graph, node *schema.Node) error {
	switch migration.kind {
	case nodeContractMigrationShapeCompatible:
		return nil
	case nodeContractMigrationRetractPlayInputClipScale:
		delete(node.Bindings, "turn-scale")
		return nil
	case nodeContractMigrationPreserveMovePointerDefaults:
		if node.Bindings == nil {
			node.Bindings = make(map[string]schema.InputBinding)
		}
		materializeLegacyDefault(graph, node, "duration", json.RawMessage(movePointerLegacyDuration))
		materializeLegacyDefault(graph, node, "motion", json.RawMessage(movePointerLegacyMotion))
		return nil
	default:
		return fmt.Errorf("node contract migration kind %d is unsupported", migration.kind)
	}
}

func materializeLegacyDefault(graph *schema.Graph, node *schema.Node, portID string, value json.RawMessage) {
	binding, bound := node.Bindings[portID]
	if bound && binding.Kind != schema.BindingDefault {
		return
	}
	if !bound && hasIncomingData(graph, node.ID, portID) {
		return
	}
	node.Bindings[portID] = schema.InputBinding{Kind: schema.BindingValue, Value: value}
}

func hasIncomingData(graph *schema.Graph, nodeID, portID string) bool {
	for _, edge := range graph.Edges {
		if edge.Channel == schema.EdgeData && edge.To.NodeID == nodeID && edge.To.PortID == portID {
			return true
		}
	}
	for _, input := range graph.Inputs {
		if input.NodeID == nodeID && input.PortID == portID {
			return true
		}
	}
	return false
}
