package services

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/yottaapp/yotta/internal/automation/browsercdp"
	automationinstalled "github.com/yottaapp/yotta/internal/automation/installed"
)

// AutomationService owns target discovery and health checks. Configured
// targets are immediately available to workflows; it exposes no input
// execution RPC.
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
		draft, err := configured.profileDraft(applications[configured.applicationSlot()])
		if err != nil {
			return AutomationTargetHealth{Code: "invalid-profile", Message: err.Error()}
		}
		installations, err := automationinstalled.Install([]automationinstalled.InstallationDraft{{
			Slot: configured.Slot, Label: configured.Label, Profile: draft,
		}})
		if err != nil {
			return AutomationTargetHealth{Code: "invalid-profile", Message: err.Error()}
		}
		defer installations.Close()
		installed := installations.Entries()[0]
		ctx, cancel := context.WithTimeout(context.Background(), time.Duration(installed.Profile.ResolveTimeoutMilliseconds())*time.Millisecond)
		err = automationinstalled.CheckInstallationHealth(ctx, installed)
		cancel()
		if err != nil {
			return automationHealthFailure(err)
		}
		return AutomationTargetHealth{OK: true, Code: "ready", Message: "target identity and adapter runtime are ready"}
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
