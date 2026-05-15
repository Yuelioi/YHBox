package actions

import (
	"errors"
	"fmt"

	"github.com/google/uuid"
)

// NormalizeAction 补默认值 + 填空 step id。所有 mutation 入口（Create/Update/录制完
// 落库）都必须先过这里再 Validate。
func NormalizeAction(a *Action) {
	if a.SchemaVersion == 0 {
		a.SchemaVersion = CurrentSchemaVersion
	}
	for i := range a.Steps {
		if a.Steps[i].ID == "" {
			a.Steps[i].ID = uuid.NewString()
		}
	}
}

// Validate 须在 NormalizeAction 之后调用。
func (a *Action) Validate() error {
	for i, s := range a.Steps {
		if err := validateStep(&s); err != nil {
			return fmt.Errorf("step[%d]: %w", i, err)
		}
	}
	return nil
}

func validateStep(s *Step) error {
	switch s.Kind {
	case StepClickLeft, StepClickMiddle, StepClickRight:
		if s.XRatio < 0 || s.XRatio > 1 || s.YRatio < 0 || s.YRatio > 1 {
			return fmt.Errorf("click ratio 必须 [0,1]，got (%f, %f)", s.XRatio, s.YRatio)
		}
		if s.DurationMs < 0 {
			return errors.New("click durationMs >= 0")
		}
	case StepKey, StepKeyDown, StepKeyUp:
		if s.Vk == "" {
			return errors.New("key step 缺 vk")
		}
		if s.DurationMs < 0 {
			return errors.New("key durationMs >= 0")
		}
	case StepSleep:
		if s.DurationMs < 0 {
			return errors.New("sleep durationMs >= 0")
		}
	case StepMouseMove:
		if s.XRatio < 0 || s.XRatio > 1 || s.YRatio < 0 || s.YRatio > 1 {
			return fmt.Errorf("mouse_move ratio 必须 [0,1]，got (%f, %f)", s.XRatio, s.YRatio)
		}
	case StepMouseDrag:
		if s.XRatio < 0 || s.XRatio > 1 || s.YRatio < 0 || s.YRatio > 1 {
			return fmt.Errorf("mouse_drag 起点 ratio 必须 [0,1]，got (%f, %f)", s.XRatio, s.YRatio)
		}
		if s.X2Ratio < 0 || s.X2Ratio > 1 || s.Y2Ratio < 0 || s.Y2Ratio > 1 {
			return fmt.Errorf("mouse_drag 终点 ratio 必须 [0,1]，got (%f, %f)", s.X2Ratio, s.Y2Ratio)
		}
		if s.DurationMs < 0 {
			return errors.New("mouse_drag durationMs >= 0")
		}
		switch s.MouseButton {
		case "", "left", "middle", "right":
		default:
			return fmt.Errorf("mouse_drag mouseButton 必须空/left/middle/right，got %q", s.MouseButton)
		}
	case StepScroll:
		if s.ScrollDelta == 0 {
			return errors.New("scroll delta 不能为 0")
		}
	case StepMouseMoveRel:
		if s.Dx == 0 && s.Dy == 0 {
			return errors.New("mouse_move_rel 至少一个轴位移非 0")
		}
		if s.DurationMs < 0 {
			return errors.New("mouse_move_rel durationMs >= 0")
		}
	default:
		return fmt.Errorf("unknown step kind: %q", s.Kind)
	}
	return nil
}
