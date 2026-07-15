package capability_test

import (
	"bytes"
	"encoding/json"
	"testing"

	"github.com/yottaapp/yotta/internal/capability"
	"github.com/yottaapp/yotta/internal/datatype"
)

func TestDefinitionRequirementAndPlanRoundTrip(t *testing.T) {
	definition := testDefinition(t)
	requirement := capability.Requirement{
		ID: "source", Capability: definition.Ref(), Operations: []string{"read"},
		TargetSlot: "input", Scope: json.RawMessage(`{"root":"workspace"}`),
	}
	normalized, err := definition.NormalizeRequirement(requirement)
	if err != nil {
		t.Fatal(err)
	}
	plan, err := capability.SealPlan([]capability.PlanEntry{{GraphID: "main", NodeID: "node-1", Requirement: normalized}})
	if err != nil {
		t.Fatal(err)
	}
	opened, err := capability.OpenPlan(plan.Bytes())
	if err != nil {
		t.Fatal(err)
	}
	if opened.Digest() != plan.Digest() || !bytes.Equal(opened.Bytes(), plan.Bytes()) {
		t.Fatal("plan identity changed after strict open")
	}
	definitionOpened, err := capability.OpenDefinition(definition.Bytes())
	if err != nil || definitionOpened.Ref() != definition.Ref() {
		t.Fatalf("definition open = %#v, %v", definitionOpened.Ref(), err)
	}
}

func TestDefinitionRejectsAuthorityWidening(t *testing.T) {
	definition := testDefinition(t)
	tests := []capability.Requirement{
		{ID: "source", Capability: definition.Ref(), Operations: []string{"write"}, TargetSlot: "input", Scope: json.RawMessage(`{"root":"workspace"}`)},
		{ID: "source", Capability: definition.Ref(), Operations: []string{"read"}, TargetSlot: "input", CredentialSlot: "secret", Scope: json.RawMessage(`{"root":"workspace"}`)},
		{ID: "source", Capability: definition.Ref(), Operations: []string{"read"}, TargetSlot: "input", Scope: json.RawMessage(`{"root":3}`)},
	}
	for _, requirement := range tests {
		if _, err := definition.NormalizeRequirement(requirement); err == nil {
			t.Fatalf("accepted widened requirement %#v", requirement)
		}
	}
}

func TestPlanRejectsDuplicateAttributionAndTampering(t *testing.T) {
	definition := testDefinition(t)
	requirement, err := definition.NormalizeRequirement(capability.Requirement{
		ID: "source", Capability: definition.Ref(), Operations: []string{"read"}, TargetSlot: "input", Scope: json.RawMessage(`{"root":"workspace"}`),
	})
	if err != nil {
		t.Fatal(err)
	}
	entry := capability.PlanEntry{GraphID: "main", NodeID: "node-1", Requirement: requirement}
	if _, err := capability.SealPlan([]capability.PlanEntry{entry, entry}); err == nil {
		t.Fatal("accepted duplicate attributed requirement")
	}
	plan, err := capability.SealPlan([]capability.PlanEntry{entry})
	if err != nil {
		t.Fatal(err)
	}
	tampered := bytes.Replace(plan.Bytes(), []byte(`"read"`), []byte(`"reed"`), 1)
	if _, err := capability.OpenPlan(tampered); err == nil {
		t.Fatal("accepted tampered plan")
	}
}

func testDefinition(t *testing.T) capability.Definition {
	t.Helper()
	const id = "https://schemas.yotta.dev/capabilities/blob/read/v1"
	definition, err := capability.SealDefinition(capability.DefinitionDraft{
		CapabilityID:    id,
		Operations:      []string{"read"},
		TargetKinds:     []string{"blob-store"},
		ScopeSchemaRoot: id + "/scope",
		ScopeSchemaBundle: []datatype.SchemaResource{{ID: id + "/scope", Schema: json.RawMessage(`{
			"$id":"https://schemas.yotta.dev/capabilities/blob/read/v1/scope",
			"$schema":"https://json-schema.org/draft/2020-12/schema",
			"type":"object","properties":{"root":{"const":"workspace"}},"required":["root"],"additionalProperties":false
		}`)}},
		Credential: capability.CredentialNone, Risk: capability.RiskLow,
		Consent: capability.ConsentNone, ProviderABI: "https://schemas.yotta.dev/provider-abi/resource/v1",
	})
	if err != nil {
		t.Fatal(err)
	}
	return definition
}
