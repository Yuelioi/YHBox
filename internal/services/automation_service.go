package services

import (
	"context"
	"errors"
	"time"

	"github.com/yottaapp/yotta/internal/automation/browsercdp"
	automationinstalled "github.com/yottaapp/yotta/internal/automation/installed"
)

// AutomationService owns target discovery and health checks. Configured
// targets are immediately available to workflows; it exposes no input
// execution RPC.
type AutomationService struct{ app *App }

type AutomationTargetHealth struct {
	OK     bool           `json:"ok"`
	ID     string         `json:"id"`
	Params map[string]any `json:"params,omitempty"`
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

func (service *AutomationService) ListAndroidApps(serial string) ([]automationinstalled.AndroidAppDescriptor, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 8*time.Second)
	defer cancel()
	return automationinstalled.DiscoverAndroidApps(ctx, serial)
}

func (service *AutomationService) ListBrowserTargets(endpoint string) ([]browsercdp.TargetInfo, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	return browsercdp.NewService(endpoint).ListTargets(ctx, endpoint)
}

func (service *AutomationService) CheckTargetHealth(slot string) AutomationTargetHealth {
	if service == nil || service.app == nil {
		return AutomationTargetHealth{ID: "automation.health.unavailable"}
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
		draft, err := configured.profileDraft(applications[configured.applicationSlot()])
		if err != nil {
			return AutomationTargetHealth{ID: "automation.health.invalid_profile", Params: map[string]any{"slot": slot}}
		}
		installations, err := automationinstalled.Install([]automationinstalled.InstallationDraft{{
			Slot: configured.Slot, Label: configured.Label, Profile: draft,
		}})
		if err != nil {
			return AutomationTargetHealth{ID: "automation.health.invalid_profile", Params: map[string]any{"slot": slot}}
		}
		defer installations.Close()
		installed := installations.Entries()[0]
		ctx, cancel := context.WithTimeout(context.Background(), time.Duration(installed.Profile.ResolveTimeoutMilliseconds())*time.Millisecond)
		err = automationinstalled.CheckInstallationHealth(ctx, installed)
		cancel()
		if err != nil {
			return automationHealthFailure(err)
		}
		return AutomationTargetHealth{OK: true, ID: "automation.health.ready", Params: map[string]any{"slot": slot}}
	}
	return AutomationTargetHealth{ID: "automation.health.not_found", Params: map[string]any{"slot": slot}}
}

func automationHealthFailure(err error) AutomationTargetHealth {
	var classified *automationinstalled.Failure
	code := "unavailable"
	if errors.As(err, &classified) {
		code = classified.Code
	}
	return AutomationTargetHealth{ID: code}
}
