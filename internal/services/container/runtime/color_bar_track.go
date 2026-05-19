package runtime

import (
	"context"
	"fmt"

	"github.com/lxn/win"

	"yhbox/internal/services/container"
	"yhbox/pkg/vision"
)

// confBarV2 置信度阈值 — 与 tools/fish/constants.go::confBar 同值 (0.50).
// ColorBarTrack 语义: cursor+target 都找到 且 conf >= confBarV2 → found 出口.
const confBarV2 = 0.50

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
	frame, capErr := r.rt.Capture.FrameROI(win.HWND(r.rt.Window.HWND), roi.X, roi.Y, roi.W, roi.H)
	if capErr != nil || frame == nil {
		r.updateLastBarTrack(SysBarTrackResult{})
		return r.edges.next(n.ID+".missing", tok.LoopStack), nil
	}
	result := vision.AnalyzeBar(frame)
	if result == nil {
		r.updateLastBarTrack(SysBarTrackResult{})
		return r.edges.next(n.ID+".missing", tok.LoopStack), nil
	}
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
		return r.edges.next(n.ID+".missing", tok.LoopStack), nil
	}
	return r.edges.next(n.ID+".found", tok.LoopStack), nil
}

func (r *ContainerRunner) updateLastBarTrack(s SysBarTrackResult) {
	r.rt.UpdateSys(func(sys *SysState) {
		sys.LastBarTrack = s
	})
}
