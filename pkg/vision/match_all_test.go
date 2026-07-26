// pkg/vision/match_all_test.go
package vision

import "testing"

// makeGray 返回 iw×ih 的均匀灰度图 (背景值 bg)。
func makeGray(iw, ih int, bg float32) []float32 {
	g := make([]float32, iw*ih)
	for i := range g {
		g[i] = bg
	}
	return g
}

// patternTemplate 造一个 k×k 有方差的模板 (ramp+checker, 保证 NCC 非退化)。
func patternTemplate(k int) *Template {
	g := make([]float32, k*k)
	for y := 0; y < k; y++ {
		for x := 0; x < k; x++ {
			v := float32(x+y) / float32(2*k) // 0..~1 ramp
			if (x+y)%2 == 0 {
				v = 1 - v
			}
			g[y*k+x] = v
		}
	}
	return &Template{Gray: g, W: k, H: k}
}

// placePattern 把模板像素写进 img 的 (px,py) 左上角。
func placePattern(img []float32, iw int, tpl *Template, px, py int) {
	for y := 0; y < tpl.H; y++ {
		for x := 0; x < tpl.W; x++ {
			img[(py+y)*iw+(px+x)] = tpl.Gray[y*tpl.W+x]
		}
	}
}

func TestMatch_FindsPattern(t *testing.T) {
	iw, ih := 60, 60
	img := makeGray(iw, ih, 0.3)
	tpl := patternTemplate(8)
	placePattern(img, iw, tpl, 20, 15)

	x, y, conf := Match(img, iw, ih, tpl, 2)
	if x != 20 || y != 15 {
		t.Fatalf("Match loc=(%d,%d), want (20,15)", x, y)
	}
	if conf < 0.99 {
		t.Errorf("conf=%v, want ~1.0", conf)
	}
}

func TestMatch_NoMatchUniform(t *testing.T) {
	// 全均匀图 → 所有窗口 patch 无方差 → corrSkip → Match 返回 not-found。
	iw, ih := 40, 40
	img := makeGray(iw, ih, 0.5)
	tpl := patternTemplate(8)
	x, y, _ := Match(img, iw, ih, tpl, 2)
	if x != -1 || y != -1 {
		t.Errorf("uniform image Match=(%d,%d), want (-1,-1)", x, y)
	}
}

func TestMatchAll_FindsAllInstances(t *testing.T) {
	iw, ih := 100, 100
	img := makeGray(iw, ih, 0.3)
	tpl := patternTemplate(8)
	locs := [][2]int{{10, 10}, {60, 10}, {10, 60}}
	for _, l := range locs {
		placePattern(img, iw, tpl, l[0], l[1])
	}

	hits := MatchAll(img, iw, ih, tpl, 2, 0.9, 0)
	if len(hits) != 3 {
		t.Fatalf("got %d hits, want 3: %+v", len(hits), hits)
	}
	// conf 降序
	for i := 1; i < len(hits); i++ {
		if hits[i].Conf > hits[i-1].Conf {
			t.Errorf("not conf-descending at %d: %v > %v", i, hits[i].Conf, hits[i-1].Conf)
		}
	}
	// 三个位置都被找到 (间距 50 > 模板 8, NMS 不合并)
	found := map[[2]int]bool{}
	for _, h := range hits {
		found[[2]int{h.X, h.Y}] = true
	}
	for _, l := range locs {
		if !found[[2]int{l[0], l[1]}] {
			t.Errorf("missed instance at %v; hits=%+v", l, hits)
		}
	}
}

func TestMatchAll_SinglePeakNoDuplicate(t *testing.T) {
	// 单个 pattern + 低阈值: 3×3 局部极大 + NMS 应只返回 1 个命中 (峰), 不重复收峰周围。
	iw, ih := 50, 50
	img := makeGray(iw, ih, 0.3)
	tpl := patternTemplate(8)
	placePattern(img, iw, tpl, 18, 22)

	hits := MatchAll(img, iw, ih, tpl, 2, 0.5, 0)
	if len(hits) != 1 {
		t.Fatalf("single pattern got %d hits, want 1: %+v", len(hits), hits)
	}
	if hits[0].X != 18 || hits[0].Y != 22 {
		t.Errorf("peak=(%d,%d), want (18,22)", hits[0].X, hits[0].Y)
	}
}

func TestMatchAll_NMSMergesNearby(t *testing.T) {
	// 两个 pattern 间距 5 (< minDist) → NMS 合并成 1; 间距足够 → 2。
	iw, ih := 60, 30
	tpl := patternTemplate(6)
	// 间距 5 < minDist(=20): 期望合并
	img := makeGray(iw, ih, 0.3)
	placePattern(img, iw, tpl, 10, 10)
	placePattern(img, iw, tpl, 15, 10)
	hits := MatchAll(img, iw, ih, tpl, 2, 0.9, 20)
	if len(hits) != 1 {
		t.Errorf("nearby (dist 5, minDist 20) got %d, want 1 (NMS merged): %+v", len(hits), hits)
	}
}

// TestMatch_EqualsMatchAllTop: Match 的 argmax 必须等于 MatchAll (无阈值过滤) 的最高 conf 命中,
// 证明 Match 重构后与 correlationMap 共享核、行为一致。
func TestMatch_EqualsMatchAllTop(t *testing.T) {
	iw, ih := 80, 80
	img := makeGray(iw, ih, 0.25)
	tpl := patternTemplate(10)
	placePattern(img, iw, tpl, 30, 40)

	mx, my, _ := Match(img, iw, ih, tpl, 2)
	// threshold=-1 收所有非退化局部极大; minDist=0 自动。
	hits := MatchAll(img, iw, ih, tpl, 2, -1, 0)
	if len(hits) == 0 {
		t.Fatal("MatchAll returned no hits")
	}
	if hits[0].X != mx || hits[0].Y != my {
		t.Errorf("MatchAll top=(%d,%d) != Match argmax=(%d,%d)", hits[0].X, hits[0].Y, mx, my)
	}
}

func TestMatchAll_CandidateCapNoPanic(t *testing.T) {
	// 病态低阈值 (-2 不行, 用 -1) + 高方差噪声图制造大量候选, 验不 panic + 截断生效。
	iw, ih := 120, 120
	img := make([]float32, iw*ih)
	for i := range img {
		img[i] = float32((i*2654435761)%1000) / 1000.0 // 伪随机方差
	}
	tpl := patternTemplate(6)
	hits := MatchAll(img, iw, ih, tpl, 2, -1, 1) // minDist 1: 几乎不抑制 → 候选量大
	if len(hits) > matchAllCandidateCap {
		t.Errorf("hits %d exceeds cap %d", len(hits), matchAllCandidateCap)
	}
}
