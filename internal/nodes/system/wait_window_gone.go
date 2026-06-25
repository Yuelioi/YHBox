// WaitWindowGone — 轮询等待匹配窗口从系统消失 (对称 WaitWindow)。
package system

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"yotta/internal/node"
	"yotta/pkg/winutil"
)

func init() { node.Register(&WaitWindowGone{}) }

// WaitWindowGone 按 Title/Class/ProcessName 轮询等待窗口消失, 消失走 Gone, 超时仍在走 Timeout
// (不报错, 供分支兜底)。常接在「关闭程序」后确认窗口真的关掉。
// 空 spec / regex 非法 / ctx 取消 = 裸冒泡 (配置错, 不应兜底)。
type WaitWindowGone struct{}

const (
	wwgInExec        = "In"
	wwgInTitle       = "Title"
	wwgInClass       = "Class"
	wwgInProcessName = "ProcessName"
	wwgInTitleMatch  = "TitleMatch"
	wwgInTimeoutMs   = "TimeoutMs"
	wwgOutGone       = "Gone"
	wwgOutTimeout    = "Timeout"
)

// wwgPollInterval 轮询间隔, 与 WaitWindow 一致。
const wwgPollInterval = 500 * time.Millisecond

// waitWindowGone 可注入 seam (测试替换 mock, 生产用 winutil.WaitWindowGone)。
var waitWindowGone = func(ctx context.Context, spec winutil.MatchSpec, timeout, interval time.Duration) error {
	return winutil.WaitWindowGone(ctx, spec, timeout, interval)
}

func (WaitWindowGone) Spec() node.Spec {
	return node.Spec{
		Kind:     "WaitWindowGone",
		Category: "Window",
		Inputs: []node.InputSpec{
			{Name: wwgInExec, Type: node.TypeExec},
			{Name: wwgInTitle, Type: "String", Default: "", Widget: node.WidgetSpec{Kind: "text"}},
			{Name: wwgInClass, Type: "String", Default: "", Widget: node.WidgetSpec{Kind: "text"}},
			{Name: wwgInProcessName, Type: "String", Default: "", Widget: node.WidgetSpec{Kind: "text"}},
			{Name: wwgInTitleMatch, Type: "String", Default: "exact",
				Widget: node.WidgetSpec{Kind: "dropdown",
					Props: node.MarshalProps(node.DropdownProps{
						Options: []node.EnumOption{
							{Value: "exact"},
							{Value: "regex"},
						}})}},
			{Name: wwgInTimeoutMs, Type: "Number", Default: json.Number("10000"),
				Widget: node.WidgetSpec{Kind: "number"}},
		},
		Outputs: []node.OutputSpec{
			{Name: wwgOutGone, Type: node.TypeExec},
			{Name: wwgOutTimeout, Type: node.TypeExec},
			// 无 Fail 出口: 空 match / regex 非法 / ctx 取消 = 配置错裸冒泡。
			// 超时仍在 = Timeout 分支 (兜底)。
		},
	}
}

func (WaitWindowGone) Run(ctx node.Ctx, in node.Inputs) (node.Outputs, error) {
	spec := winutil.MatchSpec{
		Title:       in.String(wwgInTitle),
		Class:       in.String(wwgInClass),
		ProcessName: in.String(wwgInProcessName),
		TitleMatch:  in.String(wwgInTitleMatch),
	}
	timeout := time.Duration(in.Int(wwgInTimeoutMs)) * time.Millisecond

	err := waitWindowGone(ctx.Context(), spec, timeout, wwgPollInterval)
	if err == nil {
		return ctx.Out(wwgOutGone).Fire(), nil
	}
	if errors.Is(err, winutil.ErrWindowStillPresent) {
		return ctx.Out(wwgOutTimeout).Fire(), nil // 超时仍在 → 兜底分支, 不当节点失败
	}
	return nil, err // 空 match / regex 非法 / ctx 取消 → 真错
}
