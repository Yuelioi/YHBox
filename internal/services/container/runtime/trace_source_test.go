package runtime

import (
	"context"
	"testing"

	automationtrace "yotta/internal/automation/trace"
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

func TestExecNodeViaFramework_InputTextTraceIncludesNodeSource(t *testing.T) {
	c := &container.Container{
		SchemaVersion: 1,
		ID:            "trace-text-container",
		Name:          "trace-text-container",
		Graph: container.Graph{
			Nodes: []container.GraphNode{
				{
					ID:   "text-1",
					Kind: "InputText",
					Config: map[string]any{
						"Text": "hello ae",
					},
				},
			},
		},
	}
	rt := NewRuntimeContext(c, execution.NewInputBus(), NoopMatcher{}, nil, nil, nil, 0)
	rt.SetActiveWindow(winutil.WindowHandle{HWND: 84, Title: "After Effects", ClientW: 1920, ClientH: 1080})
	input := &recordingRuntimeInput{}
	rt.Input = input
	r := NewContainerRunner(rt)

	if _, err := r.execNodeViaFramework(context.Background(), r.nodesByID["text-1"], ExecToken{NodeID: "text-1", InPin: "In"}); err != nil {
		t.Fatalf("execNodeViaFramework error = %v", err)
	}

	if len(input.textHWND) != 1 || input.textHWND[0] != 84 || input.textValues[0] != "hello ae" {
		t.Fatalf("backend TypeText = hwnds %#v texts %#v", input.textHWND, input.textValues)
	}
	records := rt.TraceRecords()
	if len(records) != 1 {
		t.Fatalf("trace len = %d, want 1", len(records))
	}
	record := records[0]
	if record.Action != "text" || record.Status != automationtrace.StatusSuccess {
		t.Fatalf("trace action/status = %s/%s", record.Action, record.Status)
	}
	if record.Source.ContainerID != "trace-text-container" ||
		record.Source.NodeID != "text-1" ||
		record.Source.NodeKind != "InputText" ||
		record.Source.InPin != "In" {
		t.Fatalf("record source = %#v", record.Source)
	}
}
