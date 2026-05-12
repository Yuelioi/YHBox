package fish

import (
	"testing"

	"yhbox"
	"yhbox/pkg/vision"
)

// TestEmbedTemplatesValid 验证 assets/templates.toml 能解析 + 所有 file 真的 embed。
func TestEmbedTemplatesValid(t *testing.T) {
	tomlBytes, err := yhbox.AssetBytes("templates.toml")
	if err != nil {
		t.Fatalf("read templates.toml: %v", err)
	}
	cfg, warns, err := vision.LoadTemplatesConfig(tomlBytes)
	if err != nil {
		t.Fatalf("LoadTemplatesConfig: %v", err)
	}
	for _, w := range warns {
		t.Errorf("warning in embed templates: %s", w)
	}
	for tool, slots := range cfg {
		for slot, specs := range slots {
			for _, s := range specs {
				if _, err := yhbox.AssetBytes(s.File); err != nil {
					t.Errorf("[%s.%s] file=%q: 没找到 embed (%v)", tool, slot, s.File, err)
				}
			}
		}
	}
}

// TestRequiredSlotsHaveTemplate 验证 fish 必需 slot 都有候选。
func TestRequiredSlotsHaveTemplate(t *testing.T) {
	d, err := NewDetector()
	if err != nil {
		t.Fatalf("NewDetector: %v", err)
	}
	for _, slot := range requiredSlots {
		if len(d.templates[slot]) == 0 {
			t.Errorf("slot %q 无候选模板", slot)
		}
	}
}
