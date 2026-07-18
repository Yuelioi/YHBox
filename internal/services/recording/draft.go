package recording

import (
	"errors"
	"fmt"
	"sort"
	"strings"

	"github.com/yottaapp/yotta/internal/blob"
	"github.com/yottaapp/yotta/internal/services/inputclip"
)

const (
	pressKeysNodeID     = "https://schemas.yotta.dev/nodes/automation/press-keys"
	clickPointerNodeID  = "https://schemas.yotta.dev/nodes/automation/click-pointer"
	scrollPointerNodeID = "https://schemas.yotta.dev/nodes/automation/scroll-pointer"
	playClipNodeID      = "https://schemas.yotta.dev/nodes/automation/play-input-clip"
	delayNodeID         = "https://schemas.yotta.dev/nodes/control/delay"
	maxPreviewSteps     = 32
	maxEditableActions  = 4096
)

type RecordingPreview struct {
	Mode          string                 `json:"mode"`
	DurationUs    uint64                 `json:"durationUs"`
	EventCount    int                    `json:"eventCount"`
	KeyActions    int                    `json:"keyActions"`
	ClickActions  int                    `json:"clickActions"`
	PointerMoves  int                    `json:"pointerMoves"`
	RawDeltas     int                    `json:"rawDeltas"`
	ScrollActions int                    `json:"scrollActions"`
	Steps         []RecordingPreviewStep `json:"steps"`
}

type RecordingPreviewStep struct {
	Kind       string   `json:"kind"`
	AtUs       uint64   `json:"atUs"`
	DurationUs uint64   `json:"durationUs"`
	Keys       []string `json:"keys,omitempty"`
	Button     string   `json:"button,omitempty"`
	Point      *Point   `json:"point,omitempty"`
	Notches    int32    `json:"notches,omitempty"`
	Samples    int      `json:"samples,omitempty"`
}

type RecordingAction struct {
	Kind       string   `json:"kind"`
	DelayUs    uint64   `json:"delayUs"`
	DurationUs uint64   `json:"durationUs"`
	Keys       []string `json:"keys,omitempty"`
	Button     string   `json:"button,omitempty"`
	Point      *Point   `json:"point,omitempty"`
	Notches    int32    `json:"notches,omitempty"`
}

type Point struct {
	X    float64 `json:"x"`
	Y    float64 `json:"y"`
	Unit string  `json:"unit"`
}

type WorkflowDraft struct {
	Mode  inputclip.RecordingMode `json:"mode"`
	Nodes []WorkflowDraftNode     `json:"nodes"`
}

type WorkflowDraftNode struct {
	NodeTypeID string                  `json:"nodeTypeID"`
	Config     map[string]any          `json:"config"`
	Values     map[string]any          `json:"values"`
	Blobs      map[string]blob.BlobRef `json:"blobs"`
	ExecInput  string                  `json:"execInput"`
	ExecOutput string                  `json:"execOutput"`
}

type recordedAction struct {
	kind    string
	startUs uint64
	endUs   uint64
	keys    []string
	button  string
	point   Point
	notches int32
}

func recordingPreview(result *StopResult) RecordingPreview {
	actions, counts := analyzeRecording(result)
	preview := RecordingPreview{
		Mode: string(result.Meta.RecordingMode), EventCount: len(result.Events), KeyActions: counts.keyActions,
		ClickActions: counts.clickActions, PointerMoves: counts.pointerMoves,
		RawDeltas: counts.rawDeltas, ScrollActions: counts.scrollActions,
		Steps: []RecordingPreviewStep{},
	}
	if len(result.Events) != 0 {
		preview.DurationUs = result.Events[len(result.Events)-1].TUs
	}
	steps := make([]RecordingPreviewStep, 0, len(actions)+1)
	for _, action := range actions {
		step := RecordingPreviewStep{
			Kind: action.kind, AtUs: action.startUs, DurationUs: action.endUs - action.startUs,
			Keys: append([]string(nil), action.keys...), Button: action.button, Notches: action.notches,
		}
		if action.kind == "click" || action.kind == "scroll" {
			point := action.point
			step.Point = &point
		}
		steps = append(steps, step)
	}
	if result.Meta.RecordingMode == inputclip.RecordingModePrecise && counts.pointerMoves+counts.rawDeltas > 0 {
		first, last, ok := movementRange(result.Events)
		if ok {
			steps = append(steps, RecordingPreviewStep{
				Kind: "move-path", AtUs: first, DurationUs: last - first,
				Samples: counts.pointerMoves + counts.rawDeltas,
			})
		}
	}
	sort.SliceStable(steps, func(i, j int) bool { return steps[i].AtUs < steps[j].AtUs })
	if len(steps) > maxPreviewSteps {
		steps = steps[:maxPreviewSteps]
	}
	preview.Steps = steps
	return preview
}

