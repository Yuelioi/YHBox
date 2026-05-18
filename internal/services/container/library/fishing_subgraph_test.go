package library

import (
	"encoding/json"
	"io/fs"
	"testing"
)

func TestFishingFightSubgraph_Loads(t *testing.T) {
	b, err := fs.ReadFile(builtinFS, "builtin_library/subgraphs/fishing-fight.json")
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	var sg struct {
		ID    string `json:"id"`
		Kind  string `json:"kind"`
		Graph struct {
			Nodes []struct {
				ID   string `json:"id"`
				Kind string `json:"kind"`
			} `json:"nodes"`
			Edges []struct {
				From string `json:"from"`
				To   string `json:"to"`
			} `json:"edges"`
		} `json:"graph"`
	}
	if err := json.Unmarshal(b, &sg); err != nil {
		t.Fatalf("parse: %v", err)
	}
	if sg.ID != "fishing-fight" {
		t.Errorf("expected id 'fishing-fight', got %q", sg.ID)
	}
	// Subgraph JSON uses "label" not "kind"; kind check is skipped (not in schema)
	if len(sg.Graph.Nodes) < 30 {
		t.Errorf("expected ~37 nodes, got %d", len(sg.Graph.Nodes))
	}
	kinds := map[string]int{}
	for _, n := range sg.Graph.Nodes {
		kinds[n.Kind]++
	}
	if kinds["KeyHoldStop"] < 4 {
		t.Errorf("expected at least 4 KeyHoldStop (double cleanup pairs + center), got %d", kinds["KeyHoldStop"])
	}
	if kinds["Throw"] < 1 {
		t.Errorf("expected at least 1 Throw, got %d", kinds["Throw"])
	}
	if kinds["ROIColorScan"] != 2 {
		t.Errorf("expected exactly 2 ROIColorScan (yellow + green), got %d", kinds["ROIColorScan"])
	}
	if kinds["SetVar"] < 8 {
		t.Errorf("expected >=8 SetVar (state machine vars), got %d", kinds["SetVar"])
	}
}

func TestFishingShopsellSubgraph_Loads(t *testing.T) {
	b, err := fs.ReadFile(builtinFS, "builtin_library/subgraphs/fishing-shopsell.json")
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	var sg struct {
		ID    string `json:"id"`
		Graph struct {
			Nodes []struct{ Kind string `json:"kind"` } `json:"nodes"`
		} `json:"graph"`
	}
	if err := json.Unmarshal(b, &sg); err != nil {
		t.Fatalf("parse: %v", err)
	}
	if sg.ID != "fishing-shopsell" {
		t.Errorf("id = %q", sg.ID)
	}
	kinds := map[string]int{}
	for _, n := range sg.Graph.Nodes {
		kinds[n.Kind]++
	}
	if kinds["WaitTemplate"] < 2 {
		t.Errorf("expected ≥2 WaitTemplate, got %d", kinds["WaitTemplate"])
	}
	if kinds["ClickTemplate"] < 2 {
		t.Errorf("expected ≥2 ClickTemplate, got %d", kinds["ClickTemplate"])
	}
	if kinds["Throw"] < 1 {
		t.Errorf("expected ≥1 Throw, got %d", kinds["Throw"])
	}
}

func TestFishingBuybaitSubgraph_Loads(t *testing.T) {
	b, err := fs.ReadFile(builtinFS, "builtin_library/subgraphs/fishing-buybait.json")
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	var sg struct {
		ID    string `json:"id"`
		Graph struct {
			Nodes []struct{ Kind string `json:"kind"` } `json:"nodes"`
		} `json:"graph"`
	}
	if err := json.Unmarshal(b, &sg); err != nil {
		t.Fatalf("parse: %v", err)
	}
	if sg.ID != "fishing-buybait" {
		t.Errorf("id = %q", sg.ID)
	}
	kinds := map[string]int{}
	for _, n := range sg.Graph.Nodes {
		kinds[n.Kind]++
	}
	if kinds["WaitTemplate"] < 4 {
		t.Errorf("expected ≥4 WaitTemplate, got %d", kinds["WaitTemplate"])
	}
	if kinds["ClickTemplate"] < 4 {
		t.Errorf("expected ≥4 ClickTemplate, got %d", kinds["ClickTemplate"])
	}
	if kinds["Throw"] < 2 {
		t.Errorf("expected ≥2 Throw (noMoney + noUI), got %d", kinds["Throw"])
	}
}

func TestFishingChangebaitSubgraph_Loads(t *testing.T) {
	b, err := fs.ReadFile(builtinFS, "builtin_library/subgraphs/fishing-changebait.json")
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	var sg struct {
		ID    string `json:"id"`
		Graph struct {
			Nodes []struct{ Kind string `json:"kind"` } `json:"nodes"`
		} `json:"graph"`
	}
	if err := json.Unmarshal(b, &sg); err != nil {
		t.Fatalf("parse: %v", err)
	}
	if sg.ID != "fishing-changebait" {
		t.Errorf("id = %q", sg.ID)
	}
	if len(sg.Graph.Nodes) < 8 {
		t.Errorf("expected ≥8 nodes, got %d", len(sg.Graph.Nodes))
	}
	kinds := map[string]int{}
	for _, n := range sg.Graph.Nodes {
		kinds[n.Kind]++
	}
	if kinds["Throw"] < 1 {
		t.Errorf("expected ≥1 Throw, got %d", kinds["Throw"])
	}
}

func TestFishingResultSubgraph_Loads(t *testing.T) {
	b, err := fs.ReadFile(builtinFS, "builtin_library/subgraphs/fishing-result.json")
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	var sg struct {
		ID    string `json:"id"`
		Kind  string `json:"kind"`
		Graph struct {
			Nodes []struct{ Kind string `json:"kind"` } `json:"nodes"`
			Edges []struct{ From, To string } `json:"edges"`
		} `json:"graph"`
	}
	if err := json.Unmarshal(b, &sg); err != nil {
		t.Fatalf("parse: %v", err)
	}
	if sg.ID != "fishing-result" {
		t.Errorf("id = %q, want fishing-result", sg.ID)
	}
	if len(sg.Graph.Nodes) != 4 {
		t.Errorf("expected 4 nodes, got %d", len(sg.Graph.Nodes))
	}
	if len(sg.Graph.Edges) != 3 {
		t.Errorf("expected 3 edges, got %d", len(sg.Graph.Edges))
	}
}
