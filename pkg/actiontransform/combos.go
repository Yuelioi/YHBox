package actiontransform

import "yhbox/internal/services/actions"

// CollapseCombos v1 no-op。
// v2 识别 "key_down(Ctrl) + key(F) + key_up(Ctrl)" 类组合合并成 step_key_combo。
func CollapseCombos(steps []actions.Step) []actions.Step {
	return steps
}
