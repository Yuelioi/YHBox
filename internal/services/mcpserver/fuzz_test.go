package mcpserver

import (
	"encoding/json"
	"testing"

	"github.com/yottaapp/yotta/internal/workflow/authoring"
)

func FuzzMCPPatchCommands(f *testing.F) {
	f.Add([]byte(`{"workflowId":"wf","baseRevision":0,"commands":[{"kind":"rename-workflow","renameWorkflow":{"name":"ok"}}]}`))
	f.Add([]byte(`{"workflowId":"wf","baseRevision":0,"commands":[{"kind":"add-node","moveNode":{}}]}`))
	f.Fuzz(func(t *testing.T, raw []byte) {
		if len(raw) > 1<<20 {
			t.Skip()
		}
		var request WorkflowApplyPatchRequest
		if json.Unmarshal(raw, &request) != nil {
			return
		}
		if len(request.Commands) > authoring.MaxCommands {
			return
		}
		for _, command := range request.Commands {
			_ = command.Validate()
		}
	})
}
