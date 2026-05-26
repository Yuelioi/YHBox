// tools/fish/bar.go — v1 fish bot 业务术语 wrapper over pkg/vision 通用双色条算法.
//
// pkg/vision 包不带任何 fishing-specific 知识 (HSV 阈值 / 字段名). 这文件:
//   - 包级常量 fishingCursorHSV / fishingTargetHSV: v1 fishing UI 实测出来的色阈
//   - BarResult struct: 保留 v1 业务术语 (cursor=光标 / target=目标条 / yellow/green
//     像素), 跟 vision.DualColorBarResult (inner/outer/innerPx/outerPx) 字段名映射
//   - analyzeBar: wrap vision.AnalyzeDualColorBar + 字段映射 + 走 vision 默认 Options
package fish

import (
	"image"

	"yhbox/pkg/vision"
)

// 复刻 Fishing v1 实测 HSV 阈值. 见 pkg/vision/bar_track.go 算法描述.
//   - cursor (浅黄高亮): H ∈ [45, 70], S ≥ 40, V ≥ 200
//   - target (高饱和青色): H ∈ [160, 180], S ≥ 140, V ≥ 100
// (debug/溜鱼/1080 实测: cursor H 50-60 S 67-127 V 253-255; target H 164-172 S 190-209 V 175-250)
var (
	fishingCursorHSV = vision.HSVRange{HMin: 45, HMax: 70, SMin: 40, SMax: 255, VMin: 200, VMax: 255}
	fishingTargetHSV = vision.HSVRange{HMin: 160, HMax: 180, SMin: 140, SMax: 255, VMin: 100, VMax: 255}
)

// BarResult: v1 fishing 业务术语视图 — cursor=光标 / target=目标条. 跟通用
// vision.DualColorBarResult 字段名映射不同 (业务代码 phase.go / states_*.go 引用
// 业务术语, 不动).
type BarResult struct {
	CursorX    int
	TargetX    int
	TargetW    int
	Confidence float64
	YellowPx   int
	GreenPx    int
}

// analyzeBar: wrap vision.AnalyzeDualColorBar + 字段映射. ROI 太小返 nil.
func analyzeBar(roi *image.RGBA) *BarResult {
	r := vision.AnalyzeDualColorBar(roi, fishingCursorHSV, fishingTargetHSV, vision.DualBarOptions{})
	if r == nil {
		return nil
	}
	return &BarResult{
		CursorX:    r.InnerX,
		TargetX:    r.OuterX,
		TargetW:    r.OuterWidth,
		Confidence: r.Confidence,
		YellowPx:   r.InnerPx,
		GreenPx:    r.OuterPx,
	}
}
