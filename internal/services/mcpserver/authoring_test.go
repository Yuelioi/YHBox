package mcpserver

import (
	"encoding/json"
	"testing"

	"yotta/internal/services/container"

	// Anonymous imports — trigger init() node registration so all nodes are
	// available in the registry when catalog/graph tools are called.
	_ "yotta/internal/nodes/collection"
	_ "yotta/internal/nodes/control"
	_ "yotta/internal/nodes/detect"
	_ "yotta/internal/nodes/event"
	_ "yotta/internal/nodes/input"
	_ "yotta/internal/nodes/io"
	_ "yotta/internal/nodes/purefunc"
	_ "yotta/internal/nodes/random"
	_ "yotta/internal/nodes/stopwatch"
	_ "yotta/internal/nodes/system"
	_ "yotta/internal/nodes/variable"
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
