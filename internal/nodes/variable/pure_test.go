package variable

import (
	"testing"

	"yhbox/internal/node"
)

// IsPureData spec flag 校验.
func TestPureData_Flag(t *testing.T) {
	for _, n := range []node.Node{&GetVar{}, &GetSys{}, &GetParam{}} {
		s := n.Spec()
		if !s.IsPureData {
			t.Errorf("%s.IsPureData = false, want true", s.Kind)
		}
	}
}
