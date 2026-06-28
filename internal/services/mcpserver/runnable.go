package mcpserver

import "yotta/internal/node"

// isRunnable: run_node 只接「对窗口做一件事」的动作节点。闸 = NeedsWindow 且非纯数据。
// 数据驱动 (读 Spec 能力位), 不写死 kind 名单 —— 节点增删自动跟随。
// Win32WindowTarget 例外: 它职责被 find_window 取代, 显式排除。
func isRunnable(spec node.Spec) bool {
	if spec.Kind == "Win32WindowTarget" {
		return false
	}
	return spec.NeedsWindow && !spec.IsPureData
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
