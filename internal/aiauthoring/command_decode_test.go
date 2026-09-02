package aiauthoring

import (
	"testing"

	"github.com/yottaapp/yotta/internal/workflow/authoring"
)

func TestDecodeAuthoringCommandsAcceptsArrayAndCommandsEnvelope(t *testing.T) {
	for _, raw := range []string{
		`[{"kind":"rename-workflow","renameWorkflow":{"name":"Updated"}}]`,
		`{"workflowId":"workflow-1","baseRevision":40,"graphId":"main","commands":[{"kind":"rename-workflow","renameWorkflow":{"name":"Updated"}}]}`,
	} {
		var commands []authoring.Command
		if err := decodeAuthoringCommands([]byte(raw), &commands); err != nil {
			t.Fatalf("decode %s: %v", raw, err)
		}
		if len(commands) != 1 || commands[0].Kind != authoring.CommandRenameWorkflow {
			t.Fatalf("commands = %#v", commands)
		}
	}
}

func TestDecodeAuthoringCommandsNormalizesFlatModelBinding(t *testing.T) {
	var commands []authoring.Command
	raw := `{"workflowId":"workflow-1","baseRevision":40,"commands":[{"kind":"set-node-binding","graphId":"main","nodeId":"node-1","portId":"threshold","binding":{"kind":"value","value":0.5}}]}`
	if err := decodeAuthoringCommands([]byte(raw), &commands); err != nil {
		t.Fatal(err)
	}
	if len(commands) != 1 || commands[0].Kind != authoring.CommandBindValue || commands[0].BindValue == nil || commands[0].BindValue.PortID != "threshold" {
		t.Fatalf("commands = %#v", commands)
	}
}

func TestDecodeAuthoringCommandsAcceptsInputIDAlias(t *testing.T) {
	var commands []authoring.Command
	raw := `{"commands":[{"kind":"set-node-binding","graphId":"main","nodeId":"node-1","inputId":"threshold","binding":{"kind":"value","value":0.75}}]}`
	if err := decodeAuthoringCommands([]byte(raw), &commands); err != nil {
		t.Fatal(err)
	}
	if commands[0].BindValue == nil || commands[0].BindValue.PortID != "threshold" {
		t.Fatalf("commands = %#v", commands)
	}
}

func TestDecodeAuthoringCommandsRejectsUnknownEnvelopeFields(t *testing.T) {
	var commands []authoring.Command
	if err := decodeAuthoringCommands([]byte(`{"commands":[],"extra":true}`), &commands); err == nil {
		t.Fatal("accepted unknown authoring command envelope field")
	}
}
