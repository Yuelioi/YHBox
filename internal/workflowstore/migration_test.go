package workflowstore

import (
	"bytes"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/yottaapp/yotta/internal/workflow/schema"
)

func TestSourceMigrationPlanRunsOnlyRegisteredVersionChain(t *testing.T) {
	plan, err := newSourceMigrationPlan(sourceContract{Format: schema.Format, Version: "3.3"}, []sourceMigrationStep{
		{
			From:  sourceContract{Format: schema.Format, Version: "3.1"},
			To:    sourceContract{Format: schema.Format, Version: "3.2"},
			Apply: rewriteSourceVersion("3.2"),
		},
		{
			From:  sourceContract{Format: schema.Format, Version: "3.2"},
			To:    sourceContract{Format: schema.Format, Version: "3.3"},
			Apply: rewriteSourceVersion("3.3"),
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	original := []byte(`{"format":"yotta.workflow","version":"3.1","kept":true}`)
	migrated, changed, err := plan.Migrate(original)
	if err != nil {
		t.Fatal(err)
	}
	if !changed || bytes.Equal(migrated, original) {
		t.Fatalf("migration result changed=%v artifact=%s", changed, migrated)
	}
	var document struct {
		Version string `json:"version"`
		Kept    bool   `json:"kept"`
	}
	if err := json.Unmarshal(migrated, &document); err != nil {
		t.Fatal(err)
	}
	if document.Version != "3.3" || !document.Kept {
		t.Fatalf("migrated document = %#v", document)
	}
}

func TestSourceMigrationPlanRejectsUnavailableAndConfusedMigrations(t *testing.T) {
	current := sourceContract{Format: schema.Format, Version: schema.Version}
	plan, err := newSourceMigrationPlan(current, nil)
	if err != nil {
		t.Fatal(err)
	}
	original := []byte(`{"format":"yotta.workflow","version":"3.0"}`)
	if _, _, err := plan.Migrate(original); !errors.Is(err, errSourceMigrationUnavailable) {
		t.Fatalf("unregistered migration error = %v", err)
	}

	confused, err := newSourceMigrationPlan(current, []sourceMigrationStep{{
		From:  sourceContract{Format: schema.Format, Version: "3.0"},
		To:    current,
		Apply: rewriteSourceVersion("9.9"),
	}})
	if err != nil {
		t.Fatal(err)
	}
	if _, _, err := confused.Migrate(original); err == nil {
		t.Fatal("migration output with an undeclared version was accepted")
	}
}

func TestOpenSourceStoreAtomicallyPublishesValidatedMigration(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, sourceMarker), []byte(sourceMarkerContents), 0o600); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(root, "wf-migrated.json")
	legacy := []byte(`{"format":"yotta.workflow","version":"3.0","workflowId":"wf-migrated"}`)
	if err := os.WriteFile(path, legacy, 0o600); err != nil {
		t.Fatal(err)
	}
	current := sourceContract{Format: schema.Format, Version: schema.Version}
	plan, err := newSourceMigrationPlan(current, []sourceMigrationStep{{
		From: sourceContract{Format: schema.Format, Version: "3.0"},
		To:   current,
		Apply: func(raw []byte) ([]byte, error) {
			if !bytes.Equal(raw, legacy) {
				return nil, errors.New("migration did not receive exact legacy bytes")
			}
			return migrationTestSource(), nil
		},
	}})
	if err != nil {
		t.Fatal(err)
	}
	store, err := openSourceStore(root, SourceStoreOptions{MaxSources: 2}, plan)
	if err != nil {
		t.Fatal(err)
	}
	loaded, err := store.Load("wf-migrated")
	if err != nil {
		t.Fatal(err)
	}
	durable, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(durable, loaded.Artifact()) || bytes.Equal(durable, legacy) {
		t.Fatalf("durable migration = %s; snapshot = %s", durable, loaded.Artifact())
	}
	if len(store.ListRecoveries()) != 0 {
		t.Fatalf("validated migration was quarantined: %#v", store.ListRecoveries())
	}
}

func TestCurrentSourceMigrationPlanDoesNotRepairDevelopmentArtifacts(t *testing.T) {
	plan, err := currentSourceMigrationPlan()
	if err != nil {
		t.Fatal(err)
	}
	developmentArtifact := []byte(`{"format":"yotta.workflow","version":"3.1","workflow":{"id":"old-dev","name":"Old development shape"},"revision":0,"entryGraph":"main","graphs":[],"variables":[],"secretRefs":[]}`)
	got, changed, err := plan.Migrate(developmentArtifact)
	if err != nil {
		t.Fatal(err)
	}
	if changed || !bytes.Equal(got, developmentArtifact) {
		t.Fatalf("development artifact was migrated: changed=%v artifact=%s", changed, got)
	}
	if _, err := openSourceArtifact(got, true); err == nil {
		t.Fatal("development artifact unexpectedly passed the current strict schema")
	}
}

func TestOpenSourceStoreQuarantinesInvalidCurrentDevelopmentArtifact(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, sourceMarker), []byte(sourceMarkerContents), 0o600); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(root, "old-dev.json")
	developmentArtifact := []byte(`{"format":"yotta.workflow","version":"3.1","workflow":{"id":"old-dev","name":"Old development shape"},"revision":0,"entryGraph":"main","graphs":[],"variables":[],"secretRefs":[]}`)
	if err := os.WriteFile(path, developmentArtifact, 0o600); err != nil {
		t.Fatal(err)
	}
	store, err := OpenSourceStore(root, SourceStoreOptions{MaxSources: 2})
	if err != nil {
		t.Fatal(err)
	}
	recoveries := store.ListRecoveries()
	if len(recoveries) != 1 || !bytes.Equal(recoveries[0].Artifact(), developmentArtifact) {
		t.Fatalf("development recovery = %#v", recoveries)
	}
	if _, err := os.Stat(path); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("invalid current artifact remained active: %v", err)
	}
}

