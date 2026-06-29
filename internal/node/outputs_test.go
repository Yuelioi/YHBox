// internal/node/outputs_test.go
package node

import "testing"

func TestOutBuilder_HappyPath(t *testing.T) {
	spec := &Spec{Kind: "T", Outputs: []OutputSpec{{Name: "out1"}}}
	b := &outBuilderImpl{exitName: "out1", spec: spec, data: map[string]any{}}
	o := b.Set("x", 42).Set("y", "hi").Fire()
	name, data := o.exit()
	if name != "out1" || data["x"] != 42 || data["y"] != "hi" {
		t.Errorf("got %v %v", name, data)
	}
}

func TestOutBuilder_DoubleFire_Panics(t *testing.T) {
	spec := &Spec{Kind: "T", Outputs: []OutputSpec{{Name: "out1"}}}
	b := &outBuilderImpl{exitName: "out1", spec: spec, data: map[string]any{}}
	b.Fire()
	defer func() {
		if r := recover(); r == nil {
			t.Error("expected panic on second Fire")
		}
	}()
	b.Fire()
}

func TestValidateExitName_Unknown_Panics(t *testing.T) {
	spec := &Spec{Kind: "T", Outputs: []OutputSpec{{Name: "out1"}}}
	defer func() {
		if r := recover(); r == nil {
			t.Error("expected panic on unknown exit name")
		}
	}()
	validateExitName(spec, "wrong")
}
