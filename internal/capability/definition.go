// Package capability owns capability definitions, attributed
// requirements, and immutable least-privilege plans.
package capability

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/url"
	"regexp"
	"sort"
	"strings"
	"unicode"

	runtimejsonschema "github.com/santhosh-tekuri/jsonschema/v6"
	"github.com/yottaapp/yotta/internal/artifact"
	"github.com/yottaapp/yotta/internal/contractschema"
	"github.com/yottaapp/yotta/internal/datatype"
)

const (
	DefinitionFormat       = "yotta.capability-definition"
	DefinitionVersion      = "1"
	MaxDefinitionBytes     = 1 << 20
	definitionDigestDomain = "yotta/capability-definition/v1"
)

var (
	operationPattern  = regexp.MustCompile(`^[a-z][a-z0-9._/-]{0,63}$`)
	identifierPattern = regexp.MustCompile(`^[a-z][a-z0-9]*(?:[-_][a-z0-9]+)*$`)
	versionPattern    = regexp.MustCompile(`^v[1-9][0-9]*$`)
)

type Ref struct {
	CapabilityID   string          `json:"capabilityId"`
	SemanticDigest artifact.Digest `json:"semanticDigest"`
}

func (r Ref) Validate() error {
	if err := validateVersionedURI(r.CapabilityID); err != nil || !r.SemanticDigest.Valid() {
		return errors.New("invalid capability reference")
	}
	return nil
}

type CredentialMode string

const (
	CredentialNone     CredentialMode = "none"
	CredentialRequired CredentialMode = "required"
)

type RiskClass string

const (
	RiskLow       RiskClass = "low"
	RiskSensitive RiskClass = "sensitive"
	RiskDangerous RiskClass = "dangerous"
)

type ConsentClass string

const (
	ConsentNone     ConsentClass = "none"
	ConsentOnce     ConsentClass = "once"
	ConsentEveryRun ConsentClass = "every-run"
)

type DefinitionDraft struct {
	CapabilityID      string
	Operations        []string
	TargetKinds       []string
	ScopeSchemaRoot   string
	ScopeSchemaBundle []datatype.SchemaResource
	Credential        CredentialMode
	Risk              RiskClass
	Consent           ConsentClass
	ProviderABI       string
}

type MachineDefinition struct {
	CapabilityID      string                    `json:"capabilityId"`
	Operations        []string                  `json:"operations"`
	TargetKinds       []string                  `json:"targetKinds"`
	ScopeSchemaRoot   string                    `json:"scopeSchemaRoot"`
	ScopeSchemaBundle []datatype.SchemaResource `json:"scopeSchemaBundle"`
	Credential        CredentialMode            `json:"credential"`
	Risk              RiskClass                 `json:"risk"`
	Consent           ConsentClass              `json:"consent"`
	ProviderABI       string                    `json:"providerAbi"`
}

type definitionDocument struct {
	Format   string            `json:"format"`
	Version  string            `json:"version"`
	Ref      Ref               `json:"ref"`
	Semantic MachineDefinition `json:"semantic"`
}

type definitionState struct {
	ref      Ref
	machine  MachineDefinition
	semantic []byte
	bytes    []byte
}

type Definition struct{ state *definitionState }

func SealDefinition(draft DefinitionDraft) (Definition, error) {
	machine, err := normalizeDefinition(draft)
	if err != nil {
		return Definition{}, err
	}
	return sealDefinition(machine)
}

