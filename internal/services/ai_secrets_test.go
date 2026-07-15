package services

import (
	"errors"
	"testing"

	"github.com/yottaapp/yotta/internal/ai"
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

func TestAISecretsResolveOnlyFrozenCredentialBindingIDs(t *testing.T) {
	store := newFakeSecretStore()
	secrets := NewAISecrets(store)
	if err := secrets.SetSlot("primary", "secret-value"); err != nil {
		t.Fatal(err)
	}
	if got := store.values[aiCredentialTargetPrefix+"primary"]; got != "secret-value" {
		t.Fatalf("stored credential = %q", got)
	}
	credential, err := secrets.Get(ai.CredentialBindingID("primary"))
	if err != nil || credential != "secret-value" {
		t.Fatalf("Get = %q, %v", credential, err)
	}
	if _, err := secrets.Get("primary"); err == nil {
		t.Fatal("accepted a workflow slot as a credential binding ID")
	}
	if err := secrets.SetSlot("Primary", "secret"); err == nil {
		t.Fatal("accepted an invalid installation slot")
	}
	if err := secrets.DeleteSlot("primary"); err != nil {
		t.Fatal(err)
	}
	if _, err := secrets.Get(ai.CredentialBindingID("primary")); !errors.Is(err, securestore.ErrNotFound) {
		t.Fatalf("deleted credential error = %v", err)
	}
}
