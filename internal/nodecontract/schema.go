package nodecontract

import (
	"encoding/json"

	"github.com/invopop/jsonschema"
	"github.com/yottaapp/yotta/internal/datatype"
)

const SchemaID = "https://yottaapp.dev/contracts/node/3.1/node-contract.schema.json"

// GenerateSchema returns the tracked JSON Schema projection of Node Contract 3.1.
// Seal and Open remain the authority for semantic invariants that JSON Schema
// cannot express, such as content digests and execution-class combinations.
func GenerateSchema() ([]byte, error) {
	reflector := &jsonschema.Reflector{Anonymous: true, ExpandedStruct: true}
	reflected := reflector.Reflect(&document{})
	raw, err := json.Marshal(reflected)
	if err != nil {
		return nil, err
	}
	var schema map[string]any
	if err := json.Unmarshal(raw, &schema); err != nil {
		return nil, err
	}
	schema["$id"] = SchemaID
	schema["title"] = "Yotta Node Contract"
	closeObjectSchemas(schema)
	tuneKnownDefinitions(schema)
	formatted, err := json.MarshalIndent(schema, "", "  ")
	if err != nil {
		return nil, err
	}
	return append(formatted, '\n'), nil
}

func tuneKnownDefinitions(schema map[string]any) {
	definitions := schema["$defs"].(map[string]any)
	datatype.TuneTypeExpressionDefinitions(definitions)
	TuneInstructionDefinitions(definitions)
	resource := definitions["Resource"].(map[string]any)
	resource["properties"].(map[string]any)["schema"] = map[string]any{
		"type":                 "object",
		"additionalProperties": true,
	}
}

// TuneInstructionDefinitions converts the reflected pointer payload shape into
// the exact tagged union shared by every projection that embeds InstructionSpec.
func TuneInstructionDefinitions(definitions map[string]any) {
	instruction := definitions["InstructionSpec"].(map[string]any)
	properties := instruction["properties"].(map[string]any)
	variants := []struct {
		kind    InstructionKind
		payload string
	}{
		{InstructionInvoke, "invoke"},
		{InstructionRunRoot, "runRoot"},
		{InstructionCountedLoop, "countedLoop"},
		{InstructionForEach, "forEach"},
		{InstructionRetry, "retry"},
	}
	oneOf := make([]any, 0, len(variants))
	for _, variant := range variants {
		oneOf = append(oneOf, map[string]any{
			"type":                 "object",
			"additionalProperties": false,
			"properties": map[string]any{
				"kind":          map[string]any{"const": string(variant.kind)},
				variant.payload: properties[variant.payload],
			},
			"required": []string{"kind", variant.payload},
		})
	}
	definitions["InstructionSpec"] = map[string]any{"oneOf": oneOf}
}

func closeObjectSchemas(value any) {
	switch typed := value.(type) {
	case map[string]any:
		if typed["type"] == "object" || typed["properties"] != nil {
			typed["additionalProperties"] = false
		}
		for _, child := range typed {
			closeObjectSchemas(child)
		}
	case []any:
		for _, child := range typed {
			closeObjectSchemas(child)
		}
	}
}
