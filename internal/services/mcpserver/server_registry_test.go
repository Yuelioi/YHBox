package mcpserver

import (
	"testing"

	"github.com/yottaapp/yotta/internal/node"
	"github.com/yottaapp/yotta/internal/services/container"
)

type registryTestNode struct{ kind string }

func (n registryTestNode) Spec() node.Spec { return node.Spec{Kind: n.kind} }
func (registryTestNode) Run(node.Ctx, node.Inputs) (node.Outputs, error) { return nil, nil }

func TestNewServerUsesStoreRegistryAsAuthority(t *testing.T) {
	storeRegistry := node.NewRegistry()
	storeRegistry.Register(registryTestNode{kind: "StoreNode"})
	store, err := container.NewStoreWithRegistry(t.TempDir(), storeRegistry)
	if err != nil {
		t.Fatal(err)
	}
	explicit := node.NewRegistry()
	explicit.Register(registryTestNode{kind: "DifferentNode"})

	server := NewServer(Deps{Store: store, Registry: explicit})
	if _, ok := server.deps.Registry.Get("StoreNode"); !ok {
		t.Fatal("server did not use store registry")
	}
	if _, ok := server.deps.Registry.Get("DifferentNode"); ok {
		t.Fatal("server mixed an explicit registry with the store generation")
	}
}
