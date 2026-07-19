package nodes

import (
	"testing"

	"github.com/yottaapp/yotta/internal/datatype"
	"github.com/yottaapp/yotta/internal/nodecontract"
)

func TestVisionAnalysisCatalogUsesTypedDataAndOneReadAuthority(t *testing.T) {
	builtins, err := Build()
	if err != nil {
		t.Fatal(err)
	}
	for _, typeID := range []string{TemplateMatchTypeID, QRCodeTypeID, ColorRangeTypeID, ColorBlobTypeID} {
		definition, ok := builtins.Catalog.LookupType(typeID)
		if !ok || len(definition.Machine().Representations) != 1 || definition.Machine().Representations[0].Kind != datatype.RepresentationInlineJSON {
			t.Fatalf("vision type %q = %#v", typeID, definition)
		}
		if typeID == ColorRangeTypeID {
			authoring := definition.Authoring()
			if authoring.EditorAdapter != "color-range" || authoring.Importance != "primary" ||
				authoring.InlinePriority != 100 || authoring.Preset != "sample-target" {
				t.Fatalf("color range authoring = %#v", authoring)
			}
		}
	}
	nodes := []struct {
		id     string
		effect nodecontract.EffectID
		output string
	}{
		{FindTemplateMatchesNodeID, FindTemplateMatchesEffectID, "matches"},
		{CompareImagesNodeID, CompareImagesEffectID, "changed-ratio"},
		{DecodeQRNodeID, DecodeQREffectID, "codes"},
		{AnalyzeColorNodeID, AnalyzeColorEffectID, "pixel-count"},
		{FindColorBlobsNodeID, FindColorBlobsEffectID, "blobs"},
	}
	for _, want := range nodes {
		definition, ok := builtins.Definition(want.id)
		if !ok {
			t.Fatalf("missing vision node %q", want.id)
		}
		machine := definition.Contract.Machine()
		if machine.Execution.Class != nodecontract.ExecutionEffect || machine.Execution.Evaluation != nodecontract.EvaluationPull ||
			machine.Execution.Determinism != nodecontract.Recorded || len(machine.Execution.Effects) != 1 || machine.Execution.Effects[0] != want.effect {
			t.Fatalf("vision execution %q = %#v", want.id, machine.Execution)
		}
		if len(machine.Ports.ExecInputs)+len(machine.Ports.ExecOutputs)+len(machine.Ports.ErrorOutputs) != 0 {
			t.Fatalf("vision analysis %q invented signal pins: %#v", want.id, machine.Ports)
		}
		if len(machine.Ports.DataOutputs) == 0 || machine.Ports.DataOutputs[0].ID != want.output {
			t.Fatalf("vision outputs %q = %#v", want.id, machine.Ports.DataOutputs)
		}
		if len(machine.CapabilityRequirements) != 1 || machine.CapabilityRequirements[0].ID != "blob-read" ||
			machine.CapabilityRequirements[0].Capability.CapabilityID != BlobReadCapabilityID {
			t.Fatalf("vision requirements %q = %#v", want.id, machine.CapabilityRequirements)
		}
	}
}

func TestVisionCollectionOutputsUseNominalElementTypes(t *testing.T) {
	builtins, err := Build()
	if err != nil {
		t.Fatal(err)
	}
	checks := map[string]string{
		FindTemplateMatchesNodeID: TemplateMatchTypeID,
		DecodeQRNodeID:            QRCodeTypeID,
		FindColorBlobsNodeID:      ColorBlobTypeID,
	}
	for nodeID, elementTypeID := range checks {
		definition, _ := builtins.Definition(nodeID)
		expression := definition.Contract.Machine().Ports.DataOutputs[0].Type
		if expression.Kind != datatype.TypeExpressionList || expression.Element == nil || expression.Element.Kind != datatype.TypeExpressionRef ||
			expression.Element.Ref == nil || expression.Element.Ref.TypeID != elementTypeID {
			t.Fatalf("collection output %q = %#v", nodeID, expression)
		}
	}
}
