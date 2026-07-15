package nodeauthoring

import (
	"encoding/json"

	"github.com/invopop/jsonschema"
	"github.com/yottaapp/yotta/internal/datatype"
)

const SchemaID = "https://yottaapp.dev/contracts/node/3.1/authoring-projection.schema.json"

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
	schema["title"] = "Yotta Node Authoring Projection 3.1"
	closeProjectionObjects(schema)
	if definitions, ok := schema["$defs"].(map[string]any); ok {
		datatype.TuneTypeExpressionDefinitions(definitions)
	}
	formatted, err := json.MarshalIndent(schema, "", "  ")
	if err != nil {
		return nil, err
	}
	return append(formatted, '\n'), nil
}

func closeProjectionObjects(value any) {
	switch typed := value.(type) {
	case map[string]any:
		if typed["type"] == "object" || typed["properties"] != nil {
			typed["additionalProperties"] = false
		}
		for _, child := range typed {
			closeProjectionObjects(child)
		}
	case []any:
		for _, child := range typed {
			closeProjectionObjects(child)
		}
	}
}
