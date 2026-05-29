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
		Kind:     "CommentBox",
		Category: "System",
		Inputs: []node.InputSpec{
			{Name: cbInLabel, Type: "String", Default: "注释",
				Widget: node.WidgetSpec{Kind: "text"}},
			{Name: cbInColor, Type: "String", Default: "#fbbf24",
				Widget: node.WidgetSpec{Kind: "text"}},
			{Name: cbInWidth, Type: "Number", Default: json.Number("200"),
				Widget: node.WidgetSpec{Kind: "text"}},
			{Name: cbInHeight, Type: "Number", Default: json.Number("150"),
				Widget: node.WidgetSpec{Kind: "text"}},
		},
		// no Outputs — render-only
		IsVisualOnly: true,
	}
}
