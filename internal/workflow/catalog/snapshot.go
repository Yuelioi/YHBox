package catalog

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"sort"

	"github.com/yottaapp/yotta/internal/artifact"
	"github.com/yottaapp/yotta/internal/node"
)

const (
	SnapshotFormat         = "yotta.catalog"
	SnapshotVersion        = 1
	MaxCatalogNodes        = 4096
	MaxNodePins            = 4096
	MaxNodeContractBytes   = 1 << 20
	MaxCatalogBytes        = 16 << 20
	MaxFieldSchemaDepth    = 64
	MaxFieldSchemaEntries  = 4096
	catalogHashDomain      = "yotta/catalog/v1"
	nodeContractHashDomain = "yotta/node-contract/v1"
)

type ExecutionMode string

const (
	ExecutionRun      ExecutionMode = "run"
	ExecutionRegion   ExecutionMode = "region"
	ExecutionEvaluate ExecutionMode = "evaluate"
	ExecutionMarker   ExecutionMode = "marker"
	ExecutionVisual   ExecutionMode = "visual"
)

// NodeContract is the compiler-facing projection of node.Spec. Presentation
// metadata such as category, widgets, hints and supported-target badges is not
// part of the executable catalog identity.
type NodeContract struct {
	Kind                string                   `json:"kind"`
	Execution           ExecutionMode            `json:"execution"`
	Inputs              []InputContract          `json:"inputs"`
	Outputs             []OutputContract         `json:"outputs"`
	IsPureData          bool                     `json:"isPureData"`
	IsNonDeterministic  bool                     `json:"isNonDeterministic"`
	IsVisualOnly        bool                     `json:"isVisualOnly"`
	NeedsTarget         bool                     `json:"needsTarget"`
	TargetCapabilities  []node.TargetCapability  `json:"targetCapabilities"`
	RuntimeCapabilities []node.RuntimeCapability `json:"runtimeCapabilities"`
	NeedsWindow         bool                     `json:"needsWindow"`
	NeedsForeground     bool                     `json:"needsForeground"`
	IsGraphMarker       bool                     `json:"isGraphMarker"`
	DynamicOutputs      bool                     `json:"dynamicOutputs"`
	DynamicInputs       bool                     `json:"dynamicInputs"`
	DynamicDataFields   bool                     `json:"dynamicDataFields"`
	HasCustomValidation bool                     `json:"hasCustomValidation"`
	HasDependencies     bool                     `json:"hasDependencies"`
	ContractHash        artifact.Digest          `json:"contractHash"`
}

type InputContract struct {
	Name     string         `json:"name"`
	Type     string         `json:"type"`
	Semantic string         `json:"semantic"`
	Required bool           `json:"required"`
	Default  any            `json:"default"`
	Schema   *FieldContract `json:"schema"`
}

type FieldContract struct {
	Type    string               `json:"type"`
	Shape   string               `json:"shape"`
	Fields  []FieldEntryContract `json:"fields"`
	Items   *FieldContract       `json:"items"`
	Options []any                `json:"options"`
}

type FieldEntryContract struct {
	Key      string         `json:"key"`
	Schema   *FieldContract `json:"schema"`
	Required bool           `json:"required"`
}

type OutputContract struct {
	Name     string           `json:"name"`
	Type     string           `json:"type"`
	Semantic string           `json:"semantic"`
	Data     []node.DataField `json:"data"`
}

type snapshotDocument struct {
	Format            string          `json:"format"`
	Version           int             `json:"version"`
	ImplementationSet artifact.Digest `json:"implementationSet"`
	Nodes             []NodeContract  `json:"nodes"`
}

type snapshotState struct {
	document snapshotDocument
	hash     artifact.Digest
	bytes    []byte
	byKind   map[string]NodeContract
}

// Snapshot is an immutable catalog generation. The implementation set locks
// built-in behavior; individual contract hashes lock compiler-visible specs.
type Snapshot struct{ state *snapshotState }

