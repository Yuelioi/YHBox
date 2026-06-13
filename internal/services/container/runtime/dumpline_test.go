package runtime

import (
	"strings"
	"testing"
	"time"

	nodepkg "yotta/internal/node"
)

func TestFormatDumpLine_Basic(t *testing.T) {
	spec := &nodepkg.Spec{
		Kind:    "CheckTemplate",
		Inputs:  []nodepkg.InputSpec{{Name: "key"}, {Name: "thr"}},
		Outputs: []nodepkg.OutputSpec{{Name: "hit"}, {Name: "x"}, {Name: "y"}},
	}
	in := map[string]any{"key": "hook", "thr": 0.9}
	out := map[string]any{"hit": true, "x": 820, "y": 440}
	line, key := FormatDumpLine(spec, "checkHook", "n_a3f", in, out, "hit", 1500*time.Millisecond, nil)
	if !strings.HasPrefix(line, "CheckTemplate(checkHook, n_a3f) in{") {
		t.Fatalf("line prefix wrong: %q", line)
	}
	if !strings.Contains(line, "in{key=hook, thr=0.9}") || !strings.Contains(line, "out{hit=true, x=820, y=440}") {
		t.Fatalf("line body wrong: %q", line)
	}
	if !strings.Contains(line, "→hit") {
		t.Fatalf("exit name must render: %q", line)
	}
	if !strings.Contains(line, "took=1.5s") {
		t.Fatalf("elapsed must render: %q", line)
	}
	if strings.Contains(key, "took") {
		t.Fatalf("took 不该进 key (会害合并): %q", key)
	}
	if strings.Contains(key, "checkHook") || strings.Contains(key, "n_a3f") {
		t.Fatalf("key must exclude name/id: %q", key)
	}
}

func TestFormatDumpLine_ErrorAndEmptySegments(t *testing.T) {
	spec := &nodepkg.Spec{Kind: "K", Inputs: []nodepkg.InputSpec{{Name: "a"}}}
	line, _ := FormatDumpLine(spec, "", "n1", map[string]any{"a": 1}, nil, "", 0, fmtErr("boom"))
	if strings.Contains(line, "out{") {
		t.Fatalf("empty out must be omitted: %q", line)
	}
	if !strings.Contains(line, "err=boom") {
		t.Fatalf("err must appear: %q", line)
	}
	if !strings.HasPrefix(line, "K(n1)") {
		t.Fatalf("empty name should render Kind(id): %q", line)
	}
}

func TestDumpValue_TruncateAndStable(t *testing.T) {
	long := strings.Repeat("x", 100)
	if v := dumpValue(long); len(v) > 70 || !strings.HasSuffix(v, "…") {
		t.Fatalf("long string must truncate with …: len=%d", len(v))
	}
	m := map[string]any{"b": 2, "a": 1}
	got := dumpValue(m)
	if !strings.Contains(got, "a") || strings.Index(got, "a") > strings.Index(got, "b") {
		t.Fatalf("map must be key-sorted: %q", got)
	}
	defer func() {
		if r := recover(); r != nil {
			t.Fatalf("dumpValue panicked: %v", r)
		}
	}()
	_ = dumpValue(make(chan int))
}

func fmtErr(s string) error { return &simpleErr{s} }

type simpleErr struct{ s string }

func (e *simpleErr) Error() string { return e.s }
