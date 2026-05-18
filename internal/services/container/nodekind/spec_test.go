// internal/services/container/nodekind/spec_test.go
package nodekind

import (
	"strings"
	"testing"
)

func TestRegisterAndGet(t *testing.T) {
	saved := byKind
	byKind = map[string]*Spec{}
	defer func() { byKind = saved }()

	Register(&Spec{Kind: "TestKind", Group: "test", ExecIn: []string{"in"}, ExecOut: []string{"out"}})
	s, ok := Get("TestKind")
	if !ok {
		t.Fatal("Get returned !ok for registered kind")
	}
	if s.Kind != "TestKind" || s.Group != "test" {
		t.Errorf("Get returned wrong spec: %+v", s)
	}
	if !IsKnown("TestKind") {
		t.Error("IsKnown=false for registered kind")
	}
	if IsKnown("Nope") {
		t.Error("IsKnown=true for unregistered kind")
	}
}

func TestRegisterDuplicatePanics(t *testing.T) {
	saved := byKind
	byKind = map[string]*Spec{}
	defer func() { byKind = saved }()

	Register(&Spec{Kind: "Dup"})
	defer func() {
		r := recover()
		if r == nil {
			t.Fatal("expected panic on duplicate Register")
		}
		msg, ok := r.(string)
		if !ok {
			t.Fatalf("panic value is not a string: %T %v", r, r)
		}
		if !strings.Contains(msg, "duplicate Kind") {
			t.Errorf("wrong panic msg: %v", msg)
		}
	}()
	Register(&Spec{Kind: "Dup"})
}

func TestSpecHelpers(t *testing.T) {
	s := &Spec{
		Kind:    "X",
		ExecOut: []string{"a", "b"},
		ExecOutFn: func(cfg map[string]any) []string {
			return []string{"dyn0", "dyn1"}
		},
		DataIn:  map[string]PinType{"n": PinNumber},
		DataOut: map[string]PinType{"v": PinAny},
	}
	if got := s.ExecOutPins(nil); !equalStrs(got, []string{"dyn0", "dyn1"}) {
		t.Errorf("ExecOutFn not preferred: got %v", got)
	}
	s2 := &Spec{Kind: "Y", ExecOut: []string{"a", "b"}}
	if got := s2.ExecOutPins(nil); !equalStrs(got, []string{"a", "b"}) {
		t.Errorf("static ExecOut not returned: got %v", got)
	}
	if s.DataInType("n", nil) != PinNumber {
		t.Errorf("DataInType wrong")
	}
	if s.DataInType("missing", nil) != "" {
		t.Errorf("DataInType non-empty for missing pin")
	}
	if s.DataOutType("v") != PinAny {
		t.Errorf("DataOutType wrong")
	}
}

func equalStrs(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}
