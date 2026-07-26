package contractschema

import (
	"encoding/json"
	"strings"
	"testing"
)

const testDialect = "https://json-schema.org/draft/2020-12/schema"

func TestNormalizeCanonicalizesSortsAndResolvesOfflineBundle(t *testing.T) {
	resources := []Resource{
		{ID: "https://schemas.example.test/child", Schema: json.RawMessage(`{
			"type":"object","$schema":"https://json-schema.org/draft/2020-12/schema",
			"$id":"https://schemas.example.test/child","properties":{"value":{"type":"string"}}
		}`)},
		{ID: "https://schemas.example.test/root", Schema: json.RawMessage(`{
			"$schema":"https://json-schema.org/draft/2020-12/schema","$id":"https://schemas.example.test/root",
			"allOf":[{"$ref":"child"}],"$defs":{"local":{"$id":"nested","type":"number"}},
			"properties":{"nested":{"$ref":"nested"}}
		}`)},
	}
	got, err := Normalize(testDialect, "https://schemas.example.test/root", resources)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 2 || got[0].ID != "https://schemas.example.test/child" || got[1].ID != "https://schemas.example.test/root" {
		t.Fatalf("normalized bundle = %#v", got)
	}
	if string(got[0].Schema) != `{"$id":"https://schemas.example.test/child","$schema":"https://json-schema.org/draft/2020-12/schema","properties":{"value":{"type":"string"}},"type":"object"}` {
		t.Fatalf("child schema is not canonical: %s", got[0].Schema)
	}
	resources[0].Schema[0] = '['
	if got[0].Schema[0] != '{' {
		t.Fatal("Normalize retained caller-owned schema bytes")
	}
}

func TestNormalizeRejectsInvalidBundleIdentityAndEnvelope(t *testing.T) {
	valid := func(id string) Resource {
		return Resource{ID: id, Schema: json.RawMessage(`{"$id":"` + id + `","$schema":"` + testDialect + `","type":"object"}`)}
	}
	cases := []struct {
		name      string
		dialect   string
		root      string
		resources []Resource
		contains  string
	}{
		{name: "relative root", dialect: testDialect, root: "root", resources: []Resource{valid("https://schemas.example.test/root")}, contains: "invalid schema bundle root"},
		{name: "empty", dialect: testDialect, resources: nil, contains: "resource budget"},
		{name: "relative id", dialect: testDialect, resources: []Resource{{ID: "child", Schema: json.RawMessage(`{}`)}}, contains: "invalid schema resource id"},
		{name: "duplicate id", dialect: testDialect, resources: []Resource{valid("https://schemas.example.test/root"), valid("https://schemas.example.test/root")}, contains: "duplicate schema resource"},
		{name: "missing root", dialect: testDialect, root: "https://schemas.example.test/missing", resources: []Resource{valid("https://schemas.example.test/root")}, contains: "root is not in bundle"},
		{name: "non object", dialect: testDialect, resources: []Resource{{ID: "https://schemas.example.test/root", Schema: json.RawMessage(`true`)}}, contains: "must be a JSON object"},
		{name: "mismatched id", dialect: testDialect, resources: []Resource{{ID: "https://schemas.example.test/root", Schema: json.RawMessage(`{"$id":"https://schemas.example.test/other","$schema":"` + testDialect + `"}`)}}, contains: "mismatched $id"},
		{name: "mismatched dialect", dialect: testDialect, resources: []Resource{{ID: "https://schemas.example.test/root", Schema: json.RawMessage(`{"$id":"https://schemas.example.test/root","$schema":"other"}`)}}, contains: "mismatched $schema"},
	}
	for _, test := range cases {
		t.Run(test.name, func(t *testing.T) {
			_, err := Normalize(test.dialect, test.root, test.resources)
			if err == nil || !strings.Contains(err.Error(), test.contains) {
				t.Fatalf("Normalize() error = %v, want %q", err, test.contains)
			}
		})
	}
}

