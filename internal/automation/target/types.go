package target

import "fmt"

const (
	KindWin32Window = "win32-window"
	KindWin32Screen = "win32-screen"
	KindAndroidADB  = "android-adb"
	KindBrowserCDP  = "browser-cdp"
	KindDebugReplay = "debug-replay"
	KindMock        = "mock"
)

const (
	SpaceNormalized    CoordinateSpace = "normalized"
	SpaceScreen        CoordinateSpace = "screen"
	SpaceWindowClient  CoordinateSpace = "window-client"
	SpaceCaptureFrame  CoordinateSpace = "capture-frame"
	SpaceAndroidDevice CoordinateSpace = "android-device"
	SpaceBrowserView   CoordinateSpace = "browser-viewport"
)

type CoordinateSpace string

type Point struct {
	X     float64
	Y     float64
	Space CoordinateSpace
}

func NewNormalizedPoint(x, y float64) Point {
	return Point{X: x, Y: y, Space: SpaceNormalized}
}

type Rect struct {
	X int
	Y int
	W int
	H int
}

type Size struct {
	W int
	H int
}

type DPIMeta struct {
	ScaleX float64
	ScaleY float64
}

type TargetRef struct {
	HWND        uintptr
	ADBSerial   string
	BrowserID   string
	ReplayPath  string
	MockImageID string
}

type Target struct {
	ID          string
	Kind        string
	DisplayName string
	Ref         TargetRef
	Bounds      Rect
	Resolution  Size
	DPI         DPIMeta
	Metadata    map[string]any
}

func (t Target) Validate() error {
	if t.ID == "" {
		return fmt.Errorf("target id is required")
	}
	if t.Kind == "" {
		return fmt.Errorf("target kind is required")
	}
	switch t.Kind {
	case KindWin32Window:
		if t.Ref.HWND == 0 {
			return fmt.Errorf("win32-window target requires hwnd")
		}
	case KindAndroidADB:
		if t.Ref.ADBSerial == "" {
			return fmt.Errorf("android-adb target requires adb serial")
		}
	case KindBrowserCDP:
		if t.Ref.BrowserID == "" {
			return fmt.Errorf("browser-cdp target requires browser id")
		}
	}
	return nil
}
