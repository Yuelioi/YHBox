package dependency

import (
	"reflect"
	"testing"

	_ "github.com/yottaapp/yotta/internal/nodes/system"
)

func TestClosureContainsTransitiveSubgraphs(t *testing.T) {
	root := []NodeInfo{
		{Kind: "Subgraph", Config: map[string]any{"literal": map[string]any{"SubgraphID": "sg-a"}}},
	}
	subgraphs := map[string][]NodeInfo{
		"sg-a": {
			{Kind: "Subgraph", Config: map[string]any{"literal": map[string]any{"SubgraphID": "sg-b"}}},
		},
	}

	got, err := Closure(root, func(id string) ([]NodeInfo, error) {
		return subgraphs[id], nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(got.Subgraphs, []string{"sg-a", "sg-b"}) {
		t.Fatalf("subgraphs: got %v", got.Subgraphs)
	}
}
