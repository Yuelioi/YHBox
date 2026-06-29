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
	if r.OutputData[wtDataMatched] != true {
		t.Errorf("Matched = %v, want true", r.OutputData[wtDataMatched])
	}
}

func TestWaitTemplate_SettleMs_RedetectThenFound(t *testing.T) {
	node.ResetRegistryForTest()
	node.Register(&WaitTemplate{})
	rn, _ := node.Get("WaitTemplate")

	pt := node.Point{X: 0.4, Y: 0.6}
	vision := &mockVision{point: &pt, conf: 0.91, hitOnCall: 1}
	r := node.RunNode(context.Background(), rn, nil,
		map[string]any{wtInTemplates: []string{"fishing.hook_icon"}, wtInTimeoutMs: 100, wtInThreshold: 0.85, wtInSettleMs: 5},
		nil, withVision(vision), false)

	if r.Error != nil {
		t.Fatal(r.Error)
	}
	if r.ExitName != wtOutFound {
		t.Errorf("exit = %q, want Found", r.ExitName)
	}
	// SettleMs>0 → 命中后等一下再 re-detect 一次 → WaitMatch 共调 2 次 (初次 + 重定位)。
	if vision.callCount != 2 {
		t.Errorf("WaitMatch callCount = %d, want 2 (initial + re-detect)", vision.callCount)
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
	if r.OutputData[wtDataMatched] != false {
		t.Errorf("Matched = %v, want false", r.OutputData[wtDataMatched])
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

	if r.Error != nil {
		t.Fatalf("unexpected runtime error: %v", r.Error)
	}
	if r.ExitName != tmplOutFail {
		t.Fatalf("exit = %q, want Fail", r.ExitName)
	}
	if r.OutputData[tmplDataCode] != string(node.CodeCaptureFailed) {
		t.Errorf("Code = %v, want %s", r.OutputData[tmplDataCode], node.CodeCaptureFailed)
	}
}

func TestWaitTemplate_PassesROIAndPollsWithSingleFrameMatch(t *testing.T) {
	node.ResetRegistryForTest()
	node.Register(&WaitTemplate{})
	rn, _ := node.Get("WaitTemplate")

	pt := node.Point{X: 0.4, Y: 0.6}
	roi := node.Geometry{Pct: node.Rect{X: 0.2, Y: 0.3, W: 0.4, H: 0.5}}
	vision := &mockVision{point: &pt, conf: 0.91, hitOnCall: 2}
	r := node.RunNode(context.Background(), rn, nil,
		map[string]any{
			wtInTemplates:      []string{"fishing.hook_icon"},
			wtInTimeoutMs:      50,
			wtInThreshold:      0.85,
			wtInROI:            roi,
			wtInPollIntervalMs: 1,
		},
		nil, withVision(vision), false)

	if r.Error != nil {
		t.Fatal(r.Error)
	}
	if r.ExitName != wtOutFound {
		t.Fatalf("exit = %q, want Found", r.ExitName)
	}
	if vision.callCount != 2 {
		t.Fatalf("WaitMatch callCount = %d, want 2", vision.callCount)
	}
	if vision.lastWaitTimeout != 0 {
		t.Fatalf("WaitMatch timeout = %v, want single-frame 0", vision.lastWaitTimeout)
	}
	if vision.lastWaitMatchROI.Pct != roi.Pct {
		t.Fatalf("roi = %+v, want %+v", vision.lastWaitMatchROI.Pct, roi.Pct)
	}
}
