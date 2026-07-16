package tools

// WindowKind identifies a semantic tools window. The GUI adapter owns its
// concrete title, route, dimensions, decoration, and color policy.
type WindowKind string

const (
	WindowMouseHUD      WindowKind = "mouse-hud"
	WindowRecordingHUD  WindowKind = "recording-hud"
	WindowLauncher      WindowKind = "launcher"
	WindowCalibratorHUD WindowKind = "calibrator-hud"
	WindowScreenPicker  WindowKind = "screen-picker"
)

// WindowRequest asks the presentation layer to open a semantic tools window.
// Only fields relevant to Kind are consumed; presentation policy stays in the
// GUI adapter.
type WindowRequest struct {
	Kind       WindowKind
	TargetSlot string
	RequestID  string
	Mode       string
	ColorSpace string
	GUID       string
}

// Window is the lifecycle surface required by the tools RPC service.
type Window interface {
	Focus()
	Show()
	Hide()
	Close()
	SetAlwaysOnTop(bool)
	SetSize(width, height int)
	OnClosing(func())
}

// Presenter translates semantic window requests and events for a GUI runtime.
// Ready must remain false until it can create windows and emit events.
type Presenter interface {
	Ready() bool
	OpenWindow(WindowRequest) (Window, error)
	Emit(name string, data any)
}
