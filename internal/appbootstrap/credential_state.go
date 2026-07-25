package appbootstrap

import (
	"context"
	"errors"

	"github.com/yottaapp/yotta/internal/ai"
	"github.com/yottaapp/yotta/internal/workflowinstallation"
)

type AICredentialAvailability interface {
	HasSlot(string) (bool, error)
}

// AICredentialStates projects non-secret credential metadata from installed
// AI profiles and checks only whether each logical slot currently resolves in
// secure storage. Secret values never enter the returned projection.
func AICredentialStates(
	ctx context.Context,
	installations ai.Installations,
	availability AICredentialAvailability,
) ([]workflowinstallation.CredentialState, error) {
	if ctx == nil {
		return nil, errors.New("AI credential state requires a context")
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if !installations.Valid() || availability == nil {
		return nil, errors.New("AI credential state requires trusted installations and secure storage")
	}
	entries := installations.Entries()
	result := make([]workflowinstallation.CredentialState, 0, len(entries))
	for _, installed := range entries {
		available, err := availability.HasSlot(installed.Slot)
		if err != nil {
			return nil, err
		}
		profile := installed.Profile.Machine()
		result = append(result, workflowinstallation.CredentialState{
			CredentialBindingID: installed.CredentialBindingID,
			Kind:                ai.CredentialKindAPIKey,
			Label:               installed.Slot + " · " + profile.Model,
			Available:           available,
		})
	}
	return result, nil
}
