package ai

import (
	"fmt"
	"regexp"
)

// placeholderRe 匹配可选反斜杠转义前缀 + {{Name}}。单花括号字面不触发。
var placeholderRe = regexp.MustCompile(`(\\?)\{\{(\w+)\}\}`)

// renderPromptTemplate 渲染提示词模板的 {{Name}} 占位:
//   - 反斜杠转义 \{{Name}} → 字面 {{Name}}
//   - Name ∈ imageNames(Image 类型输入)→ "[image]" 标记(图像走多模态块, 不进文本)
//   - Name ∈ inputs → 字符串化值
//   - 未声明 → error(validator 已先拦, 运行期兜底)
func renderPromptTemplate(tmpl string, inputs map[string]any, imageNames map[string]bool) (string, error) {
	var firstErr error
	out := placeholderRe.ReplaceAllStringFunc(tmpl, func(m string) string {
		sub := placeholderRe.FindStringSubmatch(m)
		esc, name := sub[1], sub[2]
		if esc == `\` {
			return "{{" + name + "}}"
		}
		if imageNames[name] {
			return "[image]"
		}
		if v, ok := inputs[name]; ok {
			return fmt.Sprintf("%v", v)
		}
		if firstErr == nil {
			firstErr = fmt.Errorf("undeclared template placeholder {{%s}}", name)
		}
		return m
	})
	if firstErr != nil {
		return "", firstErr
	}
	return out, nil
}
