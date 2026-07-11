package container

import (
	"testing"

	"github.com/yottaapp/yotta/internal/services/container/dependency"
)

func TestBuildYottaLockHashesAndBindingSlots(t *testing.T) {
	manifest := PackageManifest{
		SchemaVersion: PackageSchemaVersion,
		Kind:          PackageKindContainer,
		Name:          "@yotta/daily-fishing",
		DisplayName:   "每日钓鱼",
		Version:       "0.1.0",
		Author:        PackagePerson{Name: "yl"},
		Publisher:     PackagePublisher{ID: "yotta"},
		Yotta: PackageYotta{
			PackageID:  "pkg_01jz_daily_fishing",
			EntryGraph: "graph.json",
			Publication: Publication{
				State:      PublicationDraft,
				Visibility: VisibilityPrivate,
			},
		},
	}
	graph := Graph{
		ID:            "g-main",
		SchemaVersion: GraphSchemaVersion,
		Nodes: []GraphNode{
			{ID: "target", Kind: "Win32WindowTarget", Config: map[string]any{"Target": "game"}},
			{ID: "ai", Kind: "AI", Config: map[string]any{"Connection": "main"}},
		},
	}
	closure := dependency.ClosureResult{
		Templates: []string{"tpl-a"},
		Clips:     []string{"clip-a"},
		Subgraphs: []string{"sg-a"},
	}

	lock, err := BuildYottaLock(manifest, graph, closure, "2026-06-30T00:00:00Z")
	if err != nil {
		t.Fatal(err)
	}
	if lock.PackageID != "pkg_01jz_daily_fishing" || lock.PackageName != "@yotta/daily-fishing" {
		t.Fatalf("package identity missing: %+v", lock)
	}
	if lock.ManifestHash == "" || lock.GraphHash == "" || lock.ClosureHash == "" {
		t.Fatalf("hashes must be populated: %+v", lock)
	}
	if lock.Dependencies.Templates[0] != "tpl-a" || lock.Dependencies.Clips[0] != "clip-a" || lock.Dependencies.Subgraphs[0] != "sg-a" {
		t.Fatalf("closure dependencies missing: %+v", lock.Dependencies)
	}
	if len(lock.Dependencies.TargetSlots) != 1 || lock.Dependencies.TargetSlots[0] != "game" {
		t.Fatalf("target slots not derived: %+v", lock.Dependencies.TargetSlots)
	}
	if len(lock.Dependencies.AISlots) != 1 || lock.Dependencies.AISlots[0] != "main" {
		t.Fatalf("ai slots not derived: %+v", lock.Dependencies.AISlots)
	}

	manifest.Version = "0.1.1"
	next, err := BuildYottaLock(manifest, graph, closure, "2026-06-30T00:00:00Z")
	if err != nil {
		t.Fatal(err)
	}
	if next.ManifestHash == lock.ManifestHash {
		t.Fatalf("manifest hash must change when manifest changes")
	}
	if next.GraphHash != lock.GraphHash || next.ClosureHash != lock.ClosureHash {
		t.Fatalf("manifest-only change should not alter graph/closure hashes")
	}
}
