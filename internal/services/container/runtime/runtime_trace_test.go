package runtime

import (
	"testing"
	"time"

	"github.com/yottaapp/yotta/internal/automation/target"
	automationtrace "github.com/yottaapp/yotta/internal/automation/trace"
	"github.com/yottaapp/yotta/internal/services/container"
	"github.com/yottaapp/yotta/internal/services/execution"
)

func newTraceRuntime() *RuntimeContext {
	return NewRuntimeContext(&container.Container{ID: "trace-container"}, execution.NewInputBus(), NoopMatcher{}, nil, nil, nil, 0)
}

func traceTestRecord(action string) automationtrace.ActionRecord {
	started := time.UnixMilli(1000)
	ended := time.UnixMilli(1250)
	return automationtrace.ActionRecord{
		Action: action,
		Source: automationtrace.ActionSource{
			ContainerID: "trace-container",
			NodeID:      "node-1",
			NodeKind:    "ClickAt",
			InPin:       "In",
		},
		Target: target.Target{
			ID:   "win32:100",
			Kind: target.KindWin32Window,
			Ref:  target.TargetRef{HWND: 100},
		},
		Backend:   "sendinput",
		Request:   map[string]any{"x": 0.25},
		Result:    map[string]any{"ok": true},
		Status:    automationtrace.StatusSuccess,
		StartedAt: started,
		EndedAt:   ended,
	}
}

func TestRuntimeContextTraceRecorderStoresRecords(t *testing.T) {
	rt := newTraceRuntime()

	rt.TraceRecorder().Record(traceTestRecord("click"))

	records := rt.TraceRecords()
	if len(records) != 1 {
		t.Fatalf("len(records) = %d, want 1", len(records))
	}
	if records[0].Action != "click" {
		t.Fatalf("records[0].Action = %q, want click", records[0].Action)
	}
}

func TestRuntimeContextTraceRecordsReturnsCopy(t *testing.T) {
	rt := newTraceRuntime()
	rt.TraceRecorder().Record(traceTestRecord("key-chord"))

	records := rt.TraceRecords()
	records[0].Action = "mutated"

	next := rt.TraceRecords()
	if next[0].Action != "key-chord" {
		t.Fatalf("stored action = %q, want key-chord", next[0].Action)
	}
}

func TestRuntimeContextTraceIsPerRuntime(t *testing.T) {
	rt1 := newTraceRuntime()
	rt2 := newTraceRuntime()

	rt1.TraceRecorder().Record(traceTestRecord("scroll"))

	if records := rt2.TraceRecords(); len(records) != 0 {
		t.Fatalf("rt2 len(records) = %d, want 0", len(records))
	}
}

func TestRuntimeContextClearTrace(t *testing.T) {
	rt1 := newTraceRuntime()
	rt2 := newTraceRuntime()
	rt1.TraceRecorder().Record(traceTestRecord("text"))
	rt2.TraceRecorder().Record(traceTestRecord("click"))

	rt1.ClearTrace()

	if records := rt1.TraceRecords(); len(records) != 0 {
		t.Fatalf("rt1 len(records) after clear = %d, want 0", len(records))
	}
	if records := rt2.TraceRecords(); len(records) != 1 {
		t.Fatalf("rt2 len(records) after rt1 clear = %d, want 1", len(records))
	}
}

func TestRuntimeContextTraceRecorderEmitsActionTraceEvent(t *testing.T) {
	rt := newTraceRuntime()
	var gotName string
	var gotData any
	rt.Emit = func(name string, data any) {
		gotName = name
		gotData = data
	}

	rt.TraceRecorder().Record(traceTestRecord("click"))

	if gotName != "container:action-trace" {
		t.Fatalf("event name = %q, want container:action-trace", gotName)
	}
	payload, ok := gotData.(map[string]any)
	if !ok {
		t.Fatalf("payload type = %T, want map[string]any", gotData)
	}
	if payload["containerId"] != "trace-container" ||
		payload["action"] != "click" ||
		payload["backend"] != "sendinput" ||
		payload["status"] != automationtrace.StatusSuccess ||
		payload["durationMs"] != int64(250) {
		t.Fatalf("payload = %#v", payload)
	}
	source, ok := payload["source"].(automationtrace.ActionSource)
	if !ok || source.NodeID != "node-1" || source.NodeKind != "ClickAt" || source.InPin != "In" {
		t.Fatalf("source payload = %#v", payload["source"])
	}
	if records := rt.TraceRecords(); len(records) != 1 || records[0].Action != "click" {
		t.Fatalf("memory trace records = %#v", records)
	}
}

func TestRuntimeContextTraceRecorderSkipsPresentationWhenDiagnosticsDisabled(t *testing.T) {
	rt := newTraceRuntime()
	rt.SetDiagnosticsEnabled(func() bool { return false })
	emitted := false
	rt.Emit = func(string, any) { emitted = true }

	rt.TraceRecorder().Record(traceTestRecord("click"))

	if emitted {
		t.Fatal("disabled diagnostics emitted an action trace payload")
	}
	if records := rt.TraceRecords(); len(records) != 0 {
		t.Fatalf("disabled diagnostics retained %d action records", len(records))
	}
}
