// internal/node/inputs_test.go
package node

import "testing"

func TestInputs_PriorityOrder(t *testing.T) {
	in := newInputs(
		map[string]any{"X": "from-wire"},
		map[string]any{"X": "from-config", "Y": "from-config"},
		map[string]any{"X": "from-exec", "Y": "from-exec", "Z": "from-exec"},
		map[string]any{"X": "default", "Y": "default", "Z": "default", "W": "default"},
	)
	if in.String("X") != "from-wire" {
		t.Errorf("X = %q, want from-wire", in.String("X"))
	}
	if in.String("Y") != "from-config" {
		t.Errorf("Y = %q, want from-config", in.String("Y"))
	}
	if in.String("Z") != "from-exec" {
		t.Errorf("Z = %q, want from-exec", in.String("Z"))
	}
	if in.String("W") != "default" {
		t.Errorf("W = %q, want default", in.String("W"))
	}
}

func TestInputs_Has(t *testing.T) {
	in := newInputs(nil, map[string]any{"X": 1}, nil, nil)
	if !in.Has("X") {
		t.Error("Has(X) should be true")
	}
	if in.Has("Y") {
		t.Error("Has(Y) should be false")
	}
}

func TestInputs_TypeCast(t *testing.T) {
	in := newInputs(nil, nil, nil, map[string]any{"i": 42, "f": 3.14})
	if in.Float64("i") != 42.0 {
		t.Error("int→float cast failed")
	}
	if in.Int("f") != 3 {
		t.Error("float→int cast failed")
	}
}
