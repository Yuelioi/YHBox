package nodecontract

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestGeneratedMetaSchemaPins31AndRequiresExplicitPortArrays(t *testing.T) {
	raw, err := GenerateSchema()
	if err != nil {
		t.Fatal(err)
	}
	var schema map[string]any
	if err := json.Unmarshal(raw, &schema); err != nil {
		t.Fatal(err)
	}
	if schema["$id"] != SchemaID || schema["additionalProperties"] != false {
		t.Fatalf("meta-schema identity or closure = %#v", schema)
	}
	properties := schema["properties"].(map[string]any)
	assertSingleEnum(t, properties["format"], Format)
	assertSingleEnum(t, properties["version"], Version)

	semantic := resolveSchemaRefForTest(t, schema, properties["semantic"])
	semanticProperties := semantic["properties"].(map[string]any)
	ports := resolveSchemaRefForTest(t, schema, semanticProperties["ports"])
	required := stringSetForTest(t, ports["required"])
	for _, name := range []string{"dataInputs", "dataOutputs", "execInputs", "execOutputs", "errorOutputs"} {
		if !required[name] {
			t.Fatalf("port array %q is not required", name)
		}
		property := ports["properties"].(map[string]any)[name].(map[string]any)
		if property["type"] != "array" {
			t.Fatalf("port %q schema = %#v", name, property)
		}
	}
	if _, exists := ports["properties"].(map[string]any)["statusOutputs"]; exists {
		t.Fatal("statusEvents must not be encoded as connectable ports")
	}
	if !stringSetForTest(t, semantic["required"])["statusEvents"] {
		t.Fatal("semantic statusEvents declaration is not required")
	}

	definitions := schema["$defs"].(map[string]any)
	typeExpression := definitions["TypeExpression"].(map[string]any)
	if variants, ok := typeExpression["oneOf"].([]any); !ok || len(variants) != 4 {
		t.Fatalf("TypeExpression schema is not the four-way tagged union: %#v", typeExpression)
	}
	resource := definitions["Resource"].(map[string]any)
	schemaValue := resource["properties"].(map[string]any)["schema"].(map[string]any)
	if schemaValue["type"] != "object" || schemaValue["additionalProperties"] != true {
		t.Fatalf("bundled schema resource payload = %#v", schemaValue)
	}
}

func TestTrackedMetaSchemaMatchesGenerator(t *testing.T) {
	generated, err := GenerateSchema()
	if err != nil {
		t.Fatal(err)
	}
	tracked, err := os.ReadFile(filepath.Join("..", "..", "contracts", "node", "3.1", "node-contract.schema.json"))
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(tracked, generated) {
		t.Fatal("tracked Node Contract schema differs from generator")
	}
}

func assertSingleEnum(t *testing.T, value any, want string) {
	t.Helper()
	property := value.(map[string]any)
	values := property["enum"].([]any)
	if len(values) != 1 || values[0] != want {
		t.Fatalf("enum = %#v, want %q", values, want)
	}
}

func resolveSchemaRefForTest(t *testing.T, root map[string]any, value any) map[string]any {
	t.Helper()
	current := value.(map[string]any)
	if ref, ok := current["$ref"].(string); ok {
		const prefix = "#/$defs/"
		if len(ref) <= len(prefix) || ref[:len(prefix)] != prefix {
			t.Fatalf("unsupported schema ref %q", ref)
		}
		current = root["$defs"].(map[string]any)[ref[len(prefix):]].(map[string]any)
	}
	return current
}

func stringSetForTest(t *testing.T, value any) map[string]bool {
	t.Helper()
	result := map[string]bool{}
	for _, item := range value.([]any) {
		result[item.(string)] = true
	}
	return result
}
