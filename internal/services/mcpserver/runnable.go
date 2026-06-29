package mcpserver

import "yotta/internal/node"

// isRunnable: run_node 只接「对当前自动化目标做一件事」的动作节点。
// 闸 = NeedsTarget/NeedsWindow 且非纯数据。
// 数据驱动 (读 Spec 能力位), 不写死 kind 名单 —— 节点增删自动跟随。
// target selection nodes 例外: 它们职责由 find_window/window 参数或显式容器图承担, 显式排除。
func isRunnable(spec node.Spec) bool {
	if spec.Kind == "Win32WindowTarget" || spec.Kind == "AndroidTarget" {
		return false
	}
	return (spec.NeedsTarget || spec.NeedsWindow) && !spec.IsPureData
}

// execInPin 返该节点的 exec 输入 pin 名 (Type==Exec 的首个输入); 无则 "".
func execInPin(spec node.Spec) string {
	for _, in := range spec.Inputs {
		if in.Type == node.TypeExec {
			return in.Name
		}
	}
	return ""
}
