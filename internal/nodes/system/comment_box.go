// internal/nodes/system/comment_box.go
// CommentBox — render-only annotation node. 老 spec ExecIn=nil/ExecOut=nil +
// IsVisualOnly=true; runtime/nodes.go::execNode 上层 gatekeep `spec.IsVisualOnly`
// 直接 return nil, nil — Run 永不调.
//
// 新框架同套路: Spec.IsVisualOnly=true → Phase 5 runner 应跳过 Run. 万一
// runner 忘记 gatekeep 调进来, 返 errVisualOnlyNotRunnable sentinel (从
// sentinels.go) 让 framework 显式报错而不是 silently no-op.
package system

import "yhbox/internal/node"

func init() { node.Register(&CommentBox{}) }

type CommentBox struct{}

const (
	cbInLabel  = "label"
	cbInColor  = "color"
	cbInWidth  = "width"
	cbInHeight = "height"
)

func (CommentBox) Spec() node.Spec {
	return node.Spec{
		Kind:        "CommentBox",
		Version:     1,
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
			{Name: cbInWidth, Type: "Number", Default: 200,
				DisplayName: "宽度",
				Widget:      node.WidgetSpec{Kind: "text"}},
			{Name: cbInHeight, Type: "Number", Default: 150,
				DisplayName: "高度",
				Widget:      node.WidgetSpec{Kind: "text"}},
		},
		// no Outputs — render-only
		IsVisualOnly: true,
	}
}

func (CommentBox) Run(ctx node.Ctx, in node.Inputs) (node.Outputs, error) {
	// Framework Spec.IsVisualOnly gatekeep 应阻止调到这里; 防御性返 sentinel.
	return nil, errVisualOnlyNotRunnable
}
