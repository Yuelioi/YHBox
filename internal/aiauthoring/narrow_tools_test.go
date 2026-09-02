package aiauthoring

import (
	"testing"

	"github.com/yottaapp/yotta/internal/workflow/authoring"
)

func TestNormalizeCandidateNodeReferencesAcceptsBareModelHandle(t *testing.T) {
	commands := []authoring.Command{{Kind: authoring.CommandConnect, Connect: &authoring.EdgeCommand{
		GraphID: "main", Edge: authoring.PatchEdge{
			From: authoring.PatchEndpoint{NodeID: "existing", PortID: "completed"},
			To:   authoring.PatchEndpoint{NodeID: "new-key", PortID: "in"},
		},
	}}}
	normalizeCandidateNodeReferences(commands, map[string]struct{}{"new-key": {}}, nil)
	if commands[0].Connect.Edge.From.NodeID != "existing" || commands[0].Connect.Edge.To.NodeID != "$new-key" {
		t.Fatalf("commands = %#v", commands)
	}
}

func TestNormalizeCandidateNodeReferencesAcceptsPriorGeneratedID(t *testing.T) {
	commands := []authoring.Command{{Kind: authoring.CommandBindValue, BindValue: &authoring.BindValueCommand{
		GraphID: "main", NodeID: "generated-id", PortID: "keys", Value: []any{"F"},
	}}}
	normalizeCandidateNodeReferences(commands, nil, map[string]string{"generated-id": "$new-key"})
	if commands[0].BindValue.NodeID != "$new-key" {
		t.Fatalf("commands = %#v", commands)
	}
}
