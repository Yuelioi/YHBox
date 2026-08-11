package authoring

import (
	"strings"
	"testing"

	"github.com/yottaapp/yotta/internal/artifact"
	"github.com/yottaapp/yotta/internal/nodecontract"
)

func TestNodeContractUpgradePathFollowsAdjacentSteps(t *testing.T) {
	const nodeTypeID = "https://schemas.yotta.dev/nodes/test/chain"
	one := testNodeRef(nodeTypeID, "1.0.0", 'a')
	two := testNodeRef(nodeTypeID, "1.1.0", 'b')
	three := testNodeRef(nodeTypeID, "1.2.0", 'c')
	registry := []nodeContractMigration{
		testNodeMigration(one, two),
		testNodeMigration(two, three),
	}
	path, ok := nodeContractUpgradePath(registry, one, three)
	if !ok || len(path) != 2 || migrationToRef(path[1]) != three {
		t.Fatalf("path = %#v, ok=%v", path, ok)
	}
	if _, ok := nodeContractUpgradePath(append(registry, testNodeMigration(one, three)), one, three); ok {
		t.Fatal("ambiguous migration path was admitted")
	}
}

func TestValidateReleasedNodeContractsRejectsSilentDriftAndRemoval(t *testing.T) {
	previous := testNodeRef("https://schemas.yotta.dev/nodes/test/released", "1.0.0", 'a')
	if err := ValidateReleasedNodeContracts(
		[]nodecontract.NodeRef{previous},
		[]nodecontract.NodeRef{testNodeRef(previous.NodeTypeID, "1.0.0", 'b')},
	); err == nil || !strings.Contains(err.Error(), "without a newer stable version") {
		t.Fatalf("same-version drift error = %v", err)
	}
	if err := ValidateReleasedNodeContracts([]nodecontract.NodeRef{previous}, nil); err == nil ||
		!strings.Contains(err.Error(), "removed") {
		t.Fatalf("removal error = %v", err)
	}
}

func testNodeMigration(from, to nodecontract.NodeRef) nodeContractMigration {
	return nodeContractMigration{
		nodeTypeID: from.NodeTypeID,
		from:       from.Version, fromDigest: from.SemanticDigest.String(),
		to: to.Version, toDigest: to.SemanticDigest.String(),
		kind: nodeContractMigrationShapeCompatible,
	}
}

func testNodeRef(nodeTypeID, version string, digestByte byte) nodecontract.NodeRef {
	return nodecontract.NodeRef{
		NodeTypeID: nodeTypeID, Version: version,
		SemanticDigest: artifact.Digest("sha256:" + strings.Repeat(string(digestByte), 64)),
	}
}