func NewSnapshot(reader node.RegistryReader, implementationSet artifact.Digest) (Snapshot, error) {
	if reader == nil {
		return Snapshot{}, errors.New("catalog registry is required")
	}
	if !implementationSet.Valid() {
		return Snapshot{}, errors.New("catalog implementation set must be a content digest")
	}
	registered := reader.All()
	if len(registered) > MaxCatalogNodes {
		return Snapshot{}, errors.New("catalog exceeds node budget")
	}
	contracts := make([]NodeContract, 0, len(registered))
	seenKinds := map[string]bool{}
	for _, entry := range registered {
		if entry == nil {
			return Snapshot{}, errors.New("catalog contains nil node")
		}
		if entry.Spec.Kind == "" || seenKinds[entry.Spec.Kind] {
			return Snapshot{}, fmt.Errorf("catalog contains empty or duplicate node kind %q", entry.Spec.Kind)
		}
		seenKinds[entry.Spec.Kind] = true
		contract, err := projectNode(entry)
		if err != nil {
			return Snapshot{}, err
		}
		contracts = append(contracts, contract)
	}
	sort.Slice(contracts, func(i, j int) bool { return contracts[i].Kind < contracts[j].Kind })
	document := snapshotDocument{
		Format: SnapshotFormat, Version: SnapshotVersion,
		ImplementationSet: implementationSet, Nodes: contracts,
	}
	canonical, err := artifact.Marshal(document)
	if err != nil {
		return Snapshot{}, fmt.Errorf("encode catalog snapshot: %w", err)
	}
	if len(canonical) > MaxCatalogBytes {
		return Snapshot{}, errors.New("catalog exceeds byte budget")
	}
	hash, err := artifact.Sum(catalogHashDomain, canonical)
	if err != nil {
		return Snapshot{}, err
	}
	byKind := make(map[string]NodeContract, len(contracts))
	for _, contract := range contracts {
		byKind[contract.Kind] = contract
	}
	return Snapshot{state: &snapshotState{document: document, hash: hash, bytes: canonical, byKind: byKind}}, nil
}

