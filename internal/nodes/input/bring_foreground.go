package input

import (
	"fmt"

	"yhbox/internal/node"
)

func init() { node.Register(&BringGameForeground{}) }

// BringGameForeground 把游戏窗口置前台. "是否重试 / 失败处理策略"下沉到
// WindowService 适配层 — 节点本身只做一次调用, error 直接走报错路径
// (跟其他 input 节点一致).
type BringGameForeground struct{}

const (
	bgfInExec  = "In"
	bgfOutDone = "Done"
)

func (BringGameForeground) Spec() node.Spec {
	return node.Spec{
		Kind:        "BringGameForeground",
		Category:    "Input",
		NeedsWindow: true,
		Inputs: []node.InputSpec{
			{Name: bgfInExec, Type: "Exec"},
		},
		Outputs: []node.OutputSpec{
			{Name: bgfOutDone, Type: "Exec"},
		},
	}
}

func (BringGameForeground) Run(ctx node.Ctx, in node.Inputs) (node.Outputs, error) {
	// 老 runtime: 重试 3×50ms, 全 fail 仅 warn log, 不阻塞流程.
	// 新实现: 由 WindowService 适配层封装重试. 节点拿到 error 时 warn log 然后继续走 Done.
	if err := ctx.Window().BringForeground(); err != nil {
		ctx.Log().Warn("BringGameForeground: %v (游戏窗口可能是全屏独占)", err)
	}
	return ctx.Out(bgfOutDone).Fire(), nil
}

func (BringGameForeground) Display(in node.Inputs, exitName string, out node.OutputData) string {
	return fmt.Sprintf("bring foreground")
}
