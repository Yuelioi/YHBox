package services

import (
	"errors"
	"fmt"

	"github.com/yottaapp/yotta/internal/httpegress"
)

// NetworkService owns explicit workflow consent for installed HTTP origins.
// It never performs network requests and never returns provider internals.
type NetworkService struct{ app *App }

func NewNetworkService(app *App) *NetworkService { return &NetworkService{app: app} }

func (s *NetworkService) GrantHTTPWorkflowConsent(slot string) (string, error) {
	if s == nil || s.app == nil {
		return "", errors.New("network service is unavailable")
	}
	var granted string
	_, _, err := s.app.MutateSettings(func(settings *Settings) error {
		for index := range settings.Network.HTTPOrigins {
			origin := &settings.Network.HTTPOrigins[index]
			if origin.Slot != slot {
				continue
			}
			profile, err := httpegress.SealProfile(origin.profileDraft())
			if err != nil {
				return err
			}
			digest, err := httpegress.WorkflowConsentDigest(slot, profile)
			if err != nil {
				return err
			}
			origin.WorkflowConsent = digest
			granted = digest.String()
			return nil
		}
		return fmt.Errorf("HTTP origin slot %q is not installed", slot)
	}, nil)
	if err != nil {
		return "", err
	}
	s.app.Emit("settings:changed", map[string]any{})
	return granted, nil
}

func (s *NetworkService) RevokeHTTPWorkflowConsent(slot string) error {
	if s == nil || s.app == nil {
		return errors.New("network service is unavailable")
	}
	_, _, err := s.app.MutateSettings(func(settings *Settings) error {
		for index := range settings.Network.HTTPOrigins {
			origin := &settings.Network.HTTPOrigins[index]
			if origin.Slot != slot {
				continue
			}
			origin.WorkflowConsent = ""
			return nil
		}
		return fmt.Errorf("HTTP origin slot %q is not installed", slot)
	}, nil)
	if err == nil {
		s.app.Emit("settings:changed", map[string]any{})
	}
	return err
}
