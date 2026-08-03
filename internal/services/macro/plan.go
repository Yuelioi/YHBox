package macro

import (
	"fmt"
	"math"
)

const autoMoveSkipDistancePixels = 5.0

type plannedAction struct {
	Action      Action
	SourceIndex int
}

// ExecutionPlan expands document-level playback policy into explicit semantic
// actions without mutating the persisted macro. Adapters receive only the
// resulting actions and do not need to understand macro policy.
func ExecutionPlan(document Document) ([]Action, error) {
	if err := Validate(document); err != nil {
		return nil, err
	}
	planned := expandActions(document)
	actions := make([]Action, 0, len(planned))
	for _, item := range planned {
		actions = append(actions, cloneAction(item.Action))
	}
	return actions, nil
}

func CloneDocument(document Document) Document {
	document.Actions = append([]Action(nil), document.Actions...)
	for index := range document.Actions {
		document.Actions[index] = cloneAction(document.Actions[index])
	}
	return document
}

func expandActions(document Document) []plannedAction {
	result := make([]plannedAction, 0, len(document.Actions)*2)
	var cursor *Point
	for index, action := range document.Actions {
		if action.Kind == ActionClick && shouldAutoMove(document, cursor, action.Point) {
			result = append(result, plannedAction{Action: Action{
				ID:         fmt.Sprintf("auto-move-%s", action.ID),
				Kind:       ActionMove,
				Point:      clonePoint(action.Point),
				DurationUs: uint64(document.Meta.AutoMove.DurationMilliseconds) * 1000,
				Motion:     document.Meta.AutoMove.Mode,
			}, SourceIndex: index})
		}
		result = append(result, plannedAction{Action: cloneAction(action), SourceIndex: index})
		if point := pointerResult(action); point != nil {
			cursor = clonePoint(point)
		}
	}
	return result
}

func shouldAutoMove(document Document, cursor, target *Point) bool {
	if !document.Meta.AutoMove.Enabled || !validAutoMove(document.Meta.AutoMove) || target == nil {
		return false
	}
	if cursor == nil {
		return true
	}
	return pointDistancePixels(cursor, target, document.BaseResolution) >= autoMoveSkipDistancePixels
}

func pointerResult(action Action) *Point {
	switch action.Kind {
	case ActionMove, ActionClick, ActionMouseDown, ActionMouseUp, ActionScroll:
		return action.Point
	case ActionDrag:
		return action.Point
	default:
		return nil
	}
}

func pointDistancePixels(left, right *Point, resolution [2]int) float64 {
	if left == nil || right == nil || left.Unit != "ratio" || right.Unit != "ratio" {
		return math.Inf(1)
	}
	dx := (left.X - right.X) * float64(resolution[0])
	dy := (left.Y - right.Y) * float64(resolution[1])
	return math.Hypot(dx, dy)
}

func cloneAction(action Action) Action {
	action.From = clonePoint(action.From)
	action.Point = clonePoint(action.Point)
	return action
}

func clonePoint(point *Point) *Point {
	if point == nil {
		return nil
	}
	clone := *point
	return &clone
}
