package input

import (
	"yotta/internal/node"
)

func init() { node.Register(&BringWindowForeground{}) }

// BringWindowForeground 把 Win32 HWND 窗口置前台. "是否重试 / 失败处理策略"
// 下沉到 WindowService 适配层 — 节点本身只做一次调用, error 直接走报错路径
// (跟其他 input 节点一致). Browser/Android target 不复用这个前台语义。
type BringWindowForeground struct{}

const (
	bgfInExec  = "In"
	bgfOutDone = "Done"
)

func (BringWindowForeground) Spec() node.Spec {
	return node.Spec{
		Kind:            "BringWindowForeground",
		Category:        "Window",
		NeedsWindow:     true,
		NeedsForeground: true,
		Inputs: append([]node.InputSpec{
			{Name: bgfInExec, Type: "Exec"},
		}, node.WindowInputSpec()),
		Outputs: []node.OutputSpec{
			{Name: bgfOutDone, Type: "Exec"},
		},
	}
}

func (BringWindowForeground) Run(ctx node.Ctx, in node.Inputs) (node.Outputs, error) {
	// 重试由 WindowService 适配层封装. 拿到 error 仅 warn log 不阻塞 — Win32 窗口全屏独占时
	// 常拉不动, 不该卡住流程, 继续走 Done.
	if err := ctx.Window().BringForeground(); err != nil {
		ctx.Log().Warn("BringWindowForeground: %v (Win32 窗口可能是全屏独占)", err)
	}
	return ctx.Out(bgfOutDone).Fire(), nil
}
