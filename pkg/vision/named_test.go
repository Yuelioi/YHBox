package vision

import "testing"

// 注：TestPickBest 和 TestMatchTextROI 已移除，因为它们依赖旧的 TemplateSpec.Slot 字段
// 和旧的 LoadTemplatesConfig 签名（返回 2 values）。
// 这些字段和签名在 Task 1 中已被重构。
// 待 Task 2-3 完成 TemplatesConfig 的加载和遍历逻辑后，会重新编写这些测试。

// TestPickBest_RequiresExactMatch 精确匹配 — 没精确分辨率返回 nil。
func TestPickBest_RequiresExactMatch(t *testing.T) {
	tpls := []*NamedTemplate{
		{BaseW: 1920, BaseH: 1080},
		{BaseW: 1280, BaseH: 720},
	}

	if got := PickBest(tpls, 1920, 1080); got == nil || got.BaseH != 1080 {
		t.Errorf("精确 1080p 应该匹配 1080 模板，got %+v", got)
	}
	if got := PickBest(tpls, 1280, 720); got == nil || got.BaseH != 720 {
		t.Errorf("精确 720p 应该匹配 720 模板，got %+v", got)
	}
	if got := PickBest(tpls, 2560, 1440); got != nil {
		t.Errorf("1440p 无精确匹配应该返回 nil，got %+v", got)
	}
	if got := PickBest(tpls, 1366, 768); got != nil {
		t.Errorf("768p 无精确匹配应该返回 nil，got %+v", got)
	}
}
