package vision

import (
	"image"
	"image/color"
	"image/draw"
	"image/png"
	"math"
	"os"
	"path/filepath"
	"sort"
	"testing"
)

// TestHookTextAlgoCompare 在 debug/hook@full@*.png 上跑 4 种灰度预处理 + NCC，
// 输出 conf 表，决策上钩文字检测要不要换算法。
//
// 算法：
//
//	A plain  BT.601 (基线)
//	B vchan  max(R,G,B)
//	C otsu   Otsu 二值化（背景方差归零）
//	D sobel  Sobel 梯度幅值（边缘）
//
// 模板：assets/fish/hook_text_1080.png （1920x1080 native，bbox 见 templates.toml）
// 不 assert 任何 conf 值——这是探索性测试，t.Logf 输出即可。
func TestHookTextAlgoCompare(t *testing.T) {
	const (
		// 1920x1080 hook_text 在原帧里的位置（见 assets/templates.toml）
		bboxX1, bboxY1, bboxX2, bboxY2 = 776, 245, 1172, 282
		padPx                          = 40 // ROI 上下左右扩边
		dumpDir                        = "../../debug/algo_dump"
		tplPath                        = "../../assets/fish/hook_text_1080.png"
	)

	// 加载模板
	tplRGBA := loadPNGRGBA(t, tplPath)
	tw, th := tplRGBA.Bounds().Dx(), tplRGBA.Bounds().Dy()
	t.Logf("template %dx%d (%s)", tw, th, filepath.Base(tplPath))

	// 收集测试图
	matches, err := filepath.Glob("../../debug/hook@full@*.png")
	if err != nil {
		t.Fatalf("glob: %v", err)
	}
	if len(matches) == 0 {
		t.Skip("no debug/hook@full@*.png found")
	}
	sort.Strings(matches)

	if err := os.MkdirAll(dumpDir, 0o755); err != nil {
		t.Fatalf("mkdir dump: %v", err)
	}

	algos := []struct {
		id   string
		name string
		// preprocess 把 RGBA 转成 [0,1] float32 数组 + (w,h)
		preprocess func(*image.RGBA) ([]float32, int, int)
	}{
		{"A", "plain ", func(r *image.RGBA) ([]float32, int, int) { return RGBAToGray(r) }},
		{"B", "vchan ", func(r *image.RGBA) ([]float32, int, int) { return VChannel(r) }},
		{"C", "otsu  ", otsuPreprocess},
		{"D", "sobel ", sobelPreprocess},
	}

	// 表头
	t.Logf("")
	header := "image"
	for _, a := range algos {
		header += "  | " + a.id + " " + a.name
	}
	t.Logf("%-44s %s", header, "")
	t.Logf("%s", lineSep(44+len(algos)*12))

	for _, imgPath := range matches {
		base := filepath.Base(imgPath)
		frame := loadPNGRGBA(t, imgPath)
		fw, fh := frame.Bounds().Dx(), frame.Bounds().Dy()
		if fw != 1920 || fh != 1080 {
			t.Logf("skip %s: %dx%d (test 假设 1920x1080)", base, fw, fh)
			continue
		}

		// 抠 ROI（bbox + pad）
		rx := bboxX1 - padPx
		ry := bboxY1 - padPx
		rw := (bboxX2 - bboxX1) + 2*padPx
		rh := (bboxY2 - bboxY1) + 2*padPx
		if rx < 0 {
			rw += rx
			rx = 0
		}
		if ry < 0 {
			rh += ry
			ry = 0
		}
		roiRGBA := image.NewRGBA(image.Rect(0, 0, rw, rh))
		draw.Draw(roiRGBA, roiRGBA.Bounds(), frame, image.Point{X: rx, Y: ry}, draw.Src)
		dumpGrayRGBA(filepath.Join(dumpDir, base+"_roi.png"), roiRGBA)
		dumpGrayRGBA(filepath.Join(dumpDir, base+"_tpl.png"), tplRGBA)

		row := base
		row = padRight(row, 44)
		for _, a := range algos {
			roiArr, riw, rih := a.preprocess(roiRGBA)
			tplArr, ttw, tth := a.preprocess(tplRGBA)
			tpl := &Template{Gray: tplArr, W: ttw, H: tth}
			x, y, conf := MatchFast(roiArr, riw, rih, tpl, DefaultParallel())

			// 命中可视化
			vis := image.NewRGBA(roiRGBA.Bounds())
			draw.Draw(vis, vis.Bounds(), roiRGBA, image.Point{}, draw.Src)
			if x >= 0 {
				drawRect(vis, image.Rect(x, y, x+ttw, y+tth), color.RGBA{255, 0, 0, 255})
			}
			outName := base + "_" + algoFileTag(a.id) + "_hit.png"
			dumpRGBA(filepath.Join(dumpDir, outName), vis)

			row += "  | " + formatConf(conf)
		}
		t.Logf("%s", row)
	}

	t.Logf("")
	t.Logf("dump: %s", dumpDir)
}

