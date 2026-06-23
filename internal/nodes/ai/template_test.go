package ai

import "testing"

func TestRenderPromptTemplate(t *testing.T) {
	inputs := map[string]any{"x": float64(1), "name": "bob"}
	imgs := map[string]bool{"pic": true}

	cases := []struct {
		tmpl    string
		want    string
		wantErr bool
	}{
		{"a {{x}} b", "a 1 b", false},
		{"hi {{name}}", "hi bob", false},
		{"see {{pic}}", "see [image]", false}, // Image 输入 → [image] 标记
		{`lit \{{x}}`, "lit {{x}}", false},    // 反斜杠转义 → 字面
		{"{{nope}}", "", true},                // 未声明 → error
		{"no placeholder", "no placeholder", false},
		{"json {a:1}", "json {a:1}", false}, // 单花括号字面, 不触发
	}
	for _, c := range cases {
		got, err := renderPromptTemplate(c.tmpl, inputs, imgs)
		if c.wantErr {
			if err == nil {
				t.Errorf("%q: expected error, got nil", c.tmpl)
			}
			continue
		}
		if err != nil {
			t.Errorf("%q: unexpected error %v", c.tmpl, err)
			continue
		}
		if got != c.want {
			t.Errorf("%q: got %q, want %q", c.tmpl, got, c.want)
		}
	}
}
