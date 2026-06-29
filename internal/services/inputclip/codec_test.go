package inputclip

import (
	"bytes"
	"testing"
)

func TestEncodeEmptyClip(t *testing.T) {
	clip := &InputClip{
		ID: "test", Label: "empty",
		Meta: ClipMeta{MouseMode: "relative", BaseResolution: [2]int{1920, 1080}},
	}
	var buf bytes.Buffer
	if err := Encode(&buf, clip); err != nil {
		t.Fatalf("Encode failed: %v", err)
	}
	got := buf.Bytes()
	if len(got) < 12 {
		t.Fatalf("encoded length %d < 12 (Magic+Version)", len(got))
	}
	if string(got[:4]) != "ICLP" {
		t.Errorf("magic = %q, want ICLP", got[:4])
	}
	// version 4 字节 little-endian
	if got[4] != 1 || got[5] != 0 || got[6] != 0 || got[7] != 0 {
		t.Errorf("version = % x, want 01 00 00 00", got[4:8])
	}
}

func TestEncodeWithEvents(t *testing.T) {
	clip := &InputClip{
		ID: "t1", Label: "with-events",
		Meta: ClipMeta{MouseMode: "relative", BaseResolution: [2]int{1920, 1080}},
		Events: []Event{
			{TUs: 0, Seq: 0, Type: EventTypeKeyDown, A: 0x57},
			{TUs: 100000, Seq: 1, Type: EventTypeKeyUp, A: 0x57},
		},
	}
	clip.UpdateDuration()
	var buf bytes.Buffer
	if err := Encode(&buf, clip); err != nil {
		t.Fatalf("Encode failed: %v", err)
	}
	// 解码 round-trip
	got, err := Decode(bytes.NewReader(buf.Bytes()))
	if err != nil {
		t.Fatalf("Decode failed: %v", err)
	}
	if got.ID != "t1" || got.Label != "with-events" {
		t.Errorf("header round-trip 错: %+v", got)
	}
	if len(got.Events) != 2 {
		t.Fatalf("events len = %d, want 2", len(got.Events))
	}
	if got.Events[0].A != 0x57 || got.Events[1].TUs != 100000 {
		t.Errorf("events 字段错: %+v", got.Events)
	}
}

func TestCodecFooterOffsets(t *testing.T) {
	// 200 events → 2 chunks (127 + 73)
	events := make([]Event, 200)
	for i := range events {
		events[i] = Event{TUs: uint64(i * 1000), Seq: uint32(i), Type: EventTypeKeyDown, A: 0x57}
	}
	clip := &InputClip{
		ID: "f", Label: "footer-test",
		Meta:   ClipMeta{MouseMode: "relative", BaseResolution: [2]int{1920, 1080}},
		Events: events,
	}
	clip.UpdateDuration()
	var buf bytes.Buffer
	if err := Encode(&buf, clip); err != nil {
		t.Fatalf("Encode: %v", err)
	}
	got, err := Decode(bytes.NewReader(buf.Bytes()))
	if err != nil {
		t.Fatalf("Decode: %v", err)
	}
	if len(got.Events) != 200 {
		t.Fatalf("event count = %d, want 200", len(got.Events))
	}
	for i := 0; i < 200; i++ {
		if got.Events[i].TUs != uint64(i*1000) {
			t.Fatalf("event[%d].TUs = %d, want %d", i, got.Events[i].TUs, i*1000)
		}
	}
}
