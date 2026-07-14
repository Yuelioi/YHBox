package services

import (
	"errors"
	"fmt"

	"github.com/yottaapp/yotta/internal/securestore"
)

const aiCredentialTargetPrefix = "Yotta/AI/"

// AISecrets owns AI connection credentials independently from settings.json.
type AISecrets struct {
	store securestore.Store
}

func NewAISecrets(store securestore.Store) *AISecrets {
	return &AISecrets{store: store}
}

func (s *AISecrets) Get(connectionID string) (string, error) {
	if s == nil || s.store == nil {
		return "", securestore.ErrUnavailable
	}
	return s.store.Get(aiCredentialTargetPrefix + connectionID)
}

func (s *AISecrets) Set(connectionID, apiKey string) error {
	if connectionID == "" {
		return errors.New("connection id is required")
	}
	if s == nil || s.store == nil {
		return securestore.ErrUnavailable
	}
	if apiKey == "" {
		return s.Delete(connectionID)
	}
	return s.store.Set(aiCredentialTargetPrefix+connectionID, apiKey)
}

func (s *AISecrets) Delete(connectionID string) error {
	if connectionID == "" {
		return errors.New("connection id is required")
	}
	if s == nil || s.store == nil {
		return securestore.ErrUnavailable
	}
	return s.store.Delete(aiCredentialTargetPrefix + connectionID)
}

func (s *AISecrets) Has(connectionID string) (bool, error) {
	_, err := s.Get(connectionID)
	if err == nil {
		return true, nil
	}
	if errors.Is(err, securestore.ErrNotFound) {
		return false, nil
	}
	return false, err
}

// MigrateLegacy moves every plaintext key before clearing any of them. A
// failed credential write leaves settings.json untouched so startup never
// trades confidentiality for data loss.
func (s *AISecrets) MigrateLegacy(app *App) (int, error) {
	if app == nil {
		return 0, errors.New("app is nil")
	}
	current := app.Settings()
	pending := make([]AIConnection, 0, len(current.AI.Connections))
	for _, connection := range current.AI.Connections {
		if connection.APIKey != "" {
			pending = append(pending, connection)
		}
	}
	if len(pending) == 0 {
		return 0, nil
	}
	for _, connection := range pending {
		if err := s.Set(connection.ID, connection.APIKey); err != nil {
			return 0, fmt.Errorf("store credential for %q: %w", connection.Label, err)
		}
	}
	_, _, err := app.MutateSettings(func(settings *Settings) error {
		for index := range settings.AI.Connections {
			settings.AI.Connections[index].APIKey = ""
		}
		return nil
	})
	if err != nil {
		return 0, fmt.Errorf("clear migrated plaintext credentials: %w", err)
	}
	return len(pending), nil
}
