// Package nodecatalog defines the immutable machine Catalog 3.1 artifact.
package nodecatalog

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/url"
	"reflect"
	"regexp"
	"sort"
	"strings"

	"github.com/yottaapp/yotta/internal/artifact"
	"github.com/yottaapp/yotta/internal/capability"
	"github.com/yottaapp/yotta/internal/datatype"
	"github.com/yottaapp/yotta/internal/nodecontract"
)

const (
	Format                 = "yotta.catalog"
	Version                = "3.1"
	MaxCatalogBytes        = 16 << 20
	MaxCatalogDepth        = 128
	MaxCatalogNodes        = 4_096
	MaxCatalogTypes        = 4_096
	MaxCatalogCapabilities = 4_096
	MaxCatalogJSONNode     = 1_048_576

	catalogDigestDomain = "yotta/catalog/v1"
)

var versionPattern = regexp.MustCompile(`^v[1-9][0-9]*$`)

type ImplementationLock struct {
	PackageID      string                      `json:"packageId"`
	ArtifactDigest artifact.Digest             `json:"artifactDigest"`
	ABI            nodecontract.ABIRequirement `json:"abi"`
	Entrypoint     string                      `json:"entrypoint"`
}

type Binding struct {
	Contract       nodecontract.Contract
	Implementation ImplementationLock
}

type Entry struct {
	Contract       nodecontract.Contract
	Implementation ImplementationLock
}

type typeRecord struct {
	TypeRef  datatype.TypeRef `json:"typeRef"`
	Semantic json.RawMessage  `json:"semantic"`
}

type capabilityRecord struct {
	Ref      capability.Ref  `json:"ref"`
	Semantic json.RawMessage `json:"semantic"`
}

type bindingRecord struct {
	NodeRef        nodecontract.NodeRef `json:"nodeRef"`
	Semantic       json.RawMessage      `json:"semantic"`
	Implementation ImplementationLock   `json:"implementation"`
}

type document struct {
	Format           string             `json:"format"`
	Version          string             `json:"version"`
	GeneratorVersion string             `json:"generatorVersion"`
	Types            []typeRecord       `json:"types"`
	Capabilities     []capabilityRecord `json:"capabilities"`
	Nodes            []bindingRecord    `json:"nodes"`
}

type state struct {
	document     document
	hash         artifact.Digest
	bytes        []byte
	bindings     map[string]Entry
	types        map[string]datatype.Definition
	capabilities map[string]capability.Definition
}

type Snapshot struct{ state *state }