func TestOpenSourceStorePreservesOriginalWhenRegisteredMigrationFails(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, sourceMarker), []byte(sourceMarkerContents), 0o600); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(root, "wf-migration-failure.json")
	legacy := []byte(`{"format":"yotta.workflow","version":"3.0","workflowId":"wf-migration-failure"}`)
	if err := os.WriteFile(path, legacy, 0o600); err != nil {
		t.Fatal(err)
	}
	current := sourceContract{Format: schema.Format, Version: schema.Version}
	plan, err := newSourceMigrationPlan(current, []sourceMigrationStep{{
		From: sourceContract{Format: schema.Format, Version: "3.0"},
		To:   current,
		Apply: func([]byte) ([]byte, error) {
			return nil, errors.New("synthetic migration failure")
		},
	}})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := openSourceStore(root, SourceStoreOptions{MaxSources: 2}, plan); err == nil {
		t.Fatal("migration failure did not fail store open")
	}
	durable, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(durable, legacy) {
		t.Fatalf("migration failure changed original bytes: %s", durable)
	}
	if _, err := os.Stat(filepath.Join(root, sourceRecoveryDir)); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("migration failure was incorrectly quarantined: %v", err)
	}
}

func rewriteSourceVersion(version string) sourceMigration {
	return func(raw []byte) ([]byte, error) {
		var document map[string]json.RawMessage
		if err := json.Unmarshal(raw, &document); err != nil {
			return nil, err
		}
		encoded, err := json.Marshal(version)
		if err != nil {
			return nil, err
		}
		document["version"] = encoded
		return json.Marshal(document)
	}
}

func migrationTestSource() []byte {
	return []byte(`{
		"format":"yotta.workflow","version":"3.1","workflow":{"id":"wf-migrated","name":"Migrated"},
		"revision":0,"entryGraph":"main","graphs":[{"id":"main","kind":"main","nodes":[],"edges":[],"inputs":[],"outputs":[]}],
		"variables":[],"resources":[],"targetProfileDefinitions":[],"credentialRequirements":[],"dependencies":[]
	}`)
}