func TestNormalizeRejectsInvalidSubschemasAndReferences(t *testing.T) {
	resource := func(body string) []Resource {
		return []Resource{{ID: "https://schemas.example.test/root", Schema: json.RawMessage(`{
			"$id":"https://schemas.example.test/root","$schema":"` + testDialect + `",` + body + `
		}`)}}
	}
	cases := []struct {
		name     string
		body     string
		contains string
	}{
		{name: "fragment id", body: `"$defs":{"bad":{"$id":"child#fragment"}}`, contains: "must not contain a fragment"},
		{name: "duplicate nested id", body: `"$defs":{"a":{"$id":"child"},"b":{"$id":"child"}}`, contains: "duplicate bundled schema id"},
		{name: "missing ref", body: `"$ref":"missing"`, contains: "not in the offline schema bundle"},
		{name: "non string ref", body: `"$ref":42`, contains: "$ref must be a string"},
		{name: "invalid single schema", body: `"items":42`, contains: "must contain a schema"},
		{name: "invalid schema array", body: `"allOf":{}`, contains: "must contain a schema array"},
		{name: "invalid array member", body: `"anyOf":[42]`, contains: "must contain only schemas"},
		{name: "invalid schema map", body: `"properties":[]`, contains: "must contain a schema map"},
		{name: "invalid map member", body: `"$defs":{"bad":42}`, contains: "must contain only schemas"},
	}
	for _, test := range cases {
		t.Run(test.name, func(t *testing.T) {
			_, err := Normalize(testDialect, "https://schemas.example.test/root", resource(test.body))
			if err == nil || !strings.Contains(err.Error(), test.contains) {
				t.Fatalf("Normalize() error = %v, want %q", err, test.contains)
			}
		})
	}
}

func TestNormalizeEnforcesStructuralBudgets(t *testing.T) {
	id := "https://schemas.example.test/root"
	tooDeep := `{"$id":"` + id + `","$schema":"` + testDialect + `","allOf":[` + strings.Repeat(`{"allOf":[`, MaxDepth) + `true` + strings.Repeat(`]}`, MaxDepth) + `]}`
	tooLarge := `{"$id":"` + id + `","$schema":"` + testDialect + `","description":"` + strings.Repeat("x", MaxResourceBytes) + `"}`
	for _, test := range []struct {
		name     string
		schema   string
		contains string
	}{
		{name: "depth", schema: tooDeep, contains: "depth budget"},
		{name: "bytes", schema: tooLarge, contains: "byte budget"},
	} {
		t.Run(test.name, func(t *testing.T) {
			_, err := Normalize(testDialect, id, []Resource{{ID: id, Schema: json.RawMessage(test.schema)}})
			if err == nil || !strings.Contains(err.Error(), test.contains) {
				t.Fatalf("Normalize() error = %v, want %q", err, test.contains)
			}
		})
	}
}

func TestNormalizeEnforcesResourceAndReferenceBudgets(t *testing.T) {
	id := "https://schemas.example.test/root"
	resources := make([]Resource, MaxResources+1)
	for index := range resources {
		resourceID := "https://schemas.example.test/resource/" + string(rune('a'+index%26))
		resources[index] = Resource{ID: resourceID, Schema: json.RawMessage(`{}`)}
	}
	if _, err := Normalize(testDialect, "", resources); err == nil || !strings.Contains(err.Error(), "resource budget") {
		t.Fatalf("resource budget error = %v", err)
	}

	references := make([]string, MaxReferences+1)
	for index := range references {
		references[index] = `{"$ref":"#"}`
	}
	schema := `{"$id":"` + id + `","$schema":"` + testDialect + `","allOf":[` + strings.Join(references, ",") + `]}`
	if _, err := Normalize(testDialect, id, []Resource{{ID: id, Schema: json.RawMessage(schema)}}); err == nil || !strings.Contains(err.Error(), "reference budget") {
		t.Fatalf("reference budget error = %v", err)
	}
	if _, err := Normalize(testDialect, id, []Resource{{ID: id, Schema: json.RawMessage(`{`)}}); err == nil || !strings.Contains(err.Error(), "structural budget") {
		t.Fatalf("invalid JSON error = %v", err)
	}
}
