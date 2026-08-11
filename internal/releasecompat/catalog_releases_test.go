package releasecompat

import (
	"strings"
	"testing"

	"github.com/yottaapp/yotta/internal/artifact"
	"github.com/yottaapp/yotta/internal/capability"
	"github.com/yottaapp/yotta/internal/datatype"
)

func TestCatalogReleasesRequireExactReleasedRefs(t *testing.T) {
	releases := CatalogReleases{Root: t.TempDir()}
	types := []datatype.TypeRef{{
		TypeID:         "https://schemas.yotta.dev/types/test/value/v1",
		SemanticDigest: releaseDigest('a'),
	}}
	capabilities := []capability.Ref{{
		CapabilityID:   "https://schemas.yotta.dev/capabilities/test/read/v1",
		SemanticDigest: releaseDigest('b'),
	}}
	if err := releases.Freeze("4.0.0", types, capabilities); err != nil {
		t.Fatal(err)
	}
	if checked, err := releases.Check("4.0.0", types, capabilities, true); err != nil || checked != 1 {
		t.Fatalf("Check() = %d, %v", checked, err)
	}
	changed := append([]datatype.TypeRef(nil), types...)
	changed[0].SemanticDigest = releaseDigest('c')
	if _, err := releases.Check("4.0.1", changed, capabilities, false); err == nil || !strings.Contains(err.Error(), "changed semantic digest") {
		t.Fatalf("changed type error = %v", err)
	}
	if _, err := releases.Check("4.0.1", types, []capability.Ref{{
		CapabilityID:   "https://schemas.yotta.dev/capabilities/test/write/v1",
		SemanticDigest: releaseDigest('d'),
	}}, false); err == nil || !strings.Contains(err.Error(), "was removed") {
		t.Fatalf("removed capability error = %v", err)
	}
}

func releaseDigest(value byte) artifact.Digest {
	return artifact.Digest("sha256:" + strings.Repeat(string(value), 64))
}
