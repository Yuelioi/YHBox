// internal/nodes/detect/wait_template_gone_test.go
package detect

import (
	"context"
	"errors"
	"testing"

	"yotta/internal/node"
)

// TestWaitTemplateGone_Gone: 第 1 帧命中（模板还在），第 2 帧消失 → Gone。
// missAfterCall=1: callCount > 1 后返 miss; hitOnCall=1: 第 1 次调就命中。
// TimeoutMs=500 进轮询路径; 第 2 次 matchOnce 返 miss → 走 Gone。
func TestWaitTemplateGone_Gone(t *testing.T) {
	node.ResetRegistryForTest()
	node.Register(&WaitTemplateGone{})
	rn, _ := node.Get("WaitTemplateGone")

	pt := node.Point{X: 0.5, Y: 0.5}
	// hitOnCall=1: 第 1 次 WaitMatch 返命中
	// missAfterCall=1: callCount > 1 后返 miss（第 2 次 matchOnce → 消失）
	vision := &mockVision{point: &pt, conf: 0.9, hitOnCall: 1, missAfterCall: 1}
	r := node.RunNode(context.Background(), rn, nil,
		map[string]any{
			wtgInTemplates: []string{"ns.icon"},
			wtgInTimeoutMs: 500,
			wtgInThreshold: 0.85,
		},
		nil, withVision(vision), false)

	if r.Error != nil {
		t.Fatal(r.Error)
	}
	if r.ExitName != wtgOutGone {
		t.Fatalf("exit=%s want Gone", r.ExitName)
	}
}

// TestWaitTemplateGone_Timeout_SingleFrame: TimeoutMs=0 单帧路径 — 当帧仍命中 → Timeout，带 Conf。
func TestWaitTemplateGone_Timeout_SingleFrame(t *testing.T) {
	node.ResetRegistryForTest()
	node.Register(&WaitTemplateGone{})
	rn, _ := node.Get("WaitTemplateGone")

	pt := node.Point{X: 0.5, Y: 0.5}
	// hitOnCall=1: 第 1 次调就命中（模板还在），TimeoutMs=0 → 单帧 → Timeout
	vision := &mockVision{point: &pt, conf: 0.93, hitOnCall: 1}
	r := node.RunNode(context.Background(), rn, nil,
		map[string]any{
			wtgInTemplates: []string{"ns.icon"},
			wtgInTimeoutMs: 0,
			wtgInThreshold: 0.85,
		},
		nil, withVision(vision), false)

	if r.Error != nil {
		t.Fatal(r.Error)
	}
	if r.ExitName != wtgOutTimeout {
		t.Fatalf("exit=%s want Timeout", r.ExitName)
	}
}

// TestWaitTemplateGone_Gone_SingleFrame: TimeoutMs=0 单帧路径 — 当帧未命中（模板已消失）→ Gone。
func TestWaitTemplateGone_Gone_SingleFrame(t *testing.T) {
	node.ResetRegistryForTest()
	node.Register(&WaitTemplateGone{})
	rn, _ := node.Get("WaitTemplateGone")

	// hitOnCall=-1 → 永不命中（模板不在）→ matchOnce 返 !hit.Found → 立即走 Gone
	vision := &mockVision{hitOnCall: -1, conf: 0.2}
	r := node.RunNode(context.Background(), rn, nil,
		map[string]any{
			wtgInTemplates: []string{"ns.icon"},
			wtgInTimeoutMs: 0,
			wtgInThreshold: 0.85,
		},
		nil, withVision(vision), false)

	if r.Error != nil {
		t.Fatal(r.Error)
	}
	if r.ExitName != wtgOutGone {
		t.Fatalf("exit=%s want Gone", r.ExitName)
	}
}

// TestWaitTemplateGone_Error: vision 报错 → 节点返 error。
func TestWaitTemplateGone_Error(t *testing.T) {
	node.ResetRegistryForTest()
	node.Register(&WaitTemplateGone{})
	rn, _ := node.Get("WaitTemplateGone")

	vision := &mockVision{err: errors.New("window closed")}
	r := node.RunNode(context.Background(), rn, nil,
		map[string]any{wtgInTemplates: []string{"ns.icon"}},
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

func TestWaitTemplateGone_PassesROIAndUsesPollInterval(t *testing.T) {
	node.ResetRegistryForTest()
	node.Register(&WaitTemplateGone{})
	rn, _ := node.Get("WaitTemplateGone")

	pt := node.Point{X: 0.5, Y: 0.5}
	roi := node.Geometry{Pct: node.Rect{X: 0.1, Y: 0.1, W: 0.6, H: 0.6}}
	vision := &mockVision{point: &pt, conf: 0.9, hitOnCall: 1, missAfterCall: 1}
	r := node.RunNode(context.Background(), rn, nil,
		map[string]any{
			wtgInTemplates:      []string{"ns.icon"},
			wtgInTimeoutMs:      50,
			wtgInThreshold:      0.85,
			wtgInROI:            roi,
			wtgInPollIntervalMs: 1,
		},
		nil, withVision(vision), false)

	if r.Error != nil {
		t.Fatal(r.Error)
	}
	if r.ExitName != wtgOutGone {
		t.Fatalf("exit=%s want Gone", r.ExitName)
	}
	if vision.callCount != 2 {
		t.Fatalf("WaitMatch callCount = %d, want 2", vision.callCount)
	}
	if vision.lastWaitMatchROI.Pct != roi.Pct {
		t.Fatalf("roi = %+v, want %+v", vision.lastWaitMatchROI.Pct, roi.Pct)
	}
}
