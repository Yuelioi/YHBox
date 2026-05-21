package runtime

import (
	"context"
	"fmt"
	"sync/atomic"

	"github.com/lxn/win"

	"yhbox/internal/services/container"
	"yhbox/pkg/vision"
)

// confBarV2 置信度阈值 — 与 tools/fish/constants.go::confBar 同值 (0.50).
// ColorBarTrack 语义: cursor+target 都找到 且 conf >= confBarV2 → found 出口.
const confBarV2 = 0.50

// barTrackDiagThrottleMs ColorBarTrack 诊断 emit 节流, 单 process 共享.
// 500ms 一条够看 trend, 又不刷爆日志 (FISHING 30ms tick 否则 1 秒 ~30 条).
const barTrackDiagThrottleMs = 500

var lastBarTrackDiagMs atomic.Int64

// execColorBarTrack 抓 ROI 帧 → vision.AnalyzeBar 双 cluster HSV 检测 → 写 sys.lastBarTrack.
//
// 出口:
//   - found:   cursor 和 target 都找到, confidence >= confBarV2
//   - missing: 任一未找到 / confidence 不足
//
// 算法移植自 tools/fish/bar.go (analyzeBar), 见 pkg/vision/bar_track.go.
func (r *ContainerRunner) execColorBarTrack(ctx context.Context, n *container.GraphNode, tok ExecToken) ([]ExecToken, error) {
	roi, err := configROI(n, "roi")
	if err != nil {
		return nil, fmt.Errorf("ColorBarTrack %s: %w", n.ID, err)
	}
	hwnd := win.HWND(r.rt.Window.HWND)
	clientW, clientH, _ := r.rt.Capture.ClientSize(hwnd)
	frame, capErr := r.rt.Capture.FrameROI(hwnd, roi.X, roi.Y, roi.W, roi.H)

	diag := map[string]any{
		"nodeId":  n.ID,
		"hwnd":    uint64(hwnd),
		"clientW": clientW,
		"clientH": clientH,
		"roiX":    roi.X,
		"roiY":    roi.Y,
		"roiW":    roi.W,
		"roiH":    roi.H,
	}

	if capErr != nil || frame == nil {
		if capErr != nil {
			diag["captureErr"] = capErr.Error()
		} else {
			diag["captureErr"] = "nil frame"
		}
		diag["route"] = "missing"
		emitBarTrackDiag(r, diag)
		r.updateLastBarTrack(SysBarTrackResult{})
		return r.edges.next(n.ID+".missing", tok.LoopStack), nil
	}
	diag["frameW"] = frame.Bounds().Dx()
	diag["frameH"] = frame.Bounds().Dy()

	result := vision.AnalyzeBar(frame)
	if result == nil {
		diag["route"] = "missing"
		diag["reason"] = "AnalyzeBar nil"
		emitBarTrackDiag(r, diag)
		r.updateLastBarTrack(SysBarTrackResult{})
		return r.edges.next(n.ID+".missing", tok.LoopStack), nil
	}
	diag["cursorX"] = result.CursorX
	diag["targetX"] = result.TargetX
	diag["targetW"] = result.TargetW
	diag["confidence"] = result.Confidence
	diag["yellowPx"] = result.YellowPx
	diag["greenPx"] = result.GreenPx

	sys := SysBarTrackResult{
		CursorX:    result.CursorX,
		TargetX:    result.TargetX,
		TargetW:    result.TargetW,
		Confidence: result.Confidence,
		YellowPx:   result.YellowPx,
		GreenPx:    result.GreenPx,
	}
	r.updateLastBarTrack(sys)
	if result.CursorX < 0 || result.TargetX < 0 || result.Confidence < confBarV2 {
		diag["route"] = "missing"
		emitBarTrackDiag(r, diag)
		return r.edges.next(n.ID+".missing", tok.LoopStack), nil
	}
	diag["route"] = "found"
	emitBarTrackDiag(r, diag)
	return r.edges.next(n.ID+".found", tok.LoopStack), nil
}

// emitBarTrackDiag 节流推 container:bartrack-diag event (app.go 会镜像到 zerolog Info).
// 同 process 共享 500ms 窗口, 防 FISHING 30ms tick 刷爆.
func emitBarTrackDiag(r *ContainerRunner, diag map[string]any) {
	if r.rt.Emit == nil {
		return
	}
	now := nowMillis()
	last := lastBarTrackDiagMs.Load()
	if now-last < barTrackDiagThrottleMs {
		return
	}
	if !lastBarTrackDiagMs.CompareAndSwap(last, now) {
		return
	}
	r.rt.Emit("container:bartrack-diag", diag)
}

func (r *ContainerRunner) updateLastBarTrack(s SysBarTrackResult) {
	r.rt.UpdateSys(func(sys *SysState) {
		sys.LastBarTrack = s
	})
}
