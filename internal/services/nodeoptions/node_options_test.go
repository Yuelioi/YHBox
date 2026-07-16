package nodeoptions

import (
	"slices"
	"testing"

	"github.com/yottaapp/yotta/internal/node"
	_ "github.com/yottaapp/yotta/internal/nodes/all"
	"github.com/yottaapp/yotta/internal/services/androidadb"
	"github.com/yottaapp/yotta/internal/services/container"
)

func TestRegisterSubgraphAsyncSourceListsSubgraphs(t *testing.T) {
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
	RegisterSubgraphAsyncSource(nodeSvc, sgSvc)

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
	sgStore, err := container.NewSubgraphStore(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}

	nodeSvc := node.NewService()
	RegisterSubgraphAsyncSource(nodeSvc, container.NewSubgraphService(sgStore))
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
