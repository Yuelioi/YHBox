package actiontransform

import "testing"

func TestNormalize_SortsByTimestamp(t *testing.T) {
	in := []Event{
		{TimestampMs: 300, Vk: "C"},
		{TimestampMs: 100, Vk: "A"},
		{TimestampMs: 200, Vk: "B"},
	}
	out := Normalize(in)
	if len(out) != 3 {
		t.Fatalf("len=%d want 3", len(out))
	}
	wantOrder := []string{"A", "B", "C"}
	for i, ev := range out {
		if ev.Vk != wantOrder[i] {
			t.Errorf("idx %d: vk=%q want %q", i, ev.Vk, wantOrder[i])
		}
	}
	// 原 slice 不应被改动（Normalize 做了 copy）
	if in[0].Vk != "C" {
		t.Errorf("Normalize mutated input slice: in[0].Vk=%q", in[0].Vk)
	}
}

func TestNormalize_StableForEqualTimestamps(t *testing.T) {
	in := []Event{
		{TimestampMs: 100, Vk: "first"},
		{TimestampMs: 100, Vk: "second"},
		{TimestampMs: 100, Vk: "third"},
		{TimestampMs: 50, Vk: "earlier"},
	}
	out := Normalize(in)
	wantOrder := []string{"earlier", "first", "second", "third"}
	for i, ev := range out {
		if ev.Vk != wantOrder[i] {
			t.Errorf("idx %d: vk=%q want %q", i, ev.Vk, wantOrder[i])
		}
	}
}

func TestNormalize_Empty(t *testing.T) {
	out := Normalize(nil)
	if len(out) != 0 {
		t.Errorf("len=%d want 0", len(out))
	}
}
