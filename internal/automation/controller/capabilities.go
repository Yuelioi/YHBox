package controller

type Capability string

const (
	CapabilityScreenshot   Capability = "screenshot"
	CapabilityClick        Capability = "click"
	CapabilityMove         Capability = "move"
	CapabilityScroll       Capability = "scroll"
	CapabilityMouseButton  Capability = "mouse-button"
	CapabilityDrag         Capability = "drag"
	CapabilityMoveRelative Capability = "move-relative"
	CapabilityKeyChord     Capability = "key-chord"
	CapabilityKeyState     Capability = "key-state"
	CapabilityText         Capability = "text"
	CapabilityStartApp     Capability = "start-app"
	CapabilityStopApp      Capability = "stop-app"
)

type CapabilitySet struct {
	Screenshot   bool
	Click        bool
	Move         bool
	Scroll       bool
	MouseButton  bool
	Drag         bool
	MoveRelative bool
	KeyChord     bool
	KeyState     bool
	Text         bool
	StartApp     bool
	StopApp      bool
}

func (s CapabilitySet) Has(c Capability) bool {
	switch c {
	case CapabilityScreenshot:
		return s.Screenshot
	case CapabilityClick:
		return s.Click
	case CapabilityMove:
		return s.Move
	case CapabilityScroll:
		return s.Scroll
	case CapabilityMouseButton:
		return s.MouseButton
	case CapabilityDrag:
		return s.Drag
	case CapabilityMoveRelative:
		return s.MoveRelative
	case CapabilityKeyChord:
		return s.KeyChord
	case CapabilityKeyState:
		return s.KeyState
	case CapabilityText:
		return s.Text
	case CapabilityStartApp:
		return s.StartApp
	case CapabilityStopApp:
		return s.StopApp
	default:
		return false
	}
}
