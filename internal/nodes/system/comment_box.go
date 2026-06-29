// internal/nodes/system/comment_box.go
// CommentBox — render-only 注释节点 (sticky note). Spec.IsVisualOnly=true; 零 capability.
// 字段 (Title/Content/Color) 存 config.literal, 由前端 CommentBoxNode.vue 渲染 + 内联编辑.
// backend 不 Run / 不校验, Spec 仅为 catalog 展示 + 前端默认值来源.
package system

import (
	"encoding/json"

	"yotta/internal/node"
)

func init() { node.Register(&CommentBox{}) }

type CommentBox struct{}

const (
	cbInTitle   = "Title"
	cbInContent = "Content"
	cbInColor   = "Color"
	cbInIcon    = "Icon"
	cbInWidth   = "Width"
)

func (CommentBox) Spec() node.Spec {
	return node.Spec{
		Kind:     "CommentBox",
		Category: "System",
		Inputs: []node.InputSpec{
			{Name: cbInTitle, Type: "String", Default: "说明",
				Widget: node.WidgetSpec{Kind: "text"}},
			{Name: cbInContent, Type: "String",
				Default: "在这里写**这个容器怎么用**。\n\n支持 markdown：列表 / 链接 / 代码等。",
				Widget:  node.WidgetSpec{Kind: "textarea"}},
			// Color/Icon 走视觉注册中心选择器 (前端 color-preset / icon-preset widget).
			{Name: cbInColor, Type: "String", Default: "amber",
				Widget: node.WidgetSpec{Kind: "color-preset"}},
			{Name: cbInIcon, Type: "String", Default: "i-tabler-note",
				Widget: node.WidgetSpec{Kind: "icon-preset"}},
			{Name: cbInWidth, Type: "Number", Default: json.Number("280"),
				Widget: node.WidgetSpec{Kind: "number",
					Props: node.MarshalProps(node.SliderProps{Min: 160, Max: 800, Step: 10})}},
		},
		// no Outputs — render-only
		IsVisualOnly: true,
	}
}