func editableRecordingActions(result *StopResult) []RecordingAction {
	actions, _ := analyzeRecording(result)
	if len(actions) > maxEditableActions {
		return nil
	}
	out := make([]RecordingAction, 0, len(actions))
	var previousEnd uint64
	for _, action := range actions {
		delay := uint64(0)
		if action.startUs > previousEnd {
			delay = action.startUs - previousEnd
		}
		item := RecordingAction{
			Kind: action.kind, DelayUs: delay, DurationUs: action.endUs - action.startUs,
			Keys: append([]string(nil), action.keys...), Button: action.button, Notches: action.notches,
		}
		if action.kind == "click" || action.kind == "scroll" {
			point := action.point
			item.Point = &point
		}
		out = append(out, item)
		previousEnd = action.endUs
	}
	return out
}

func applyEditedActions(result *StopResult, actions []RecordingAction) error {
	if result == nil || result.Meta.RecordingMode != inputclip.RecordingModeSimple {
		return errors.New("edited actions are only valid for simple recordings")
	}
	if len(actions) == 0 {
		return errors.New("recording must contain at least one action")
	}
	if len(actions) > maxEditableActions {
		return errors.New("recording exceeds the editable action budget")
	}
	events := make([]inputclip.Event, 0, len(actions)*4)
	var cursor uint64
	appendEvent := func(event inputclip.Event) {
		event.Seq = uint32(len(events))
		events = append(events, event)
	}
	for index, action := range actions {
		if index == 0 && action.DelayUs != 0 {
			return errors.New("first recording action must start without delay")
		}
		if action.DelayUs > inputclip.MaxInputClipDurationUs || cursor > inputclip.MaxInputClipDurationUs-action.DelayUs {
			return fmt.Errorf("recording action %d exceeds the duration budget", index)
		}
		cursor += action.DelayUs
		if action.DurationUs > inputclip.MaxInputClipDurationUs || cursor > inputclip.MaxInputClipDurationUs-action.DurationUs {
			return fmt.Errorf("recording action %d exceeds the duration budget", index)
		}
		end := cursor + action.DurationUs
		switch action.Kind {
		case "keys":
			if len(action.Keys) == 0 || len(action.Keys) > 16 {
				return fmt.Errorf("recording action %d has invalid keys", index)
			}
			vks := make([]uint32, 0, len(action.Keys))
			seen := make(map[uint32]struct{}, len(action.Keys))
			for _, name := range action.Keys {
				vk, ok := workflowKeyCode(name)
				if !ok {
					return fmt.Errorf("recording action %d has unsupported key %q", index, name)
				}
				if _, duplicate := seen[vk]; duplicate {
					continue
				}
				seen[vk] = struct{}{}
				vks = append(vks, vk)
				appendEvent(inputclip.Event{TUs: cursor, Type: inputclip.EventTypeKeyDown, A: int32(vk)})
			}
			for keyIndex := len(vks) - 1; keyIndex >= 0; keyIndex-- {
				appendEvent(inputclip.Event{TUs: end, Type: inputclip.EventTypeKeyUp, A: int32(vks[keyIndex])})
			}
		case "click":
			button, ok := hookButton(action.Button)
			if !ok || action.Point == nil {
				return fmt.Errorf("recording action %d has invalid click data", index)
			}
			x, y, err := absolutePoint(*action.Point, result.Meta.BaseResolution)
			if err != nil {
				return fmt.Errorf("recording action %d: %w", index, err)
			}
			appendEvent(inputclip.Event{TUs: cursor, Type: inputclip.EventTypeMouseBtnDown, A: int32(button), B: x, C: y})
			appendEvent(inputclip.Event{TUs: end, Type: inputclip.EventTypeMouseBtnUp, A: int32(button), B: x, C: y})
		case "scroll":
			if action.Notches == 0 || action.Point == nil || action.DurationUs != 0 {
				return fmt.Errorf("recording action %d has invalid scroll data", index)
			}
			x, y, err := absolutePoint(*action.Point, result.Meta.BaseResolution)
			if err != nil {
				return fmt.Errorf("recording action %d: %w", index, err)
			}
			appendEvent(inputclip.Event{TUs: cursor, Type: inputclip.EventTypeScroll, A: action.Notches, B: x, C: y})
		default:
			return fmt.Errorf("recording action %d has unsupported kind %q", index, action.Kind)
		}
		cursor = end
	}
	if len(events) > inputclip.MaxInputClipEvents {
		return errors.New("recording exceeds the event budget")
	}
	result.Events = events
	return canonicalizeStopResult(result)
}

