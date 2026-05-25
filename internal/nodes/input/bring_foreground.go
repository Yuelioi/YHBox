package input

import (
	"fmt"

	"yhbox/internal/node"
)

func init() { node.Register(&BringGameForeground{}) }

// BringGameForeground 把游戏窗口置前台. 老 runtime 在节点外做 3×50ms 重试 +
// 失败仅日志不报错 (bring_foreground.go::execBringGameForegroundImpl). 新节点
// 把"是否重试 / 失败处理策略"下沉到 WindowService 适配层 — 节点本身一次调用,
// error 直接走 Done 路径前的报错 (跟其他 input 节点一致).
//
// Phase 5 wire 时 WindowService.BringForeground() 适配 GameProvider.BringToForeground
// + 内部 3×50ms 重试, 仍保持失败 → 仅日志不阻塞流程 (老语义).
type BringGameForeground struct{}

const (
	bgfInExec  = "In"
	bgfOutDone = "Done"
)

func (BringGameForeground) Spec() node.Spec {
	return node.Spec{
		Kind:        "BringGameForeground",
		Category:    "Input",
		DisplayName: "游戏窗口置前台",
		Description: "把游戏窗口设为前台焦点. 全屏独占 / 反作弊场景可能被 OS 拒绝, 失败时记日志继续.",
		Inputs: []node.InputSpec{
			{Name: bgfInExec, Type: "Exec"},
		},
		Outputs: []node.OutputSpec{
			{Name: bgfOutDone, Type: "Exec", DisplayName: "完成"},
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
