package dependency

import (
	"reflect"
	"testing"

	_ "github.com/yottaapp/yotta/internal/nodes/detect"
	_ "github.com/yottaapp/yotta/internal/nodes/io"
	_ "github.com/yottaapp/yotta/internal/nodes/system"
)

func TestClosureSplitsTemplatesAndSubgraphs(t *testing.T) {
	root := []NodeInfo{
		{Kind: "CheckTemplate", Config: map[string]any{"literal": map[string]any{"Templates": []any{"tpl-root"}}}},
		{Kind: "Subgraph", Config: map[string]any{"literal": map[string]any{"SubgraphID": "sg-a"}}},
	}
	subgraphs := map[string][]NodeInfo{
		"sg-a": {
			{Kind: "CheckTemplate", Config: map[string]any{"literal": map[string]any{"Templates": []any{"tpl-sub"}}}},
		},
	}

	got, err := Closure(root, func(id string) ([]NodeInfo, error) {
		return subgraphs[id], nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(got.Subgraphs, []string{"sg-a"}) {
		t.Fatalf("subgraphs: got %v", got.Subgraphs)
	}
	if !reflect.DeepEqual(got.Templates, []string{"tpl-root", "tpl-sub"}) {
		t.Fatalf("templates: got %v", got.Templates)
	}
}
