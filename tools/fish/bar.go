// tools/fish/bar.go — algorithm moved to pkg/vision/bar_track.go.
// This file remains as thin shim for v1 fish bot internal callers.
package fish

import (
	"image"

	"yhbox/pkg/vision"
)

// BarResult: tools/fish 内部用名, 跟 vision.BarTrackResult 同形态.
type BarResult = vision.BarTrackResult

// analyzeBar: 兼容 v1 fish bot 内部 caller — 直接 forward 到 vision.AnalyzeBar.
func analyzeBar(roi *image.RGBA) *BarResult {
	return vision.AnalyzeBar(roi)
}
