package macro

import (
	"errors"
	"fmt"
	"regexp"
	"sort"
	"strings"

	"github.com/yottaapp/yotta/internal/blob"
	"github.com/yottaapp/yotta/internal/services/inputclip"
)

const (
	SchemaVersion      = 1
	MaxActions         = 4096
	MaxDurationUs      = inputclip.MaxInputClipDurationUs
	MaxClickDurationUs = uint64(5_000_000)
)

type ActionKind string

const (
	ActionKeyDown   ActionKind = "key-down"
	ActionKeyUp     ActionKind = "key-up"
	ActionMouseDown ActionKind = "mouse-down"
	ActionMouseUp   ActionKind = "mouse-up"
	ActionClick     ActionKind = "click"
	ActionScroll    ActionKind = "scroll"
	ActionSleep     ActionKind = "sleep"
)

type Point struct {
	X    float64 `json:"x"`
	Y    float64 `json:"y"`
	Unit string  `json:"unit"`
}

type Action struct {
	ID         string     `json:"id"`
	Kind       ActionKind `json:"kind"`
	Key        string     `json:"key,omitempty"`
	Button     string     `json:"button,omitempty"`
	Point      *Point     `json:"point,omitempty"`
	Notches    int32      `json:"notches,omitempty"`
	DurationUs uint64     `json:"durationUs,omitempty"`
}

type Document struct {
	SchemaVersion  int      `json:"schemaVersion"`
	BaseResolution [2]int   `json:"baseResolution"`
	Actions        []Action `json:"actions"`
}

type Macro struct {
	ID          string       `json:"id"`
	Label       string       `json:"label"`
	Description string       `json:"description,omitempty"`
	Category    string       `json:"category,omitempty"`
	Tags        []string     `json:"tags,omitempty"`
	CreatedAt   string       `json:"createdAt"`
	Document    Document     `json:"document"`
	Blob        blob.BlobRef `json:"blob"`
}

type Summary struct {
	ID          string       `json:"id"`
	Label       string       `json:"label"`
	Description string       `json:"description,omitempty"`
	Category    string       `json:"category,omitempty"`
	Tags        []string     `json:"tags,omitempty"`
	CreatedAt   string       `json:"createdAt"`
	ActionCount int          `json:"actionCount"`
	DurationUs  uint64       `json:"durationUs"`
	Blob        blob.BlobRef `json:"blob"`
}

type Issue struct {
	Index   int    `json:"index"`
	Code    string `json:"code"`
	Message string `json:"message"`
}

type HeldState struct {
	Keys    []string `json:"keys"`
	Buttons []string `json:"buttons"`
}

type Analysis struct {
	Issues     []Issue     `json:"issues"`
	HeldAfter  []HeldState `json:"heldAfter"`
	DurationUs uint64      `json:"durationUs"`
}

