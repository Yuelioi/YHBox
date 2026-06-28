package runtime

import (
	"testing"

	"yotta/internal/automation/target"
	automationtrace "yotta/internal/automation/trace"
	"yotta/internal/services/container"
	"yotta/internal/services/execution"
)

func newTraceRuntime() *RuntimeContext {
	return NewRuntimeContext(&container.Container{}, execution.NewInputBus(), NoopMatcher{}, nil, nil, nil, 0)
}

func traceTestRecord(action string) automationtrace.ActionRecord {
	return automationtrace.ActionRecord{
		Action: action,
		Target: target.Target{
			ID:   "win32:100",
			Kind: target.KindWin32Window,
			Ref:  target.TargetRef{HWND: 100},
		},
		Status: automationtrace.StatusSuccess,
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
