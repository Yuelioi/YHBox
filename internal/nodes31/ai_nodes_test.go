package nodes31

import (
	"bytes"
	"testing"
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
}