func Seal(types []datatype.Definition, capabilities []capability.Definition, bindings []Binding, generatorVersion string) (Snapshot, error) {
	if !versionPattern.MatchString(generatorVersion) {
		return Snapshot{}, errors.New("catalog generator version must use vN form")
	}
	if len(types) > MaxCatalogTypes || len(capabilities) > MaxCatalogCapabilities || len(bindings) > MaxCatalogNodes {
		return Snapshot{}, errors.New("catalog exceeds entry budget")
	}
	typeRecords := make([]typeRecord, 0, len(types))
	typesByID := make(map[string]datatype.Definition, len(types))
	for _, definition := range types {
		if !definition.Valid() {
			return Snapshot{}, errors.New("catalog contains invalid data type definition")
		}
		ref := definition.TypeRef()
		if _, duplicate := typesByID[ref.TypeID]; duplicate {
			return Snapshot{}, fmt.Errorf("catalog contains duplicate data type %q", ref.TypeID)
		}
		typesByID[ref.TypeID] = definition
		typeRecords = append(typeRecords, typeRecord{TypeRef: ref, Semantic: definition.SemanticBytes()})
	}
	sort.Slice(typeRecords, func(i, j int) bool { return typeRecords[i].TypeRef.TypeID < typeRecords[j].TypeRef.TypeID })

	capabilityRecords := make([]capabilityRecord, 0, len(capabilities))
	capabilitiesByID := make(map[string]capability.Definition, len(capabilities))
	for _, definition := range capabilities {
		if !definition.Valid() {
			return Snapshot{}, errors.New("catalog contains invalid capability definition")
		}
		ref := definition.Ref()
		if _, duplicate := capabilitiesByID[ref.CapabilityID]; duplicate {
			return Snapshot{}, fmt.Errorf("catalog contains duplicate capability %q", ref.CapabilityID)
		}
		capabilitiesByID[ref.CapabilityID] = definition
		capabilityRecords = append(capabilityRecords, capabilityRecord{Ref: ref, Semantic: definition.SemanticBytes()})
	}
	sort.Slice(capabilityRecords, func(i, j int) bool {
		return capabilityRecords[i].Ref.CapabilityID < capabilityRecords[j].Ref.CapabilityID
	})

	bindingRecords := make([]bindingRecord, 0, len(bindings))
	bindingsByID := make(map[string]Entry, len(bindings))
	for _, binding := range bindings {
		if !binding.Contract.Valid() {
			return Snapshot{}, errors.New("catalog contains invalid node contract")
		}
		ref := binding.Contract.NodeRef()
		if _, duplicate := bindingsByID[ref.NodeTypeID]; duplicate {
			return Snapshot{}, fmt.Errorf("catalog contains duplicate node type %q", ref.NodeTypeID)
		}
		if err := validateImplementation(binding.Contract, binding.Implementation); err != nil {
			return Snapshot{}, fmt.Errorf("node %q: %w", ref.NodeTypeID, err)
		}
		if err := validatePortTypes(binding.Contract.Machine().Ports, typesByID); err != nil {
			return Snapshot{}, fmt.Errorf("node %q: %w", ref.NodeTypeID, err)
		}
		for _, requirement := range binding.Contract.Machine().CapabilityRequirements {
			definition, ok := capabilitiesByID[requirement.Capability.CapabilityID]
			if !ok || definition.Ref() != requirement.Capability {
				return Snapshot{}, fmt.Errorf("node %q: unbound capability requirement %q", ref.NodeTypeID, requirement.ID)
			}
			normalized, err := definition.NormalizeRequirement(requirement)
			if err != nil || !reflect.DeepEqual(normalized, requirement) {
				return Snapshot{}, fmt.Errorf("node %q: invalid capability requirement %q", ref.NodeTypeID, requirement.ID)
			}
		}
		entry := Entry(binding)
		bindingsByID[ref.NodeTypeID] = entry
		bindingRecords = append(bindingRecords, bindingRecord{
			NodeRef: ref, Semantic: binding.Contract.SemanticBytes(), Implementation: binding.Implementation,
		})
	}
	sort.Slice(bindingRecords, func(i, j int) bool {
		return bindingRecords[i].NodeRef.NodeTypeID < bindingRecords[j].NodeRef.NodeTypeID
	})
	return sealDocument(document{
		Format: Format, Version: Version, GeneratorVersion: generatorVersion,
		Types: typeRecords, Capabilities: capabilityRecords, Nodes: bindingRecords,
	}, bindingsByID, typesByID, capabilitiesByID)
}

