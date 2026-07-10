package runtime

import (
	"testing"

	"github.com/yottaapp/yotta/internal/node"
)

func tm(x, y, conf float64, key string) node.TemplateMatch {
	return node.TemplateMatch{
		Point:       node.Point{X: x, Y: y},
		Conf:        conf,
		BBox:        [4]float64{x - 0.05, y - 0.05, 0.1, 0.1}, // 短边 0.1 → auto radius 0.05
		TemplateKey: key,
	}
}

// 跨模板统一 NMS: 高分 A 抑制重叠低分 B (含不同模板), 远处 C 保留; 结果 conf 降序。
func TestNmsMatches_CrossTemplateSuppress(t *testing.T) {
	in := []node.TemplateMatch{
		tm(0.51, 0.50, 0.80, "B"), // 距 A 0.01 < min(0.05,0.05) → 被 A 抑制
		tm(0.50, 0.50, 0.95, "A"),
		tm(0.90, 0.90, 0.70, "C"), // 远 → 保留
	}
	out := nmsMatches(in, 0, 100) // auto radius
	if len(out) != 2 {
		t.Fatalf("want 2 (A,C), got %d: %+v", len(out), out)
	}
	if out[0].TemplateKey != "A" {
		t.Errorf("primary=%s, want A (highest conf)", out[0].TemplateKey)
	}
	for _, m := range out {
		if m.TemplateKey == "B" {
			t.Error("B should be suppressed by overlapping higher-conf A (cross-template)")
		}
	}
	// conf 降序
	for i := 1; i < len(out); i++ {
		if out[i].Conf > out[i-1].Conf {
			t.Error("not conf-descending")
		}
	}
}

// 显式 minDistance (像素): radius = minDistance/frameW 归一化。
func TestNmsMatches_ExplicitMinDistance(t *testing.T) {
	in := []node.TemplateMatch{
		tm(0.50, 0.50, 0.90, "A"),
		tm(0.60, 0.50, 0.85, "B"), // 距 0.10; minDistance 20/frameW 100 = 0.20 → 0.10<0.20 抑制
	}
	out := nmsMatches(in, 20, 100)
	if len(out) != 1 {
		t.Fatalf("explicit minDist should merge to 1, got %d: %+v", len(out), out)
	}
	// 间距足够: minDistance 5 → 0.05 < 0.10 → 两个都留
	out2 := nmsMatches(in, 5, 100)
	if len(out2) != 2 {
		t.Fatalf("far enough (minDist 5) should keep 2, got %d", len(out2))
	}
}

func TestNmsMatches_EmptyAndSingle(t *testing.T) {
	if got := nmsMatches(nil, 0, 100); got != nil {
		t.Errorf("nil → nil, got %+v", got)
	}
	one := []node.TemplateMatch{tm(0.1, 0.1, 0.9, "A")}
	if got := nmsMatches(one, 0, 100); len(got) != 1 {
		t.Errorf("single → 1, got %d", len(got))
	}
}
