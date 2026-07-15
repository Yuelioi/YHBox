package capability

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"regexp"
	"sort"
	"time"

	"github.com/yottaapp/yotta/internal/artifact"
	"github.com/yottaapp/yotta/internal/runid"
)

const (
	RunGrantFormat       = "yotta.run-grant"
	RunGrantVersion      = "3.1"
	MaxRunGrantBytes     = 4 << 20
	runGrantDigestDomain = "yotta/run-grant/v1"
)

var authorityIDPattern = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9._:/-]{0,127}$`)

type DefinitionCatalog interface {
	LookupCapability(string) (Definition, bool)
}

type Binding struct {
	GraphID             string `json:"graphId"`
	NodeID              string `json:"nodeId"`
	RequirementID       string `json:"requirementId"`
	ProviderID          string `json:"providerId"`
	TargetID            string `json:"targetId"`
	TargetKind          string `json:"targetKind"`
	ResourceKind        string `json:"resourceKind"`
	PluginInstanceID    string `json:"pluginInstanceId"`
	SessionID           string `json:"sessionId"`
	CredentialBindingID string `json:"credentialBindingId,omitempty"`
}

type GrantEntry struct {
	Binding    Binding         `json:"binding"`
	Capability Ref             `json:"capability"`
	Operations []string        `json:"operations"`
	Scope      json.RawMessage `json:"scope"`
}

type GrantRequest struct {
	ProgramHash      artifact.Digest
	Plan             Plan
	RunID            string
	Principal        string
	PolicyGeneration string
	IssuedAt         time.Time
	ExpiresAt        time.Time
	Bindings         []Binding
	ConsentLineage   []artifact.Digest
}

type runGrantDocument struct {
	Format             string            `json:"format"`
	Version            string            `json:"version"`
	GrantDigest        artifact.Digest   `json:"grantDigest"`
	ProgramHash        artifact.Digest   `json:"programHash"`
	CapabilityPlanHash artifact.Digest   `json:"capabilityPlanDigest"`
	RunID              string            `json:"runId"`
	Principal          string            `json:"principal"`
	PolicyGeneration   string            `json:"policyGeneration"`
	IssuedAt           time.Time         `json:"issuedAt"`
	ExpiresAt          time.Time         `json:"expiresAt"`
	Entries            []GrantEntry      `json:"entries"`
	ConsentLineage     []artifact.Digest `json:"consentLineage"`
}

type runGrantState struct {
	document runGrantDocument
	bytes    []byte
}

type RunGrant struct{ state *runGrantState }

func SealRunGrant(request GrantRequest, catalog DefinitionCatalog) (RunGrant, error) {
	if !request.ProgramHash.Valid() || !request.Plan.Valid() || runid.Validate(request.RunID) != nil || !authorityIDPattern.MatchString(request.Principal) ||
		!authorityIDPattern.MatchString(request.PolicyGeneration) || request.IssuedAt.Location() != time.UTC || request.ExpiresAt.Location() != time.UTC ||
		!request.ExpiresAt.After(request.IssuedAt) || catalog == nil {
		return RunGrant{}, errors.New("invalid run grant identity or lifetime")
	}
	planEntries := request.Plan.Entries()
	if len(request.Bindings) != len(planEntries) {
		return RunGrant{}, errors.New("run grant must bind every planned requirement exactly once")
	}
	bindings := make(map[string]Binding, len(request.Bindings))
	for _, binding := range request.Bindings {
		key, err := validateBinding(binding)
		if err != nil {
			return RunGrant{}, err
		}
		if _, duplicate := bindings[key]; duplicate {
			return RunGrant{}, errors.New("duplicate run grant binding")
		}
		bindings[key] = binding
	}
	entries := make([]GrantEntry, 0, len(planEntries))
	for _, planned := range planEntries {
		key := planEntryKey(planned.GraphID, planned.NodeID, planned.Requirement.ID)
		binding, ok := bindings[key]
		if !ok {
			return RunGrant{}, fmt.Errorf("missing run grant binding for %s", key)
		}
		definition, ok := catalog.LookupCapability(planned.Requirement.Capability.CapabilityID)
		if !ok || definition.Ref() != planned.Requirement.Capability {
			return RunGrant{}, errors.New("run grant references an untrusted capability definition")
		}
		normalized, err := definition.NormalizeRequirement(planned.Requirement)
		if err != nil {
			return RunGrant{}, err
		}
		machine := definition.Machine()
		if !contains(machine.TargetKinds, binding.TargetKind) {
			return RunGrant{}, fmt.Errorf("target kind %q is not allowed by capability", binding.TargetKind)
		}
		if machine.Credential == CredentialRequired && binding.CredentialBindingID == "" ||
			machine.Credential == CredentialNone && binding.CredentialBindingID != "" {
			return RunGrant{}, errors.New("credential binding does not match capability definition")
		}
		entries = append(entries, GrantEntry{
			Binding: binding, Capability: normalized.Capability,
			Operations: append([]string(nil), normalized.Operations...), Scope: append([]byte(nil), normalized.Scope...),
		})
		delete(bindings, key)
	}
	if len(bindings) != 0 {
		return RunGrant{}, errors.New("run grant contains an unplanned binding")
	}
	lineage := append([]artifact.Digest(nil), request.ConsentLineage...)
	for _, digest := range lineage {
		if !digest.Valid() {
			return RunGrant{}, errors.New("invalid consent lineage digest")
		}
	}
	sort.Slice(lineage, func(i, j int) bool { return lineage[i] < lineage[j] })
	for index := 1; index < len(lineage); index++ {
		if lineage[index-1] == lineage[index] {
			return RunGrant{}, errors.New("duplicate consent lineage digest")
		}
	}
	document := runGrantDocument{
		Format: RunGrantFormat, Version: RunGrantVersion, ProgramHash: request.ProgramHash,
		CapabilityPlanHash: request.Plan.Digest(), RunID: request.RunID, Principal: request.Principal,
		PolicyGeneration: request.PolicyGeneration, IssuedAt: request.IssuedAt, ExpiresAt: request.ExpiresAt,
		Entries: entries, ConsentLineage: lineage,
	}
	return sealRunGrantDocument(document)
}

func OpenRunGrant(raw []byte, plan Plan, catalog DefinitionCatalog) (RunGrant, error) {
	if len(raw) == 0 || len(raw) > MaxRunGrantBytes || !plan.Valid() {
		return RunGrant{}, errors.New("run grant exceeds byte budget or has no trusted plan")
	}
	if err := artifact.InspectJSONBudget(raw, 96, 524288, 1<<20); err != nil {
		return RunGrant{}, err
	}
	canonical, err := artifact.Canonicalize(raw)
	if err != nil || !bytes.Equal(canonical, raw) {
		return RunGrant{}, errors.New("run grant is not canonical")
	}
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.DisallowUnknownFields()
	var document runGrantDocument
	if err := decoder.Decode(&document); err != nil {
		return RunGrant{}, fmt.Errorf("decode run grant: %w", err)
	}
	var trailing any
	if err := decoder.Decode(&trailing); !errors.Is(err, io.EOF) {
		return RunGrant{}, errors.New("run grant contains trailing values")
	}
	if document.Format != RunGrantFormat || document.Version != RunGrantVersion || document.CapabilityPlanHash != plan.Digest() {
		return RunGrant{}, errors.New("unsupported run grant or plan mismatch")
	}
	bindings := make([]Binding, 0, len(document.Entries))
	for _, entry := range document.Entries {
		bindings = append(bindings, entry.Binding)
	}
	sealed, err := SealRunGrant(GrantRequest{
		ProgramHash: document.ProgramHash, Plan: plan, RunID: document.RunID, Principal: document.Principal,
		PolicyGeneration: document.PolicyGeneration, IssuedAt: document.IssuedAt, ExpiresAt: document.ExpiresAt,
		Bindings: bindings, ConsentLineage: document.ConsentLineage,
	}, catalog)
	if err != nil {
		return RunGrant{}, err
	}
	if sealed.Digest() != document.GrantDigest || !bytes.Equal(sealed.Bytes(), raw) {
		return RunGrant{}, errors.New("run grant digest mismatch")
	}
	return sealed, nil
}

func sealRunGrantDocument(document runGrantDocument) (RunGrant, error) {
	body := document
	body.GrantDigest = ""
	bodyBytes, err := artifact.Marshal(body)
	if err != nil {
		return RunGrant{}, err
	}
	digest, err := artifact.Sum(runGrantDigestDomain, bodyBytes)
	if err != nil {
		return RunGrant{}, err
	}
	document.GrantDigest = digest
	raw, err := artifact.Marshal(document)
	if err != nil {
		return RunGrant{}, err
	}
	if len(raw) > MaxRunGrantBytes {
		return RunGrant{}, errors.New("run grant exceeds byte budget")
	}
	return RunGrant{state: &runGrantState{document: document, bytes: raw}}, nil
}

func validateBinding(binding Binding) (string, error) {
	if !validAttributionID(binding.GraphID) || !validAttributionID(binding.NodeID) || !identifierPattern.MatchString(binding.RequirementID) ||
		!authorityIDPattern.MatchString(binding.ProviderID) || !authorityIDPattern.MatchString(binding.TargetID) || !identifierPattern.MatchString(binding.TargetKind) ||
		!authorityIDPattern.MatchString(binding.ResourceKind) || !authorityIDPattern.MatchString(binding.PluginInstanceID) || !authorityIDPattern.MatchString(binding.SessionID) ||
		binding.CredentialBindingID != "" && !authorityIDPattern.MatchString(binding.CredentialBindingID) {
		return "", errors.New("invalid run grant binding")
	}
	return planEntryKey(binding.GraphID, binding.NodeID, binding.RequirementID), nil
}

func planEntryKey(graphID, nodeID, requirementID string) string {
	return graphID + "\x00" + nodeID + "\x00" + requirementID
}

func contains(values []string, wanted string) bool {
	index := sort.SearchStrings(values, wanted)
	return index < len(values) && values[index] == wanted
}

func (g RunGrant) Valid() bool { return g.state != nil && g.state.document.GrantDigest.Valid() }
func (g RunGrant) Digest() artifact.Digest {
	if !g.Valid() {
		return ""
	}
	return g.state.document.GrantDigest
}
func (g RunGrant) ProgramHash() artifact.Digest {
	if !g.Valid() {
		return ""
	}
	return g.state.document.ProgramHash
}
func (g RunGrant) PlanDigest() artifact.Digest {
	if !g.Valid() {
		return ""
	}
	return g.state.document.CapabilityPlanHash
}
func (g RunGrant) RunID() string {
	if !g.Valid() {
		return ""
	}
	return g.state.document.RunID
}
func (g RunGrant) Principal() string {
	if !g.Valid() {
		return ""
	}
	return g.state.document.Principal
}
func (g RunGrant) PolicyGeneration() string {
	if !g.Valid() {
		return ""
	}
	return g.state.document.PolicyGeneration
}
func (g RunGrant) ExpiresAt() time.Time {
	if !g.Valid() {
		return time.Time{}
	}
	return g.state.document.ExpiresAt
}
func (g RunGrant) Bytes() []byte {
	if !g.Valid() {
		return nil
	}
	return append([]byte(nil), g.state.bytes...)
}
func (g RunGrant) Entries() []GrantEntry {
	if !g.Valid() {
		return nil
	}
	result := make([]GrantEntry, len(g.state.document.Entries))
	for index, entry := range g.state.document.Entries {
		result[index] = GrantEntry{
			Binding: entry.Binding, Capability: entry.Capability,
			Operations: append([]string(nil), entry.Operations...), Scope: append([]byte(nil), entry.Scope...),
		}
	}
	return result
}