func otsuPreprocess(r *image.RGBA) ([]float32, int, int) {
	g, w, h := RGBAToGray(r)
	thr := OtsuThreshold(g)
	return Binarize(g, thr), w, h
}

func sobelPreprocess(r *image.RGBA) ([]float32, int, int) {
	g, w, h := RGBAToGray(r)
	return SobelMag(g, w, h), w, h
}

// --- helpers ---

func loadPNGRGBA(t *testing.T, path string) *image.RGBA {
	t.Helper()
	f, err := os.Open(path)
	if err != nil {
		t.Fatalf("open %s: %v", path, err)
	}
	defer f.Close()
	src, err := png.Decode(f)
	if err != nil {
		t.Fatalf("decode %s: %v", path, err)
	}
	b := src.Bounds()
	rgba := image.NewRGBA(image.Rect(0, 0, b.Dx(), b.Dy()))
	draw.Draw(rgba, rgba.Bounds(), src, b.Min, draw.Src)
	return rgba
}

func dumpGrayRGBA(path string, r *image.RGBA) {
	gray, w, h := RGBAToGray(r)
	dumpGray8(path, gray, w, h)
}

func dumpGray8(path string, gray []float32, w, h int) {
	img := image.NewGray(image.Rect(0, 0, w, h))
	for i, v := range gray {
		if v < 0 {
			v = 0
		} else if v > 1 {
			v = 1
		}
		img.Pix[i] = uint8(v * 255)
	}
	f, err := os.Create(path)
	if err != nil {
		return
	}
	defer f.Close()
	_ = png.Encode(f, img)
}

func dumpRGBA(path string, img *image.RGBA) {
	f, err := os.Create(path)
	if err != nil {
		return
	}
	defer f.Close()
	_ = png.Encode(f, img)
}

func drawRect(img *image.RGBA, r image.Rectangle, c color.RGBA) {
	for x := r.Min.X; x < r.Max.X; x++ {
		img.Set(x, r.Min.Y, c)
		img.Set(x, r.Max.Y-1, c)
	}
	for y := r.Min.Y; y < r.Max.Y; y++ {
		img.Set(r.Min.X, y, c)
		img.Set(r.Max.X-1, y, c)
	}
}

func formatConf(c float32) string {
	if c < 0 || math.IsNaN(float64(c)) {
		return " miss "
	}
	return padRight(formatFloat(c), 7)
}

func formatFloat(c float32) string {
	// 3 位小数
	s := []byte("0.000")
	v := int(c*1000 + 0.5)
	if v < 0 {
		v = 0
	}
	if v > 9999 {
		v = 9999
	}
	if v >= 1000 {
		s = []byte("1.000")
	} else {
		s[2] = byte('0' + v/100)
		s[3] = byte('0' + (v/10)%10)
		s[4] = byte('0' + v%10)
	}
	return string(s)
}

func padRight(s string, n int) string {
	for len(s) < n {
		s += " "
	}
	return s
}

func lineSep(n int) string {
	out := make([]byte, n)
	for i := range out {
		out[i] = '-'
	}
	return string(out)
}

func algoFileTag(id string) string {
	switch id {
	case "A":
		return "plain"
	case "B":
		return "vchan"
	case "C":
		return "otsu"
	case "D":
		return "sobel"
	}
	return id
}
