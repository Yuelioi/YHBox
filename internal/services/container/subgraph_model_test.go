package container

import (
	"encoding/json"
	"strings"
	"testing"
	"time"
)

func TestSubgraphOutputDeclShape(t *testing.T) {
	d := SubgraphOutputDecl{ID: "decl-uuid", Name: "found"}
	b, err := json.Marshal(d)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	if !strings.Contains(string(b), `"id":"decl-uuid"`) {
		t.Errorf("missing id: %s", string(b))
	}
	if !strings.Contains(string(b), `"name":"found"`) {
		t.Errorf("missing name: %s", string(b))
	}
}

func TestSubgraphSerialization(t *testing.T) {
	sg := Subgraph{
		ID:          "sg-abc",
		Label:       "录制 14:32",
		Description: "",
		Graph: Graph{
			ID:            "g-sg-abc",
			SchemaVersion: GraphSchemaVersion,
			Nodes:         []GraphNode{{ID: "in", Kind: "SubgraphInput", CreatedAt: time.Now().UTC()}},
		},
		OutputPins: []SubgraphOutputDecl{
			{ID: "decl-1", Name: "done"},
			{ID: "decl-2", Name: "timeout"},
		},
		Tags:      []string{"录制"},
		CreatedAt: time.Now().UTC(),
	}
	b, err := json.Marshal(sg)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	var back Subgraph
	if err := json.Unmarshal(b, &back); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if len(back.OutputPins) != 2 || back.OutputPins[0].ID != "decl-1" {
		t.Errorf("OutputPins lost: %+v", back.OutputPins)
	}
}
