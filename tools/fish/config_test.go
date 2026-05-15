package fish

import (
	"io/fs"
	"testing"

	"yhbox/pkg/vision"
)

// TestEmbedTemplatesValid 验证 configs/zh/templates.toml 能解析 + 所有 file 真的 embed。
func TestEmbedTemplatesValid(t *testing.T) {
	tomlBytes := Assets().TemplatesTOML
	if len(tomlBytes) == 0 {
		t.Fatal("Assets().TemplatesTOML 为空（LoadConfig 没跑？）")
	}
	cfg, warns, err := vision.LoadTemplatesConfig(tomlBytes)
	if err != nil {
		t.Fatalf("LoadTemplatesConfig: %v", err)
	}
	for _, w := range warns {
		t.Errorf("warning in embed templates: %s", w)
	}
	tplFS := Assets().Templates
	if tplFS == nil {
		t.Fatal("Assets().Templates 为 nil")
	}
	for tool, slots := range cfg {
		for slot, specs := range slots {
			for _, s := range specs {
				if _, err := fs.ReadFile(tplFS, s.File); err != nil {
					t.Errorf("[%s.%s] file=%q: 找不到 embed (%v)", tool, slot, s.File, err)
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
