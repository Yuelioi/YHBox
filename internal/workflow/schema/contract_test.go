package schema

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strconv"
	"testing"
)

func TestTrackedSchemasAreClosedAndPinnedTo31(t *testing.T) {
	root := filepath.Join("..", "..", "..", "contracts", "workflow", "3.1")
	for _, name := range []string{"workflow-source.schema.json", "diagnostic.schema.json"} {
		raw, err := os.ReadFile(filepath.Join(root, name))
		if err != nil {
			t.Fatal(err)
		}
		var document map[string]any
		if err := json.Unmarshal(raw, &document); err != nil {
			t.Fatal(err)
		}
		assertClosedObjects(t, document, name)
	}

	raw, err := os.ReadFile(filepath.Join(root, "workflow-source.schema.json"))
	if err != nil {
		t.Fatal(err)
	}
	var sourceSchema map[string]any
	if err := json.Unmarshal(raw, &sourceSchema); err != nil {
		t.Fatal(err)
	}
	properties := sourceSchema["properties"].(map[string]any)
	if got := properties["format"].(map[string]any)["enum"].([]any); len(got) != 1 || got[0] != Format {
		t.Fatalf("format enum = %#v", got)
	}
	if got := properties["version"].(map[string]any)["enum"].([]any); len(got) != 1 || got[0] != Version {
		t.Fatalf("version enum = %#v", got)
	}
	typeExpression := sourceSchema["$defs"].(map[string]any)["TypeExpression"].(map[string]any)
	if variants, ok := typeExpression["oneOf"].([]any); !ok || len(variants) != 4 {
		t.Fatalf("workflow TypeExpression is not the shared four-way union: %#v", typeExpression)
	}
}

func assertClosedObjects(t *testing.T, value any, path string) {
	t.Helper()
	switch typed := value.(type) {
	case map[string]any:
		if typed["type"] == "object" {
			if _, isOpenConfig := typed["properties"]; isOpenConfig {
				if open, ok := typed["additionalProperties"].(bool); !ok || open {
					t.Fatalf("object schema is open at %s", path)
				}
			}
		}
		for key, child := range typed {
			assertClosedObjects(t, child, path+"/"+key)
		}
	case []any:
		for index, child := range typed {
			assertClosedObjects(t, child, path+"/"+strconv.Itoa(index))
		}
	}
}
