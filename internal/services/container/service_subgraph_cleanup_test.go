package container

import "testing"

func TestSubgraphServiceCleanupOnlyDeletesSelectedUnusedBlueprints(t *testing.T) {
	store, _ := newTestSubgraphStore(t)
	for _, sg := range []*Subgraph{
		{ID: "unused", Label: "Unused"},
		{ID: "used", Label: "Used"},
		{ID: "anonymous", Label: "Anonymous", IsAnonymous: true},
	} {
		if err := store.Create(sg); err != nil {
			t.Fatal(err)
		}
	}

	refs := map[string]int{"used": 1}
	svc := NewSubgraphService(store)
	ConfigureSubgraphReferrerScanner(svc, func(id string) []SubgraphReferrer {
		out := make([]SubgraphReferrer, refs[id])
		return out
	})

	preview := svc.PreviewCleanup()
	if len(preview.Unused) != 1 || preview.Unused[0].ID != "unused" {
		t.Fatalf("unused preview = %+v", preview.Unused)
	}
	if len(preview.Referenced) != 1 || preview.Referenced[0].ID != "used" {
		t.Fatalf("referenced preview = %+v", preview.Referenced)
	}

	refs["unused"] = 1
	result := svc.CleanupUnused(CleanupArgs{IDs: []string{"unused"}})
	if len(result.Skipped) != 1 || result.Skipped[0].ID != "unused" {
		t.Fatalf("cleanup should recheck references: %+v", result)
	}
	if _, ok := store.Get("unused"); !ok {
		t.Fatal("newly referenced blueprint was deleted")
	}

	refs["unused"] = 0
	result = svc.CleanupUnused(CleanupArgs{IDs: []string{"unused"}})
	if len(result.Deleted) != 1 || result.Deleted[0] != "unused" {
		t.Fatalf("cleanup result = %+v", result)
	}
	if _, ok := store.Get("unused"); ok {
		t.Fatal("unused blueprint still exists")
	}
	if _, ok := store.Get("anonymous"); !ok {
		t.Fatal("anonymous implementation subgraph must not be exposed to cleanup")
	}
}
