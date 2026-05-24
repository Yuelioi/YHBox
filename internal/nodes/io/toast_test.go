package io

import (
	"context"
	"strings"
	"testing"

	"yhbox/internal/node"
)

func TestToast_HappyPath(t *testing.T) {
	node.ResetRegistryForTest()
	node.Register(&Toast{})
	rn, _ := node.Get("Toast")

	cap := &captureLog{}
	r := node.RunNode(context.Background(), rn, nil,
		map[string]any{
			toastInTitle:   "提示",
			toastInMessage: "钓鱼开始",
			toastInColor:   "success",
		},
		nil, withLog(cap))

	if r.Error != nil {
		t.Fatal(r.Error)
	}
	if r.ExitName != toastOutDone {
		t.Errorf("exit = %q, want Done", r.ExitName)
	}
	if len(cap.entries) != 1 {
		t.Fatalf("entries = %v, want 1 log line", cap.entries)
	}
	if !strings.HasPrefix(cap.entries[0], "INFO ") {
		t.Errorf("entry[0] = %q, want INFO-prefixed", cap.entries[0])
	}
	if r.DisplayText == "" {
		t.Error("Display should emit non-empty text")
	}
}

func TestToast_RequiredMessageMissing(t *testing.T) {
	node.ResetRegistryForTest()
	node.Register(&Toast{})
	rn, _ := node.Get("Toast")

	// 不传 Message → REQUIRED_FIELD_MISSING
	r := node.RunNode(context.Background(), rn, nil, nil, nil, node.StubServices())
	if len(r.Validation) == 0 {
		t.Fatal("expected REQUIRED_FIELD_MISSING for Message")
	}
	if r.Validation[0].Code != "REQUIRED_FIELD_MISSING" {
		t.Errorf("validation[0].Code = %q, want REQUIRED_FIELD_MISSING", r.Validation[0].Code)
	}
}

func TestToast_WildcardMessage(t *testing.T) {
	node.ResetRegistryForTest()
	node.Register(&Toast{})
	rn, _ := node.Get("Toast")

	// Message 是 wildcard — 接 number / Point / etc.
	cap := &captureLog{}
	r := node.RunNode(context.Background(), rn, nil,
		map[string]any{
			toastInTitle:   "Count",
			toastInMessage: 42,
		},
		nil, withLog(cap))

	if r.Error != nil {
		t.Fatal(r.Error)
	}
	if len(cap.entries) != 1 {
		t.Errorf("entries = %v, want 1", cap.entries)
	}
}
