package io

import (
	"context"
	"testing"

	"yotta/internal/node"
)

// captureLog 实现 node.LogService, 把调用累积到 entries.
type captureLog struct {
	entries []string
}

func (c *captureLog) Debug(format string, args ...any) {
	c.entries = append(c.entries, "DEBUG "+format)
}
func (c *captureLog) Info(format string, args ...any) {
	c.entries = append(c.entries, "INFO "+format)
}
func (c *captureLog) Warn(format string, args ...any) {
	c.entries = append(c.entries, "WARN "+format)
}

// withLog 构造一个用 captureLog 替换 LogService 的 ServiceBundle, 其余 stub.
func withLog(cap *captureLog) node.ServiceBundle {
	b := node.StubServices()
	b.Log = cap
	return b
}

func TestLog_Info(t *testing.T) {
	node.ResetRegistryForTest()
	node.Register(&Log{})
	rn, _ := node.Get("Log")

	cap := &captureLog{}
	r := node.RunNode(context.Background(), rn, nil,
		map[string]any{logInMessage: "hello", logInLevel: "info"},
		nil, withLog(cap))

	if r.Error != nil {
		t.Fatal(r.Error)
	}
	if len(cap.entries) != 1 || cap.entries[0] != "INFO %s" {
		t.Errorf("entries = %v, want 1 INFO line", cap.entries)
	}
}

func TestLog_Warn(t *testing.T) {
	node.ResetRegistryForTest()
	node.Register(&Log{})
	rn, _ := node.Get("Log")

	cap := &captureLog{}
	node.RunNode(context.Background(), rn, nil,
		map[string]any{logInMessage: "danger", logInLevel: "warn"},
		nil, withLog(cap))

	if len(cap.entries) != 1 || cap.entries[0] != "WARN %s" {
		t.Errorf("entries = %v, want 1 WARN line", cap.entries)
	}
}

func TestLog_WildcardMessage(t *testing.T) {
	node.ResetRegistryForTest()
	node.Register(&Log{})
	rn, _ := node.Get("Log")

	cap := &captureLog{}
	// Message 是 wildcard "*", 接任意类型. 这里传 number.
	r := node.RunNode(context.Background(), rn, nil,
		map[string]any{logInMessage: 42, logInLevel: "info"},
		nil, withLog(cap))

	if r.Error != nil {
		t.Fatal(r.Error)
	}
	if r.DisplayText == "" {
		t.Error("Display should emit")
	}
}
