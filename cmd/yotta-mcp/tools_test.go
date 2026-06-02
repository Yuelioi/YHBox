package main

import (
	"encoding/json"
	"testing"

	"yotta/internal/services/container"
)

func TestSchemaExamples_AllValid(t *testing.T) {
	exs := schemaExamples()
	if len(exs) < 2 {
		t.Fatalf("need >=2 examples, got %d", len(exs))
	}
	sawNeedsWindow := false
	for i, raw := range exs {
		var c container.Container
		if err := json.Unmarshal(raw, &c); err != nil {
			t.Fatalf("example %d not valid JSON: %v", i, err)
		}
		c.Normalize()
		for _, e := range container.ValidateContainer(&c) {
			if e.Severity == container.SeverityError {
				t.Fatalf("example %d has validation error: %s @ %s", i, e.Code, e.NodeID)
			}
		}
		for _, n := range c.Graph.Nodes {
			if n.Kind == "WindowTarget" {
				sawNeedsWindow = true
			}
		}
	}
	if !sawNeedsWindow {
		t.Fatal("examples must cover a needsWindow scenario (WindowTarget present)")
	}
}

func TestListNodesJSON_NonEmpty(t *testing.T) {
	b := listNodesJSON()
	var arr []map[string]any
	if err := json.Unmarshal(b, &arr); err != nil {
		t.Fatalf("listNodesJSON not valid JSON: %v", err)
	}
	if len(arr) == 0 {
		t.Fatal("empty catalog — node packages not registered?")
	}
	// 抽样确认结构: 找到 KeyPress 且 needsWindow
	found := false
	for _, n := range arr {
		if n["kind"] == "KeyPress" {
			found = true
			if n["needsWindow"] != true {
				t.Error("KeyPress should have needsWindow=true")
			}
		}
	}
	if !found {
		t.Fatal("KeyPress not in list_nodes output")
	}
}
