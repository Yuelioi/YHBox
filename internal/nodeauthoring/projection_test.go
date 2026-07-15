package nodeauthoring_test

import (
	"bytes"
	"testing"

	"github.com/yottaapp/yotta/internal/nodeauthoring"
	"github.com/yottaapp/yotta/internal/nodes31"
)

func TestProjectionDerivesEditorFactsFromTrustedContracts(t *testing.T) {
	builtins, err := nodes31.Build()
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

	concat, ok := projection.Node(nodes31.ConcatNodeID)
	if !ok || len(concat.DataInputs) != 2 || len(concat.DataOutputs) != 1 || len(concat.Signals) != 0 {
		t.Fatalf("concat projection = %#v", concat)
	}
	if concat.DataInputs[0].ID != "a" || concat.DataInputs[0].Binding != nodeauthoring.BindingRequired ||
		concat.DataInputs[0].Type.TitleKey != "type.core.string.title" ||
		concat.DataInputs[0].Type.Lifecycle != nodeauthoring.LifecycleDurable {
		t.Fatalf("concat input projection = %#v", concat.DataInputs[0])
	}
	if len(concat.ConfigFields) != 0 || concat.Availability != nodeauthoring.AvailabilityPortable {
		t.Fatalf("concat config/availability = %#v / %q", concat.ConfigFields, concat.Availability)
	}

	conversion, ok := projection.Node(nodes31.StreamToBlobNodeID)
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
	binary, ok := projection.Type(nodes31.BinaryTypeID)
	if !ok || binary.Lifecycle != nodeauthoring.LifecycleMixed || len(binary.Representations) != 2 {
		t.Fatalf("binary type projection = %#v", binary)
	}
}

func TestProjectionRejectsIncompletePresentationAndTampering(t *testing.T) {
	builtins, err := nodes31.Build()
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
