package purefunc

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/yottaapp/yotta/internal/node"
)

func evalJSONNode(t *testing.T, n node.Node, dataWire map[string]any) any {
	t.Helper()
	node.ResetRegistryForTest()
	node.Register(n)
	rn, ok := node.Get(n.Spec().Kind)
	if !ok {
		t.Fatalf("%s not registered", n.Spec().Kind)
	}
	got, err := node.EvaluatePureData(context.Background(), rn, dataWire, nil, node.StubServices())
	if err != nil {
		t.Fatalf("EvaluatePureData %s: %v", n.Spec().Kind, err)
	}
	return got
}

func TestParseJSON_ObjectAndArray(t *testing.T) {
	obj := evalJSONNode(t, &ParseJSON{}, map[string]any{"Text": `{"ok":true,"n":2}`})
	m, ok := obj.(map[string]any)
	if !ok || m["ok"] != true {
		t.Fatalf("ParseJSON object = %#v", obj)
	}
	if n, ok := m["n"].(json.Number); !ok || n.String() != "2" {
		t.Fatalf("number = %#v, want json.Number(2)", m["n"])
	}

	arr := evalJSONNode(t, &ParseJSON{}, map[string]any{"Text": `[{"url":"a"}, "b"]`})
	items, ok := arr.([]any)
	if !ok || len(items) != 2 {
		t.Fatalf("ParseJSON array = %#v", arr)
	}
}

func TestParseJSON_InvalidReturnsNil(t *testing.T) {
	got := evalJSONNode(t, &ParseJSON{}, map[string]any{"Text": `{"bad"`})
	if got != nil {
		t.Fatalf("invalid ParseJSON = %#v, want nil", got)
	}
}

func TestParseJSON_TrailingGarbageReturnsNil(t *testing.T) {
	got := evalJSONNode(t, &ParseJSON{}, map[string]any{"Text": `{"ok":true} trailing`})
	if got != nil {
		t.Fatalf("ParseJSON with trailing garbage = %#v, want nil", got)
	}
}

func TestToJSON(t *testing.T) {
	got := evalJSONNode(t, &ToJSON{}, map[string]any{"Value": map[string]any{"a": 1, "b": true}})
	if got != `{"a":1,"b":true}` {
		t.Fatalf("ToJSON object = %q", got)
	}

	got = evalJSONNode(t, &ToJSON{}, map[string]any{"Value": []any{"x", 2}})
	if got != `["x",2]` {
		t.Fatalf("ToJSON array = %q", got)
	}
}

func TestJsonPath(t *testing.T) {
	value := map[string]any{
		"user": map[string]any{"name": "Lin"},
		"items": []any{
			map[string]any{"id": "a"},
			map[string]any{"id": "b"},
		},
	}
	cases := []struct {
		name string
		path string
		want any
	}{
		{"root", "$", value},
		{"field", "$.user.name", "Lin"},
		{"index", "$.items[0].id", "a"},
		{"wildcard", "$.items[*].id", []any{"a", "b"}},
		{"missing", "$.user.missing", nil},
		{"badIndex", "$.items[9]", nil},
		{"invalidPath", "items[0]", nil},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := evalJSONNode(t, &JsonPath{}, map[string]any{"JSON": value, "Path": tc.path})
			if !jsonPathEqual(got, tc.want) {
				t.Fatalf("JsonPath(%s) = %#v, want %#v", tc.path, got, tc.want)
			}
		})
	}
}

func jsonPathEqual(a, b any) bool {
	ab, _ := json.Marshal(a)
	bb, _ := json.Marshal(b)
	return string(ab) == string(bb)
}
