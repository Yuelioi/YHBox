package runtime

import (
	"context"
	"image"
	"image/color"
	"testing"

	automationtrace "github.com/yottaapp/yotta/internal/automation/trace"
	"github.com/yottaapp/yotta/internal/node"
	_ "github.com/yottaapp/yotta/internal/nodes/image"
	_ "github.com/yottaapp/yotta/internal/nodes/input"
	"github.com/yottaapp/yotta/internal/services/container"
	"github.com/yottaapp/yotta/internal/services/execution"
	"github.com/yottaapp/yotta/pkg/winutil"
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
	installTestWin32Input(rt, &recordingRuntimeInput{})
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
	installTestWin32Input(rt, input)
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

func TestExecNodeViaFramework_MouseHoldStartTraceIncludesNodeSource(t *testing.T) {
	rt, input, runner := newTraceSourceRuntime(t, "trace-mouse-down-container", container.GraphNode{
		ID:   "down-1",
		Kind: "MouseHoldStart",
		Config: map[string]any{
			"Point":  node.Point{X: 0.3, Y: 0.7},
			"Button": "right",
		},
	})

	if _, err := runner.execNodeViaFramework(context.Background(), runner.nodesByID["down-1"], ExecToken{NodeID: "down-1", InPin: "In"}); err != nil {
		t.Fatalf("execNodeViaFramework error = %v", err)
	}

	if len(input.mouseDownHWND) != 1 || input.mouseDownHWND[0] != 84 || input.mouseDownX[0] != 0.3 || input.mouseDownY[0] != 0.7 || input.mouseDownButton[0] != "right" {
		t.Fatalf("backend MouseDown = hwnds %#v x %#v y %#v buttons %#v", input.mouseDownHWND, input.mouseDownX, input.mouseDownY, input.mouseDownButton)
	}
	assertSingleTraceSource(t, rt.TraceRecords(), "mouse-down", "trace-mouse-down-container", "down-1", "MouseHoldStart")
}

func TestExecNodeViaFramework_MouseHoldStopTraceIncludesNodeSource(t *testing.T) {
	rt, input, runner := newTraceSourceRuntime(t, "trace-mouse-up-container", container.GraphNode{
		ID:   "up-1",
		Kind: "MouseHoldStop",
		Config: map[string]any{
			"Button": "middle",
		},
	})

	if _, err := runner.execNodeViaFramework(context.Background(), runner.nodesByID["up-1"], ExecToken{NodeID: "up-1", InPin: "In"}); err != nil {
		t.Fatalf("execNodeViaFramework error = %v", err)
	}

	if len(input.mouseUpHWND) != 1 || input.mouseUpHWND[0] != 84 || input.mouseUpButton[0] != "middle" {
		t.Fatalf("backend MouseUp = hwnds %#v buttons %#v", input.mouseUpHWND, input.mouseUpButton)
	}
	assertSingleTraceSource(t, rt.TraceRecords(), "mouse-up", "trace-mouse-up-container", "up-1", "MouseHoldStop")
}

func TestExecNodeViaFramework_SwipeTraceIncludesNodeSource(t *testing.T) {
	rt, input, runner := newTraceSourceRuntime(t, "trace-drag-container", container.GraphNode{
		ID:   "swipe-1",
		Kind: "Swipe",
		Config: map[string]any{
			"Begin":      node.Point{X: 0.1, Y: 0.2},
			"End":        node.Point{X: 0.8, Y: 0.9},
			"Button":     "left",
			"DurationMs": 300,
		},
	})

	if _, err := runner.execNodeViaFramework(context.Background(), runner.nodesByID["swipe-1"], ExecToken{NodeID: "swipe-1", InPin: "In"}); err != nil {
		t.Fatalf("execNodeViaFramework error = %v", err)
	}

	if len(input.dragHWND) != 1 || input.dragHWND[0] != 84 ||
		input.dragX1[0] != 0.1 || input.dragY1[0] != 0.2 ||
		input.dragX2[0] != 0.8 || input.dragY2[0] != 0.9 ||
		input.dragButton[0] != "left" || input.dragDurationMs[0] != 300 {
		t.Fatalf("backend Drag = hwnds %#v from (%#v,%#v) to (%#v,%#v) buttons %#v durations %#v",
			input.dragHWND, input.dragX1, input.dragY1, input.dragX2, input.dragY2, input.dragButton, input.dragDurationMs)
	}
	records := rt.TraceRecords()
	assertSingleTraceSource(t, records, "drag", "trace-drag-container", "swipe-1", "Swipe")
	if len(records[0].CoordinateSteps) != 2 {
		t.Fatalf("coordinate steps len = %d, want 2", len(records[0].CoordinateSteps))
	}
}

func TestExecNodeViaFramework_MouseMoveRelTraceIncludesNodeSource(t *testing.T) {
	rt, input, runner := newTraceSourceRuntime(t, "trace-move-relative-container", container.GraphNode{
		ID:   "move-rel-1",
		Kind: "MouseMoveRel",
		Config: map[string]any{
			"Dx":         10,
			"Dy":         -20,
			"DurationMs": 150,
		},
	})

	if _, err := runner.execNodeViaFramework(context.Background(), runner.nodesByID["move-rel-1"], ExecToken{NodeID: "move-rel-1", InPin: "In"}); err != nil {
		t.Fatalf("execNodeViaFramework error = %v", err)
	}

	if len(input.moveRelHWND) != 1 || input.moveRelHWND[0] != 84 || input.moveRelDx[0] != 10 || input.moveRelDy[0] != -20 || input.moveRelDuration[0] != 150 {
		t.Fatalf("backend MouseMoveRel = hwnds %#v dx %#v dy %#v durations %#v", input.moveRelHWND, input.moveRelDx, input.moveRelDy, input.moveRelDuration)
	}
	records := rt.TraceRecords()
	assertSingleTraceSource(t, records, "move-relative", "trace-move-relative-container", "move-rel-1", "MouseMoveRel")
	if len(records[0].CoordinateSteps) != 0 {
		t.Fatalf("coordinate steps len = %d, want 0", len(records[0].CoordinateSteps))
	}
}

func TestExecNodeViaFramework_CaptureTraceIncludesNodeSource(t *testing.T) {
	rt, _, runner := newTraceSourceRuntime(t, "trace-capture-container", container.GraphNode{
		ID:     "capture-1",
		Kind:   "Capture",
		Config: map[string]any{},
	})
	img := image.NewRGBA(image.Rect(0, 0, 2, 2))
	img.SetRGBA(0, 0, color.RGBA{R: 255, A: 255})
	installTestWin32Capture(rt, fakeCapture{img: img})

	if _, err := runner.execNodeViaFramework(context.Background(), runner.nodesByID["capture-1"], ExecToken{NodeID: "capture-1", InPin: "In"}); err != nil {
		t.Fatalf("execNodeViaFramework error = %v", err)
	}

	records := rt.TraceRecords()
	assertSingleTraceSource(t, records, "screenshot", "trace-capture-container", "capture-1", "Capture")
	if records[0].Target.ID != "win32:84" {
		t.Fatalf("trace target id = %q, want win32:84", records[0].Target.ID)
	}
}

func newTraceSourceRuntime(t *testing.T, containerID string, graphNode container.GraphNode) (*RuntimeContext, *recordingRuntimeInput, *ContainerRunner) {
	t.Helper()
	c := &container.Container{
		SchemaVersion: 1,
		ID:            containerID,
		Name:          containerID,
		Graph: container.Graph{
			Nodes: []container.GraphNode{graphNode},
		},
	}
	rt := NewRuntimeContext(c, execution.NewInputBus(), NoopMatcher{}, nil, nil, nil, 0)
	rt.SetActiveWindow(winutil.WindowHandle{HWND: 84, Title: "After Effects", ClientW: 1920, ClientH: 1080})
	input := &recordingRuntimeInput{}
	installTestWin32Input(rt, input)
	return rt, input, NewContainerRunner(rt)
}

func assertSingleTraceSource(t *testing.T, records []automationtrace.ActionRecord, action, containerID, nodeID, nodeKind string) {
	t.Helper()
	if len(records) != 1 {
		t.Fatalf("trace len = %d, want 1", len(records))
	}
	record := records[0]
	if record.Action != action || record.Status != automationtrace.StatusSuccess {
		t.Fatalf("trace action/status = %s/%s", record.Action, record.Status)
	}
	if record.Source.ContainerID != containerID ||
		record.Source.NodeID != nodeID ||
		record.Source.NodeKind != nodeKind ||
		record.Source.InPin != "In" {
		t.Fatalf("record source = %#v", record.Source)
	}
}
