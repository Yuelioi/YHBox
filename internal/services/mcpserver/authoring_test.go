package mcpserver

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/yottaapp/yotta/internal/services/container"

	_ "github.com/yottaapp/yotta/internal/nodes/all"
)

func TestSchemaExamples_AllValid(t *testing.T) {
	exs := schemaExamples()
	if len(exs) < 2 {
		t.Fatalf("need >=2 examples, got %d", len(exs))
	}
	sawWin32TargetExample := false
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
			if n.Kind == "Win32WindowTarget" {
				sawWin32TargetExample = true
			}
		}
	}
	if !sawWin32TargetExample {
		t.Fatal("examples must cover a Win32 target scenario (Win32WindowTarget present)")
	}
}

func TestSaveContainerReturnsPackageDirectory(t *testing.T) {
	root := t.TempDir()
	st, err := container.NewStore(root)
	if err != nil {
		t.Fatal(err)
	}
	raw := []byte(`{
		"schemaVersion": 1,
		"name": "mcp saved",
		"graph": {
			"id": "g",
			"schemaVersion": 1,
			"nodes": [
				{"id": "start", "kind": "Start"},
				{"id": "target", "kind": "Win32WindowTarget", "config": {"Title": "Game"}},
				{"id": "stop", "kind": "Stop"}
			],
			"edges": [
				{"from": "start.Done", "to": "target.In"},
				{"from": "target.Done", "to": "stop.In"}
			]
		}
	}`)

	res, errs := saveContainer(st, raw)
	if errs != nil {
		t.Fatalf("save returned validation errors: %s", string(errs))
	}
	if res.Path != res.ID+"/" {
		t.Fatalf("path = %q, want package directory", res.Path)
	}
	for _, name := range []string{"package.json", "graph.json", "installation.json", "yotta-lock.json"} {
		if _, err := os.Stat(filepath.Join(root, res.ID, name)); err != nil {
			t.Fatalf("%s missing: %v", name, err)
		}
	}
}
