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

// clusterEntry 描述沿扫描轴方向上的一个连续命中段。
type clusterEntry struct {
	StartPx  int `json:"startPx"`
	EndPx    int `json:"endPx"`
	CenterPx int `json:"centerPx"`
	PxCount  int `json:"pxCount"`
}

// execROIColorScan 沿 scanAxis 轴扫描 ROI, 统计落在 HSV 区间内的连续像素段（cluster）。
//
// 出口:
//   - found:   cluster 数量 >= minClusterCount（在 timeout 前命中）
//   - notFound: 首次扫描即不足 minClusterCount 且 timeout <= 0
//   - timeout: 超时仍未满足
//
// $sys 输出: lastROIScan.clusterCount / lastROIScan.clusters（最后一次扫描结果）.
func (r *ContainerRunner) execROIColorScan(ctx context.Context, n *container.GraphNode, tok ExecToken) ([]ExecToken, error) {
	roi, err := configROI(n, "roi")
	if err != nil {
		return nil, fmt.Errorf("ROIColorScan %s: %w", n.ID, err)
	}
	hsv, err := configHSV(n, "hsv")
	if err != nil {
		return nil, fmt.Errorf("ROIColorScan %s: %w", n.ID, err)
	}
	axis := configString(n, "scanAxis")
	if axis != "x" && axis != "y" {
		return nil, fmt.Errorf("ROIColorScan %s: scanAxis must be x or y", n.ID)
	}
	minCluster := int(r.configFloat(n, "minClusterPx", 2))
	maxCluster := int(r.configFloat(n, "maxClusterPx", 0))
	if maxCluster <= 0 {
		if axis == "x" {
			maxCluster = roi.W / 3
		} else {
			maxCluster = roi.H / 3
		}
	}
	minClusterCount := int(r.configFloat(n, "minClusterCount", 1))
	pollMs := int(r.configFloat(n, "pollIntervalMs", defaultPollMs))
	if pollMs < pollClampMs {
		pollMs = pollClampMs
	}
	timeoutMs := int(r.configFloat(n, "timeoutMs", defaultTOMs))

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
			clusters := scanClusters(frame, hsv, axis, minCluster, maxCluster)
			r.rt.UpdateSys(func(s *SysState) {
				s.LastROIScan.Clusters = clusters
				s.LastROIScan.ClusterCount = len(clusters)
			})
			if len(clusters) >= minClusterCount {
				return r.edges.next(n.ID+".found", tok.LoopStack), nil
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

// scanClusters 沿 axis 方向统计每条线上的命中像素数，然后把连续非零段合并为 cluster。
// 仅保留长度在 [minPx, maxPx] 之间的段。
func scanClusters(img *image.RGBA, h hsvRange, axis string, minPx, maxPx int) []clusterEntry {
	bounds := img.Bounds()
	w, hh := bounds.Dx(), bounds.Dy()

	var lineCount []int
	if axis == "x" {
		lineCount = make([]int, w)
		for y := 0; y < hh; y++ {
			off := y * img.Stride
			for x := 0; x < w; x++ {
				i := off + x*4
				if hsvInRange(img.Pix[i], img.Pix[i+1], img.Pix[i+2], h) {
					lineCount[x]++
				}
			}
		}
	} else {
		lineCount = make([]int, hh)
		for y := 0; y < hh; y++ {
			off := y * img.Stride
			for x := 0; x < w; x++ {
				i := off + x*4
				if hsvInRange(img.Pix[i], img.Pix[i+1], img.Pix[i+2], h) {
					lineCount[y]++
				}
			}
		}
	}

	var clusters []clusterEntry
	i := 0
	for i < len(lineCount) {
		if lineCount[i] == 0 {
			i++
			continue
		}
		start := i
		sum := 0
		for i < len(lineCount) && lineCount[i] > 0 {
			sum += lineCount[i]
			i++
		}
		runLen := i - start
		if runLen < minPx || runLen > maxPx {
			continue
		}
		clusters = append(clusters, clusterEntry{
			StartPx:  start,
			EndPx:    i,
			CenterPx: (start + i) / 2,
			PxCount:  sum,
		})
	}
	return clusters
}

// hsvInRange 判断 RGB 像素是否落在 hsvRange 内。
// 复用 vision.RGBToHSV，避免重复转换逻辑。
func hsvInRange(rv, gv, bv uint8, h hsvRange) bool {
	hv, sv, vv := vision.RGBToHSV(rv, gv, bv)
	return hv >= h.hMin && hv <= h.hMax &&
		sv >= h.sMin && sv <= h.sMax &&
		vv >= h.vMin && vv <= h.vMax
}
