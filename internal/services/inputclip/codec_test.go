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
	if got[4] != 2 || got[5] != 0 || got[6] != 0 || got[7] != 0 {
		t.Errorf("version = % x, want 02 00 00 00", got[4:8])
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
	if got.ID != "" || got.Label != "" {
		t.Errorf("content carrier leaked mutable asset metadata: %+v", got)
	}
	if len(got.Events) != 2 {
		t.Fatalf("events len = %d, want 2", len(got.Events))
	}
	if got.Events[0].A != 0x57 || got.Events[1].TUs != 100000 {
		t.Errorf("events 字段错: %+v", got.Events)
	}
}

func TestCodecContentIdentityExcludesPresentationMetadata(t *testing.T) {
	events := []Event{{TUs: 0, Type: EventTypeKeyDown, A: 0x41}}
	encode := func(id, label string) []byte {
		var buffer bytes.Buffer
		clip := &InputClip{ID: id, Label: label, Meta: ClipMeta{MouseMode: "absolute", BaseResolution: [2]int{1920, 1080}}, Events: events}
		if err := Encode(&buffer, clip); err != nil {
			t.Fatal(err)
		}
		return buffer.Bytes()
	}
	if first, second := encode("clip-a", "First"), encode("clip-b", "Second"); !bytes.Equal(first, second) {
		t.Fatal("presentation metadata changed InputClip content identity")
	}
}

func TestDecodeRejectsNonCanonicalOrCorruptCarrier(t *testing.T) {
	clip := &InputClip{Meta: ClipMeta{MouseMode: "absolute", BaseResolution: [2]int{1920, 1080}}, Events: []Event{{TUs: 0, Type: EventTypeKeyDown, A: 0x41}}}
	var buffer bytes.Buffer
	if err := Encode(&buffer, clip); err != nil {
		t.Fatal(err)
	}
	valid := buffer.Bytes()
	for _, mutate := range []func([]byte){
		func(data []byte) { data[4] = 1 },
		func(data []byte) { data[len(data)-1] ^= 1 },
		func(data []byte) { data[12] = ' ' },
	} {
		candidate := append([]byte(nil), valid...)
		mutate(candidate)
		if _, err := Decode(bytes.NewReader(candidate)); err == nil {
			t.Fatal("Decode accepted a corrupt or non-canonical carrier")
		}
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