func OpenDefinition(raw []byte) (Definition, error) {
	if len(raw) == 0 || len(raw) > MaxDefinitionBytes {
		return Definition{}, errors.New("capability definition exceeds byte budget")
	}
	if err := artifact.InspectJSONBudget(raw, 96, 262144, 1<<20); err != nil {
		return Definition{}, fmt.Errorf("inspect capability definition: %w", err)
	}
	canonical, err := artifact.Canonicalize(raw)
	if err != nil || !bytes.Equal(canonical, raw) {
		return Definition{}, errors.New("capability definition is not canonical")
	}
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.DisallowUnknownFields()
	var document definitionDocument
	if err := decoder.Decode(&document); err != nil {
		return Definition{}, fmt.Errorf("decode capability definition: %w", err)
	}
	var trailing any
	if err := decoder.Decode(&trailing); !errors.Is(err, io.EOF) {
		return Definition{}, errors.New("capability definition contains trailing values")
	}
	if document.Format != DefinitionFormat || document.Version != DefinitionVersion {
		return Definition{}, errors.New("unsupported capability definition")
	}
	machine, err := normalizeDefinition(DefinitionDraft{
		CapabilityID: document.Semantic.CapabilityID, Operations: document.Semantic.Operations,
		TargetKinds: document.Semantic.TargetKinds, ScopeSchemaRoot: document.Semantic.ScopeSchemaRoot,
		ScopeSchemaBundle: document.Semantic.ScopeSchemaBundle, Credential: document.Semantic.Credential,
		Risk: document.Semantic.Risk, Consent: document.Semantic.Consent, ProviderABI: document.Semantic.ProviderABI,
	})
	if err != nil {
		return Definition{}, err
	}
	sealed, err := sealDefinition(machine)
	if err != nil {
		return Definition{}, err
	}
	if sealed.Ref() != document.Ref || !bytes.Equal(sealed.Bytes(), raw) {
		return Definition{}, errors.New("capability definition digest mismatch")
	}
	return sealed, nil
}

func OpenSemanticDefinition(ref Ref, semantic []byte) (Definition, error) {
	canonical, err := artifact.Canonicalize(semantic)
	if err != nil || !bytes.Equal(canonical, semantic) {
		return Definition{}, errors.New("capability semantic definition is not canonical")
	}
	decoder := json.NewDecoder(bytes.NewReader(semantic))
	decoder.DisallowUnknownFields()
	var machine MachineDefinition
	if err := decoder.Decode(&machine); err != nil {
		return Definition{}, fmt.Errorf("decode capability semantic definition: %w", err)
	}
	var trailing any
	if err := decoder.Decode(&trailing); !errors.Is(err, io.EOF) {
		return Definition{}, errors.New("capability semantic definition contains trailing values")
	}
	normalized, err := normalizeDefinition(DefinitionDraft(machine))
	if err != nil {
		return Definition{}, err
	}
	sealed, err := sealDefinition(normalized)
	if err != nil {
		return Definition{}, err
	}
	if sealed.Ref() != ref || !bytes.Equal(sealed.SemanticBytes(), semantic) {
		return Definition{}, errors.New("capability semantic definition digest mismatch")
	}
	return sealed, nil
}

func sealDefinition(machine MachineDefinition) (Definition, error) {
	semantic, err := artifact.Marshal(machine)
	if err != nil {
		return Definition{}, err
	}
	digest, err := artifact.Sum(definitionDigestDomain, semantic)
	if err != nil {
		return Definition{}, err
	}
	ref := Ref{CapabilityID: machine.CapabilityID, SemanticDigest: digest}
	raw, err := artifact.Marshal(definitionDocument{Format: DefinitionFormat, Version: DefinitionVersion, Ref: ref, Semantic: machine})
	if err != nil {
		return Definition{}, err
	}
	if len(raw) > MaxDefinitionBytes {
		return Definition{}, errors.New("capability definition exceeds byte budget")
	}
	return Definition{state: &definitionState{ref: ref, machine: machine, semantic: semantic, bytes: raw}}, nil
}

