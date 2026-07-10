package variable

import (
	"context"
	"testing"

	"github.com/yottaapp/yotta/internal/node"
)

func TestIncVar_HappyPath(t *testing.T) {
	node.ResetRegistryForTest()
	node.Register(&IncVar{})
	rn, _ := node.Get("IncVar")
	vars := &recordingVars{store: map[string]any{"counter": 10.0}}
	svc := node.StubServices()
	svc.Vars = vars
	r := node.RunNode(context.Background(), rn, nil,
		map[string]any{ivInVarName: "counter", ivInDelta: 3.0},
		nil, svc, false)
	if r.Error != nil {
		t.Fatal(r.Error)
	}
	if vars.store["counter"] != 13.0 {
		t.Errorf("counter = %v, want 13", vars.store["counter"])
	}
}

func TestIncVar_DefaultDelta(t *testing.T) {
	node.ResetRegistryForTest()
	node.Register(&IncVar{})
	rn, _ := node.Get("IncVar")
	vars := &recordingVars{}
	svc := node.StubServices()
	svc.Vars = vars
	r := node.RunNode(context.Background(), rn, nil,
		map[string]any{ivInVarName: "x"},
		nil, svc, false)
	if r.Error != nil {
		t.Fatal(r.Error)
	}
	if vars.store["x"] != 1.0 {
		t.Errorf("x = %v, want 1 (default delta)", vars.store["x"])
	}
}
