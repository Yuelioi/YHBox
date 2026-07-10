package input

import (
	"context"
	"errors"
	"fmt"
	"testing"

	"github.com/yottaapp/yotta/internal/node"
)

// TypeText 记录格式: "TypeText:<text>"
func (r *recordingInput) TypeText(s string) error {
	r.calls = append(r.calls, fmt.Sprintf("TypeText:%s", s))
	return r.err
}

func TestInputText_HappyPath(t *testing.T) {
	node.ResetRegistryForTest()
	node.Register(&InputText{})
	rn, _ := node.Get("InputText")

	rec := &recordingInput{}
	r := node.RunNode(context.Background(), rn, nil,
		map[string]any{itInText: "hello"},
		nil, withInput(rec), false)

	if r.Error != nil {
		t.Fatal(r.Error)
	}
	if r.ExitName != itOutDone {
		t.Errorf("exit = %q, want Done", r.ExitName)
	}
	if len(rec.calls) != 1 || rec.calls[0] != "TypeText:hello" {
		t.Errorf("calls = %v, want [TypeText:hello]", rec.calls)
	}
}

func TestInputText_EmptyText_ValidationError(t *testing.T) {
	node.ResetRegistryForTest()
	node.Register(&InputText{})
	rn, _ := node.Get("InputText")

	r := node.RunNode(context.Background(), rn, nil,
		map[string]any{itInText: ""},
		nil, withInput(&recordingInput{}), false)

	if len(r.Validation) == 0 {
		t.Fatal("expected validation error for empty text")
	}
	found := false
	for _, e := range r.Validation {
		if e.Code == "MISSING_TEXT" {
			found = true
		}
	}
	if !found {
		t.Errorf("validation = %v, want MISSING_TEXT", r.Validation)
	}
}

func TestInputText_BackendError_Propagates(t *testing.T) {
	node.ResetRegistryForTest()
	node.Register(&InputText{})
	rn, _ := node.Get("InputText")

	rec := &recordingInput{err: errors.New("hwnd closed")}
	r := node.RunNode(context.Background(), rn, nil,
		map[string]any{itInText: "hello"},
		nil, withInput(rec), false)

	if r.Error == nil {
		t.Fatal("expected backend error to propagate")
	}
}

func TestInputText_Unicode_Injected(t *testing.T) {
	node.ResetRegistryForTest()
	node.Register(&InputText{})
	rn, _ := node.Get("InputText")

	rec := &recordingInput{}
	r := node.RunNode(context.Background(), rn, nil,
		map[string]any{itInText: "你好世界"},
		nil, withInput(rec), false)

	if r.Error != nil {
		t.Fatal(r.Error)
	}
	if len(rec.calls) != 1 || rec.calls[0] != "TypeText:你好世界" {
		t.Errorf("calls = %v, want [TypeText:你好世界]", rec.calls)
	}
}
