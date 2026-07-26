// pkg/vision/match_all.go
// MatchAll — Match 的"全部命中"变体: 收集 3×3 局部极大且 conf≥threshold 的位置 + NMS。
// 与 Match 共享 correlationMap (CCOEFF_NORMED 全图), 算法一致不重复维护。spec §节点2。
package vision

import (
	"math"
	"sort"
	"sync"
)

// corrSkip 哨兵: 低于 NCC 有效域 [-1,1], 标记 uniform patch / 跳过的搜索位。
const corrSkip = float32(-2)

// matchAllCandidateCap NMS 前候选硬上限 (病态低阈值防爆); 超出按 conf 降序保前 N。
const matchAllCandidateCap = 4096

// MatchHit 一个命中 (ROI 内左上角像素 + conf)。
type MatchHit struct {
	X, Y int
	Conf float32
}

// correlationMap 计算 tpl 在 img 每个搜索位置的 CCOEFF_NORMED conf, 返回完整 conf 图
// (行主序, sw×sh)。uniform patch 位置 = corrSkip。模板单色 / 搜索空间为空 → nil,0,0。
// 积分图 O(1) 算 patch 均值/方差 + 行级并行 (同 Match 旧实现, 只是不丢弃非极大值)。
func correlationMap(img []float32, iw, ih int, tpl *Template, parallel int) ([]float32, int, int) {
	tw, th := tpl.W, tpl.H
	if iw < tw || ih < th {
		return nil, 0, 0
	}
	var tplSum float64
	for _, v := range tpl.Gray {
		tplSum += float64(v)
	}
	tplMean := tplSum / float64(tw*th)
	tplPrime := make([]float32, len(tpl.Gray))
	var tplSqSum float64
	for i, v := range tpl.Gray {
		d := float64(v) - tplMean
		tplPrime[i] = float32(d)
		tplSqSum += d * d
	}
	if tplSqSum < 1e-12 {
		return nil, 0, 0
	}
	tplNorm := math.Sqrt(tplSqSum)

	stride := iw + 1
	sumI := make([]float64, stride*(ih+1))
	sumI2 := make([]float64, stride*(ih+1))
	for y := 0; y < ih; y++ {
		for x := 0; x < iw; x++ {
			v := float64(img[y*iw+x])
			i := (y+1)*stride + (x + 1)
			sumI[i] = v + sumI[i-stride] + sumI[i-1] - sumI[i-stride-1]
			sumI2[i] = v*v + sumI2[i-stride] + sumI2[i-1] - sumI2[i-stride-1]
		}
	}

	searchW := iw - tw + 1
	searchH := ih - th + 1
	if searchW <= 0 || searchH <= 0 {
		return nil, 0, 0
	}
	confMap := make([]float32, searchW*searchH)

	var wg sync.WaitGroup
	if parallel < 1 {
		parallel = 1
	}
	sema := make(chan struct{}, parallel)
	for sy := 0; sy < searchH; sy++ {
		wg.Add(1)
		sema <- struct{}{}
		go func(sy int) {
			defer wg.Done()
			defer func() { <-sema }()
			for sx := 0; sx < searchW; sx++ {
				topL := sy*stride + sx
				topR := sy*stride + sx + tw
				botL := (sy+th)*stride + sx
				botR := (sy+th)*stride + sx + tw
				pSum := sumI[botR] - sumI[topR] - sumI[botL] + sumI[topL]
				pSum2 := sumI2[botR] - sumI2[topR] - sumI2[botL] + sumI2[topL]
				patchSqDiff := pSum2 - pSum*pSum/float64(tw*th)
				if patchSqDiff < 1e-9 {
					confMap[sy*searchW+sx] = corrSkip
					continue
				}
				var cross float64
				for ty := 0; ty < th; ty++ {
					iRow := (sy + ty) * iw
					tRow := ty * tw
					for tx := 0; tx < tw; tx++ {
						cross += float64(tplPrime[tRow+tx]) * float64(img[iRow+sx+tx])
					}
				}
				confMap[sy*searchW+sx] = float32(cross / (math.Sqrt(patchSqDiff) * tplNorm))
			}
		}(sy)
	}
	wg.Wait()
	return confMap, searchW, searchH
}

// MatchAll: 收集 conf≥threshold 且为 3×3 局部极大的候选, 做 NMS (minDist 像素中心距;
// <=0 → 模板 min(W,H)/2)。返回按 conf 降序 (并列 y,x) 的命中 (ROI 内左上角坐标)。
func MatchAll(img []float32, iw, ih int, tpl *Template, parallel int, threshold float32, minDist int) []MatchHit {
	confMap, sw, sh := correlationMap(img, iw, ih, tpl, parallel)
	if sw <= 0 || sh <= 0 {
		return nil
	}
	var cand []MatchHit
	for sy := 0; sy < sh; sy++ {
		for sx := 0; sx < sw; sx++ {
			c := confMap[sy*sw+sx]
			if c < threshold || c <= corrSkip {
				continue
			}
			if !isLocalMax(confMap, sw, sh, sx, sy, c) {
				continue
			}
			cand = append(cand, MatchHit{X: sx, Y: sy, Conf: c})
		}
	}
	// conf 降序, 并列 (y,x) — 全确定性。
	sort.Slice(cand, func(i, j int) bool {
		if cand[i].Conf != cand[j].Conf {
			return cand[i].Conf > cand[j].Conf
		}
		if cand[i].Y != cand[j].Y {
			return cand[i].Y < cand[j].Y
		}
		return cand[i].X < cand[j].X
	})
	if len(cand) > matchAllCandidateCap {
		cand = cand[:matchAllCandidateCap] // 病态低阈值: 保前 N (已按 conf 降序)。
	}
	if minDist <= 0 {
		minDist = minInt(tpl.W, tpl.H) / 2
	}
	return nmsHits(cand, tpl.W, tpl.H, minDist)
}

// isLocalMax: c ≥ 全部存在的 8 邻域 (边缘缩减邻域); plateau (邻域相等) 仅在 (y,x) 最小处算极大。
func isLocalMax(m []float32, sw, sh, x, y int, c float32) bool {
	for dy := -1; dy <= 1; dy++ {
		for dx := -1; dx <= 1; dx++ {
			if dx == 0 && dy == 0 {
				continue
			}
			nx, ny := x+dx, y+dy
			if nx < 0 || nx >= sw || ny < 0 || ny >= sh {
				continue
			}
			nc := m[ny*sw+nx]
			if nc > c {
				return false
			}
			if nc == c && (ny < y || (ny == y && nx < x)) {
				return false // plateau: 让位给更靠前的代表点
			}
		}
	}
	return true
}

// nmsHits: cand 已按 conf 降序; 贪心保留, 中心欧氏距 < minDist 的低分被抑制。
func nmsHits(cand []MatchHit, tw, th, minDist int) []MatchHit {
	if minDist <= 0 {
		return cand
	}
	kept := make([]MatchHit, 0, len(cand))
	md := float64(minDist)
	for _, c := range cand {
		cx, cy := float64(c.X+tw/2), float64(c.Y+th/2)
		ok := true
		for _, k := range kept {
			kx, ky := float64(k.X+tw/2), float64(k.Y+th/2)
			if math.Hypot(cx-kx, cy-ky) < md {
				ok = false
				break
			}
		}
		if ok {
			kept = append(kept, c)
		}
	}
	return kept
}

func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}
