package recording

import (
	"sort"
	"strings"

	"github.com/yottaapp/yotta/internal/blob"
	"github.com/yottaapp/yotta/internal/services/inputclip"
)

const (
	pressKeysNodeID    = "https://schemas.yotta.dev/nodes/automation/press-keys"
	clickPointerNodeID = "https://schemas.yotta.dev/nodes/automation/click-pointer"
	playClipNodeID     = "https://schemas.yotta.dev/nodes/automation/play-input-clip"
	delayNodeID        = "https://schemas.yotta.dev/nodes/control/delay"
	maxPreviewSteps    = 32
	minimumDelayUs     = 50_000
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
	for _, action := range actions {
		if len(preview.Steps) == maxPreviewSteps {
			break
		}
		step := RecordingPreviewStep{
			Kind: action.kind, AtUs: action.startUs, DurationUs: action.endUs - action.startUs,
			Keys: append([]string(nil), action.keys...), Button: action.button,
		}
		if action.kind == "click" {
			point := action.point
			step.Point = &point
		}
		preview.Steps = append(preview.Steps, step)
	}
	return preview
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
		if index != 0 && action.startUs > previousEnd && action.startUs-previousEnd >= minimumDelayUs {
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
		}
	}
	sort.SliceStable(actions, func(i, j int) bool { return actions[i].startUs < actions[j].startUs })
	return actions, counts
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
