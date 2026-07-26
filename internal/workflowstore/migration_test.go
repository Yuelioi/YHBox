package workflowstore

import (
	"bytes"
	"encoding/json"
	"errors"
	"testing"

	"github.com/yottaapp/yotta/internal/workflow/schema"
)

func TestSourceMigrationPlanRunsOnlyRegisteredVersionChain(t *testing.T) {
	plan, err := newSourceMigrationPlan(sourceContract{Format: schema.Format, Version: "3"}, []sourceMigrationStep{
		{
			From:  sourceContract{Format: schema.Format, Version: "1"},
			To:    sourceContract{Format: schema.Format, Version: "2"},
			Apply: rewriteSourceVersion("2"),
		},
		{
			From:  sourceContract{Format: schema.Format, Version: "2"},
			To:    sourceContract{Format: schema.Format, Version: "3"},
			Apply: rewriteSourceVersion("3"),
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	original := []byte(`{"format":"yotta.workflow","version":"1","kept":true}`)
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
	if document.Version != "3" || !document.Kept {
		t.Fatalf("migrated document = %#v", document)
	}
}

func TestSourceMigrationPlanRejectsUnavailableAndConfusedMigrations(t *testing.T) {
	current := sourceContract{Format: schema.Format, Version: schema.Version}
	plan, err := newSourceMigrationPlan(current, nil)
	if err != nil {
		t.Fatal(err)
	}
	original := []byte(`{"format":"yotta.workflow","version":"0"}`)
	if _, _, err := plan.Migrate(original); !errors.Is(err, errSourceMigrationUnavailable) {
		t.Fatalf("unregistered migration error = %v", err)
	}

	confused, err := newSourceMigrationPlan(current, []sourceMigrationStep{{
		From:  sourceContract{Format: schema.Format, Version: "0"},
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

func TestCurrentSourceMigrationPlanDoesNotRepairDevelopmentArtifacts(t *testing.T) {
	plan, err := currentSourceMigrationPlan()
	if err != nil {
		t.Fatal(err)
	}
	developmentArtifact := []byte(`{"format":"yotta.workflow","version":"3.1"}`)
	if _, _, err := plan.Migrate(developmentArtifact); !errors.Is(err, errSourceMigrationUnavailable) {
		t.Fatalf("development artifact migration error = %v", err)
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