func projectNode(entry *node.RegisteredNode) (NodeContract, error) {
	spec := entry.Spec
	if len(spec.Inputs)+len(spec.Outputs) > MaxNodePins {
		return NodeContract{}, fmt.Errorf("node %q exceeds pin budget", spec.Kind)
	}
	mode := ExecutionMode("")
	switch {
	case entry.Run != nil:
		mode = ExecutionRun
	case entry.RunRegion != nil:
		mode = ExecutionRegion
	case entry.Evaluate != nil:
		mode = ExecutionEvaluate
	case spec.IsGraphMarker:
		mode = ExecutionMarker
	case spec.IsVisualOnly:
		mode = ExecutionVisual
	default:
		return NodeContract{}, fmt.Errorf("node %q has no compiler execution mode", spec.Kind)
	}
	contract := NodeContract{
		Kind: spec.Kind, Execution: mode,
		Inputs: make([]InputContract, 0, len(spec.Inputs)), Outputs: make([]OutputContract, 0, len(spec.Outputs)),
		IsPureData: spec.IsPureData, IsNonDeterministic: spec.IsNonDeterministic,
		IsVisualOnly: spec.IsVisualOnly, NeedsTarget: spec.NeedsTarget,
		TargetCapabilities:  append(make([]node.TargetCapability, 0, len(spec.TargetCapabilities)), spec.TargetCapabilities...),
		RuntimeCapabilities: append(make([]node.RuntimeCapability, 0, len(spec.RuntimeCapabilities)), spec.RuntimeCapabilities...),
		NeedsWindow:         spec.NeedsWindow, NeedsForeground: spec.NeedsForeground,
		IsGraphMarker: spec.IsGraphMarker, DynamicOutputs: spec.DynamicOutputs,
		DynamicInputs: spec.DynamicInputs, DynamicDataFields: spec.DynamicDataFields,
		HasCustomValidation: entry.Validate != nil, HasDependencies: entry.Dependencies != nil,
	}
	if err := validateUniqueCapabilities(spec.Kind, spec.TargetCapabilities, spec.RuntimeCapabilities); err != nil {
		return NodeContract{}, err
	}
	inputNames := map[string]bool{}
	for _, input := range spec.Inputs {
		if input.Name == "" || input.Type == "" || inputNames[input.Name] {
			return NodeContract{}, fmt.Errorf("node %q contains invalid or duplicate input %q", spec.Kind, input.Name)
		}
		inputNames[input.Name] = true
		fieldContract, err := projectFieldContract(input.Schema)
		if err != nil {
			return NodeContract{}, fmt.Errorf("node %q input %q: %w", spec.Kind, input.Name, err)
		}
		contract.Inputs = append(contract.Inputs, InputContract{
			Name: input.Name, Type: input.Type, Semantic: input.Semantic, Required: input.Required,
			Default: input.Default, Schema: fieldContract,
		})
	}
	outputNames := map[string]bool{}
	for _, output := range spec.Outputs {
		if output.Name == "" || output.Type == "" || outputNames[output.Name] {
			return NodeContract{}, fmt.Errorf("node %q contains invalid or duplicate output %q", spec.Kind, output.Name)
		}
		outputNames[output.Name] = true
		dataNames := map[string]bool{}
		for _, field := range output.Data {
			if field.Name == "" || field.Type == "" || dataNames[field.Name] {
				return NodeContract{}, fmt.Errorf("node %q output %q contains invalid data field %q", spec.Kind, output.Name, field.Name)
			}
			dataNames[field.Name] = true
		}
		contract.Outputs = append(contract.Outputs, OutputContract{
			Name: output.Name, Type: output.Type, Semantic: output.Semantic,
			Data: append(make([]node.DataField, 0, len(output.Data)), output.Data...),
		})
	}
	sort.Slice(contract.TargetCapabilities, func(i, j int) bool { return contract.TargetCapabilities[i] < contract.TargetCapabilities[j] })
	sort.Slice(contract.RuntimeCapabilities, func(i, j int) bool { return contract.RuntimeCapabilities[i] < contract.RuntimeCapabilities[j] })
	canonical, err := artifact.Marshal(contract)
	if err != nil {
		return NodeContract{}, fmt.Errorf("encode node contract %q: %w", spec.Kind, err)
	}
	if len(canonical) > MaxNodeContractBytes {
		return NodeContract{}, fmt.Errorf("node %q contract exceeds byte budget", spec.Kind)
	}
	contract.ContractHash, err = artifact.Sum(nodeContractHashDomain, canonical)
	if err != nil {
		return NodeContract{}, err
	}
	return cloneContract(contract)
}

func projectFieldContract(source *node.FieldSchema) (*FieldContract, error) {
	entries := 0
	return projectFieldContractAt(source, 0, &entries)
}

