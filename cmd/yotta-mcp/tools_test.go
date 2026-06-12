package main

import (
	"encoding/json"
	"os"
	"path/filepath"
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
		for _, e := range container.ValidateContainer(&c, nil) {
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

func TestValidateContainerJSON_GoodAndBad(t *testing.T) {
	good := schemaExamples()[1] // rich example (already known-valid)
	out, hadErr := validateContainerJSON(good)
	if hadErr {
		t.Fatalf("rich example should have no error-level issues, got: %s", string(out))
	}

	// 缺 WindowTarget (KeyPress needsWindow) + KeyPress 缺 VK 来源 → 该有 error。
	bad := []byte(`{"schemaVersion":1,"name":"x","graph":{"version":1,"nodes":[
      {"id":"s","kind":"Start"},
      {"id":"k","kind":"KeyPress","config":{}},
      {"id":"t","kind":"Stop"}],
      "edges":[{"from":"s.Done","to":"k.In"},{"from":"k.Done","to":"t.In"}]}}`)
	out2, hadErr2 := validateContainerJSON(bad)
	if !hadErr2 {
		t.Fatalf("bad container should have error-level issues, got: %s", string(out2))
	}
	var errs []container.ValidationError
	if err := json.Unmarshal(out2, &errs); err != nil {
		t.Fatalf("output not a []ValidationError JSON: %v", err)
	}
	if len(errs) == 0 {
		t.Fatal("expected validation errors in output")
	}
}

func TestSaveContainer_RoundTrip(t *testing.T) {
	dir := t.TempDir()
	st, err := container.NewStore(filepath.Join(dir, "containers"))
	if err != nil {
		t.Fatal(err)
	}
	res, saveErrs := saveContainer(st, schemaExamples()[1]) // rich, known-valid
	if saveErrs != nil {
		t.Fatalf("rich example should save clean, got: %s", string(saveErrs))
	}
	if res.ID == "" {
		t.Fatal("expected a generated id")
	}
	if _, err := os.Stat(filepath.Join(dir, "containers", res.ID, "container.json")); err != nil {
		t.Fatalf("saved file missing: %v", err)
	}
}

func TestSaveContainer_RejectsError(t *testing.T) {
	dir := t.TempDir()
	st, _ := container.NewStore(filepath.Join(dir, "containers"))
	bad := []byte(`{"schemaVersion":1,"name":"x","graph":{"version":1,"nodes":[
      {"id":"k","kind":"KeyPress","config":{}}],"edges":[]}}`) // KeyPress, no WindowTarget, no Start
	_, saveErrs := saveContainer(st, bad)
	if saveErrs == nil {
		t.Fatal("error-level container must be rejected (saveErrs should be non-nil)")
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
