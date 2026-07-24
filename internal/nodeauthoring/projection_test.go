package nodeauthoring_test

import (
	"bytes"
	"testing"

	"github.com/yottaapp/yotta/internal/datatype"
	"github.com/yottaapp/yotta/internal/nodeauthoring"
	"github.com/yottaapp/yotta/internal/nodes"
)

func TestProjectionDerivesEditorFactsFromTrustedContracts(t *testing.T) {
	builtins, err := nodes.Build()
	if err != nil {
		t.Fatal(err)
	}
	input := nodeauthoring.Input{
		Catalog: builtins.Catalog, Types: builtins.Types, Capabilities: builtins.Capabilities,
		Contracts: builtins.Contracts, GeneratorVersion: "v1",
	}
	projection, err := nodeauthoring.Project(input)
	if err != nil {
		t.Fatal(err)
	}
	if !projection.Valid() || projection.CatalogHash() != builtins.Catalog.Hash() {
		t.Fatal("projection did not bind the trusted machine Catalog")
	}
	opened, err := nodeauthoring.Open(projection.Bytes(), input)
	if err != nil || opened.Digest() != projection.Digest() || !bytes.Equal(opened.Bytes(), projection.Bytes()) {
		t.Fatalf("strict open changed projection identity: %v", err)
	}

	concat, ok := projection.Node(nodes.ConcatNodeID)
	if !ok || len(concat.DataInputs) != 2 || len(concat.DataOutputs) != 1 || len(concat.Signals) != 0 {
		t.Fatalf("concat projection = %#v", concat)
	}
	if concat.DataInputs[0].ID != "a" || concat.DataInputs[0].Binding != nodeauthoring.BindingRequired ||
		concat.DataInputs[0].Type.TitleKey != "type.core.string.title" ||
		concat.DataInputs[0].Type.Lifecycle != nodeauthoring.LifecycleDurable {
		t.Fatalf("concat input projection = %#v", concat.DataInputs[0])
	}
	matchTemplate, ok := projection.Node(nodes.MatchTemplateNodeID)
	if !ok || len(matchTemplate.DataInputs) < 2 || matchTemplate.DataInputs[1].ID != "template" ||
		matchTemplate.DataInputs[1].EditorAdapter != "template-image" || matchTemplate.DataInputs[1].TitleKey == "" ||
		matchTemplate.DataInputs[1].DescriptionKey == "" {
		t.Fatalf("template port authoring projection = %#v", matchTemplate.DataInputs)
	}
	clickTemplate, ok := projection.Node(nodes.ClickTemplateNodeID)
	if !ok || len(clickTemplate.Capabilities) != 3 || len(clickTemplate.ConfigFields) != 1 ||
		clickTemplate.ConfigFields[0].ID != "slot" || clickTemplate.DataInputs[0].ID != "template" ||
		clickTemplate.DataInputs[0].EditorAdapter != "template-image" {
		t.Fatalf("click template authoring projection = %#v", clickTemplate)
	}
	boundTargets := 0
	for _, requirement := range clickTemplate.Capabilities {
		if requirement.TargetSlot == "target" {
			boundTargets++
			if requirement.TargetSlotConfigKey != "slot" {
				t.Fatalf("click template target binding = %#v", requirement)
			}
		}
	}
	if boundTargets != 2 {
		t.Fatalf("click template target capabilities = %#v", clickTemplate.Capabilities)
	}
	if len(concat.ConfigFields) != 0 || concat.Availability != nodeauthoring.AvailabilityPortable {
		t.Fatalf("concat config/availability = %#v / %q", concat.ConfigFields, concat.Availability)
	}

	conversion, ok := projection.Node(nodes.StreamToBlobNodeID)
	if !ok || len(conversion.ConfigFields) != 1 || len(conversion.Capabilities) != 2 {
		t.Fatalf("conversion projection = %#v", conversion)
	}
	mediaType := conversion.ConfigFields[0]
	if mediaType.ID != "mediaType" || mediaType.TitleKey != "node.conversion.streamToBlob.config.mediaType.title" ||
		mediaType.DescriptionKey != "node.conversion.streamToBlob.config.mediaType.description" ||
		mediaType.Control != nodeauthoring.ControlText || !mediaType.Required ||
		mediaType.Constraints.MinLength == nil || *mediaType.Constraints.MinLength != 3 ||
		mediaType.Constraints.MaxLength == nil || *mediaType.Constraints.MaxLength != 255 || mediaType.Constraints.Pattern == "" {
		t.Fatalf("mediaType field = %#v", mediaType)
	}
	if conversion.Availability != nodeauthoring.AvailabilityTargetRequired ||
		conversion.DataInputs[0].Carrier != nodeauthoring.CarrierRuntime ||
		conversion.DataOutputs[0].Carrier != nodeauthoring.CarrierDurable {
		t.Fatalf("conversion availability/carriers = %#v", conversion)
	}
	binary, ok := projection.Type(nodes.BinaryTypeID)
	if !ok || binary.Lifecycle != nodeauthoring.LifecycleMixed || len(binary.Representations) != 2 {
		t.Fatalf("binary type projection = %#v", binary)
	}
	integer, ok := projection.Type(nodes.IntegerTypeID)
	if !ok || len(integer.AssignableTo) != 1 || integer.AssignableTo[0].TypeID != nodes.NumberTypeID || len(integer.Traits) == 0 {
		t.Fatalf("integer type relations = %#v", integer)
	}
	stateTypes := 0
	for _, definition := range builtins.Types {
		projected, ok := projection.Type(definition.TypeRef().TypeID)
		if !ok {
			t.Fatalf("missing projected type %q", definition.TypeRef().TypeID)
		}
		if !builtins.Catalog.TypeSystem().HasTrait(definition.TypeRef(), datatype.TraitDurable) {
			if len(projected.StateInitial) != 0 {
				t.Fatalf("non-durable type %q has a state initializer", definition.TypeRef().TypeID)
			}
			continue
		}
		if len(projected.StateInitial) == 0 {
			t.Fatalf("durable inline type %q has no state initializer", definition.TypeRef().TypeID)
		}
		if _, err := datatype.SealInlineJSON(
			builtins.Catalog,
			datatype.RefResolvedType(definition.TypeRef()),
			projected.StateInitial,
		); err != nil {
			t.Fatalf("state initializer for %q: %v", definition.TypeRef().TypeID, err)
		}
		stateTypes++
	}
	if stateTypes == 0 {
		t.Fatal("projection did not expose any state-initializable data types")
	}
	stateRead, ok := projection.Node(nodes.StateReadNodeID)
	if !ok || len(stateRead.StateAccesses) != 1 || stateRead.StateAccesses[0].Mode != "read" ||
		stateRead.StateAccesses[0].SlotConfigKey != "variable" || stateRead.StateAccesses[0].Type.Label != "$T" {
		t.Fatalf("state read projection = %#v", stateRead)
	}
	nodes := projection.Nodes()
	if len(nodes) == 0 || len(nodes[0].Tags) == 0 {
		t.Fatal("projection did not enumerate nodes")
	}
	nodes[0].Tags[0] = "mutated"
	unchanged, ok := projection.Node(nodes[0].NodeRef.NodeTypeID)
	if !ok || unchanged.Tags[0] == "mutated" {
		t.Fatal("node enumeration leaked mutable trusted projection state")
	}
}

func TestProjectionRejectsIncompletePresentationAndTampering(t *testing.T) {
	builtins, err := nodes.Build()
	if err != nil {
		t.Fatal(err)
	}
	input := nodeauthoring.Input{
		Catalog: builtins.Catalog, Types: builtins.Types, Capabilities: builtins.Capabilities,
		Contracts: builtins.Contracts, GeneratorVersion: "v1",
	}
	missing := input
	missing.Contracts = missing.Contracts[:len(missing.Contracts)-1]
	if _, err := nodeauthoring.Project(missing); err == nil {
		t.Fatal("accepted an authoring projection missing a Catalog node")
	}
	projection, err := nodeauthoring.Project(input)
	if err != nil {
		t.Fatal(err)
	}
	tampered := bytes.Replace(projection.Bytes(), []byte(`"portable"`), []byte(`"target-required"`), 1)
	if _, err := nodeauthoring.Open(tampered, input); err == nil {
		t.Fatal("accepted a tampered authoring projection")
	}
}
