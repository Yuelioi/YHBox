package services

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/rs/zerolog"
	"github.com/yottaapp/yotta/internal/securestore"
)

type fakeSecretStore struct {
	values map[string]string
	setErr error
}

func newFakeSecretStore() *fakeSecretStore {
	return &fakeSecretStore{values: make(map[string]string)}
}

func (s *fakeSecretStore) Get(target string) (string, error) {
	value, ok := s.values[target]
	if !ok {
		return "", securestore.ErrNotFound
	}
	return value, nil
}

func (s *fakeSecretStore) Set(target, value string) error {
	if s.setErr != nil {
		return s.setErr
	}
	s.values[target] = value
	return nil
}

func (s *fakeSecretStore) Delete(target string) error {
	delete(s.values, target)
	return nil
}

func TestAISecretsMigrateLegacyClearsPlaintextAfterCredentialCommit(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "settings.json")
	settings := defaultSettings()
	settings.AI.Connections = []AIConnection{{
		ID: "primary", Label: "Primary", Protocol: "openai", APIKey: "secret-value",
	}}
	if err := SaveSettings(path, settings); err != nil {
		t.Fatal(err)
	}
	app := NewApp(path, nil, zerolog.Nop())
	store := newFakeSecretStore()
	secrets := NewAISecrets(store)

	count, err := secrets.MigrateLegacy(app)
	if err != nil || count != 1 {
		t.Fatalf("MigrateLegacy = count %d, err %v", count, err)
	}
	if got := store.values[aiCredentialTargetPrefix+"primary"]; got != "secret-value" {
		t.Fatalf("stored credential = %q", got)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if string(data) == "" || containsJSONSecret(data, "secret-value") {
		t.Fatalf("settings still contains plaintext credential: %s", data)
	}
	if got := app.Settings().AI.Connections[0].APIKey; got != "" {
		t.Fatalf("live settings retained plaintext credential: %q", got)
	}
}

func TestAISecretsMigrationFailurePreservesPlaintext(t *testing.T) {
	path := filepath.Join(t.TempDir(), "settings.json")
	settings := defaultSettings()
	settings.AI.Connections = []AIConnection{{
		ID: "primary", Label: "Primary", Protocol: "openai", APIKey: "keep-me",
	}}
	if err := SaveSettings(path, settings); err != nil {
		t.Fatal(err)
	}
	app := NewApp(path, nil, zerolog.Nop())
	store := newFakeSecretStore()
	store.setErr = errors.New("vault unavailable")

	if _, err := NewAISecrets(store).MigrateLegacy(app); err == nil {
		t.Fatal("migration succeeded with unavailable credential store")
	}
	if got := app.Settings().AI.Connections[0].APIKey; got != "keep-me" {
		t.Fatalf("plaintext was cleared after failed migration: %q", got)
	}
}

func TestSettingsServiceGetRedactsLegacyAPIKey(t *testing.T) {
	app := NewApp(filepath.Join(t.TempDir(), "settings.json"), nil, zerolog.Nop())
	_, _, err := app.MutateSettings(func(settings *Settings) error {
		settings.AI.Connections = []AIConnection{{
			ID: "primary", Label: "Primary", Protocol: "openai", APIKey: "never-return",
		}}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	got := NewSettingsService(app).Get()
	if got.AI.Connections[0].APIKey != "" {
		t.Fatal("settings RPC returned plaintext API key")
	}
	if app.Settings().AI.Connections[0].APIKey != "never-return" {
		t.Fatal("redaction mutated live settings")
	}
}

func TestSettingsServiceMetadataUpdatePreservesUnmigratedLegacyKey(t *testing.T) {
	app := NewApp(filepath.Join(t.TempDir(), "settings.json"), nil, zerolog.Nop())
	_, _, err := app.MutateSettings(func(settings *Settings) error {
		settings.AI.Connections = []AIConnection{{
			ID: "primary", Label: "Before", Protocol: "openai", APIKey: "keep-until-migrated",
		}}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	service := NewSettingsService(app)
	if err := service.Update(`{"ai":{"connections":[{"id":"primary","label":"After","protocol":"openai","baseURL":""}]}}`); err != nil {
		t.Fatal(err)
	}
	connection := app.Settings().AI.Connections[0]
	if connection.Label != "After" || connection.APIKey != "keep-until-migrated" {
		t.Fatalf("metadata update lost legacy key: %+v", connection)
	}
}

func containsJSONSecret(data []byte, secret string) bool {
	for index := 0; index+len(secret) <= len(data); index++ {
		if string(data[index:index+len(secret)]) == secret {
			return true
		}
	}
	return false
}
