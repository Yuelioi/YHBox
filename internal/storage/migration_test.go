package storage

import (
	"errors"
	"testing"

	"github.com/yottaapp/yotta/internal/artifact"
)

func migrationChecksum(digit byte) artifact.Digest {
	raw := make([]byte, 64)
	for index := range raw {
		raw[index] = digit
	}
	return artifact.Digest("sha256:" + string(raw))
}

func TestMigrationRegistryPlansOneDeterministicReleasedChain(t *testing.T) {
	registry, err := NewMigrationRegistry("3", []MigrationStep{
		{ID: "layout-1-to-2", From: "1", To: "2", Checksum: migrationChecksum('1')},
		{ID: "layout-2-to-3", From: "2", To: "3", Checksum: migrationChecksum('2')},
	})
	if err != nil {
		t.Fatalf("NewMigrationRegistry: %v", err)
	}
	plan, err := registry.Plan("1")
	if err != nil {
		t.Fatalf("Plan: %v", err)
	}
	if len(plan) != 2 || plan[0].ID != "layout-1-to-2" || plan[1].ID != "layout-2-to-3" {
		t.Fatalf("unexpected plan: %#v", plan)
	}
	current, err := registry.Plan("3")
	if err != nil || len(current) != 0 {
		t.Fatalf("current plan = %#v, %v", current, err)
	}
}

func TestMigrationRegistryRejectsAmbiguityGapsAndMutableIdentity(t *testing.T) {
	for name, steps := range map[string][]MigrationStep{
		"ambiguous": {
			{ID: "layout-1-to-2", From: "1", To: "2", Checksum: migrationChecksum('1')},
			{ID: "layout-1-to-3", From: "1", To: "3", Checksum: migrationChecksum('2')},
		},
		"gap": {
			{ID: "layout-1-to-2", From: "1", To: "2", Checksum: migrationChecksum('1')},
		},
		"bad checksum": {
			{ID: "layout-1-to-3", From: "1", To: "3", Checksum: "sha256:not-a-digest"},
		},
		"backward": {
			{ID: "layout-3-to-2", From: "3", To: "2", Checksum: migrationChecksum('1')},
		},
	} {
		t.Run(name, func(t *testing.T) {
			if _, err := NewMigrationRegistry("3", steps); err == nil {
				t.Fatal("expected invalid registry to be rejected")
			}
		})
	}
}

func TestMigrationRegistryDistinguishesUnknownOldAndFutureLayouts(t *testing.T) {
	registry, err := NewMigrationRegistry("3", []MigrationStep{
		{ID: "layout-2-to-3", From: "2", To: "3", Checksum: migrationChecksum('2')},
	})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := registry.Plan("1"); !errors.Is(err, ErrNoMigrationPath) {
		t.Fatalf("old layout error = %v", err)
	}
	if _, err := registry.Plan("4"); !errors.Is(err, ErrFutureLayout) {
		t.Fatalf("future layout error = %v", err)
	}
}
