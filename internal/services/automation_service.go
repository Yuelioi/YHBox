package services

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/yottaapp/yotta/internal/appcontrol"
	"github.com/yottaapp/yotta/internal/automation/browsercdp"
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

// BulkWorkflowConsentResult reports the current installation snapshot covered
// by one explicit bulk authorization action.
type BulkWorkflowConsentResult struct {
	Applications int `json:"applications"`
	Targets      int `json:"targets"`
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

func (service *AutomationService) ListBrowserTargets(endpoint string) ([]browsercdp.TargetInfo, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	return browsercdp.NewService(endpoint).ListTargets(ctx, endpoint)
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
		if configured.isBrowser() {
			ctx, cancel := context.WithTimeout(context.Background(), time.Duration(configured.ResolveTimeoutMilliseconds)*time.Millisecond)
			probe, probeErr := automationinstalled.NewBrowserHealthProbe(profile)
			if probeErr == nil {
				_, probeErr = probe.Resolve(ctx)
			}
			cancel()
			if probeErr != nil {
				return automationHealthFailure(probeErr)
			}
			return AutomationTargetHealth{OK: true, Code: "ready", Message: "browser page identity and viewport are ready"}
		}
		if !configured.isAndroid() {
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
			return automationHealthFailure(err)
		}
		return AutomationTargetHealth{OK: true, Code: "ready", Message: "ADB device identity and display are ready"}
	}
	return AutomationTargetHealth{Code: "not-found", Message: fmt.Sprintf("automation target slot %q is not installed", slot)}
}

func automationHealthFailure(err error) AutomationTargetHealth {
	var classified *automationinstalled.Failure
	code := "unavailable"
	if errors.As(err, &classified) {
		code = classified.Code
	}
	return AutomationTargetHealth{Code: code, Message: err.Error()}
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
			if configured.isDesktop() {
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

// GrantAllWorkflowConsents authorizes every currently installed desktop
// application and automation target in one atomic settings mutation. This is
// deliberately a snapshot operation: installing or changing an identity later
// produces a different digest and therefore requires fresh consent.
func (service *AutomationService) GrantAllWorkflowConsents() (BulkWorkflowConsentResult, error) {
	if service == nil || service.app == nil {
		return BulkWorkflowConsentResult{}, errors.New("automation service is unavailable")
	}
	var result BulkWorkflowConsentResult
	_, _, err := service.app.MutateSettings(func(settings *Settings) error {
		applications := make(map[string]InstalledApplicationSettings, len(settings.Applications.Profiles))
		for index := range settings.Applications.Profiles {
			configured := &settings.Applications.Profiles[index]
			profile, err := appcontrol.SealProfile(configured.profileDraft())
			if err != nil {
				return err
			}
			if err := appcontrol.VerifyProfile(profile); err != nil {
				return err
			}
			digest, err := appcontrol.WorkflowConsentDigest(configured.Slot, profile)
			if err != nil {
				return err
			}
			configured.WorkflowConsent = digest
			applications[configured.Slot] = *configured
			result.Applications++
		}
		for index := range settings.Automation.Targets {
			configured := &settings.Automation.Targets[index]
			var application InstalledApplicationSettings
			if configured.isDesktop() {
				var ok bool
				application, ok = applications[configured.ApplicationSlot]
				if !ok {
					return fmt.Errorf("automation target %q references unknown application slot %q", configured.Slot, configured.ApplicationSlot)
				}
			}
			profile, err := automationinstalled.SealProfile(configured.profileDraft(application))
			if err != nil {
				return err
			}
			if err := automationinstalled.VerifyProfile(profile); err != nil {
				return err
			}
			digest, err := automationinstalled.WorkflowConsentDigest(configured.Slot, profile)
			if err != nil {
				return err
			}
			configured.WorkflowConsent = digest
			result.Targets++
		}
		return nil
	}, nil)
	if err != nil {
		return BulkWorkflowConsentResult{}, err
	}
	service.app.Emit("settings:changed", map[string]any{})
	return result, nil
}

// RevokeAllWorkflowConsents clears the same application + automation consent
// family atomically. Capability, package admission, and arm boundaries are not
// changed by either bulk operation.
func (service *AutomationService) RevokeAllWorkflowConsents() (BulkWorkflowConsentResult, error) {
	if service == nil || service.app == nil {
		return BulkWorkflowConsentResult{}, errors.New("automation service is unavailable")
	}
	var result BulkWorkflowConsentResult
	_, _, err := service.app.MutateSettings(func(settings *Settings) error {
		for index := range settings.Applications.Profiles {
			settings.Applications.Profiles[index].WorkflowConsent = ""
			result.Applications++
		}
		for index := range settings.Automation.Targets {
			settings.Automation.Targets[index].WorkflowConsent = ""
			result.Targets++
		}
		return nil
	}, nil)
	if err != nil {
		return BulkWorkflowConsentResult{}, err
	}
	service.app.Emit("settings:changed", map[string]any{})
	return result, nil
}
