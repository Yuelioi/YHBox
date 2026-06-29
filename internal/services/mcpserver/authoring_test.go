package mcpserver

import (
	"encoding/json"
	"testing"

	"yotta/internal/services/container"

	_ "yotta/internal/nodes/all"
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
