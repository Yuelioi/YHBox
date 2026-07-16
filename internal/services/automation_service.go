package services

import (
	"errors"
	"fmt"

	automationinstalled "github.com/yottaapp/yotta/internal/automation/installed"
)

// AutomationService owns explicit workflow consent for exact installed window
// targets. It exposes no input execution RPC.
type AutomationService struct{ app *App }

func NewAutomationService(app *App) *AutomationService { return &AutomationService{app: app} }

func (service *AutomationService) GrantWorkflowConsent(slot string) (string, error) {
	if service == nil || service.app == nil {
		return "", errors.New("automation service is unavailable")
	}
	var granted string
	_, _, err := service.app.MutateSettings(func(settings *Settings) error {
		applications := make(map[string]InstalledApplicationSettings, len(settings.Applications.Profiles))
		for _, configured := range settings.Applications.Profiles {
			applications[configured.Slot] = configured
		}
		for index := range settings.Automation.Win32Targets {
			configured := &settings.Automation.Win32Targets[index]
			if configured.Slot != slot {
				continue
			}
			application, ok := applications[configured.ApplicationSlot]
			if !ok {
				return fmt.Errorf("automation target %q references unknown application slot %q", slot, configured.ApplicationSlot)
			}
			profile, err := automationinstalled.SealProfile(configured.profileDraft(application))
			if err != nil {
				return err
			}
			if err := automationinstalled.VerifyProfile(profile); err != nil {
				return err
			}
			digest, err := automationinstalled.WorkflowConsentDigest(slot, profile)
			if err != nil {
				return err
			}
			configured.WorkflowConsent, granted = digest, digest.String()
			return nil
		}
		return fmt.Errorf("automation target slot %q is not installed", slot)
	}, nil)
	if err != nil {
		return "", err
	}
	service.app.Emit("settings:changed", map[string]any{})
	return granted, nil
}

func (service *AutomationService) RevokeWorkflowConsent(slot string) error {
	if service == nil || service.app == nil {
		return errors.New("automation service is unavailable")
	}
	_, _, err := service.app.MutateSettings(func(settings *Settings) error {
		for index := range settings.Automation.Win32Targets {
			configured := &settings.Automation.Win32Targets[index]
			if configured.Slot == slot {
				configured.WorkflowConsent = ""
				return nil
			}
		}
		return fmt.Errorf("automation target slot %q is not installed", slot)
	}, nil)
	if err == nil {
		service.app.Emit("settings:changed", map[string]any{})
	}
	return err
}