func projectFieldContractAt(source *node.FieldSchema, depth int, entries *int) (*FieldContract, error) {
	if source == nil {
		return nil, nil
	}
	*entries = *entries + 1
	if depth > MaxFieldSchemaDepth || *entries > MaxFieldSchemaEntries {
		return nil, errors.New("field schema exceeds structural budget")
	}
	shape := ""
	if source.Widget == "point" || source.Widget == "geometry" {
		shape = source.Widget
	}
	if shape != "" && source.Type != "object" {
		return nil, errors.New("field shape requires object type")
	}
	contract := &FieldContract{Type: source.Type, Shape: shape, Fields: make([]FieldEntryContract, 0), Options: make([]any, 0)}
	switch source.Type {
	case "number", "string", "bool":
		if len(source.Fields) != 0 || source.Items != nil || len(source.Options) != 0 {
			return nil, errors.New("scalar field schema contains incompatible members")
		}
	case "object", "tuple":
		if source.Items != nil || len(source.Options) != 0 {
			return nil, errors.New("structured field schema contains incompatible members")
		}
		if shape != "" && len(source.Fields) != 0 {
			return nil, errors.New("named field shape cannot also declare fields")
		}
		seen := map[string]bool{}
		for _, field := range source.Fields {
			if field.Key == "" || field.Schema == nil || seen[field.Key] {
				return nil, errors.New("field schema contains invalid or duplicate field")
			}
			seen[field.Key] = true
			child, err := projectFieldContractAt(field.Schema, depth+1, entries)
			if err != nil {
				return nil, err
			}
			contract.Fields = append(contract.Fields, FieldEntryContract{Key: field.Key, Schema: child, Required: field.Required})
		}
	case "array":
		if source.Items == nil || len(source.Fields) != 0 || len(source.Options) != 0 {
			return nil, errors.New("array field schema requires only items")
		}
		child, err := projectFieldContractAt(source.Items, depth+1, entries)
		if err != nil {
			return nil, err
		}
		contract.Items = child
	case "enum":
		if len(source.Options) == 0 || len(source.Fields) != 0 || source.Items != nil {
			return nil, errors.New("enum field schema requires only options")
		}
		seen := map[string]bool{}
		type optionValue struct {
			key   string
			value any
		}
		values := make([]optionValue, 0, len(source.Options))
		for _, option := range source.Options {
			canonical, err := artifact.Marshal(map[string]any{"value": option.Value})
			if err != nil {
				return nil, err
			}
			key := string(canonical)
			if seen[key] {
				return nil, errors.New("enum field schema contains duplicate option")
			}
			seen[key] = true
			values = append(values, optionValue{key: key, value: option.Value})
		}
		sort.Slice(values, func(i, j int) bool { return values[i].key < values[j].key })
		for _, value := range values {
			contract.Options = append(contract.Options, value.value)
		}
	default:
		return nil, fmt.Errorf("unknown field schema type %q", source.Type)
	}
	return contract, nil
}

func (s Snapshot) Valid() bool { return s.state != nil && s.state.hash.Valid() }

func (s Snapshot) Hash() artifact.Digest {
	if !s.Valid() {
		return ""
	}
	return s.state.hash
}

func (s Snapshot) ImplementationSet() artifact.Digest {
	if !s.Valid() {
		return ""
	}
	return s.state.document.ImplementationSet
}

func validateUniqueCapabilities(kind string, target []node.TargetCapability, runtime []node.RuntimeCapability) error {
	seen := map[string]bool{}
	for _, capability := range target {
		key := "target:" + string(capability)
		if capability == "" || seen[key] {
			return fmt.Errorf("node %q contains invalid or duplicate capability %q", kind, capability)
		}
		seen[key] = true
	}
	for _, capability := range runtime {
		key := "runtime:" + string(capability)
		if capability == "" || seen[key] {
			return fmt.Errorf("node %q contains invalid or duplicate capability %q", kind, capability)
		}
		seen[key] = true
	}
	return nil
}

func (s Snapshot) Bytes() []byte {
	if !s.Valid() {
		return nil
	}
	return append([]byte(nil), s.state.bytes...)
}

func (s Snapshot) Lookup(kind string) (NodeContract, bool) {
	if !s.Valid() {
		return NodeContract{}, false
	}
	contract, ok := s.state.byKind[kind]
	if !ok {
		return NodeContract{}, false
	}
	clone, err := cloneContract(contract)
	if err != nil {
		panic("catalog snapshot invariant: " + err.Error())
	}
	return clone, true
}

func (s Snapshot) All() []NodeContract {
	if !s.Valid() {
		return nil
	}
	contracts := make([]NodeContract, len(s.state.document.Nodes))
	for index, contract := range s.state.document.Nodes {
		clone, err := cloneContract(contract)
		if err != nil {
			panic("catalog snapshot invariant: " + err.Error())
		}
		contracts[index] = clone
	}
	return contracts
}

func cloneContract(contract NodeContract) (NodeContract, error) {
	raw, err := json.Marshal(contract)
	if err != nil {
		return NodeContract{}, err
	}
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.UseNumber()
	var clone NodeContract
	if err := decoder.Decode(&clone); err != nil {
		return NodeContract{}, err
	}
	return clone, nil
}
