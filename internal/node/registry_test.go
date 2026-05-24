// internal/node/registry_test.go
package node

import "testing"

type stubNode struct{ kind string }

func (n stubNode) Spec() Spec { return Spec{Kind: n.kind, Version: 1} }

func TestRegister_HappyPath(t *testing.T) {
	ResetRegistryForTest()
	Register(stubNode{kind: "Test1"})

	rn, ok := Get("Test1")
	if !ok {
		t.Fatal("Get failed")
	}
	if rn.Spec.Kind != "Test1" {
		t.Errorf("kind = %q, want Test1", rn.Spec.Kind)
	}
}

func TestRegister_Duplicate_Panics(t *testing.T) {
	ResetRegistryForTest()
	Register(stubNode{kind: "Dup"})
	defer func() {
		if r := recover(); r == nil {
			t.Error("expected panic on duplicate Register")
		}
	}()
	Register(stubNode{kind: "Dup"})
}

func TestFreeze_BlocksRegister(t *testing.T) {
	ResetRegistryForTest()
	Register(stubNode{kind: "A"})
	Freeze()
	defer func() {
		if r := recover(); r == nil {
			t.Error("expected panic on Register after Freeze")
		}
	}()
	Register(stubNode{kind: "B"})
}

func TestAll_ReturnsSnapshot(t *testing.T) {
	ResetRegistryForTest()
	Register(stubNode{kind: "X"})
	Register(stubNode{kind: "Y"})
	all := All()
	if len(all) != 2 {
		t.Errorf("All len = %d, want 2", len(all))
	}
}
