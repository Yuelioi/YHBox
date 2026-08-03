package controller

import (
	"context"
	"image"

	"github.com/yottaapp/yotta/internal/automation/pointermotion"
	"github.com/yottaapp/yotta/internal/automation/target"
)

type Controller interface {
	Target() target.Target
	Capabilities(context.Context) CapabilitySet
	HealthCheck(context.Context) HealthReport
}

type Screenshotter interface {
	Screenshot(context.Context, ScreenshotRequest) (Frame, error)
}

type PointerInput interface {
	Click(context.Context, ClickRequest) error
	Move(context.Context, MoveRequest) error
	Scroll(context.Context, ScrollRequest) error
	MouseDown(context.Context, MouseButtonRequest) error
	MouseUp(context.Context, MouseButtonRequest) error
	Drag(context.Context, DragRequest) error
	MoveRelative(context.Context, RelativeMoveRequest) error
}

// PointerLocator reports the current pointer position in the controller's
// target coordinate space. Controllers that cannot query pointer state do not
// implement this optional interface.
type PointerLocator interface {
	PointerPosition(context.Context) (target.Point, error)
}

type KeyboardInput interface {
	KeyChord(context.Context, KeyChordRequest) error
	KeyDown(context.Context, KeyRequest) error
	KeyUp(context.Context, KeyRequest) error
	Text(context.Context, TextRequest) error
}

type AppLifecycle interface {
	StartApp(context.Context, StartAppRequest) error
	StopApp(context.Context, StopAppRequest) error
}

type HealthReport struct {
	OK      bool
	Message string
}

type Frame struct {
	Image *image.RGBA
	Space target.CoordinateSpace
	Size  target.Size
}

type ScreenshotRequest struct {
	Space target.CoordinateSpace
	ROI   target.Rect
}

type ClickRequest struct {
	Point      target.Point
	Button     string
	DurationMs int
	Policy     ActionPolicy
}

type MoveRequest struct {
	Point      target.Point
	DurationMs int
	Motion     pointermotion.Kind
	Policy     ActionPolicy
}

type ScrollRequest struct {
	Point      target.Point
	Notches    int
	Horizontal bool
	Policy     ActionPolicy
}

type MouseButtonRequest struct {
	Point  target.Point
	Button string
	Policy ActionPolicy
}

type DragRequest struct {
	From       target.Point
	To         target.Point
	Button     string
	DurationMs int
	Motion     pointermotion.Kind
	Policy     ActionPolicy
}

type RelativeMoveRequest struct {
	Dx         int
	Dy         int
	DurationMs int
	Policy     ActionPolicy
}

type KeyChordRequest struct {
	Keys   []string
	Policy ActionPolicy
}

type KeyRequest struct {
	Key    string
	Policy ActionPolicy
}

type TextRequest struct {
	Text   string
	Policy ActionPolicy
}

type StartAppRequest struct {
	Intent string
}

type StopAppRequest struct {
	Intent string
}

type ActionPolicy struct {
	ForegroundRequired bool
	BackgroundAllowed  bool
	NoStealFocus       bool
}
