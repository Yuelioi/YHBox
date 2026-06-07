// WaitWindow — 轮询等待匹配窗口出现 (可配超时)。纯探测, 不改当前活动窗口。
package system

import (
	"encoding/json"
	"errors"
	"time"

	"yotta/internal/node"
	"yotta/pkg/winutil"
)

func init() { node.Register(&WaitWindow{}) }

// WaitWindow 按 Title/Class/ProcessName 轮询等待窗口出现, 出现走 Found, 超时走 Timeout
// (不报错, 供分支兜底)。本节点只探测窗口是否存在, 不改当前活动窗口 —— 要操作该窗口,
// Found 后接 WindowTarget 锁定 (此时窗口已存在, 会瞬时解析)。
type WaitWindow struct{}

const (
	wwInExec        = "In"
	wwInTitle       = "Title"
	wwInClass       = "Class"
	wwInProcessName = "ProcessName"
	wwInTitleMatch  = "TitleMatch"
	wwInTimeoutMs   = "TimeoutMs"
	wwOutFound      = "Found"
	wwOutTimeout    = "Timeout"
)

// wwPollInterval 轮询间隔, 跟 windowAdapter.SetActive 的解析间隔一致 (500ms)。
const wwPollInterval = 500 * time.Millisecond

func (WaitWindow) Spec() node.Spec {
	return node.Spec{
		Kind:     "WaitWindow",
		Category: "System",
		Inputs: []node.InputSpec{
			{Name: wwInExec, Type: node.TypeExec},
			{Name: wwInTitle, Type: "String", Default: "", Widget: node.WidgetSpec{Kind: "text"}},
			{Name: wwInClass, Type: "String", Default: "", Widget: node.WidgetSpec{Kind: "text"}},
			{Name: wwInProcessName, Type: "String", Default: "", Widget: node.WidgetSpec{Kind: "text"}},
			{Name: wwInTitleMatch, Type: "String", Default: "exact",
				Widget: node.WidgetSpec{Kind: "dropdown",
					Props: node.MarshalProps(node.DropdownProps{
						Options: []node.EnumOption{
							{Value: "exact"},
							{Value: "regex"},
						}})}},
			{Name: wwInTimeoutMs, Type: "Number", Default: json.Number("10000"),
				Widget: node.WidgetSpec{Kind: "number"}},
		},
		Outputs: []node.OutputSpec{
			{Name: wwOutFound, Type: node.TypeExec},
			{Name: wwOutTimeout, Type: node.TypeExec},
			// 无 Fail 出口: 找不到窗 = Timeout 结果分支 (:79); 空 match/regex 非法 = 配置错裸冒泡。
			// 无真·运行时 Coded error → 不给死引脚。
		},
	}
}

func (WaitWindow) Run(ctx node.Ctx, in node.Inputs) (node.Outputs, error) {
	spec := winutil.MatchSpec{
		Title:       in.String(wwInTitle),
		Class:       in.String(wwInClass),
		ProcessName: in.String(wwInProcessName),
		TitleMatch:  in.String(wwInTitleMatch),
	}
	timeout := time.Duration(in.Int(wwInTimeoutMs)) * time.Millisecond

	_, err := winutil.ResolveWindow(ctx.Context(), spec, timeout, wwPollInterval)
	if err == nil {
		return ctx.Out(wwOutFound).Fire(), nil
	}
	if errors.Is(err, winutil.ErrWindowNotFound) {
		return ctx.Out(wwOutTimeout).Fire(), nil // 超时没出现 → 兜底分支, 不当节点失败
	}
	return nil, err // 空 match / regex 非法 / ctx 取消 → 真错
}
