package nodes31

import (
	"testing"

	"github.com/yottaapp/yotta/internal/datatype"
	"github.com/yottaapp/yotta/internal/nodecontract"
)

func TestMatchTemplateIsExplicitPullAnalysisWithoutExecPins(t *testing.T) {
	builtins, err := Build()
	if err != nil {
		t.Fatal(err)
	}
	definition, ok := builtins.Definition(MatchTemplateNodeID)
	if !ok {
		t.Fatal("match template definition is missing")
	}
	machine := definition.Contract.Machine()
	if machine.Execution.Class != nodecontract.ExecutionEffect || machine.Execution.Evaluation != nodecontract.EvaluationPull ||
		machine.Execution.Determinism != nodecontract.Recorded || len(machine.Execution.Effects) != 1 || machine.Execution.Effects[0] != MatchTemplateEffectID {
		t.Fatalf("execution = %#v", machine.Execution)
	}
	if len(machine.Ports.ExecInputs)+len(machine.Ports.ExecOutputs)+len(machine.Ports.ErrorOutputs) != 0 {
		t.Fatalf("analysis node invented signal pins: %#v", machine.Ports)
	}
	if len(machine.Ports.DataInputs) != 4 || len(machine.Ports.DataOutputs) != 4 {
		t.Fatalf("ports = %#v", machine.Ports)
	}
	for _, index := range []int{0, 1} {
		expression := machine.Ports.DataInputs[index].Type
		if expression.Kind != datatype.TypeExpressionRef || expression.Ref == nil || expression.Ref.TypeID != ImageTypeID {
			t.Fatalf("image input %d = %#v", index, expression)
		}
	}
	if machine.Ports.DataInputs[2].ID != "region" || machine.Ports.DataInputs[3].ID != "threshold" ||
		machine.Ports.DataInputs[2].Default == nil || machine.Ports.DataInputs[3].Default == nil {
		t.Fatalf("analysis controls = %#v", machine.Ports.DataInputs)
	}
	if len(machine.CapabilityRequirements) != 1 || machine.CapabilityRequirements[0].ID != "blob-read" ||
		machine.CapabilityRequirements[0].Capability.CapabilityID != BlobReadCapabilityID {
		t.Fatalf("requirements = %#v", machine.CapabilityRequirements)
	}
	if definition.Implementation.Entrypoint != "vision.match-template" {
		t.Fatalf("implementation = %#v", definition.Implementation)
	}
}
