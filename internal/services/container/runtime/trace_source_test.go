package runtime

import (
	"context"
	"testing"

	_ "yotta/internal/nodes/input"
	"yotta/internal/services/container"
	"yotta/internal/services/execution"
	"yotta/pkg/winutil"
)

func TestExecNodeViaFramework_InputTraceIncludesNodeSource(t *testing.T) {
	c := &container.Container{
		SchemaVersion: 1,
		ID:            "trace-source-container",
		Name:          "trace-source-container",
		Graph: container.Graph{
			Nodes: []container.GraphNode{
				{
					ID:   "click-1",
					Kind: "ClickAt",
					Config: map[string]any{
						"Point":      map[string]any{"x": 0.25, "y": 0.75},
						"Button":     "left",
						"DurationMs": 50,
					},
				},
			},
		},
	}
	rt := NewRuntimeContext(c, execution.NewInputBus(), NoopMatcher{}, nil, nil, nil, 0)
	rt.SetActiveWindow(winutil.WindowHandle{HWND: 42, Title: "After Effects", ClientW: 1920, ClientH: 1080})
	rt.Input = &recordingRuntimeInput{}
	r := NewContainerRunner(rt)

	if _, err := r.execNodeViaFramework(context.Background(), r.nodesByID["click-1"], ExecToken{NodeID: "click-1", InPin: "In"}); err != nil {
		t.Fatalf("execNodeViaFramework error = %v", err)
	}

	records := rt.TraceRecords()
	if len(records) != 2 {
		t.Fatalf("trace len = %d, want 2 (move + click)", len(records))
	}
	for _, record := range records {
		if record.Source.ContainerID != "trace-source-container" ||
			record.Source.NodeID != "click-1" ||
			record.Source.NodeKind != "ClickAt" ||
			record.Source.InPin != "In" {
			t.Fatalf("record %s source = %#v", record.Action, record.Source)
		}
	}
}
