package nodes

import (
	"testing"

	"github.com/yottaapp/yotta/internal/capability"
	"github.com/yottaapp/yotta/internal/datatype"
	"github.com/yottaapp/yotta/internal/nodecontract"
)

func TestCaptureWindowUsesNominalImageAndSeparateAuthorities(t *testing.T) {
	builtins, err := Build()
	if err != nil {
		t.Fatal(err)
	}
	image, ok := builtins.Catalog.LookupType(ImageTypeID)
	if !ok || len(image.Machine().Representations) != 1 || image.Machine().Representations[0].Kind != datatype.RepresentationBlobRef {
		t.Fatalf("image type = %#v", image)
	}
	definition, ok := builtins.Definition(CaptureWindowNodeID)
	if !ok {
		t.Fatal("capture window definition is missing")
	}
	machine := definition.Contract.Machine()
	if machine.Execution.Class != nodecontract.ExecutionEffect || machine.Execution.Determinism != nodecontract.Recorded ||
		len(machine.Execution.Effects) != 1 || machine.Execution.Effects[0] != CaptureWindowEffectID {
		t.Fatalf("execution = %#v", machine.Execution)
	}
	if len(machine.Ports.DataOutputs) != 1 || machine.Ports.DataOutputs[0].ID != "image" ||
		!signalIDsEqual(machine.Ports.ExecInputs, []string{"in"}) || !signalIDsEqual(machine.Ports.ExecOutputs, []string{"completed"}) ||
		!signalIDsEqual(machine.Ports.ErrorOutputs, []string{"failed"}) {
		t.Fatalf("ports = %#v", machine.Ports)
	}
	if len(machine.CapabilityRequirements) != 1 {
		t.Fatalf("requirements = %#v", machine.CapabilityRequirements)
	}
	requirements := map[string]capability.Requirement{}
	for _, requirement := range machine.CapabilityRequirements {
		requirements[requirement.ID] = requirement
	}
	if requirements["blob-write"].Capability.CapabilityID != BlobWriteCapabilityID {
		t.Fatalf("requirements = %#v", machine.CapabilityRequirements)
	}
	if len(machine.ConfiguredTargets) != 1 || machine.ConfiguredTargets[0].ID != "target" || len(machine.ConfiguredTargets[0].TargetKinds) != 3 {
		t.Fatalf("configured targets = %#v", machine.ConfiguredTargets)
	}
}