var actionIDPattern = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9._-]{0,127}$`)

func Analyze(document Document) Analysis {
	analysis := Analysis{Issues: []Issue{}, HeldAfter: make([]HeldState, 0, len(document.Actions))}
	if document.SchemaVersion != SchemaVersion {
		analysis.Issues = append(analysis.Issues, Issue{Index: -1, Code: "macro.schema-version", Message: fmt.Sprintf("schemaVersion must be %d", SchemaVersion)})
	}
	if len(document.Actions) > MaxActions {
		analysis.Issues = append(analysis.Issues, Issue{Index: -1, Code: "macro.action-budget", Message: fmt.Sprintf("macro exceeds %d actions", MaxActions)})
	}
	if !validResolution(document.BaseResolution) {
		analysis.Issues = append(analysis.Issues, Issue{Index: -1, Code: "macro.base-resolution", Message: "baseResolution must contain two positive dimensions"})
	}
	ids := make(map[string]struct{}, len(document.Actions))
	heldKeys := map[string]struct{}{}
	heldButtons := map[string]struct{}{}
	for index, action := range document.Actions {
		if !actionIDPattern.MatchString(action.ID) {
			analysis.Issues = append(analysis.Issues, Issue{Index: index, Code: "macro.action-id", Message: "action id is invalid"})
		} else if _, duplicate := ids[action.ID]; duplicate {
			analysis.Issues = append(analysis.Issues, Issue{Index: index, Code: "macro.action-id-duplicate", Message: "action id is duplicated"})
		}
		ids[action.ID] = struct{}{}
		analysis.Issues = append(analysis.Issues, validateAction(index, action)...)
		switch action.Kind {
		case ActionKeyDown:
			key, _, ok := CanonicalKey(action.Key)
			if ok {
				if _, active := heldKeys[key]; active {
					analysis.Issues = append(analysis.Issues, Issue{Index: index, Code: "macro.key-already-down", Message: fmt.Sprintf("%s is already held", key)})
				} else {
					heldKeys[key] = struct{}{}
				}
			}
		case ActionKeyUp:
			key, _, ok := CanonicalKey(action.Key)
			if ok {
				if _, active := heldKeys[key]; !active {
					analysis.Issues = append(analysis.Issues, Issue{Index: index, Code: "macro.key-not-down", Message: fmt.Sprintf("%s is not held", key)})
				} else {
					delete(heldKeys, key)
				}
			}
		case ActionMouseDown:
			if validButton(action.Button) {
				if _, active := heldButtons[action.Button]; active {
					analysis.Issues = append(analysis.Issues, Issue{Index: index, Code: "macro.button-already-down", Message: fmt.Sprintf("%s mouse button is already held", action.Button)})
				} else {
					heldButtons[action.Button] = struct{}{}
				}
			}
		case ActionMouseUp:
			if validButton(action.Button) {
				if _, active := heldButtons[action.Button]; !active {
					analysis.Issues = append(analysis.Issues, Issue{Index: index, Code: "macro.button-not-down", Message: fmt.Sprintf("%s mouse button is not held", action.Button)})
				} else {
					delete(heldButtons, action.Button)
				}
			}
		case ActionClick:
			if _, active := heldButtons[action.Button]; active {
				analysis.Issues = append(analysis.Issues, Issue{Index: index, Code: "macro.click-button-held", Message: "click cannot use a mouse button that is already held"})
			}
		}
		if action.Kind == ActionSleep || action.Kind == ActionClick {
			if action.DurationUs > MaxDurationUs || analysis.DurationUs > MaxDurationUs-action.DurationUs {
				analysis.Issues = append(analysis.Issues, Issue{Index: index, Code: "macro.duration-budget", Message: "macro exceeds the duration budget"})
			} else {
				analysis.DurationUs += action.DurationUs
			}
		}
		analysis.HeldAfter = append(analysis.HeldAfter, snapshotHeld(heldKeys, heldButtons))
	}
	if len(heldKeys) != 0 || len(heldButtons) != 0 {
		analysis.Issues = append(analysis.Issues, Issue{Index: len(document.Actions) - 1, Code: "macro.held-at-end", Message: "macro ends with held keyboard or mouse input"})
	}
	return analysis
}

func Validate(document Document) error {
	analysis := Analyze(document)
	if len(analysis.Issues) == 0 {
		return nil
	}
	issue := analysis.Issues[0]
	if issue.Index >= 0 {
		return fmt.Errorf("macro action %d: %s", issue.Index, issue.Message)
	}
	return errors.New(issue.Message)
}

func validateAction(index int, action Action) []Issue {
	issues := []Issue{}
	add := func(code, message string) { issues = append(issues, Issue{Index: index, Code: code, Message: message}) }
	pointRequired := func() {
		if action.Point == nil || action.Point.Unit != "ratio" || action.Point.X < 0 || action.Point.X > 1 || action.Point.Y < 0 || action.Point.Y > 1 {
			add("macro.point", "point must be a ratio inside the target")
		}
	}
	switch action.Kind {
	case ActionKeyDown, ActionKeyUp:
		if _, _, ok := CanonicalKey(action.Key); !ok {
			add("macro.key", fmt.Sprintf("unsupported key %q", action.Key))
		}
		if action.Button != "" || action.Point != nil || action.Notches != 0 || action.DurationUs != 0 {
			add("macro.action-shape", "key action contains unrelated fields")
		}
	case ActionMouseDown, ActionMouseUp:
		if !validButton(action.Button) {
			add("macro.button", "mouse button is invalid")
		}
		pointRequired()
		if action.Key != "" || action.Notches != 0 || action.DurationUs != 0 {
			add("macro.action-shape", "mouse action contains unrelated fields")
		}
	case ActionClick:
		if !validButton(action.Button) {
			add("macro.button", "click button is invalid")
		}
		pointRequired()
		if action.DurationUs == 0 || action.DurationUs > MaxClickDurationUs {
			add("macro.click-duration", "click duration must be between 1 microsecond and 5 seconds")
		}
		if action.Key != "" || action.Notches != 0 {
			add("macro.action-shape", "click contains unrelated fields")
		}
	case ActionScroll:
		pointRequired()
		if action.Notches == 0 || action.Notches < -120 || action.Notches > 120 {
			add("macro.scroll", "scroll notches must be between -120 and 120 and cannot be zero")
		}
		if action.Key != "" || action.Button != "" || action.DurationUs != 0 {
			add("macro.action-shape", "scroll contains unrelated fields")
		}
	case ActionSleep:
		if action.DurationUs == 0 || action.DurationUs > MaxDurationUs {
			add("macro.sleep-duration", "sleep duration must be positive and inside the duration budget")
		}
		if action.Key != "" || action.Button != "" || action.Point != nil || action.Notches != 0 {
			add("macro.action-shape", "sleep contains unrelated fields")
		}
	default:
		add("macro.action-kind", fmt.Sprintf("unsupported action kind %q", action.Kind))
	}
	return issues
}

func snapshotHeld(keys, buttons map[string]struct{}) HeldState {
	state := HeldState{Keys: make([]string, 0, len(keys)), Buttons: make([]string, 0, len(buttons))}
	for key := range keys {
		state.Keys = append(state.Keys, key)
	}
	for button := range buttons {
		state.Buttons = append(state.Buttons, button)
	}
	sort.Strings(state.Keys)
	sort.Strings(state.Buttons)
	return state
}

func validResolution(resolution [2]int) bool {
	return resolution[0] > 0 && resolution[0] <= 100_000 && resolution[1] > 0 && resolution[1] <= 100_000
}

func validButton(button string) bool {
	return button == "left" || button == "middle" || button == "right"
}

var namedKeys = map[string]uint32{
	"BACKSPACE": 0x08, "TAB": 0x09, "ENTER": 0x0D, "SHIFT": 0x10, "CTRL": 0x11, "ALT": 0x12,
	"CAPSLOCK": 0x14, "ESC": 0x1B, "SPACE": 0x20, "PGUP": 0x21, "PGDN": 0x22, "END": 0x23,
	"HOME": 0x24, "LEFT": 0x25, "UP": 0x26, "RIGHT": 0x27, "DOWN": 0x28, "INSERT": 0x2D,
	"DELETE": 0x2E, ",": 0xBC, ".": 0xBE,
}

func CanonicalKey(value string) (string, uint32, bool) {
	key := strings.ToUpper(strings.TrimSpace(value))
	if len(key) == 1 && ((key[0] >= 'A' && key[0] <= 'Z') || (key[0] >= '0' && key[0] <= '9')) {
		return key, uint32(key[0]), true
	}
	if strings.HasPrefix(key, "F") {
		var number int
		if _, err := fmt.Sscanf(key, "F%d", &number); err == nil && number >= 1 && number <= 24 {
			return fmt.Sprintf("F%d", number), 0x70 + uint32(number-1), true
		}
	}
	code, ok := namedKeys[key]
	return key, code, ok
}

func KeyName(code uint32) string {
	if (code >= 'A' && code <= 'Z') || (code >= '0' && code <= '9') {
		return string(rune(code))
	}
	if code >= 0x70 && code <= 0x87 {
		return fmt.Sprintf("F%d", code-0x70+1)
	}
	for name, candidate := range namedKeys {
		if candidate == code {
			return name
		}
	}
	return ""
}

func FromInputEvents(events []inputclip.Event, resolution [2]int) (Document, error) {
	document := Document{SchemaVersion: SchemaVersion, BaseResolution: resolution, Actions: []Action{}}
	if !validResolution(resolution) {
		return Document{}, errors.New("recording resolution is invalid")
	}
	var previousUs uint64
	for index, event := range events {
		if index != 0 && event.TUs > previousUs {
			document.Actions = append(document.Actions, Action{ID: actionID(len(document.Actions)), Kind: ActionSleep, DurationUs: event.TUs - previousUs})
		}
		action := Action{ID: actionID(len(document.Actions))}
		switch event.Type {
		case inputclip.EventTypeKeyDown, inputclip.EventTypeKeyUp:
			action.Key = KeyName(uint32(event.A))
			if action.Key == "" {
				return Document{}, fmt.Errorf("recording event %d uses unsupported key code %d", index, event.A)
			}
			if event.Type == inputclip.EventTypeKeyDown {
				action.Kind = ActionKeyDown
			} else {
				action.Kind = ActionKeyUp
			}
		case inputclip.EventTypeMouseBtnDown, inputclip.EventTypeMouseBtnUp:
			action.Button = buttonName(event.A)
			if action.Button == "" {
				return Document{}, fmt.Errorf("recording event %d uses unsupported mouse button %d", index, event.A)
			}
			action.Point = ratioPoint(event.B, event.C, resolution)
			if event.Type == inputclip.EventTypeMouseBtnDown {
				action.Kind = ActionMouseDown
			} else {
				action.Kind = ActionMouseUp
			}
		case inputclip.EventTypeScroll:
			action.Kind = ActionScroll
			action.Notches = event.A
			action.Point = ratioPoint(event.B, event.C, resolution)
		default:
			return Document{}, fmt.Errorf("recording event %d is not a macro event", index)
		}
		document.Actions = append(document.Actions, action)
		previousUs = event.TUs
	}
	if err := Validate(document); err != nil {
		return Document{}, err
	}
	return document, nil
}

func actionID(index int) string {
	return fmt.Sprintf("action-%06d", index+1)
}

func buttonName(code int32) string {
	switch code {
	case 0:
		return "left"
	case 1:
		return "middle"
	case 2:
		return "right"
	default:
		return ""
	}
}

func ratioPoint(x, y int32, resolution [2]int) *Point {
	return &Point{X: float64(x) / float64(resolution[0]), Y: float64(y) / float64(resolution[1]), Unit: "ratio"}
}
