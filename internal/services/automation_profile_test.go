package services

import (
	"encoding/json"

	"github.com/yottaapp/yotta/internal/artifact"
	automationinstalled "github.com/yottaapp/yotta/internal/automation/installed"
)

type DesktopAutomationTargetSettings = automationinstalled.DesktopProfileIntent
type AndroidAutomationTargetSettings = automationinstalled.AndroidProfileIntent
type BrowserAutomationTargetSettings = automationinstalled.BrowserProfileIntent

func automationTargetProfile(payload any) json.RawMessage {
	raw, _ := artifact.Marshal(payload)
	return raw
}
