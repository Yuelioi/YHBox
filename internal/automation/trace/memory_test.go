package trace

import (
	"testing"
	"time"

	"github.com/yottaapp/yotta/internal/automation/target"
)

func TestMemoryRecorderBoundsAndPreservesNewestOrder(t *testing.T) {
	recorder := NewMemoryRecorder()
	for i := 0; i < defaultMemoryCapacity+10; i++ {
		recorder.Record(ActionRecord{Action: string(rune(i))})
	}
	records := recorder.Records()
	if len(records) != defaultMemoryCapacity {
		t.Fatalf("records=%d, want %d", len(records), defaultMemoryCapacity)
	}
	if records[0].Action != string(rune(10)) || records[len(records)-1].Action != string(rune(defaultMemoryCapacity+9)) {
		t.Fatal("bounded recorder did not retain newest records in order")
	}
}

func TestMemoryRecorderRecordAndRecords(t *testing.T) {
	rec := NewMemoryRecorder()
	started := time.UnixMilli(1000)
	ended := time.UnixMilli(1100)

	rec.Record(ActionRecord{
		Action:    "click",
		Target:    target.Target{ID: "win32:42", Kind: target.KindWin32Window, Ref: target.TargetRef{HWND: 42}},
		Backend:   "win32",
		Status:    StatusSuccess,
		StartedAt: started,
		EndedAt:   ended,
	})

	records := rec.Records()
	if len(records) != 1 {
		t.Fatalf("records len = %d, want 1", len(records))
	}
	got := records[0]
	if got.Action != "click" || got.Target.ID != "win32:42" || got.Backend != "win32" {
		t.Fatalf("unexpected record: %#v", got)
	}
	if got.Status != StatusSuccess {
		t.Fatalf("status = %q, want %q", got.Status, StatusSuccess)
	}
	if got.Duration() != 100*time.Millisecond {
		t.Fatalf("duration = %s, want 100ms", got.Duration())
	}

	records[0].Action = "mutated"
	if rec.Records()[0].Action != "click" {
		t.Fatalf("Records returned mutable backing slice")
	}
}

func TestMemoryRecorderClear(t *testing.T) {
	rec := NewMemoryRecorder()
	rec.Record(ActionRecord{Action: "click"})
	rec.Clear()
	if got := len(rec.Records()); got != 0 {
		t.Fatalf("records len after clear = %d, want 0", got)
	}
}

func TestActionRecordStoresSourceMetadata(t *testing.T) {
	rec := NewMemoryRecorder()
	rec.Record(ActionRecord{
		Action: "click",
		Source: ActionSource{
			ContainerID: "container-1",
			NodeID:      "click-1",
			NodeKind:    "ClickAt",
			InPin:       "In",
		},
	})

	records := rec.Records()
	if records[0].Source.ContainerID != "container-1" ||
		records[0].Source.NodeID != "click-1" ||
		records[0].Source.NodeKind != "ClickAt" ||
		records[0].Source.InPin != "In" {
		t.Fatalf("source = %#v", records[0].Source)
	}
}
