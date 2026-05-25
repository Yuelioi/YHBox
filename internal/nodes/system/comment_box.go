// internal/nodes/system/comment_box.go
// CommentBox — render-only annotation node. Spec.IsVisualOnly=true; framework
// runner gatekeep IsVisualOnly 不调任何执行路径. 零 capability (无 Run/RunRegion/Evaluate).
package system

import (
	"encoding/json"

	"yhbox/internal/node"
)

func init() { node.Register(&CommentBox{}) }

type CommentBox struct{}

const (
	cbInLabel  = "Label"
	cbInColor  = "Color"
	cbInWidth  = "Width"
	cbInHeight = "Height"
)

func (CommentBox) Spec() node.Spec {
	return node.Spec{
		Kind:        "CommentBox",
		Category:    "System",
		DisplayName: "注释框",
		Description: "纯渲染节点 — 在画布上画带颜色的标签框. 不参与执行, 不连边.",
		Inputs: []node.InputSpec{
			{Name: cbInLabel, Type: "String", Default: "注释",
				DisplayName: "标签",
				Widget:      node.WidgetSpec{Kind: "text"}},
			{Name: cbInColor, Type: "String", Default: "#fbbf24",
				DisplayName: "颜色",
				Widget:      node.WidgetSpec{Kind: "text"}},
			{Name: cbInWidth, Type: "Number", Default: json.Number("200"),
				DisplayName: "宽度",
				Widget:      node.WidgetSpec{Kind: "text"}},
			{Name: cbInHeight, Type: "Number", Default: json.Number("150"),
				DisplayName: "高度",
				Widget:      node.WidgetSpec{Kind: "text"}},
		},
		// no Outputs — render-only
		IsVisualOnly: true,
	}
}