func buildWorkflowDraft(result *StopResult, targetSlot string, clip blob.BlobRef) WorkflowDraft {
	actions, _ := analyzeRecording(result)
	if result.Meta.RecordingMode == inputclip.RecordingModePrecise {
		return WorkflowDraft{Mode: inputclip.RecordingModePrecise, Nodes: []WorkflowDraftNode{{
			NodeTypeID: playClipNodeID,
			Config:     map[string]any{"slot": targetSlot},
			Values:     map[string]any{},
			Blobs:      map[string]blob.BlobRef{"clip": clip},
			ExecInput:  "in", ExecOutput: "completed",
		}}}
	}
	nodes := make([]WorkflowDraftNode, 0, len(actions)*2)
	var previousEnd uint64
	for index, action := range actions {
		if index != 0 && action.startUs > previousEnd {
			nodes = append(nodes, WorkflowDraftNode{
				NodeTypeID: delayNodeID, Config: map[string]any{},
				Values: map[string]any{"duration-milliseconds": microsecondsToMilliseconds(action.startUs - previousEnd)},
				Blobs:  map[string]blob.BlobRef{}, ExecInput: "in", ExecOutput: "done",
			})
		}
		hold := microsecondsToMilliseconds(action.endUs - action.startUs)
		switch action.kind {
		case "keys":
			nodes = append(nodes, WorkflowDraftNode{
				NodeTypeID: pressKeysNodeID, Config: map[string]any{"slot": targetSlot},
				Values: map[string]any{"keys": append([]string(nil), action.keys...), "hold-duration": hold},
				Blobs:  map[string]blob.BlobRef{}, ExecInput: "in", ExecOutput: "completed",
			})
		case "click":
			nodes = append(nodes, WorkflowDraftNode{
				NodeTypeID: clickPointerNodeID, Config: map[string]any{"slot": targetSlot},
				Values: map[string]any{"point": action.point, "button": action.button, "hold-duration": hold},
				Blobs:  map[string]blob.BlobRef{}, ExecInput: "in", ExecOutput: "completed",
			})
		case "scroll":
			nodes = append(nodes, WorkflowDraftNode{
				NodeTypeID: scrollPointerNodeID, Config: map[string]any{"slot": targetSlot},
				Values: map[string]any{"point": action.point, "notches": int64(action.notches), "horizontal": false},
				Blobs:  map[string]blob.BlobRef{}, ExecInput: "in", ExecOutput: "completed",
			})
		}
		previousEnd = action.endUs
	}
	return WorkflowDraft{Mode: inputclip.RecordingModeSimple, Nodes: nodes}
}

type recordingCounts struct {
	keyActions, clickActions, pointerMoves, rawDeltas, scrollActions int
}

func analyzeRecording(result *StopResult) ([]recordedAction, recordingCounts) {
	counts := recordingCounts{}
	actions := make([]recordedAction, 0)
	activeKeys := map[int32]struct{}{}
	keyOrder := make([]string, 0)
	var keyStart uint64
	activeButtons := map[int32]inputclip.Event{}
	for _, event := range result.Events {
		switch event.Type {
		case inputclip.EventTypeKeyDown:
			name := workflowKeyName(uint32(event.A))
			if name == "" {
				continue
			}
			if len(activeKeys) == 0 {
				keyStart = event.TUs
				keyOrder = keyOrder[:0]
			}
			if _, exists := activeKeys[event.A]; !exists {
				activeKeys[event.A] = struct{}{}
				keyOrder = append(keyOrder, name)
			}
		case inputclip.EventTypeKeyUp:
			if _, exists := activeKeys[event.A]; !exists {
				continue
			}
			delete(activeKeys, event.A)
			if len(activeKeys) == 0 {
				actions = append(actions, recordedAction{kind: "keys", startUs: keyStart, endUs: event.TUs, keys: append([]string(nil), keyOrder...)})
				counts.keyActions++
			}
		case inputclip.EventTypeMouseBtnDown:
			activeButtons[event.A] = event
		case inputclip.EventTypeMouseBtnUp:
			down, exists := activeButtons[event.A]
			button := pointerButton(event.A)
			if !exists || button == "" {
				continue
			}
			delete(activeButtons, event.A)
			actions = append(actions, recordedAction{
				kind: "click", startUs: down.TUs, endUs: event.TUs, button: button,
				point: ratioPoint(down.B, down.C, result.Meta.BaseResolution),
			})
			counts.clickActions++
		case inputclip.EventTypeMouseMove:
			counts.pointerMoves++
		case inputclip.EventTypeRawDelta:
			counts.rawDeltas++
		case inputclip.EventTypeScroll:
			counts.scrollActions++
			actions = append(actions, recordedAction{
				kind: "scroll", startUs: event.TUs, endUs: event.TUs, notches: event.A,
				point: ratioPoint(event.B, event.C, result.Meta.BaseResolution),
			})
		}
	}
	sort.SliceStable(actions, func(i, j int) bool { return actions[i].startUs < actions[j].startUs })
	return actions, counts
}

