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

func TestWaitTemplate_Capture_Found(t *testing.T) {
	node.ResetRegistryForTest()
	node.Register(&WaitTemplate{})
	rn, _ := node.Get("WaitTemplate")

	pt := node.Point{X: 0.4, Y: 0.6}
	vision := &mockVision{point: &pt, conf: 0.91, hitOnCall: 1}
	vars := newRecVars()
	r := node.RunNode(context.Background(), rn, nil,
		map[string]any{wtInTemplates: []string{"fishing.hook_icon"}, wtInTimeoutMs: 100, wtInThreshold: 0.85,
			wtCapFound: "f", wtCapPoint: "p"},
		nil, withVisionVars(vision, vars), false)

	if r.Error != nil {
		t.Fatal(r.Error)
	}
	if r.ExitName != wtOutFound {
		t.Fatalf("exit = %q, want Found", r.ExitName)
	}
	if got, ok := vars.Get("f"); !ok || got != true {
		t.Errorf("capture f = %v (ok=%v), want true", got, ok)
	} else if _, isBool := got.(bool); !isBool {
		// CaptureType=bool: 断言写入值的 Go 类型是 bool.
		t.Errorf("capture f Go type = %T, want bool", got)
	}
	gp, ok := vars.Get("p")
	if !ok || gp != pt {
		t.Fatalf("capture p = %v (ok=%v), want %v", gp, ok, pt)
	}
	// CaptureType=point: 断言写入值的 Go 类型是 node.Point.
	if _, isPoint := gp.(node.Point); !isPoint {
		t.Errorf("capture p Go type = %T, want node.Point", gp)
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

