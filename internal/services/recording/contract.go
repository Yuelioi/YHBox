package recording

import (
	"errors"

	"github.com/yottaapp/yotta/internal/services/inputclip"
)

// HookMouseBtn identifies a mouse button in the native recording stream.
type HookMouseBtn int

const (
	HookBtnLeft   HookMouseBtn = 0
	HookBtnMiddle HookMouseBtn = 1
	HookBtnRight  HookMouseBtn = 2
)

// HookEvent is the platform-neutral event stream consumed by the recorder.
type HookEvent struct {
	TimestampMs  uint32
	IsKeyboard   bool
	Vk           uint32
	IsKeyDown    bool
	ScreenX      int32
	ScreenY      int32
	MouseBtn     HookMouseBtn
	IsMouseDown  bool
	IsMouseMove  bool
	IsScroll     bool
	WheelNotches int
	IsRawDelta   bool
	RawDx        int
	RawDy        int
}

// StopResult is the recorded event stream and its environment snapshot.
type StopResult struct {
	Events  []inputclip.Event
	Meta    inputclip.ClipMeta
	ClientW int
	ClientH int
	TempID  string
}

// ErrRecorderNotActive classifies a stop request without an active recording.
var ErrRecorderNotActive = errors.New("recorder not active")
