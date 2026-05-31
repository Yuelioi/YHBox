// template_common.go — WaitTemplate / ClickTemplate / CheckTemplate 共用的多模板辅助.
// 三个节点的 Template 字段已从单 key 升成 Templates 列表 (+ MatchMode any/all);
// 校验 / 依赖抽取逻辑一致, 抽这里避免三处重复.
package detect

import (
	"fmt"
	"strings"

	"yhbox/internal/node"
)

// validateTemplateKeys 逐个校验列表里的 key 是 namespace.name 格式. 空列表 / 空 key 跳过
// (未配 template 由 required / MISSING_TEMPLATE 规则负责, 跟旧单值行为一致).
func validateTemplateKeys(keys []string, field string) []node.ValidationError {
	var errs []node.ValidationError
	for _, key := range keys {
		if key != "" && !strings.Contains(key, ".") {
			errs = append(errs, node.ValidationError{
				Code:    "INVALID_TEMPLATE_KEY",
				Message: fmt.Sprintf("template key 必须 namespace.name 格式, got %q", key),
				Field:   field,
			})
		}
	}
	return errs
}

// templateDeps 把模板 key 列表转成 library scanner 用的 template 依赖 (每 key 一条).
func templateDeps(keys []string) []node.Dependency {
	deps := make([]node.Dependency, 0, len(keys))
	for _, key := range keys {
		if key != "" {
			deps = append(deps, node.Dependency{Kind: "template", Key: key})
		}
	}
	return deps
}