func movementRange(events []inputclip.Event) (uint64, uint64, bool) {
	var first, last uint64
	found := false
	for _, event := range events {
		if event.Type != inputclip.EventTypeMouseMove && event.Type != inputclip.EventTypeRawDelta {
			continue
		}
		if !found {
			first = event.TUs
			found = true
		}
		last = event.TUs
	}
	return first, last, found
}

func workflowKeyCode(name string) (uint32, bool) {
	upper := strings.ToUpper(strings.TrimSpace(name))
	if len(upper) == 1 {
		value := upper[0]
		if (value >= 'A' && value <= 'Z') || (value >= '0' && value <= '9') {
			return uint32(value), true
		}
	}
	if strings.HasPrefix(upper, "F") {
		var number int
		if _, err := fmt.Sscanf(upper, "F%d", &number); err == nil && number >= 1 && number <= 12 {
			return 0x70 + uint32(number-1), true
		}
	}
	keys := map[string]uint32{
		"SPACE": VK_SPACE, "ESC": VK_ESCAPE, "ENTER": VK_RETURN, "TAB": VK_TAB,
		"BACKSPACE": VK_BACK, "DELETE": VK_DELETE, "INSERT": VK_INSERT, "HOME": VK_HOME,
		"END": VK_END, "PGUP": VK_PRIOR, "PGDN": VK_NEXT, "UP": VK_UP, "DOWN": VK_DOWN,
		"LEFT": VK_LEFT, "RIGHT": VK_RIGHT, "CTRL": VK_CONTROL, "SHIFT": VK_SHIFT,
		"ALT": VK_MENU, "CAPSLOCK": VK_CAPITAL, ",": 0xBC, ".": 0xBE,
	}
	vk, ok := keys[upper]
	return vk, ok
}

func hookButton(name string) (HookMouseBtn, bool) {
	switch strings.ToLower(strings.TrimSpace(name)) {
	case "left":
		return HookBtnLeft, true
	case "middle":
		return HookBtnMiddle, true
	case "right":
		return HookBtnRight, true
	default:
		return 0, false
	}
}

func absolutePoint(point Point, resolution [2]int) (int32, int32, error) {
	if point.Unit != "ratio" || point.X < 0 || point.X > 1 || point.Y < 0 || point.Y > 1 || resolution[0] <= 0 || resolution[1] <= 0 {
		return 0, 0, errors.New("point must be a ratio inside the recording resolution")
	}
	x := int32(point.X * float64(resolution[0]))
	y := int32(point.Y * float64(resolution[1]))
	if x == int32(resolution[0]) {
		x--
	}
	if y == int32(resolution[1]) {
		y--
	}
	return x, y, nil
}

func workflowKeyName(vk uint32) string {
	switch vk {
	case 0xBC:
		return ","
	case 0xBE:
		return "."
	case VK_CAPITAL:
		return "CAPSLOCK"
	}
	return strings.ToUpper(vkName(vk))
}

func pointerButton(button int32) string {
	switch button {
	case int32(HookBtnLeft):
		return "left"
	case int32(HookBtnRight):
		return "right"
	case int32(HookBtnMiddle):
		return "middle"
	default:
		return ""
	}
}

func ratioPoint(x, y int32, resolution [2]int) Point {
	if resolution[0] <= 0 || resolution[1] <= 0 {
		return Point{X: 0.5, Y: 0.5, Unit: "ratio"}
	}
	return Point{
		X:    clampRatio(float64(x) / float64(resolution[0])),
		Y:    clampRatio(float64(y) / float64(resolution[1])),
		Unit: "ratio",
	}
}

func clampRatio(value float64) float64 {
	if value < 0 {
		return 0
	}
	if value > 1 {
		return 1
	}
	return value
}

func microsecondsToMilliseconds(value uint64) int64 {
	return int64((value + 500) / 1000)
}
