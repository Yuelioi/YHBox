package appbootstrap

import (
	"context"
	"testing"

	"github.com/yottaapp/yotta/internal/nodes"
	"github.com/yottaapp/yotta/internal/workflow/schema"
)

func TestProjectWorkflowCapabilityScopesResolvesExactDynamicSlotAndAuthority(t *testing.T) {
	builtins, err := nodes.Build()
	if err != nil {
		t.Fatal(err)
	}
	entry, found := builtins.Catalog.Lookup(nodes.HTTPGetNodeID)
	if !found {
		t.Fatal("HTTP GET node is missing")
	}
	source := schema.WorkflowSource{
		TargetDefaults: []schema.TargetDefault{{Target: "origin", Slot: "reporting-origin"}},
		Graphs: []schema.Graph{{
			ID: "main", Kind: schema.GraphKindMain,
			Nodes: []schema.Node{{
				ID: "request", NodeRef: entry.Contract.NodeRef(),
				Config: map[string]any{}, Bindings: map[string]schema.InputBinding{},
			}},
		}},
	}
	scopes, err := projectWorkflowCapabilityScopes(
		context.Background(), source, builtins.Catalog, nil,
	)
	if err != nil || len(scopes) != 1 {
		t.Fatalf("projectWorkflowCapabilityScopes() = %#v, %v", scopes, err)
	}
	scope := scopes[0]
	if scope.NodeTypeID != nodes.HTTPGetNodeID ||
		scope.CapabilityID != nodes.HTTPGetCapabilityID ||
		len(scope.Operations) != 1 || scope.Operations[0] != "get" ||
		scope.TargetSlot != "reporting-origin" || scope.CredentialSlot != "" ||
		scope.Scope != `{"method":"GET"}` || scope.Risk != "sensitive" || scope.Consent != "none" {
		t.Fatalf("capability scope = %#v", scope)
	}
}
