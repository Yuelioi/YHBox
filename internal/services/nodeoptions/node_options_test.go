package nodeoptions

import (
	"slices"
	"testing"
	"time"

	"github.com/yottaapp/yotta/internal/node"
	_ "github.com/yottaapp/yotta/internal/nodes/all"
	"github.com/yottaapp/yotta/internal/services/androidadb"
	"github.com/yottaapp/yotta/internal/services/asset"
	"github.com/yottaapp/yotta/internal/services/container"
)

func TestRegisterAssetAsyncSourcesListsClipsAndSubgraphs(t *testing.T) {
	assetStore, err := asset.NewStore(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	if err := assetStore.PutRecord(asset.AssetRecord{
		GUID:      "clip-1",
		Kind:      asset.KindClip,
		Name:      "Intro clip",
		Category:  "demo",
		Tags:      []string{"a"},
		CreatedAt: time.Now().UTC(),
	}); err != nil {
		t.Fatal(err)
	}
	if err := assetStore.PutRecord(asset.AssetRecord{
		GUID:      "tpl-1",
		Kind:      asset.KindTemplate,
		Name:      "Template",
		CreatedAt: time.Now().UTC(),
	}); err != nil {
		t.Fatal(err)
	}
	assetSvc := asset.NewService(assetStore, nil)

	sgStore, err := container.NewSubgraphStore(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	if err := sgStore.Create(&container.Subgraph{ID: "sg-visible", Label: "Visible"}); err != nil {
		t.Fatal(err)
	}
	if err := sgStore.Create(&container.Subgraph{ID: "sg-anon", Label: "Anon", IsAnonymous: true}); err != nil {
		t.Fatal(err)
	}
	sgSvc := container.NewSubgraphService(sgStore)

	nodeSvc := node.NewService()
	RegisterAssetAsyncSources(nodeSvc, assetSvc, sgSvc)

	clips, err := nodeSvc.AsyncOptions("", "PlayClip", AsyncSourceClipIDs, nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(clips) != 1 || clips[0].Value != "clip-1" || clips[0].Label == "" {
		t.Fatalf("clip options = %+v", clips)
	}

	visible, err := nodeSvc.AsyncOptions("", "Subgraph", AsyncSourceSubgraphIDs, nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(visible) != 1 || visible[0].Value != "sg-visible" {
		t.Fatalf("Subgraph options = %+v", visible)
	}

	all, err := nodeSvc.AsyncOptions("", "CollapsedNode", AsyncSourceSubgraphIDs, nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(all) != 2 {
		t.Fatalf("CollapsedNode options = %+v", all)
	}
}

func TestAllDeclaredAsyncSourcesRegisteredByComposition(t *testing.T) {
	assetStore, err := asset.NewStore(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	sgStore, err := container.NewSubgraphStore(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}

	nodeSvc := node.NewService()
	RegisterAssetAsyncSources(nodeSvc, asset.NewService(assetStore, nil), container.NewSubgraphService(sgStore))
	androidadb.RegisterNodeAsyncSource(nodeSvc, androidadb.NewService(nil))

	registered := map[string]bool{}
	for _, name := range nodeSvc.RegisteredAsyncSources() {
		registered[name] = true
	}

	var missing []string
	for _, rn := range node.All() {
		for _, input := range rn.Spec.Inputs {
			if input.Widget.Kind != "async-dropdown" {
				continue
			}
			source, _ := input.Widget.Props["asyncSource"].(string)
			if source == "" {
				missing = append(missing, rn.Spec.Kind+"."+input.Name+": empty asyncSource")
				continue
			}
			if !registered[source] {
				missing = append(missing, rn.Spec.Kind+"."+input.Name+": "+source)
			}
		}
	}
	slices.Sort(missing)
	if len(missing) > 0 {
		t.Fatalf("declared async sources not registered:\n  %v", missing)
	}
}
