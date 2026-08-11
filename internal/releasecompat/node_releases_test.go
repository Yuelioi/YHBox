package releasecompat

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/yottaapp/yotta/internal/artifact"
	"github.com/yottaapp/yotta/internal/nodecontract"
)

func TestNodeReleasesFreezeIsImmutableAndRequiresCurrent(t *testing.T) {
	root := t.TempDir()
	releases := NodeReleases{Root: root}
	refs := []nodecontract.NodeRef{releaseRef("https://schemas.yotta.dev/nodes/test", "1.0.0", 'a')}
	if err := releases.Freeze("4.0.0", refs); err != nil {
		t.Fatal(err)
	}
	if err := releases.Freeze("4.0.0", refs); err != nil {
		t.Fatalf("idempotent freeze: %v", err)
	}
	if checked, err := releases.Check("4.0.0", refs, true); err != nil || checked != 1 {
		t.Fatalf("Check() = %d, %v", checked, err)
	}
	changed := []nodecontract.NodeRef{releaseRef(refs[0].NodeTypeID, "1.0.0", 'b')}
	if err := releases.Freeze("4.0.0", changed); err == nil || !strings.Contains(err.Error(), "immutable") {
		t.Fatalf("changed freeze error = %v", err)
	}
	if _, err := releases.Check("4.0.1", refs, true); err == nil || !strings.Contains(err.Error(), "no frozen") {
		t.Fatalf("missing current release error = %v", err)
	}
}

func TestNodeReleasesRejectsChangedOrMalformedFloor(t *testing.T) {
	root := t.TempDir()
	releases := NodeReleases{Root: root}
	previous := []nodecontract.NodeRef{releaseRef("https://schemas.yotta.dev/nodes/test", "1.0.0", 'a')}
	if err := releases.Freeze("4.0.0", previous); err != nil {
		t.Fatal(err)
	}
	changed := []nodecontract.NodeRef{releaseRef(previous[0].NodeTypeID, "1.0.0", 'b')}
	if _, err := releases.Check("4.0.1", changed, false); err == nil || !strings.Contains(err.Error(), "without a newer stable version") {
		t.Fatalf("same-version drift error = %v", err)
	}
	path := filepath.Join(root, "4.0.0", builtinNodeReleaseFile)
	if err := os.WriteFile(path, []byte(`{"format":"wrong"}`), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := releases.Check("4.0.1", previous, false); err == nil {
		t.Fatal("malformed release snapshot was accepted")
	}
}

func releaseRef(nodeTypeID, version string, digestByte byte) nodecontract.NodeRef {
	return nodecontract.NodeRef{
		NodeTypeID: nodeTypeID, Version: version,
		SemanticDigest: artifact.Digest("sha256:" + strings.Repeat(string(digestByte), 64)),
	}
}
