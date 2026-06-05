// internal/nodes/detect/wait_template_test.go
package detect

import (
	"context"
	"errors"
	"testing"

	"yotta/internal/node"
)

func TestWaitTemplate_Found(t *testing.T) {
	node.ResetRegistryForTest()
	node.Register(&WaitTemplate{})
	rn, _ := node.Get("WaitTemplate")

	pt := node.Point{X: 0.4, Y: 0.6}
	vision := &mockVision{point: &pt, conf: 0.91, hitOnCall: 1}
	r := node.RunNode(context.Background(), rn, nil,
		map[string]any{wtInTemplates: []string{"fishing.hook_icon"}, wtInTimeoutMs: 100, wtInThreshold: 0.85},
		nil, withVision(vision), false)

	if r.Error != nil {
		t.Fatal(r.Error)
	}
	if r.ExitName != wtOutFound {
		t.Errorf("exit = %q, want Found", r.ExitName)
	}
}

func TestWaitTemplate_Timeout(t *testing.T) {
	node.ResetRegistryForTest()
	node.Register(&WaitTemplate{})
	rn, _ := node.Get("WaitTemplate")

	// hitOnCall=-1 → 永不命中, 走 timeout
	vision := &mockVision{hitOnCall: -1, conf: 0.4}
	r := node.RunNode(context.Background(), rn, nil,
		map[string]any{wtInTemplates: []string{"fishing.hook_icon"}, wtInTimeoutMs: 30, wtInThreshold: 0.85},
		nil, withVision(vision), false)

	if r.Error != nil {
		t.Fatal(r.Error)
	}
	if r.ExitName != wtOutTimeout {
		t.Errorf("exit = %q, want Timeout", r.ExitName)
	}
}

func TestWaitTemplate_Error(t *testing.T) {
	node.ResetRegistryForTest()
	node.Register(&WaitTemplate{})
	rn, _ := node.Get("WaitTemplate")

	vision := &mockVision{err: errors.New("window closed")}
	r := node.RunNode(context.Background(), rn, nil,
		map[string]any{wtInTemplates: []string{"fishing.x"}},
		nil, withVision(vision), false)

	if r.Error == nil {
		t.Error("expected error propagation")
	}
}

func TestWaitTemplate_InvalidKey_ValidationError(t *testing.T) {
	node.ResetRegistryForTest()
	node.Register(&WaitTemplate{})
	rn, _ := node.Get("WaitTemplate")

	r := node.RunNode(context.Background(), rn, nil,
		map[string]any{wtInTemplates: []string{"no_dot"}},
		nil, withVision(&mockVision{}), false)

	if len(r.Validation) != 1 || r.Validation[0].Code != "INVALID_TEMPLATE_KEY" {
		t.Errorf("validation = %v, want INVALID_TEMPLATE_KEY", r.Validation)
	}
}
