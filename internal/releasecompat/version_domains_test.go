package releasecompat

import (
	"strings"
	"testing"
)

func TestVersionDomainReleasesFreezeAndRequireReleasedReaders(t *testing.T) {
	releases := VersionDomainReleases{Root: t.TempDir()}
	v1 := []CurrentVersionDomain{{
		Name: "yotta.workflow", CurrentVersion: "1", Class: "user-data", ReadableVersions: []string{"1"},
	}}
	if err := releases.Freeze("4.0.0", v1); err != nil {
		t.Fatal(err)
	}
	if err := releases.Freeze("4.0.0", v1); err != nil {
		t.Fatalf("idempotent freeze: %v", err)
	}
	if checked, err := releases.Check("4.0.0", v1, true); err != nil || checked != 1 {
		t.Fatalf("Check() = %d, %v", checked, err)
	}
	v2WithoutReader := []CurrentVersionDomain{{
		Name: "yotta.workflow", CurrentVersion: "2", Class: "user-data", ReadableVersions: []string{"2"},
	}}
	if _, err := releases.Check("4.0.1", v2WithoutReader, false); err == nil || !strings.Contains(err.Error(), "not declared readable") {
		t.Fatalf("missing released reader error = %v", err)
	}
	v2WithReader := []CurrentVersionDomain{{
		Name: "yotta.workflow", CurrentVersion: "2", Class: "user-data", ReadableVersions: []string{"1", "2"},
	}}
	if _, err := releases.Check("4.0.1", v2WithReader, false); err != nil {
		t.Fatalf("retained released reader: %v", err)
	}
}

func TestVersionDomainReleasesRejectRemovalClassDriftAndOverwrite(t *testing.T) {
	releases := VersionDomainReleases{Root: t.TempDir()}
	current := []CurrentVersionDomain{{
		Name: "schedule-schema", CurrentVersion: "5", Class: "user-data", ReadableVersions: []string{"5"},
	}}
	if err := releases.Freeze("4.0.0", current); err != nil {
		t.Fatal(err)
	}
	changed := []CurrentVersionDomain{{
		Name: "schedule-schema", CurrentVersion: "6", Class: "user-data", ReadableVersions: []string{"5", "6"},
	}}
	if err := releases.Freeze("4.0.0", changed); err == nil || !strings.Contains(err.Error(), "immutable") {
		t.Fatalf("changed freeze error = %v", err)
	}
	if _, err := releases.Check("4.0.1", []CurrentVersionDomain{{
		Name: "schedule-schema", CurrentVersion: "5", Class: "derived", ReadableVersions: []string{"5"},
	}}, false); err == nil || !strings.Contains(err.Error(), "changed class") {
		t.Fatalf("class drift error = %v", err)
	}
	if _, err := releases.Check("4.0.1", []CurrentVersionDomain{{
		Name: "other", CurrentVersion: "1", Class: "user-data", ReadableVersions: []string{"1"},
	}}, false); err == nil || !strings.Contains(err.Error(), "was removed") {
		t.Fatalf("removal error = %v", err)
	}
}