func normalizeDefinition(draft DefinitionDraft) (MachineDefinition, error) {
	if err := validateVersionedURI(draft.CapabilityID); err != nil {
		return MachineDefinition{}, fmt.Errorf("invalid capability id: %w", err)
	}
	operations, err := normalizeStrings(draft.Operations, operationPattern, "operation", 64)
	if err != nil || len(operations) == 0 {
		return MachineDefinition{}, errors.New("capability requires 1..64 canonical operations")
	}
	targetKinds, err := normalizeStrings(draft.TargetKinds, identifierPattern, "target kind", 64)
	if err != nil || len(targetKinds) == 0 {
		return MachineDefinition{}, errors.New("capability requires 1..64 target kinds")
	}
	bundle, err := contractschema.Normalize(datatype.JSONSchemaDialect, draft.ScopeSchemaRoot, draft.ScopeSchemaBundle)
	if err != nil {
		return MachineDefinition{}, fmt.Errorf("normalize capability scope schema: %w", err)
	}
	if draft.Credential != CredentialNone && draft.Credential != CredentialRequired {
		return MachineDefinition{}, errors.New("invalid capability credential mode")
	}
	if draft.Risk != RiskLow && draft.Risk != RiskSensitive && draft.Risk != RiskDangerous {
		return MachineDefinition{}, errors.New("invalid capability risk class")
	}
	if draft.Consent != ConsentNone && draft.Consent != ConsentOnce && draft.Consent != ConsentEveryRun {
		return MachineDefinition{}, errors.New("invalid capability consent class")
	}
	if err := validateVersionedURI(draft.ProviderABI); err != nil {
		return MachineDefinition{}, fmt.Errorf("invalid capability provider ABI: %w", err)
	}
	return MachineDefinition{
		CapabilityID: draft.CapabilityID, Operations: operations, TargetKinds: targetKinds,
		ScopeSchemaRoot: draft.ScopeSchemaRoot, ScopeSchemaBundle: bundle, Credential: draft.Credential,
		Risk: draft.Risk, Consent: draft.Consent, ProviderABI: draft.ProviderABI,
	}, nil
}

func (d Definition) Valid() bool { return d.state != nil && d.state.ref.SemanticDigest.Valid() }
func (d Definition) Ref() Ref {
	if !d.Valid() {
		return Ref{}
	}
	return d.state.ref
}
func (d Definition) Bytes() []byte {
	if !d.Valid() {
		return nil
	}
	return append([]byte(nil), d.state.bytes...)
}
func (d Definition) SemanticBytes() []byte {
	if !d.Valid() {
		return nil
	}
	return append([]byte(nil), d.state.semantic...)
}
func (d Definition) Machine() MachineDefinition {
	if !d.Valid() {
		return MachineDefinition{}
	}
	var clone MachineDefinition
	if err := json.Unmarshal(d.state.semantic, &clone); err != nil {
		panic("capability definition invariant: " + err.Error())
	}
	return clone
}

type Requirement struct {
	ID             string          `json:"id"`
	Capability     Ref             `json:"capability"`
	Operations     []string        `json:"operations"`
	TargetSlot     string          `json:"targetSlot"`
	CredentialSlot string          `json:"credentialSlot,omitempty"`
	Scope          json.RawMessage `json:"scope"`
}

func (d Definition) NormalizeRequirement(source Requirement) (Requirement, error) {
	normalized, err := NormalizeRequirementSyntax(source)
	if err != nil {
		return Requirement{}, err
	}
	if !d.Valid() || normalized.Capability != d.Ref() {
		return Requirement{}, errors.New("invalid capability requirement identity")
	}
	allowed := make(map[string]struct{}, len(d.state.machine.Operations))
	for _, operation := range d.state.machine.Operations {
		allowed[operation] = struct{}{}
	}
	for _, operation := range normalized.Operations {
		if _, ok := allowed[operation]; !ok {
			return Requirement{}, fmt.Errorf("operation %q is not defined by capability", operation)
		}
	}
	if d.state.machine.Credential == CredentialRequired {
		if normalized.CredentialSlot == "" {
			return Requirement{}, errors.New("capability requires a credential slot")
		}
	} else if normalized.CredentialSlot != "" {
		return Requirement{}, errors.New("credential-free capability cannot request a credential slot")
	}
	if err := validateSchemaValue(d.state.machine, normalized.Scope); err != nil {
		return Requirement{}, fmt.Errorf("capability scope violates definition: %w", err)
	}
	return normalized, nil
}

