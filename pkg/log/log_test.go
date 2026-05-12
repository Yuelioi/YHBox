package log

import (
	"bytes"
	"strings"
	"testing"
)

// TestLogger_MultiSink_Log Log 方法应该把同一条消息写到所有 sink。
func TestLogger_MultiSink_Log(t *testing.T) {
	var a, b bytes.Buffer
	l := New(&a, &b)
	l.Log(SYSTEM, "hello %s", "world")

	if !strings.Contains(a.String(), "[SYSTEM] hello world") {
		t.Errorf("sink a missing message: %q", a.String())
	}
	if !strings.Contains(b.String(), "[SYSTEM] hello world") {
		t.Errorf("sink b missing message: %q", b.String())
	}
}

// TestLogger_MultiSink_Printf Printf 也应该多 sink 写入。
func TestLogger_MultiSink_Printf(t *testing.T) {
	var a, b bytes.Buffer
	l := New(&a, &b)
	l.Printf("[FIGHT] elapsed=%ds", 3)

	if !strings.Contains(a.String(), "[FIGHT] elapsed=3s") {
		t.Errorf("sink a missing: %q", a.String())
	}
	if !strings.Contains(b.String(), "[FIGHT] elapsed=3s") {
		t.Errorf("sink b missing: %q", b.String())
	}
}

// TestLogger_NoSink 不传 sink 应安全，不 panic。
func TestLogger_NoSink(t *testing.T) {
	l := New()
	l.Log(SYSTEM, "noop")
	l.Printf("[X] noop")
	// 不崩就行
}

// TestLogger_TimestampFormat 每条日志应以 HH:MM:SS 开头。
func TestLogger_TimestampFormat(t *testing.T) {
	var buf bytes.Buffer
	l := New(&buf)
	l.Log(SYSTEM, "msg")
	out := buf.String()
	if len(out) < 8 || out[2] != ':' || out[5] != ':' {
		t.Errorf("missing HH:MM:SS prefix: %q", out)
	}
}
