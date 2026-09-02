package problem

import (
	"bytes"
	"testing"
)

func TestParamsAreCanonicalBoundedAndImmutable(t *testing.T) {
	params, err := New(map[string]any{"thresholdPpm": 850000, "reason": "timeout"})
	if err != nil {
		t.Fatal(err)
	}
	want := []byte(`{"reason":"timeout","thresholdPpm":850000}`)
	if !bytes.Equal(params.Bytes(), want) {
		t.Fatalf("params = %s, want %s", params.Bytes(), want)
	}
	copy := params.Bytes()
	copy[0] = '['
	if !bytes.Equal(params.Bytes(), want) {
		t.Fatal("Params.Bytes exposed mutable storage")
	}
}

func TestParamsRejectNonObjectAndNonCanonicalJSON(t *testing.T) {
	for _, raw := range [][]byte{[]byte(`[]`), []byte(`{"b":1,"a":2}`), []byte(`{"bad key":1}`)} {
		if _, err := Open(raw); err == nil {
			t.Fatalf("Open(%s) succeeded", raw)
		}
	}
}
