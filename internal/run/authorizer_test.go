package run_test

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/yottaapp/yotta/internal/artifact"
	"github.com/yottaapp/yotta/internal/capability"
	"github.com/yottaapp/yotta/internal/datatype"
	"github.com/yottaapp/yotta/internal/resource"
	run31 "github.com/yottaapp/yotta/internal/run"
	"github.com/yottaapp/yotta/internal/stream"
)

type catalog map[string]capability.Definition

const testRunID = "0190c7d4-1e40-7cc5-a783-57b16d5c8e3a"

func (c catalog) LookupCapability(id string) (capability.Definition, bool) {
	definition, ok := c[id]
	return definition, ok
}

func TestGrantAuthorizerDrivesBrokerAndRevokesCalls(t *testing.T) {
	now := time.Date(2026, 7, 15, 1, 0, 0, 0, time.UTC)
	definition := streamCapability(t)
	plan := streamPlan(t, definition)
	grant, err := capability.SealRunGrant(capability.GrantRequest{
		ProgramHash: digest("program"), Plan: plan, RunID: testRunID, Principal: "user-1", PolicyGeneration: "policy-1",
		IssuedAt: now, ExpiresAt: now.Add(time.Minute), Bindings: []capability.Binding{{
			GraphID: "main", NodeID: "producer", RequirementID: "stream", ProviderID: stream.ProviderID,
			TargetID: "memory", TargetKind: "stream-session", ResourceKind: stream.Kind,
			PluginInstanceID: "builtin", SessionID: "session-1",
		}},
	}, catalog{definition.Ref().CapabilityID: definition})
	if err != nil {
		t.Fatal(err)
	}
	authorizer, err := run31.NewGrantAuthorizer(grant, func() time.Time { return now })
	if err != nil {
		t.Fatal(err)
	}
	provider, err := stream.NewProvider(stream.Limits{MaxCapacity: 2, MaxChunkBytes: 32})
	if err != nil {
		t.Fatal(err)
	}
	broker, err := resource.New(authorizer, map[string]resource.Provider{stream.ProviderID: provider}, resource.Options{Now: func() time.Time { return now }})
	if err != nil {
		t.Fatal(err)
	}
	scope := resource.Scope{
		ProgramHash: grant.ProgramHash(), CapabilityPlanDigest: grant.PlanDigest(), GrantDigest: grant.Digest(),
		PolicyGeneration: grant.PolicyGeneration(), RunID: grant.RunID(), Principal: grant.Principal(),
		PluginInstanceID: "builtin", SessionID: "session-1", GraphID: "main", NodeID: "producer",
		RequirementID: "stream", InvocationID: "invoke-1",
	}
	config, _ := json.Marshal(stream.Config{Capacity: 1, MaxChunkBytes: 16})
	handle, err := broker.Open(context.Background(), resource.OpenRequest{
		Scope: scope, ProviderID: stream.ProviderID, TargetID: "memory", Kind: stream.Kind,
		Operations: []string{stream.OperationSend, stream.OperationCancel}, ExpiresAt: now.Add(30 * time.Second), Config: config,
	})
	if err != nil {
		t.Fatal(err)
	}
	authorizer.Revoke()
	if _, err := broker.Invoke(context.Background(), resource.Call{Scope: scope, Handle: handle, Operation: stream.OperationSend, Payload: []byte("x")}); !errors.Is(err, run31.ErrGrantDenied) {
		t.Fatalf("call after grant revoke = %v", err)
	}
	if err := broker.RevokeRun(context.Background(), grant.RunID()); err != nil {
		t.Fatal(err)
	}
}

func TestGrantAuthorizerRejectsWrongRequirementBeforeProviderOpen(t *testing.T) {
	// The full open path is covered above; a forged requirement cannot select
	// another entry even when the provider/target strings happen to match.
	authorizer, grant, now := grantAuthorizer(t)
	scope := resource.Scope{
		ProgramHash: grant.ProgramHash(), CapabilityPlanDigest: grant.PlanDigest(), GrantDigest: grant.Digest(), PolicyGeneration: grant.PolicyGeneration(),
		RunID: grant.RunID(), Principal: grant.Principal(), PluginInstanceID: "builtin", SessionID: "session-1",
		GraphID: "main", NodeID: "producer", RequirementID: "forged", InvocationID: "invoke-1",
	}
	_, err := authorizer.AuthorizeOpen(context.Background(), resource.OpenRequest{
		Scope: scope, ProviderID: stream.ProviderID, TargetID: "memory", Kind: stream.Kind,
		Operations: []string{stream.OperationSend}, ExpiresAt: now.Add(time.Second),
	})
	if !errors.Is(err, run31.ErrGrantDenied) {
		t.Fatalf("forged requirement = %v", err)
	}
}

func TestGrantAuthorizerReturnsCanonicalPlannedScope(t *testing.T) {
	authorizer, _, now := grantAuthorizer(t)
	scope, err := authorizer.Scope("main", "producer", "stream", "invoke-1")
	if err != nil {
		t.Fatal(err)
	}
	authorization, err := authorizer.AuthorizeOpen(context.Background(), resource.OpenRequest{
		Scope: scope, ProviderID: stream.ProviderID, TargetID: "memory", Kind: stream.Kind,
		Operations: []string{stream.OperationSend}, ExpiresAt: now.Add(time.Second),
	})
	if err != nil {
		t.Fatal(err)
	}
	if string(authorization.CapabilityScope) != `{}` || authorization.CredentialBindingID != "" {
		t.Fatalf("authorization = %s / %q", authorization.CapabilityScope, authorization.CredentialBindingID)
	}
}

