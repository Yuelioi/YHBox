// internal/nodes/mock/form.go
package mock

import "yhbox/internal/node"

func init() {
	node.Register(&Form{})
}

type Form struct{}

const (
	formInExec        = "in"
	formInTitle       = "Title"
	formInDescription = "Description"
	formInMode        = "Mode"
	formInEnableLog   = "EnableLog"
	formInDetail      = "Detail"
	formInPassword    = "Password"
	formOutSubmit     = "Submit"
)

func (Form) Spec() node.Spec {
	return node.Spec{
		Kind:        "Mock_Form",
		Version:     1,
		Category:    "Mock",
		DisplayName: "Mock 表单",
		Description: "Phase 0 测试节点 — 验证 string/textarea/dropdown/Required/AND-OR conditional show widget.",
		Inputs: []node.InputSpec{
			{Name: formInExec, Type: "Exec"},
			{Name: formInTitle, Type: "String", Required: true,
				DisplayName: "标题",
				Doc:         "必填 — Phase 0 验证 Required 红星 + 提交 ValidationError",
				Widget:      node.WidgetSpec{Kind: "text"}},
			{Name: formInDescription, Type: "String",
				DisplayName: "描述",
				Widget: node.WidgetSpec{Kind: "textarea",
					Props: node.MarshalProps(node.TextareaProps{Rows: 4, MaxLength: 500})}},
			{Name: formInMode, Type: "String", Default: "simple",
				DisplayName: "模式",
				Widget: node.WidgetSpec{Kind: "dropdown",
					Props: node.MarshalProps(node.DropdownProps{
						Options: []node.EnumOption{
							{Value: "simple", Label: "简单"},
							{Value: "advanced", Label: "高级"},
						}})}},
			{Name: formInEnableLog, Type: "Bool", Default: false,
				DisplayName: "启用日志",
				Widget:      node.WidgetSpec{Kind: "checkbox"}},
			{Name: formInDetail, Type: "String",
				DisplayName: "详细信息",
				Doc:         "仅 Mode=advanced AND EnableLog=true 时显示 — Phase 0 验证 AND/OR 条件",
				VisibleWhen: &node.VisibleRule{
					And: []*node.VisibleRule{
						{Field: formInMode, Equals: "advanced"},
						{Field: formInEnableLog, Equals: true},
					}},
				Widget: node.WidgetSpec{Kind: "textarea"}},
			{Name: formInPassword, Type: "String",
				DisplayName: "密码", Advanced: true,
				Widget: node.WidgetSpec{Kind: "password"}},
		},
		Outputs: []node.OutputSpec{
			{Name: formOutSubmit, Type: "Exec", DisplayName: "提交"},
		},
	}
}
