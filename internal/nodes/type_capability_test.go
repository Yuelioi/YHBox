package nodes

import (
	"strings"
	"testing"

	"github.com/yottaapp/yotta/internal/nodecontract"
)

func TestTypeCapabilityClosureRejectsMissingApplicableCapabilities(t *testing.T) {
	builtins, err := Build()
	if err != nil {
		t.Fatal(err)
	}
	withoutBreak := make([]nodecontract.Contract, 0, len(builtins.Contracts)-1)
	for _, contract := range builtins.Contracts {
		if contract.NodeRef().NodeTypeID != BreakPointNodeID {
			withoutBreak = append(withoutBreak, contract)
		}
	}
	if _, err := validateTypeCapabilityClosure(builtins.Types, withoutBreak); err == nil || !strings.Contains(err.Error(), "break node is missing") {
		t.Fatalf("missing break capability error = %v", err)
	}

	withoutMath := make([]nodecontract.Contract, 0, len(builtins.Contracts))
	for _, contract := range builtins.Contracts {
		if contract.Authoring().Category != "math" {
			withoutMath = append(withoutMath, contract)
		}
	}
	if _, err := validateTypeCapabilityClosure(builtins.Types, withoutMath); err == nil || !strings.Contains(err.Error(), "no math consumer") {
		t.Fatalf("missing numeric capability error = %v", err)
	}
}

func TestTypeCapabilityMatrixDocumentsWaiversAndStructures(t *testing.T) {
	builtins, err := Build()
	if err != nil {
		t.Fatal(err)
	}
	rows, err := validateTypeCapabilityClosure(builtins.Types, builtins.Contracts)
	if err != nil {
		t.Fatal(err)
	}
	matrix := renderTypeCapabilityMatrix(rows)
	for _, required := range []string{InputClipTypeID, "recording asset library", PointTypeID, "Structure break"} {
		if !strings.Contains(matrix, required) {
			t.Fatalf("type capability matrix is missing %q", required)
		}
	}
}
