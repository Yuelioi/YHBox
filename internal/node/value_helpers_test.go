package node

import "testing"

func TestFormatValue(t *testing.T) {
	cases := []struct {
		in   any
		want string
	}{
		{nil, "null"},
		{true, "true"},
		{false, "false"},
		{3.14, "3.14"},
		{42, "42"},
		{int64(7), "7"},
		{"s", "s"},
	}
	for _, tc := range cases {
		if got := FormatValue(tc.in); got != tc.want {
			t.Errorf("FormatValue(%v) = %q, want %q", tc.in, got, tc.want)
		}
	}
}

func TestLooseEqual(t *testing.T) {
	if !LooseEqual(nil, nil) {
		t.Error("nil,nil want true")
	}
	if LooseEqual(nil, "") {
		t.Error(`nil,"" want false (FormatValue(nil)="null")`)
	}
	if !LooseEqual(1.0, 1.0) || LooseEqual(1.0, 2.0) {
		t.Error("same-type compare broken")
	}
	if !LooseEqual(1.0, "1") {
		t.Error(`cross-type 1.0 vs "1" want true (串比)`)
	}
}

// 防护: slice/map 同类型直比是 Go 运行时 panic — LooseEqual 必须退 FormatValue 串比.
func TestLooseEqual_UncomparableNoPanic(t *testing.T) {
	if !LooseEqual([]any{1, "a"}, []any{1, "a"}) {
		t.Error("equal slices (string-repr) want true")
	}
	if LooseEqual([]any{1}, []any{2}) {
		t.Error("different slices want false")
	}
	if !LooseEqual(map[string]any{"k": 1}, map[string]any{"k": 1}) {
		t.Error("equal maps (string-repr) want true")
	}
}