func TestGrantAuthorizerRejectsBorrowAcrossDifferentCanonicalScopes(t *testing.T) {
	now := time.Date(2026, 7, 15, 1, 0, 0, 0, time.UTC)
	definition := streamCapability(t)
	requirement := func(id, lane string) capability.Requirement {
		t.Helper()
		result, err := definition.NormalizeRequirement(capability.Requirement{
			ID: id, Capability: definition.Ref(), Operations: []string{stream.OperationSend}, TargetSlot: "stream",
			Scope: []byte(`{"lane":"` + lane + `"}`),
		})
		if err != nil {
			t.Fatal(err)
		}
		return result
	}
	plan, err := capability.SealPlan([]capability.PlanEntry{
		{GraphID: "main", NodeID: "owner", Requirement: requirement("stream-owner", "wide")},
		{GraphID: "main", NodeID: "borrower", Requirement: requirement("stream-borrower", "narrow")},
	})
	if err != nil {
		t.Fatal(err)
	}
	binding := func(nodeID, requirementID string) capability.Binding {
		return capability.Binding{
			GraphID: "main", NodeID: nodeID, RequirementID: requirementID, ProviderID: stream.ProviderID,
			TargetID: "memory", TargetKind: "stream-session", ResourceKind: stream.Kind,
			PluginInstanceID: "builtin", SessionID: "session-1",
		}
	}
	grant, err := capability.SealRunGrant(capability.GrantRequest{
		ProgramHash: digest("program"), Plan: plan, RunID: testRunID, Principal: "user-1", PolicyGeneration: "policy-1",
		IssuedAt: now, ExpiresAt: now.Add(time.Minute), Bindings: []capability.Binding{
			binding("owner", "stream-owner"), binding("borrower", "stream-borrower"),
		},
	}, catalog{definition.Ref().CapabilityID: definition})
	if err != nil {
		t.Fatal(err)
	}
	authorizer, err := run31.NewGrantAuthorizer(grant, func() time.Time { return now })
	if err != nil {
		t.Fatal(err)
	}
	ownerScope, err := authorizer.Scope("main", "owner", "stream-owner", "invoke-owner")
	if err != nil {
		t.Fatal(err)
	}
	borrowerScope, err := authorizer.Scope("main", "borrower", "stream-borrower", "invoke-borrower")
	if err != nil {
		t.Fatal(err)
	}
	err = authorizer.AuthorizeBorrow(context.Background(), resource.BorrowRequest{
		Owner: ownerScope, Borrower: borrowerScope, ProviderID: stream.ProviderID, TargetID: "memory", Kind: stream.Kind,
		Operations: []string{stream.OperationSend}, ExpiresAt: now.Add(30 * time.Second),
	})
	if !errors.Is(err, run31.ErrGrantDenied) {
		t.Fatalf("cross-scope borrow = %v", err)
	}
}

func grantAuthorizer(t *testing.T) (*run31.GrantAuthorizer, capability.RunGrant, time.Time) {
	t.Helper()
	now := time.Date(2026, 7, 15, 1, 0, 0, 0, time.UTC)
	definition := streamCapability(t)
	plan := streamPlan(t, definition)
	grant, err := capability.SealRunGrant(capability.GrantRequest{
		ProgramHash: digest("program"), Plan: plan, RunID: testRunID, Principal: "user-1", PolicyGeneration: "policy-1",
		IssuedAt: now, ExpiresAt: now.Add(time.Minute), Bindings: []capability.Binding{{
			GraphID: "main", NodeID: "producer", RequirementID: "stream", ProviderID: stream.ProviderID, TargetID: "memory",
			TargetKind: "stream-session", ResourceKind: stream.Kind, PluginInstanceID: "builtin", SessionID: "session-1",
		}},
	}, catalog{definition.Ref().CapabilityID: definition})
	if err != nil {
		t.Fatal(err)
	}
	authorizer, err := run31.NewGrantAuthorizer(grant, func() time.Time { return now })
	if err != nil {
		t.Fatal(err)
	}
	return authorizer, grant, now
}

func streamCapability(t *testing.T) capability.Definition {
	t.Helper()
	const id = "https://schemas.yotta.dev/capabilities/stream/session/v1"
	definition, err := capability.SealDefinition(capability.DefinitionDraft{
		CapabilityID: id, Operations: []string{stream.OperationCancel, stream.OperationSend}, TargetKinds: []string{"stream-session"},
		ScopeSchemaRoot: id + "/scope", ScopeSchemaBundle: []datatype.SchemaResource{{ID: id + "/scope", Schema: []byte(`{"$id":"https://schemas.yotta.dev/capabilities/stream/session/v1/scope","$schema":"https://json-schema.org/draft/2020-12/schema","type":"object","properties":{"lane":{"type":"string"}},"additionalProperties":false}`)}},
		Credential: capability.CredentialNone, Risk: capability.RiskLow, Consent: capability.ConsentNone,
		ProviderABI: "https://schemas.yotta.dev/provider-abi/resource/v1",
	})
	if err != nil {
		t.Fatal(err)
	}
	return definition
}

func streamPlan(t *testing.T, definition capability.Definition) capability.Plan {
	t.Helper()
	requirement, err := definition.NormalizeRequirement(capability.Requirement{
		ID: "stream", Capability: definition.Ref(), Operations: []string{stream.OperationCancel, stream.OperationSend}, TargetSlot: "stream", Scope: []byte(`{}`),
	})
	if err != nil {
		t.Fatal(err)
	}
	plan, err := capability.SealPlan([]capability.PlanEntry{{GraphID: "main", NodeID: "producer", Requirement: requirement}})
	if err != nil {
		t.Fatal(err)
	}
	return plan
}

func digest(label string) artifact.Digest {
	digest, _ := artifact.Sum("test/run-authorizer/v1", []byte(strings.TrimSpace(label)))
	return digest
}
