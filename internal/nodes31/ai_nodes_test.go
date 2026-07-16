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
	if !builtins.AIAgentPrompt.Valid() || !builtins.AIAgentToolSet.Valid() {
		t.Fatal("AI Agent artifacts are missing")
	}
	if !builtins.AIAuthoringPrompt.Valid() || !builtins.AIAuthoringToolSet.Valid() {
		t.Fatal("AI authoring artifacts are missing")
	}
	agent, ok := builtins.Definition(AIAgentNodeID)
	if !ok {
		t.Fatal("AI Agent definition is missing")
	}
	expected, err := builtinImplementationDigest(
		agent.Implementation.Entrypoint, "v1",
		"provider-native-bounded-agent/"+builtins.AIAgentPrompt.Digest().String()+"/"+builtins.AIAgentToolSet.Digest().String(),
		agent.Implementation.ABI,
	)
	if err != nil {
		t.Fatal(err)
	}
	machine := agent.Contract.Machine()
	if agent.Implementation.ArtifactDigest != expected || machine.Execution.Retry != "never" || len(machine.StatusEvents) != 2 ||
		len(machine.CapabilityRequirements) != 1 || len(machine.CapabilityRequirements[0].Operations) != 2 {
		t.Fatalf("AI Agent contract/lock = %#v / %#v", agent.Implementation, machine)
	}
	for _, resource := range machine.ConfigSchemaBundle {
		if bytes.Contains(resource.Schema, []byte(`"instructions"`)) || bytes.Contains(resource.Schema, []byte(`"toolSet"`)) {
			t.Fatalf("workflow config can override trusted Agent artifacts: %s", resource.Schema)
		}
	}
}
