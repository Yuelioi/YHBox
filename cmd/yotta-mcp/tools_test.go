package main

import (
	"encoding/json"
	"testing"
)

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
