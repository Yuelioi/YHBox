package main

import (
	"testing"

	"github.com/yottaapp/yotta/internal/node"
	"github.com/yottaapp/yotta/internal/nodes/control"
	"github.com/yottaapp/yotta/internal/services/container"
)

type assetDependencyNode struct{}

func (assetDependencyNode) Spec() node.Spec {
	return node.Spec{Kind: "AssetDependencyNode", Inputs: []node.InputSpec{{Name: "In", Type: node.TypeExec}}, Outputs: []node.OutputSpec{{Name: "Done", Type: node.TypeExec}}}
}
func (assetDependencyNode) Run(node.Ctx, node.Inputs) (node.Outputs, error) { return nil, nil }
func (assetDependencyNode) Dependencies(node.Inputs) []node.Dependency {
	return []node.Dependency{{Kind: "template", Key: "template-custom"}}
}

func TestScanAssetReferrersUsesStoreRegistry(t *testing.T) {
	registry := node.NewRegistry()
	registry.Register(&control.Start{})
	registry.Register(assetDependencyNode{})
	store, err := container.NewStoreWithRegistry(t.TempDir(), registry)
	if err != nil {
		t.Fatal(err)
	}
	if err := store.Save(&container.Container{SchemaVersion: 1, ID: "custom", Name: "custom", Graph: container.Graph{
		Nodes: []container.GraphNode{{ID: "start", Kind: "Start"}, {ID: "asset", Kind: "AssetDependencyNode"}},
		Edges: []container.GraphEdge{{From: "start.Done", To: "asset.In"}},
	}}); err != nil {
		t.Fatal(err)
	}
	subgraphs, err := container.NewSubgraphStore(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	refs := scanAssetReferrers(store, subgraphs)("template-custom")
	if len(refs) != 1 || refs[0].NodeKind != "AssetDependencyNode" {
		t.Fatalf("custom dependency refs = %+v", refs)
	}
}
