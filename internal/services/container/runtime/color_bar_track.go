package runtime

import (
	"context"
	"fmt"
	"sync"
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

// missingROIThrottleMs ColorBarTrack 没找到匹配 resolution 时 emit warning 的节流窗口.
// 30s 一条窗口够 user 在前端看到不刷屏 (frame size 不变时本质同一情况).
const missingROIThrottleMs = 30000

var lastBarTrackDiagMs atomic.Int64

// missingROIThrottle key = nodeID:WxH → unixMs. sync.Map 避免 map race.
var missingROIThrottle sync.Map

// pickROIByResolution 按 (frameW, frameH) 从 rois 数组 exact match. 没匹配返 (zero, false).
// rois 来自 GraphNode.Config["rois"], runtime 拿到时是 []any (JSON 反序列化, 各项 map[string]any).
func pickROIByResolution(rois any, frameW, frameH int) (roiRect, bool) {
	arr, ok := rois.([]any)
	if !ok {
		return roiRect{}, false
	}
	for _, item := range arr {
		m, ok := item.(map[string]any)
		if !ok {
			continue
		}
		res, ok := m["resolution"].([]any)
		if !ok || len(res) != 2 {
			continue
		}
		w, _ := res[0].(float64)
		h, _ := res[1].(float64)
		if int(w) != frameW || int(h) != frameH {
			continue
		}
		x, _ := m["x"].(float64)
		y, _ := m["y"].(float64)
		rw, _ := m["w"].(float64)
		rh, _ := m["h"].(float64)
		return roiRect{X: int(x), Y: int(y), W: int(rw), H: int(rh)}, true
	}
	return roiRect{}, false
}

// execColorBarTrack 抓 ROI 帧 → vision.AnalyzeBar 双 cluster HSV 检测 → 写 sys.lastBarTrack.
//
// 出口:
//   - found:   cursor 和 target 都找到, confidence >= confBarV2
//   - missing: 任一未找到 / confidence 不足 / rois 无 frame size 匹配项
//
// 算法移植自 tools/fish/bar.go (analyzeBar), 见 pkg/vision/bar_track.go.
func (r *ContainerRunner) execColorBarTrack(ctx context.Context, n *container.GraphNode, tok ExecToken) ([]ExecToken, error) {
	hwnd := win.HWND(r.rt.Window.HWND)
	clientW, clientH, _ := r.rt.Capture.ClientSize(hwnd)

	roi, ok := pickROIByResolution(n.Config["rois"], clientW, clientH)
	if !ok {
		r.emitColorBarMissingROI(n.ID, clientW, clientH)
		r.updateLastBarTrack(SysBarTrackResult{})
		return r.edges.next(n.ID+".missing", tok.LoopStack), nil
	}

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

// emitColorBarMissingROI 当 ColorBarTrack 节点的 rois 数组里没有匹配当前 frame 分辨率的条目时,
// 推一条 container:warning event. 同 (nodeID, frameSize) 30s 节流防刷屏 (frame 不变 = 同一情况).
func (r *ContainerRunner) emitColorBarMissingROI(nodeID string, frameW, frameH int) {
	if r.rt.Emit == nil {
		return
	}
	key := fmt.Sprintf("%s:%dx%d", nodeID, frameW, frameH)
	now := nowMillis()
	if prev, ok := missingROIThrottle.Load(key); ok {
		if now-prev.(int64) < missingROIThrottleMs {
			return
		}
	}
	missingROIThrottle.Store(key, now)

	containerID := ""
	if r.rt.Container != nil {
		containerID = r.rt.Container.ID
	}
	r.rt.Emit("container:warning", map[string]any{
		"containerId": containerID,
		"code":        "MISSING_COLORBAR_ROI",
		"severity":    "warning",
		"message":     fmt.Sprintf("ColorBarTrack node %q has no rois entry matching frame %dx%d", nodeID, frameW, frameH),
		"nodeId":      nodeID,
		"frameSize":   []int{frameW, frameH},
	})
}

func (r *ContainerRunner) updateLastBarTrack(s SysBarTrackResult) {
	r.rt.UpdateSys(func(sys *SysState) {
		sys.LastBarTrack = s
	})
}
