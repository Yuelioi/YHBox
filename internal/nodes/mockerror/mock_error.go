// Package mockerror MockError 固定返 error, 验证 framework error 传播 + 节点变红 UI.
// build tag: 不在 production 注册. 测试用 -tags mockerror.
//go:build mockerror

package mockerror

import (
	"errors"

	"yhbox/internal/node"
)

func init() { node.Register(&MockError{}) }

type MockError struct{}

const (
	meInExec    = "In"
	meInMessage = "ErrorMessage"
	meOutDone   = "Done"
)

func (MockError) Spec() node.Spec {
	return node.Spec{
		Kind:        "MockError",
		Category:    "Test",
		DisplayName: "Mock Error",
		Description: "固定返 error, 验证 framework error 传播. Build-tagged: -tags mockerror.",
		Inputs: []node.InputSpec{
			{Name: meInExec, Type: "Exec"},
			{Name: meInMessage, Type: "String", Default: "deliberate failure",
				DisplayName: "错误消息",
				Widget:      node.WidgetSpec{Kind: "text"}},
		},
		Outputs: []node.OutputSpec{
			{Name: meOutDone, Type: "Exec", DisplayName: "完成 (永远走不到)"},
		},
	}
}

func (MockError) Run(ctx node.Ctx, in node.Inputs) (node.Outputs, error) {
	return nil, errors.New(in.String(meInMessage))
}