func Open(raw []byte) (Snapshot, error) {
	if len(raw) == 0 || len(raw) > MaxCatalogBytes {
		return Snapshot{}, errors.New("catalog exceeds byte budget")
	}
	if err := inspectJSON(raw); err != nil {
		return Snapshot{}, fmt.Errorf("catalog exceeds structural budget: %w", err)
	}
	canonical, err := artifact.Canonicalize(raw)
	if err != nil {
		return Snapshot{}, fmt.Errorf("canonicalize catalog: %w", err)
	}
	if !bytes.Equal(raw, canonical) {
		return Snapshot{}, errors.New("catalog is not canonical")
	}
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.DisallowUnknownFields()
	decoder.UseNumber()
	var decoded document
	if err := decoder.Decode(&decoded); err != nil {
		return Snapshot{}, fmt.Errorf("decode catalog: %w", err)
	}
	var trailing any
	if err := decoder.Decode(&trailing); !errors.Is(err, io.EOF) {
		return Snapshot{}, errors.New("catalog contains trailing JSON values")
	}
	if decoded.Format != Format || decoded.Version != Version || !versionPattern.MatchString(decoded.GeneratorVersion) {
		return Snapshot{}, errors.New("unsupported catalog format")
	}
	if len(decoded.Types) > MaxCatalogTypes || len(decoded.Capabilities) > MaxCatalogCapabilities || len(decoded.Nodes) > MaxCatalogNodes {
		return Snapshot{}, errors.New("catalog exceeds entry budget")
	}
	capabilities := make([]capability.Definition, 0, len(decoded.Capabilities))
	for _, record := range decoded.Capabilities {
		definition, err := capability.OpenSemanticDefinition(record.Ref, record.Semantic)
		if err != nil {
			return Snapshot{}, fmt.Errorf("open catalog capability %q: %w", record.Ref.CapabilityID, err)
		}
		capabilities = append(capabilities, definition)
	}
	types := make([]datatype.Definition, 0, len(decoded.Types))
	for _, record := range decoded.Types {
		definition, err := datatype.OpenSemanticDefinition(record.TypeRef, record.Semantic)
		if err != nil {
			return Snapshot{}, fmt.Errorf("open catalog type %q: %w", record.TypeRef.TypeID, err)
		}
		types = append(types, definition)
	}
	bindings := make([]Binding, 0, len(decoded.Nodes))
	for _, record := range decoded.Nodes {
		contract, err := nodecontract.OpenSemantic(record.NodeRef, record.Semantic)
		if err != nil {
			return Snapshot{}, fmt.Errorf("open catalog node %q: %w", record.NodeRef.NodeTypeID, err)
		}
		bindings = append(bindings, Binding{Contract: contract, Implementation: record.Implementation})
	}
	sealed, err := Seal(types, capabilities, bindings, decoded.GeneratorVersion)
	if err != nil {
		return Snapshot{}, err
	}
	if !bytes.Equal(sealed.Bytes(), raw) {
		return Snapshot{}, errors.New("catalog is not normalized")
	}
	return sealed, nil
}

func sealDocument(doc document, bindings map[string]Entry, types map[string]datatype.Definition, capabilities map[string]capability.Definition) (Snapshot, error) {
	canonical, err := artifact.Marshal(doc)
	if err != nil {
		return Snapshot{}, fmt.Errorf("encode catalog: %w", err)
	}
	if len(canonical) > MaxCatalogBytes {
		return Snapshot{}, errors.New("catalog exceeds byte budget")
	}
	hash, err := artifact.Sum(catalogDigestDomain, canonical)
	if err != nil {
		return Snapshot{}, err
	}
	return Snapshot{state: &state{document: doc, hash: hash, bytes: canonical, bindings: bindings, types: types, capabilities: capabilities}}, nil
}

func (s Snapshot) Valid() bool { return s.state != nil && s.state.hash.Valid() }

func (s Snapshot) Hash() artifact.Digest {
	if !s.Valid() {
		return ""
	}
	return s.state.hash
}

func (s Snapshot) Bytes() []byte {
	if !s.Valid() {
		return nil
	}
	return append([]byte(nil), s.state.bytes...)
}

func (s Snapshot) Lookup(nodeTypeID string) (Entry, bool) {
	if !s.Valid() {
		return Entry{}, false
	}
	entry, ok := s.state.bindings[nodeTypeID]
	return entry, ok
}

func (s Snapshot) LookupType(typeID string) (datatype.Definition, bool) {
	if !s.Valid() {
		return datatype.Definition{}, false
	}
	definition, ok := s.state.types[typeID]
	return definition, ok
}

func (s Snapshot) LookupCapability(capabilityID string) (capability.Definition, bool) {
	if !s.Valid() {
		return capability.Definition{}, false
	}
	definition, ok := s.state.capabilities[capabilityID]
	return definition, ok
}

