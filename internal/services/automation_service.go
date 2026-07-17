package services

import (
	"context"
	"errors"
	"fmt"
	"time"

	automationinstalled "github.com/yottaapp/yotta/internal/automation/installed"
)

// AutomationService owns explicit workflow consent for exact installed window
// targets. It exposes no input execution RPC.
type AutomationService struct{ app *App }

type AutomationTargetHealth struct {
	OK      bool   `json:"ok"`
	Code    string `json:"code"`
	Message string `json:"message"`
}

func NewAutomationService(app *App) *AutomationService { return &AutomationService{app: app} }

func (service *AutomationService) ListTargetTypes() []automationinstalled.TargetTypeDescriptor {
	return automationinstalled.TargetTypes()
}

func (service *AutomationService) ListADBDevices() ([]automationinstalled.AndroidDeviceDescriptor, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	return automationinstalled.DiscoverAndroidDevices(ctx)
}

func (service *AutomationService) CheckTargetHealth(slot string) AutomationTargetHealth {
	if service == nil || service.app == nil {
		return AutomationTargetHealth{Code: "unavailable", Message: "automation service is unavailable"}
	}
	settings := service.app.Settings()
	applications := make(map[string]InstalledApplicationSettings, len(settings.Applications.Profiles))
	for _, application := range settings.Applications.Profiles {
		applications[application.Slot] = application
	}
	for _, configured := range settings.Automation.Targets {
		if configured.Slot != slot {
			continue
		}
		profile, err := automationinstalled.SealProfile(configured.profileDraft(applications[configured.ApplicationSlot]))
		if err != nil {
			return AutomationTargetHealth{Code: "invalid-profile", Message: err.Error()}
		}
		if configured.AdapterKind != automationinstalled.AdapterKindAndroidADB {
			if err := automationinstalled.VerifyProfile(profile); err != nil {
				return AutomationTargetHealth{Code: "identity-changed", Message: err.Error()}
			}
			return AutomationTargetHealth{OK: true, Code: "ready", Message: "desktop target profile is valid"}
		}
		ctx, cancel := context.WithTimeout(context.Background(), time.Duration(configured.ResolveTimeoutMilliseconds)*time.Millisecond)
		driver, err := automationinstalled.NewAndroidHealthProbe(profile)
		if err == nil {
			_, err = driver.Resolve(ctx)
		}
		cancel()
		if err != nil {
			var classified *automationinstalled.Failure
			code := "unavailable"
			if errors.As(err, &classified) {
				code = classified.Code
			}
			return AutomationTargetHealth{Code: code, Message: err.Error()}
		}
		return AutomationTargetHealth{OK: true, Code: "ready", Message: "ADB device identity and display are ready"}
	}
	return AutomationTargetHealth{Code: "not-found", Message: fmt.Sprintf("automation target slot %q is not installed", slot)}
}

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
		for index := range settings.Automation.Targets {
			configured := &settings.Automation.Targets[index]
			if configured.Slot != slot {
				continue
			}
			var application InstalledApplicationSettings
			if configured.TargetKind != automationinstalled.TargetKindAndroidDevice || configured.AdapterKind != automationinstalled.AdapterKindAndroidADB {
				var ok bool
				application, ok = applications[configured.ApplicationSlot]
				if !ok {
					return fmt.Errorf("automation target %q references unknown application slot %q", slot, configured.ApplicationSlot)
				}
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
		for index := range settings.Automation.Targets {
			configured := &settings.Automation.Targets[index]
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
