package main

import (
	"bytes"
	"context"
	"image"
	"image/color"
	"image/png"
	"testing"

	"yotta/internal/services/asset"
)

// patternImg 造一张有空间方差的 small 图 (棋盘格), NCC 需要非常数像素才有定义.
func patternImg(w, h int) *image.RGBA {
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			if (x/2+y/2)%2 == 0 {
				img.Set(x, y, color.RGBA{R: 240, G: 30, B: 30, A: 255})
			} else {
				img.Set(x, y, color.RGBA{R: 20, G: 20, B: 200, A: 255})
			}
		}
	}
	return img
}

// makeTestPNGBlob 把一张图编码成 PNG 存进 store blob 池, 返回 sha.
func makeTestPNGBlob(t *testing.T, s *asset.Store, img *image.RGBA) string {
	t.Helper()
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		t.Fatalf("png encode: %v", err)
	}
	sha, err := s.Blobs().Put(buf.Bytes())
	if err != nil {
		t.Fatalf("blob put: %v", err)
	}
	return sha
}

// TestMatcher_DetectByGUID 验 matcher 经全局 asset store 按 guid 取 variant 解码匹配.
// 未知 guid → ok-miss 不崩; 已知 guid 命中 (frame 与 template 同像素 → conf=1.0).
func TestMatcher_DetectByGUID(t *testing.T) {
	s, err := asset.NewStore(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	m := newTemplateMatcherAdapter(s, nil)

	// 未知 guid → miss 不崩.
	frame := image.NewRGBA(image.Rect(0, 0, 200, 200))
	found, _, _, _, err := m.Detect(context.Background(), frame, "no-such-guid", 0.8, nil, 2.0)
	if err != nil {
		t.Fatalf("unknown guid should not error: %v", err)
	}
	if found {
		t.Fatal("unknown guid must miss")
	}

	// 存一个 16x16 棋盘 template 变体 (录制分辨率 200x200), frame 同尺寸在已知位置嵌同图.
	tplImg := patternImg(16, 16)
	sha := makeTestPNGBlob(t, s, tplImg)
	guid := "g-pat"
	if err := s.PutRecord(asset.AssetRecord{GUID: guid, Kind: asset.KindTemplate, Name: "pat"}); err != nil {
		t.Fatal(err)
	}
	// bbox 全屏 (0,0,200,200) 让 ROI = 全帧.
	if err := s.PutVariant(guid, [2]int{200, 200}, sha, [4]int{0, 0, 200, 200}, nil); err != nil {
		t.Fatal(err)
	}
	// frame: 在 (50,50) 嵌入同样 16x16 棋盘块 (其余黑).
	for y := 0; y < 16; y++ {
		for x := 0; x < 16; x++ {
			frame.Set(50+x, 50+y, tplImg.At(x, y))
		}
	}
	found, pt, _, conf, err := m.Detect(context.Background(), frame, guid, 0.7, nil, 2.0)
	if err != nil {
		t.Fatalf("detect: %v", err)
	}
	if !found {
		t.Fatalf("expected hit, got miss (conf=%.3f)", conf)
	}
	// 命中点应落在嵌入块中心附近 (58/200 ≈ 0.29).
	if pt.X < 0.2 || pt.X > 0.4 || pt.Y < 0.2 || pt.Y > 0.4 {
		t.Errorf("hit point off: %+v", pt)
	}
}

// TestMatcher_DecodeCacheBySha 验解码缓存按 blob sha, Invalidate 清空.
func TestMatcher_DecodeCacheBySha(t *testing.T) {
	s, err := asset.NewStore(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	m := newTemplateMatcherAdapter(s, nil)
	sha := makeTestPNGBlob(t, s, patternImg(8, 8))

	tpl1, err := m.loadDecodedTemplate(sha)
	if err != nil {
		t.Fatal(err)
	}
	tpl2, err := m.loadDecodedTemplate(sha)
	if err != nil {
		t.Fatal(err)
	}
	if tpl1 != tpl2 {
		t.Fatal("same sha should return cached *vision.Template")
	}
	m.Invalidate()
	if _, ok := m.loadCache.Load(sha); ok {
		t.Fatal("Invalidate should clear decode cache")
	}
}

func TestLongEdgeScale(t *testing.T) {
	cases := []struct {
		fw, fh, vw, vh int
		want           float64
	}{
		{1920, 1080, 1920, 1080, 1.0},
		{2560, 1440, 1280, 720, 2.0},
		{1280, 720, 2560, 1440, 0.5},
		{1366, 768, 1280, 720, 1366.0 / 1280},
		{1920, 1080, 0, 0, 0},
	}
	for _, c := range cases {
		if got := longEdgeScale(c.fw, c.fh, c.vw, c.vh); got != c.want {
			t.Errorf("longEdgeScale(%d,%d,%d,%d) = %v, want %v", c.fw, c.fh, c.vw, c.vh, got, c.want)
		}
	}
}

func TestWithinScaleTolerance(t *testing.T) {
	cases := []struct {
		scale, k float64
		want     bool
	}{
		{1.0, 2.0, true},
		{2.0, 2.0, true},
		{0.5, 2.0, true},
		{2.5, 2.0, false},
		{0.4, 2.0, false},
		{1.2, 1.0, false},
		{1.0, 1.0, true},
		{0, 2.0, false},
		{1.0, 0.5, true},  // k<1 归一到 1 → 仅精确放行
		{1.2, 0.5, false}, // k<1 归一到 1 → 非精确判否
	}
	for _, c := range cases {
		if got := withinScaleTolerance(c.scale, c.k); got != c.want {
			t.Errorf("withinScaleTolerance(%v,%v) = %v, want %v", c.scale, c.k, got, c.want)
		}
	}
}

func TestNormScaleTolerance(t *testing.T) {
	cases := []struct{ in, want float64 }{
		{0, 2.0},    // 未配 → 默认
		{-1, 2.0},   // 异常 → 默认
		{0.5, 1.0},  // (0,1) → 仅精确
		{1.0, 1.0},  // 正好 1
		{2.5, 2.5},  // 透传
	}
	for _, c := range cases {
		if got := normScaleTolerance(c.in); got != c.want {
			t.Errorf("normScaleTolerance(%v) = %v, want %v", c.in, got, c.want)
		}
	}
}