func validateImplementation(contract nodecontract.Contract, lock ImplementationLock) error {
	if err := validateVersionedURI(lock.PackageID); err != nil {
		return fmt.Errorf("invalid implementation package: %w", err)
	}
	if !lock.ArtifactDigest.Valid() || lock.Entrypoint == "" || len(lock.Entrypoint) > 1_024 || strings.TrimSpace(lock.Entrypoint) != lock.Entrypoint {
		return errors.New("invalid implementation artifact lock")
	}
	allowed := false
	for _, abi := range contract.Machine().ImplementationABI {
		if abi == lock.ABI {
			allowed = true
			break
		}
	}
	if !allowed {
		return errors.New("implementation ABI is not allowed by the node contract")
	}
	return nil
}

func validatePortTypes(ports nodecontract.PortSet, definitions map[string]datatype.Definition) error {
	for _, input := range ports.DataInputs {
		if err := validateExpressionTypes(input.Type, definitions, 0); err != nil {
			return fmt.Errorf("input %q: %w", input.ID, err)
		}
		if input.ResourceLease != nil && !supportsRuntimeReference(input.Type, definitions) {
			return fmt.Errorf("input %q: resource lease requires an exact type with a runtime representation", input.ID)
		}
	}
	for _, output := range ports.DataOutputs {
		if err := validateExpressionTypes(output.Type, definitions, 0); err != nil {
			return fmt.Errorf("output %q: %w", output.ID, err)
		}
		if output.ResourceLease != nil && !supportsRuntimeReference(output.Type, definitions) {
			return fmt.Errorf("output %q: resource lease requires an exact type with a runtime representation", output.ID)
		}
	}
	return nil
}

func supportsRuntimeReference(expression datatype.TypeExpression, definitions map[string]datatype.Definition) bool {
	if expression.Kind != datatype.TypeExpressionRef || expression.Ref == nil {
		return false
	}
	definition, ok := definitions[expression.Ref.TypeID]
	if !ok || definition.TypeRef() != *expression.Ref {
		return false
	}
	for _, representation := range definition.Machine().Representations {
		if representation.Kind == datatype.RepresentationStreamRef || representation.Kind == datatype.RepresentationHandleRef {
			return true
		}
	}
	return false
}

func validateExpressionTypes(expression datatype.TypeExpression, definitions map[string]datatype.Definition, depth int) error {
	if depth > datatype.MaxTypeDepth {
		return errors.New("type expression exceeds depth budget")
	}
	switch expression.Kind {
	case datatype.TypeExpressionRef:
		definition, ok := definitions[expression.Ref.TypeID]
		if !ok || definition.TypeRef() != *expression.Ref {
			return fmt.Errorf("unbound type reference %q", expression.Ref.TypeID)
		}
	case datatype.TypeExpressionList:
		return validateExpressionTypes(*expression.Element, definitions, depth+1)
	case datatype.TypeExpressionUnion:
		for _, member := range expression.Members {
			if err := validateExpressionTypes(member, definitions, depth+1); err != nil {
				return err
			}
		}
	case datatype.TypeExpressionVariable:
		return nil
	default:
		return fmt.Errorf("invalid type expression kind %q", expression.Kind)
	}
	return nil
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

func inspectJSON(raw []byte) error {
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.UseNumber()
	depth, nodes := 0, 0
	for {
		token, err := decoder.Token()
		if errors.Is(err, io.EOF) {
			if depth != 0 {
				return errors.New("unbalanced JSON")
			}
			return nil
		}
		if err != nil {
			return err
		}
		nodes++
		if nodes > MaxCatalogJSONNode {
			return errors.New("JSON node budget exceeded")
		}
		if delimiter, ok := token.(json.Delim); ok {
			switch delimiter {
			case '{', '[':
				depth++
				if depth > MaxCatalogDepth {
					return errors.New("JSON depth budget exceeded")
				}
			case '}', ']':
				depth--
			}
		}
	}
}
