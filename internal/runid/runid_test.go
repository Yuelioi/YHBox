package runid_test

import (
	"testing"

	"github.com/yottaapp/yotta/internal/runid"
)

func TestNewProducesCanonicalUUIDv7(t *testing.T) {
	id, err := runid.New()
	if err != nil {
		t.Fatal(err)
	}
	if err := runid.Validate(id); err != nil {
		t.Fatal(err)
	}
}

func TestValidateRejectsLegacyAndNonV7IDs(t *testing.T) {
	for _, value := range []string{"run-1", "550e8400-e29b-41d4-a716-446655440000", "0190C7D4-1E40-7CC5-A783-57B16D5C8E3A"} {
		if err := runid.Validate(value); err == nil {
			t.Fatalf("accepted %q", value)
		}
	}
}