func NormalizeRequirementSyntax(source Requirement) (Requirement, error) {
	if !identifierPattern.MatchString(source.ID) || source.Capability.Validate() != nil || !identifierPattern.MatchString(source.TargetSlot) {
		return Requirement{}, errors.New("invalid capability requirement identity")
	}
	if source.CredentialSlot != "" && !identifierPattern.MatchString(source.CredentialSlot) {
		return Requirement{}, errors.New("invalid capability credential slot")
	}
	operations, err := normalizeStrings(source.Operations, operationPattern, "operation", 64)
	if err != nil || len(operations) == 0 {
		return Requirement{}, errors.New("capability requirement needs operations")
	}
	if len(source.Scope) == 0 || len(source.Scope) > 64<<10 {
		return Requirement{}, errors.New("capability scope exceeds byte budget")
	}
	canonical, err := artifact.Canonicalize(source.Scope)
	if err != nil {
		return Requirement{}, fmt.Errorf("canonicalize capability scope: %w", err)
	}
	return Requirement{ID: source.ID, Capability: source.Capability, Operations: operations, TargetSlot: source.TargetSlot, CredentialSlot: source.CredentialSlot, Scope: canonical}, nil
}

func validateSchemaValue(definition MachineDefinition, raw []byte) error {
	compiler := runtimejsonschema.NewCompiler()
	for _, resource := range definition.ScopeSchemaBundle {
		var schema any
		decoder := json.NewDecoder(bytes.NewReader(resource.Schema))
		decoder.UseNumber()
		if err := decoder.Decode(&schema); err != nil {
			return err
		}
		if err := compiler.AddResource(resource.ID, schema); err != nil {
			return err
		}
	}
	validator, err := compiler.Compile(definition.ScopeSchemaRoot)
	if err != nil {
		return err
	}
	var value any
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.UseNumber()
	if err := decoder.Decode(&value); err != nil {
		return err
	}
	return validator.Validate(value)
}

func normalizeStrings[T ~string](source []T, pattern *regexp.Regexp, label string, maximum int) ([]T, error) {
	if len(source) > maximum {
		return nil, fmt.Errorf("%s set exceeds budget", label)
	}
	result := append([]T(nil), source...)
	sort.Slice(result, func(i, j int) bool { return result[i] < result[j] })
	for index, value := range result {
		if !pattern.MatchString(string(value)) {
			return nil, fmt.Errorf("invalid %s %q", label, value)
		}
		if index > 0 && result[index-1] == value {
			return nil, fmt.Errorf("duplicate %s %q", label, value)
		}
	}
	if result == nil {
		result = []T{}
	}
	return result, nil
}

func validateVersionedURI(value string) error {
	parsed, err := url.Parse(value)
	if err != nil || !parsed.IsAbs() || parsed.Host == "" || parsed.Fragment != "" {
		return fmt.Errorf("%q must be an absolute URI without a fragment", value)
	}
	segments := strings.Split(strings.Trim(parsed.EscapedPath(), "/"), "/")
	if len(segments) == 0 || !versionPattern.MatchString(segments[len(segments)-1]) {
		return fmt.Errorf("%q must end in /vN", value)
	}
	return nil
}

func validAttributionID(value string) bool {
	if value == "" || len(value) > 256 || strings.TrimSpace(value) != value {
		return false
	}
	for _, character := range value {
		if unicode.IsControl(character) || character == '\u061c' || character == '\u200e' || character == '\u200f' ||
			character >= '\u202a' && character <= '\u202e' || character >= '\u2066' && character <= '\u2069' {
			return false
		}
	}
	return true
}
