package capability_test

import (
	"bytes"
	"testing"
	"time"

	"github.com/yottaapp/yotta/internal/artifact"
	"github.com/yottaapp/yotta/internal/capability"
)

type definitions map[string]capability.Definition

const testRunID = "0190c7d4-1e40-7cc5-a783-57b16d5c8e3a"

func (catalog definitions) LookupCapability(id string) (capability.Definition, bool) {
	definition, ok := catalog[id]
	return definition, ok
}

func TestRunGrantBindsExactPlanAndContainsNoBearerAuthority(t *testing.T) {
	definition := testDefinition(t)
	plan := testPlan(t, definition)
	issued := time.Date(2026, 7, 15, 1, 0, 0, 0, time.UTC)
	grant, err := capability.SealRunGrant(capability.GrantRequest{
		ProgramHash: testArtifactDigest(t, "program"), Plan: plan, RunID: testRunID, Principal: "user-1",
		PolicyGeneration: "policy-1", IssuedAt: issued, ExpiresAt: issued.Add(time.Minute),
		Bindings: []capability.Binding{{
			GraphID: "main", NodeID: "node-1", RequirementID: "source",
			ProviderID: "blob", TargetID: "workspace", TargetKind: "blob-store", ResourceKind: "blob/session",
			PluginInstanceID: "builtin", SessionID: "session-1",
		}},
	}, definitions{definition.Ref().CapabilityID: definition})
	if err != nil {
		t.Fatal(err)
	}
	opened, err := capability.OpenRunGrant(grant.Bytes(), plan, definitions{definition.Ref().CapabilityID: definition})
	if err != nil {
		t.Fatal(err)
	}
	if opened.Digest() != grant.Digest() || !bytes.Equal(opened.Bytes(), grant.Bytes()) {
		t.Fatal("grant identity changed after strict open")
	}
	if bytes.Contains(grant.Bytes(), []byte("token")) || bytes.Contains(grant.Bytes(), []byte("secret")) {
		t.Fatalf("grant projection contains bearer authority: %s", grant.Bytes())
	}
}

func TestRunGrantRejectsMissingOrWrongBindings(t *testing.T) {
	definition := testDefinition(t)
	plan := testPlan(t, definition)
	now := time.Date(2026, 7, 15, 1, 0, 0, 0, time.UTC)
	base := capability.GrantRequest{
		ProgramHash: testArtifactDigest(t, "program"), Plan: plan, RunID: testRunID, Principal: "user-1",
		PolicyGeneration: "policy-1", IssuedAt: now, ExpiresAt: now.Add(time.Minute),
	}
	catalog := definitions{definition.Ref().CapabilityID: definition}
	if _, err := capability.SealRunGrant(base, catalog); err == nil {
		t.Fatal("accepted a grant missing a planned binding")
	}
	base.Bindings = []capability.Binding{{
		GraphID: "main", NodeID: "node-1", RequirementID: "source", ProviderID: "blob",
		TargetID: "workspace", TargetKind: "wrong-kind", ResourceKind: "blob/session", PluginInstanceID: "builtin", SessionID: "session-1",
	}}
	if _, err := capability.SealRunGrant(base, catalog); err == nil {
		t.Fatal("accepted a target kind outside the capability definition")
	}
}

func testPlan(t *testing.T, definition capability.Definition) capability.Plan {
	t.Helper()
	requirement, err := definition.NormalizeRequirement(capability.Requirement{
		ID: "source", Capability: definition.Ref(), Operations: []string{"read"}, TargetSlot: "input",
		Scope: []byte(`{"root":"workspace"}`),
	})
	if err != nil {
		t.Fatal(err)
	}
	plan, err := capability.SealPlan([]capability.PlanEntry{{GraphID: "main", NodeID: "node-1", Requirement: requirement}})
	if err != nil {
		t.Fatal(err)
	}
	return plan
}

func testArtifactDigest(t *testing.T, label string) artifact.Digest {
	t.Helper()
	digest, err := artifact.Sum("test/run-grant/v1", []byte(label))
	if err != nil {
		t.Fatal(err)
	}
	return digest
}
