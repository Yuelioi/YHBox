//go:build mockerror

package mockerror

import (
	"context"
	"testing"

	"yhbox/internal/node"
)

func TestMockError_AlwaysErrors(t *testing.T) {
	node.ResetRegistryForTest()
	node.Register(&MockError{})
	rn, _ := node.Get("MockError")
	r := node.RunNode(context.Background(), rn, nil,
		map[string]any{meInMessage: "boom"},
		nil, node.StubServices())
	if r.Error == nil || r.Error.Error() != "boom" {
		t.Errorf("error = %v, want boom", r.Error)
	}
}

func TestMockError_DefaultMessage(t *testing.T) {
	node.ResetRegistryForTest()
	node.Register(&MockError{})
	rn, _ := node.Get("MockError")
	r := node.RunNode(context.Background(), rn, nil, nil, nil, node.StubServices())
	if r.Error == nil || r.Error.Error() != "deliberate failure" {
		t.Errorf("error = %v, want default", r.Error)
	}
}
