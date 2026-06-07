package tools

import "testing"

func rgbN(n int, r, g, b uint8) []RGB {
	out := make([]RGB, n)
	for i := range out {
		out[i] = RGB{r, g, b}
	}
	return out
}

func TestExtractColorRange_Empty(t *testing.T) {
	if _, err := extractColorRange(nil, "hsv"); err == nil {
		t.Fatal("空样本应返回错误")
	}
}

func TestExtractColorRange_PureRed_HSV(t *testing.T) {
	got, err := extractColorRange(rgbN(100, 255, 0, 0), "hsv")
	if err != nil {
		t.Fatal(err)
	}
	want := [6]int{0, 5, 90, 100, 90, 100}
	if got.Range != want || got.HueWrap {
		t.Fatalf("got %v wrap=%v, want %v wrap=false", got.Range, got.HueWrap, want)
	}
}

func TestExtractColorRange_RGB(t *testing.T) {
	got, _ := extractColorRange(rgbN(100, 200, 50, 50), "rgb")
	want := [6]int{185, 215, 35, 65, 35, 65}
	if got.Range != want {
		t.Fatalf("got %v want %v", got.Range, want)
	}
}

func TestExtractColorRange_PercentileOutlierResistance(t *testing.T) {
	s := append(rgbN(98, 200, 50, 50), rgbN(2, 255, 255, 255)...)
	got, _ := extractColorRange(s, "rgb")
	if got.Range[1] > 230 {
		t.Fatalf("rMax=%d 被离群拉爆 (期望 ~215)", got.Range[1])
	}
}

func TestExtractColorRange_SingleSample(t *testing.T) {
	got, _ := extractColorRange(rgbN(1, 255, 0, 0), "hsv")
	if got.Range[2] != 90 || got.Range[3] != 100 {
		t.Fatalf("单样本 S 应 [90,100], got [%d,%d]", got.Range[2], got.Range[3])
	}
}

func TestExtractColorRange_GrayExcludedFromHue(t *testing.T) {
	s := append(rgbN(90, 128, 128, 128), rgbN(10, 0, 255, 0)...)
	got, _ := extractColorRange(s, "hsv")
	if got.Range[0] < 110 || got.Range[1] > 130 {
		t.Fatalf("H 应只反映绿像素 ~120, got [%d,%d]", got.Range[0], got.Range[1])
	}
	if got.Range[2] != 0 {
		t.Fatalf("sMin 应由全部像素(含灰)决定=0, got %d", got.Range[2])
	}
}

func TestExtractColorRange_HueWrap(t *testing.T) {
	s := append(append(rgbN(30, 255, 0, 0), rgbN(30, 255, 0, 20)...), rgbN(30, 255, 20, 0)...)
	got, _ := extractColorRange(s, "hsv")
	if !got.HueWrap || got.Range[0] != 0 || got.Range[1] != 360 {
		t.Fatalf("环绕应 HueWrap=true 且 H=[0,360], got wrap=%v H=[%d,%d]", got.HueWrap, got.Range[0], got.Range[1])
	}
}

func TestExtractColorRange_HueNonWrap_Tight(t *testing.T) {
	s := append(append(rgbN(30, 255, 120, 0), rgbN(30, 255, 128, 0)...), rgbN(30, 255, 136, 0)...)
	got, _ := extractColorRange(s, "hsv")
	if got.HueWrap || got.Range[0] < 20 || got.Range[1] > 40 {
		t.Fatalf("非环绕橙色应紧区间 ~[25,35], got wrap=%v H=[%d,%d]", got.HueWrap, got.Range[0], got.Range[1])
	}
}

func TestExtractColorRange_PaddingThenClamp(t *testing.T) {
	got, _ := extractColorRange(rgbN(50, 250, 5, 5), "rgb")
	if got.Range[1] != 255 {
		t.Fatalf("padding 先加后 clamp: rMax 应 255, got %d", got.Range[1])
	}
	if got.Range[0] != 235 {
		t.Fatalf("rMin 应 250-15=235, got %d", got.Range[0])
	}
}

func TestHueWraps(t *testing.T) {
	cases := []struct {
		in   []int
		want bool
	}{
		{[]int{0, 10, 350}, true},
		{[]int{20, 22, 25}, false},
		{[]int{0, 2, 358}, true},
		{[]int{30}, false},
		{[]int{}, false},
	}
	for _, c := range cases {
		if got := hueWraps(c.in); got != c.want {
			t.Errorf("hueWraps(%v)=%v want %v", c.in, got, c.want)
		}
	}
}
