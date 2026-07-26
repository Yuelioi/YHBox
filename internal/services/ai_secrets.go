package services

import (
	"errors"
	"strings"

	"github.com/yottaapp/yotta/internal/ai"
	"github.com/yottaapp/yotta/internal/securestore"
)

const aiCredentialTargetPrefix = "Yotta/AIModel/"

// AISecrets owns model credentials independently from settings.json.
type AISecrets struct {
	store securestore.Store
}

func NewAISecrets(store securestore.Store) *AISecrets {
	return &AISecrets{store: store}
}

// Get implements ai.CredentialStore. Its input is the exact credential
// binding ID frozen into the Host Profile, not arbitrary node config.
func (s *AISecrets) Get(bindingID string) (string, error) {
	if s == nil || s.store == nil {
		return "", securestore.ErrUnavailable
	}
	slot, err := slotFromCredentialBinding(bindingID)
	if err != nil {
		return "", err
	}
	return s.store.Get(aiCredentialTargetPrefix + slot)
}

func (s *AISecrets) SetSlot(slot, apiKey string) error {
	if err := ai.ValidateInstallationSlot(slot); err != nil {
		return err
	}
	if s == nil || s.store == nil {
		return securestore.ErrUnavailable
	}
	if apiKey == "" {
		return s.DeleteSlot(slot)
	}
	return s.store.Set(aiCredentialTargetPrefix+slot, apiKey)
}

func (s *AISecrets) DeleteSlot(slot string) error {
	if err := ai.ValidateInstallationSlot(slot); err != nil {
		return err
	}
	if s == nil || s.store == nil {
		return securestore.ErrUnavailable
	}
	return s.store.Delete(aiCredentialTargetPrefix + slot)
}

func (s *AISecrets) HasSlot(slot string) (bool, error) {
	if err := ai.ValidateInstallationSlot(slot); err != nil {
		return false, err
	}
	_, err := s.store.Get(aiCredentialTargetPrefix + slot)
	if err == nil {
		return true, nil
	}
	if errors.Is(err, securestore.ErrNotFound) {
		return false, nil
	}
	return false, err
}

func slotFromCredentialBinding(bindingID string) (string, error) {
	slot, ok := strings.CutPrefix(bindingID, "ai-credential/")
	if !ok || ai.ValidateInstallationSlot(slot) != nil {
		return "", errors.New("invalid AI credential binding")
	}
	return slot, nil
}
