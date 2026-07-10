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

// HookEvent is the platform-neutral tagged event stream consumed by Recorder.
// The discriminator fields define which payload fields are valid; consumers
// must not infer an event kind from zero-valued payload fields.
type HookEvent struct {
	// TimestampMs is the host event timestamp in milliseconds. Windows supplies
	// the KBDLLHOOKSTRUCT/MSLLHOOKSTRUCT tick so channel scheduling jitter does
	// not change the recorded timing.
	TimestampMs uint32

	// IsKeyboard selects Vk and IsKeyDown. When false, exactly one of the mouse
	// button, move, scroll, or raw-delta payloads is selected below.
	IsKeyboard bool
	Vk         uint32
	IsKeyDown  bool

	// ScreenX and ScreenY are absolute screen-space coordinates populated for
	// host mouse events. Recorder performs window filtering and client conversion.
	ScreenX int32
	ScreenY int32

	// MouseBtn and IsMouseDown are valid for a mouse button event, when all
	// explicit mouse-kind discriminators below are false.
	MouseBtn    HookMouseBtn
	IsMouseDown bool

	// IsMouseMove selects a pointer-move event used to retain drag trajectories.
	IsMouseMove bool

	// IsScroll selects WheelNotches; positive means wheel-up and negative means
	// wheel-down.
	IsScroll     bool
	WheelNotches int

	// IsRawDelta selects RawDx and RawDy. Their unit is device mickeys, matching
	// relative SendInput movement so playback can preserve camera displacement.
	IsRawDelta bool
	RawDx      int
	RawDy      int
}

// StopResult is the recorded event stream and its environment snapshot.
// The service persists Events and Meta as an InputClip in precise mode, while
// simple mode converts Events using ClientW and ClientH into a subgraph.
type StopResult struct {
	Events  []inputclip.Event
	Meta    inputclip.ClipMeta
	ClientW int
	ClientH int
	// TempID is the stable session identifier used to derive the persistent
	// clip-<id> or sg-<id> identifier promised to event subscribers.
	TempID string
}

// recorderLifecycle pins the service-facing Recorder surface across platform
// build variants. Platform adapters may implement unsupported operations as
// typed errors or idempotent no-ops, but they must preserve this lifecycle.
type recorderLifecycle interface {
	SetMouseCounts360Getter(func() int)
	Active() bool
	Pause()
	Resume()
	Start(uintptr, inputclip.ClipMeta) (string, error)
	Stop() (*StopResult, error)
	Cancel()
}

var _ recorderLifecycle = (*Recorder)(nil)

// ErrRecorderNotActive classifies a stop request without an active recording.
var ErrRecorderNotActive = errors.New("recorder not active")
