package schema

import (
	"encoding/json"
	"fmt"

	"github.com/invopop/jsonschema"
	"github.com/yottaapp/yotta/internal/datatype"
)

const (
	workflowSchemaID   = "https://yottaapp.dev/contracts/workflow/" + SchemaPathVersion + "/workflow-source.schema.json"
	diagnosticSchemaID = "https://yottaapp.dev/contracts/workflow/" + SchemaPathVersion + "/diagnostic.schema.json"
)

// GenerateContract returns the canonical JSON Schema document used by both the
// runtime parser and the tracked frontend contract artifacts.
func GenerateContract(name string) ([]byte, error) {
	reflector := &jsonschema.Reflector{Anonymous: true, ExpandedStruct: true}
	var contract *jsonschema.Schema
	var id, title string
	switch name {
	case "workflow":
		contract = reflector.Reflect(&WorkflowSource{})
		id = workflowSchemaID
		title = "Yotta Workflow Source"
	case "diagnostic":
		contract = reflector.Reflect(&Diagnostic{})
		id = diagnosticSchemaID
		title = "Yotta Compiler Diagnostic"
	default:
		return nil, fmt.Errorf("unknown contract %q", name)
	}

	raw, err := json.Marshal(contract)
	if err != nil {
		return nil, err
	}
	var document map[string]any
	if err := json.Unmarshal(raw, &document); err != nil {
		return nil, err
	}
	document["$id"] = id
	document["title"] = title
	if definitions, ok := document["$defs"].(map[string]any); ok {
		datatype.TuneTypeExpressionDefinitions(definitions)
	}
	formatted, err := json.MarshalIndent(document, "", "  ")
	if err != nil {
		return nil, err
	}
	return append(formatted, '\n'), nil
}
