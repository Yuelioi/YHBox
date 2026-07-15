package services

import (
	"errors"
	"fmt"

	"github.com/yottaapp/yotta/internal/appcontrol"
)

// ApplicationService owns executable inspection and explicit workflow consent.
// It never launches or terminates an application through the settings RPC.
type ApplicationService struct{ app *App }

func NewApplicationService(app *App) *ApplicationService { return &ApplicationService{app: app} }

func (s *ApplicationService) InspectExecutable(path string) (appcontrol.ExecutableInspection, error) {
	return appcontrol.InspectExecutable(path)
}

func (s *ApplicationService) GrantWorkflowConsent(slot string) (string, error) {
	if s == nil || s.app == nil {
		return "", errors.New("application service is unavailable")
	}
	var granted string
	_, _, err := s.app.MutateSettings(func(settings *Settings) error {
		for index := range settings.Applications.Profiles {
			configured := &settings.Applications.Profiles[index]
			if configured.Slot != slot {
				continue
			}
			profile, err := appcontrol.SealProfile(configured.profileDraft())
			if err != nil {
				return err
			}
			if err := appcontrol.VerifyProfile(profile); err != nil {
				return err
			}
			digest, err := appcontrol.WorkflowConsentDigest(slot, profile)
			if err != nil {
				return err
			}
			configured.WorkflowConsent, granted = digest, digest.String()
			return nil
		}
		return fmt.Errorf("application slot %q is not installed", slot)
	}, nil)
	if err != nil {
		return "", err
	}
	s.app.Emit("settings:changed", map[string]any{})
	return granted, nil
}

func (s *ApplicationService) RevokeWorkflowConsent(slot string) error {
	if s == nil || s.app == nil {
		return errors.New("application service is unavailable")
	}
	_, _, err := s.app.MutateSettings(func(settings *Settings) error {
		for index := range settings.Applications.Profiles {
			configured := &settings.Applications.Profiles[index]
			if configured.Slot != slot {
				continue
			}
			configured.WorkflowConsent = ""
			return nil
		}
		return fmt.Errorf("application slot %q is not installed", slot)
	}, nil)
	if err == nil {
		s.app.Emit("settings:changed", map[string]any{})
	}
	return err
}
