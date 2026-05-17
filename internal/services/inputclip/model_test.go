package inputclip

import (
	"testing"
	"unsafe"
)

func TestEventTypeConstants(t *testing.T) {
	if EventTypeKeyDown != 1 || EventTypeRawDelta != 6 {
		t.Errorf("EventType enum 值意外: KeyDown=%d, RawDelta=%d", EventTypeKeyDown, EventTypeRawDelta)
	}
}

func TestEventLessOrdering(t *testing.T) {
	a := Event{TUs: 100, Seq: 1}
	b := Event{TUs: 100, Seq: 2}
	c := Event{TUs: 200, Seq: 0}
	if !a.Less(b) {
		t.Error("同 tUs seq 升序: a should < b")
	}
	if !a.Less(c) {
		t.Error("不同 tUs: a should < c")
	}
	if b.Less(a) {
		t.Error("b should not < a")
	}
}

func TestInputClipDurationFromLastEvent(t *testing.T) {
	clip := &InputClip{
		Events: []Event{
			{TUs: 0, Seq: 0, Type: EventTypeKeyDown, A: 0x57},
			{TUs: 500000, Seq: 1, Type: EventTypeKeyUp, A: 0x57},
		},
	}
	clip.UpdateDuration()
	if clip.DurationUs != 500000 {
		t.Errorf("DurationUs = %d, want 500000", clip.DurationUs)
	}
}

func TestEventSizeFixed32Bytes(t *testing.T) {
	const want = 32
	got := unsafe.Sizeof(Event{})
	if got != want {
		t.Errorf("Event size = %d 字节, want %d (binary codec 依赖固定布局)", got, want)
	}
}
