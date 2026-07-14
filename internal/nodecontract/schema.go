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
	schema["title"] = "Yotta Node Contract 3.1"
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
	definitions["TypeExpression"] = map[string]any{
		"title": "TypeExpression",
		"oneOf": []any{
			typeExpressionVariant("ref", map[string]any{
				"ref": map[string]any{"$ref": "#/$defs/TypeRef"},
			}, "ref"),
			typeExpressionVariant("list", map[string]any{
				"element": map[string]any{"$ref": "#/$defs/TypeExpression"},
			}, "element"),
			typeExpressionVariant("union", map[string]any{
				"members": map[string]any{
					"type": "array", "minItems": 2, "maxItems": datatype.MaxUnionMembers,
					"items": map[string]any{"$ref": "#/$defs/TypeExpression"},
				},
			}, "members"),
			typeExpressionVariant("variable", map[string]any{
				"variable": map[string]any{"type": "string", "minLength": 1, "maxLength": datatype.MaxTypeStringBytes},
				"constraints": map[string]any{
					"type": "array", "maxItems": datatype.MaxTypeConstraints,
					"items": map[string]any{"type": "string", "minLength": 1, "maxLength": datatype.MaxTypeStringBytes},
				},
			}, "variable"),
		},
	}
	resource := definitions["Resource"].(map[string]any)
	resource["properties"].(map[string]any)["schema"] = map[string]any{
		"type":                 "object",
		"additionalProperties": true,
	}
}

func typeExpressionVariant(kind string, properties map[string]any, required ...string) map[string]any {
	properties["kind"] = map[string]any{"const": kind, "type": "string"}
	return map[string]any{
		"type":                 "object",
		"additionalProperties": false,
		"properties":           properties,
		"required":             append([]string{"kind"}, required...),
	}
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
