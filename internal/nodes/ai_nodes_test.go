package nodes

import (
	"bytes"
	"testing"

	"github.com/yottaapp/yotta/internal/capability"
	"github.com/yottaapp/yotta/internal/nodecontract"
)

func TestAINodesPinTrustedPromptManifestsIntoImplementationLocks(t *testing.T) {
	builtins, err := Build()
	if err != nil {
		t.Fatal(err)
	}
	if !builtins.AIGeneratePrompt.Valid() || !builtins.AIExtractPrompt.Valid() ||
		builtins.AIGeneratePrompt.Digest() == builtins.AIExtractPrompt.Digest() {
		t.Fatalf("AI prompt manifests = %#v / %#v", builtins.AIGeneratePrompt.Machine(), builtins.AIExtractPrompt.Machine())
	}
	for _, test := range []struct {
		name        string
		nodeID      string
		conformance string
	}{
		{name: "generate", nodeID: AIGenerateNodeID, conformance: "provider-native-text-generation/" + builtins.AIGeneratePrompt.Digest().String()},
		{name: "extract", nodeID: AIExtractNodeID, conformance: "provider-native-strict-structured-output/" + builtins.AIExtractPrompt.Digest().String()},
	} {
		t.Run(test.name, func(t *testing.T) {
			definition, ok := builtins.Definition(test.nodeID)
			if !ok {
				t.Fatal("AI definition is missing")
			}
			expected, err := builtinImplementationDigest(definition.Implementation.Entrypoint, aiImplementationVersion, test.conformance, definition.Implementation.ABI)
			if err != nil {
				t.Fatal(err)
			}
			if definition.Implementation.ArtifactDigest != expected {
				t.Fatalf("implementation digest = %s, want %s", definition.Implementation.ArtifactDigest, expected)
			}
			for _, resource := range definition.Contract.Machine().ConfigSchemaBundle {
				if bytes.Contains(resource.Schema, []byte(`"instructions"`)) {
					t.Fatalf("workflow config can override trusted instructions: %s", resource.Schema)
				}
			}
		})
	}
	for _, nodeID := range []string{AIGenerateNodeID, AIExtractNodeID} {
		definition, _ := builtins.Definition(nodeID)
		machine := definition.Contract.Machine()
		if len(machine.Ports.DataInputs) != 2 ||
			machine.Ports.DataInputs[0].ID != "prompt" ||
			machine.Ports.DataInputs[1].ID != "image" ||
			machine.Ports.DataInputs[1].Required ||
			machine.Ports.DataInputs[1].Type.Ref == nil ||
			machine.Ports.DataInputs[1].Type.Ref.TypeID != ImageTypeID {
			t.Fatalf("%s AI inputs = %#v", nodeID, machine.Ports.DataInputs)
		}
		requirements := map[string]capability.Requirement{}
		for _, requirement := range machine.CapabilityRequirements {
			requirements[requirement.ID] = requirement
		}
		if requirements["blob-read"].Capability.CapabilityID != BlobReadCapabilityID {
			t.Fatalf("%s blob read requirement = %#v", nodeID, requirements["blob-read"])
		}
		ports := definition.Contract.Authoring().Ports
		authoring := map[string]nodecontract.PortAuthoring{}
		for _, port := range ports {
			authoring[port.ID] = port
		}
		if len(ports) != 2 ||
			authoring["prompt"].EditorAdapter != "multiline-text" ||
			authoring["image"].Group != "common" {
			t.Fatalf("%s AI authoring = %#v", nodeID, ports)
		}
	}
	extract, _ := builtins.Definition(AIExtractNodeID)
	for _, resource := range extract.Contract.Machine().ConfigSchemaBundle {
		if bytes.Contains(resource.Schema, []byte(`"schema"`)) ||
			!bytes.Contains(resource.Schema, []byte(`"fields"`)) ||
			!bytes.Contains(resource.Schema, []byte(`"structured-output-fields"`)) {
			t.Fatalf("AI Extract exposes the wrong output configuration: %s", resource.Schema)
		}
	}
	if !builtins.AIAuthoringPrompt.Valid() || !builtins.AIAuthoringToolSet.Valid() {
		t.Fatal("AI authoring artifacts are missing")
	}
	if _, ok := builtins.Definition("https://schemas.yotta.dev/nodes/ai/agent"); ok {
		t.Fatal("removed bounded AI Agent is still present in the built-in catalog")
	}
}
