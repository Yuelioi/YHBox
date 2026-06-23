package container

import "regexp"

// aiPlaceholderRe 匹配可选反斜杠转义 + {{Name}}(与 nodes/ai 的渲染侧一致)。
var aiPlaceholderRe = regexp.MustCompile(`(\\?)\{\{(\w+)\}\}`)

// validateAI 编辑期校验 AI 节点(节点本地; 连接存在性需 settings, 不在此层做)。
//   - User 模板非空 → AI_EMPTY_USER
//   - 提示词 {{Name}} 引用未声明的动态输入 → AI_UNDECLARED_PLACEHOLDER(硬错, 抓拼写)
//   - 引用 Image 类型输入 → AI_IMAGE_IN_TEXT(warn, 图像不参与文本插值)
func validateAI(n *GraphNode) []ValidationError {
	var errs []ValidationError

	if PinString(n, "User") == "" {
		errs = append(errs, ValidationError{Severity: SeverityError, NodeID: n.ID, Code: "AI_EMPTY_USER"})
	}

	declared := map[string]bool{}
	imageNames := map[string]bool{}
	for _, d := range ParseDynamicInputDecls(n) {
		declared[d.Name] = true
		if d.Type == "Image" {
			imageNames[d.Name] = true
		}
	}

	for _, tmpl := range []string{PinString(n, "User"), PinString(n, "System")} {
		for _, m := range aiPlaceholderRe.FindAllStringSubmatch(tmpl, -1) {
			if m[1] == `\` {
				continue // 转义字面 \{{ — 不算引用
			}
			name := m[2]
			switch {
			case imageNames[name]:
				errs = append(errs, ValidationError{
					Severity: SeverityWarning, NodeID: n.ID,
					Code: "AI_IMAGE_IN_TEXT", Params: map[string]any{"name": name},
				})
			case !declared[name]:
				errs = append(errs, ValidationError{
					Severity: SeverityError, NodeID: n.ID,
					Code: "AI_UNDECLARED_PLACEHOLDER", Params: map[string]any{"name": name},
				})
			}
		}
	}
	return errs
}
