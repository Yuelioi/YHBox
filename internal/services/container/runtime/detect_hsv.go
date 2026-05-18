package runtime

import (
	"context"
	"fmt"
	"image"
	"time"

	"github.com/lxn/win"

	"yhbox/internal/services/container"
	"yhbox/pkg/vision"
)

const (
	pollClampMs   = 10   // poll < 10ms 强制夹到 10
	defaultPollMs = 100  // 默认 poll 间隔
	defaultTOMs   = 5000 // 默认超时
)

// execDetectColorHSV 在 ROI 内轮询帧, 统计落在 HSV 区间内的像素比例.
//
// 出口:
//   - yes:     pixelRatio >= minPixelRatio (在 timeout 前命中)
//   - timeout: 超时仍未命中
//
// $sys 输出: lastDetect.pixelCount / lastDetect.pixelRatio (最后一次扫描结果).
func (r *ContainerRunner) execDetectColorHSV(ctx context.Context, n *container.GraphNode, tok ExecToken) ([]ExecToken, error) {
	roi, err := configROI(n, "roi")
	if err != nil {
		return nil, fmt.Errorf("DetectColorHSV %s: %w", n.ID, err)
	}
	hsv, err := configHSV(n, "hsv")
	if err != nil {
		return nil, fmt.Errorf("DetectColorHSV %s: %w", n.ID, err)
	}
	// v4: minPixelRatio / pollIntervalMs / timeoutMs via data-in pin.
	minRatio := r.pullNumber(n, "minPixelRatio", 0.05)
	pollMs := int(r.pullNumber(n, "pollIntervalMs", float64(defaultPollMs)))
	if pollMs < pollClampMs {
		pollMs = pollClampMs
	}
	timeoutMs := int(r.pullNumber(n, "timeoutMs", float64(defaultTOMs)))

	deadline := time.Time{}
	if timeoutMs > 0 {
		deadline = time.Now().Add(time.Duration(timeoutMs) * time.Millisecond)
	}

	for {
		if ctx.Err() != nil {
			return nil, ctx.Err()
		}
		frame, capErr := r.rt.Capture.FrameROI(win.HWND(r.rt.Window.HWND), roi.X, roi.Y, roi.W, roi.H)
		if capErr == nil && frame != nil {
			count, ratio := countHSVInROI(frame, hsv)
			r.rt.UpdateSys(func(s *SysState) {
				s.LastDetect.PixelCount = count
				s.LastDetect.PixelRatio = ratio
			})
			if ratio >= minRatio {
				return r.edges.next(n.ID+".yes", tok.LoopStack), nil
			}
		}
		if !deadline.IsZero() && time.Now().After(deadline) {
			return r.edges.next(n.ID+".timeout", tok.LoopStack), nil
		}
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-time.After(time.Duration(pollMs) * time.Millisecond):
		}
	}
}

// countHSVInROI 统计 img 中落在 hsvRange 内的像素数和比例.
// 直接访问 Pix 避免 image.At() 接口开销.
func countHSVInROI(img *image.RGBA, h hsvRange) (count int, ratio float64) {
	bounds := img.Bounds()
	w, hh := bounds.Dx(), bounds.Dy()
	if w == 0 || hh == 0 {
		return 0, 0
	}
	for y := 0; y < hh; y++ {
		off := y * img.Stride
		for x := 0; x < w; x++ {
			i := off + x*4
			rv, gv, bv := img.Pix[i], img.Pix[i+1], img.Pix[i+2]
			hv, sv, vv := vision.RGBToHSV(rv, gv, bv)
			if hv >= h.hMin && hv <= h.hMax &&
				sv >= h.sMin && sv <= h.sMax &&
				vv >= h.vMin && vv <= h.vMax {
				count++
			}
		}
	}
	ratio = float64(count) / float64(w*hh)
	return
}
