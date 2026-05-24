// internal/node/service_test.go
package node

import "testing"

func TestNodeService_GetAllNodeSpecs(t *testing.T) {
	ResetRegistryForTest()
	Register(stubNode{kind: "T1"})
	Register(stubNode{kind: "T2"})

	svc := NewService()
	specs := svc.GetAllNodeSpecs()
	if len(specs) != 2 {
		t.Errorf("len = %d, want 2", len(specs))
	}
}

func TestNodeService_AsyncOptions(t *testing.T) {
	svc := NewService()
	svc.RegisterAsyncSource("test", func(nodeID, specKind string, params map[string]any) ([]EnumOption, error) {
		return []EnumOption{{Value: "x", Label: "X"}}, nil
	})

	opts, err := svc.AsyncOptions("n1", "Mock_Vision", "test", nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(opts) != 1 || opts[0].Value != "x" {
		t.Errorf("got %v", opts)
	}
}

func TestNodeService_AsyncOptions_UnknownSource(t *testing.T) {
	svc := NewService()
	_, err := svc.AsyncOptions("n1", "X", "unknown", nil)
	if err == nil {
		t.Error("expected error for unknown asyncSource")
	}
}
