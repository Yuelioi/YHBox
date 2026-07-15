// Package nodeauthoring derives the complete editor-facing projection from
// trusted Data Type, Node Contract, Capability, and machine Catalog facts.
package nodeauthoring

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"regexp"
	"sort"
	"strings"

	"github.com/yottaapp/yotta/internal/artifact"
	"github.com/yottaapp/yotta/internal/capability"
	"github.com/yottaapp/yotta/internal/datatype"
	"github.com/yottaapp/yotta/internal/nodecatalog"
	"github.com/yottaapp/yotta/internal/nodecontract"
)

const (
	Format             = "yotta.node-authoring-projection"
	Version            = "3.1"
	MaxProjectionBytes = 16 << 20
	MaxProjectedFields = 4096
	MaxProjectionRefs  = 1024
	digestDomain       = "yotta/node-authoring-projection/v1"
)

var generatorVersionPattern = regexp.MustCompile(`^v[1-9][0-9]*$`)

type Lifecycle string

const (
	LifecycleDurable   Lifecycle = "durable"
	LifecycleRuntime   Lifecycle = "runtime-only"
	LifecycleMixed     Lifecycle = "durable-or-runtime"
	LifecycleDependent Lifecycle = "resolved-at-compile"
)

type Carrier string

const (
	CarrierDurable Carrier = "durable"
	CarrierRuntime Carrier = "runtime"
)

type BindingState string

const (
	BindingRequired BindingState = "required"
	BindingOptional BindingState = "optional"
	BindingDefault  BindingState = "default-available"
	BindingOutput   BindingState = "output"
)

type Availability string

const (
	AvailabilityPortable              Availability = "portable"
	AvailabilityHostRequired          Availability = "host-required"
	AvailabilityTargetRequired        Availability = "target-required"
	AvailabilityHostAndTargetRequired Availability = "host-and-target-required"
)

type Control string

const (
	ControlText          Control = "text"
	ControlCode          Control = "code"
	ControlNumber        Control = "number"
	ControlInteger       Control = "integer"
	ControlToggle        Control = "toggle"
	ControlSelect        Control = "select"
	ControlObject        Control = "object"
	ControlList          Control = "list"
	ControlJSON          Control = "json"
	ControlStateVariable Control = "state-variable"
)

type Input struct {
	Catalog          nodecatalog.Snapshot
	Types            []datatype.Definition
	Capabilities     []capability.Definition
	Contracts        []nodecontract.Contract
	GeneratorVersion string
}

type FieldConstraints struct {
	MinLength *int              `json:"minLength,omitempty"`
	MaxLength *int              `json:"maxLength,omitempty"`
	Pattern   string            `json:"pattern,omitempty"`
	Minimum   json.RawMessage   `json:"minimum,omitempty"`
	Maximum   json.RawMessage   `json:"maximum,omitempty"`
	MinItems  *int              `json:"minItems,omitempty"`
	MaxItems  *int              `json:"maxItems,omitempty"`
	Enum      []json.RawMessage `json:"enum"`
}

type FieldProjection struct {
	ID             string            `json:"id"`
	Title          string            `json:"title,omitempty"`
	Description    string            `json:"description,omitempty"`
	TitleKey       string            `json:"titleKey,omitempty"`
	DescriptionKey string            `json:"descriptionKey,omitempty"`
	Control        Control           `json:"control" jsonschema:"required,enum=text,enum=code,enum=number,enum=integer,enum=toggle,enum=select,enum=object,enum=list,enum=json,enum=state-variable"`
	Required       bool              `json:"required"`
	HasDefault     bool              `json:"hasDefault"`
	Default        json.RawMessage   `json:"default,omitempty"`
	Examples       []json.RawMessage `json:"examples"`
	Deprecated     bool              `json:"deprecated"`
	ReadOnly       bool              `json:"readOnly"`
	Constraints    FieldConstraints  `json:"constraints"`
	Properties     []FieldProjection `json:"properties"`
	Items          *FieldProjection  `json:"items,omitempty"`
}

type TypeProjection struct {
	TypeRef         datatype.TypeRef              `json:"typeRef"`
	SchemaRoot      string                        `json:"schemaRoot"`
	Control         Control                       `json:"control" jsonschema:"required,enum=text,enum=number,enum=integer,enum=toggle,enum=select,enum=object,enum=list,enum=json"`
	TitleKey        string                        `json:"titleKey,omitempty"`
	DescriptionKey  string                        `json:"descriptionKey,omitempty"`
	Color           string                        `json:"color,omitempty"`
	Icon            string                        `json:"icon,omitempty"`
	EditorAdapter   string                        `json:"editorAdapter,omitempty"`
	Examples        []json.RawMessage             `json:"examples"`
	Representations []datatype.RepresentationSpec `json:"representations"`
	Lifecycle       Lifecycle                     `json:"lifecycle" jsonschema:"required,enum=durable,enum=runtime-only,enum=durable-or-runtime,enum=resolved-at-compile"`
	Constraints     FieldConstraints              `json:"constraints"`
}

type TypeUse struct {
	Expression      datatype.TypeExpression       `json:"expression"`
	Label           string                        `json:"label"`
	Control         Control                       `json:"control" jsonschema:"required,enum=text,enum=number,enum=integer,enum=toggle,enum=select,enum=object,enum=list,enum=json"`
	Color           string                        `json:"color,omitempty"`
	TypeIDs         []string                      `json:"typeIds"`
	TitleKey        string                        `json:"titleKey,omitempty"`
	DescriptionKey  string                        `json:"descriptionKey,omitempty"`
	Representations []datatype.RepresentationSpec `json:"representations"`
	Lifecycle       Lifecycle                     `json:"lifecycle" jsonschema:"required,enum=durable,enum=runtime-only,enum=durable-or-runtime,enum=resolved-at-compile"`
	Constraints     FieldConstraints              `json:"constraints"`
	Examples        []json.RawMessage             `json:"examples"`
	EditorAdapter   string                        `json:"editorAdapter,omitempty"`
}

type PortProjection struct {
	ID         string                             `json:"id"`
	Binding    BindingState                       `json:"binding" jsonschema:"required,enum=required,enum=optional,enum=default-available,enum=output"`
	HasDefault bool                               `json:"hasDefault"`
	Default    json.RawMessage                    `json:"default,omitempty"`
	Carrier    Carrier                            `json:"carrier" jsonschema:"required,enum=durable,enum=runtime"`
	Type       TypeUse                            `json:"type"`
	Lease      *nodecontract.ResourceLeaseBinding `json:"resourceLease,omitempty"`
}

type SignalProjection struct {
	ID        string `json:"id"`
	Channel   string `json:"channel" jsonschema:"required,enum=exec,enum=error"`
	Direction string `json:"direction" jsonschema:"required,enum=input,enum=output"`
}

type CapabilityProjection struct {
	RequirementID           string                    `json:"requirementId"`
	Capability              capability.Ref            `json:"capability"`
	Operations              []string                  `json:"operations"`
	Scope                   json.RawMessage           `json:"scope"`
	TargetSlot              string                    `json:"targetSlot"`
	CredentialSlot          string                    `json:"credentialSlot,omitempty"`
	TargetSlotConfigKey     string                    `json:"targetSlotConfigKey,omitempty"`
	CredentialSlotConfigKey string                    `json:"credentialSlotConfigKey,omitempty"`
	TargetKinds             []string                  `json:"targetKinds"`
	Credential              capability.CredentialMode `json:"credential" jsonschema:"required,enum=none,enum=required"`
	Risk                    capability.RiskClass      `json:"risk" jsonschema:"required,enum=low,enum=sensitive,enum=dangerous"`
	Consent                 capability.ConsentClass   `json:"consent" jsonschema:"required,enum=none,enum=once,enum=every-run"`
}

type StateAccessProjection struct {
	ID            string                       `json:"id"`
	SlotConfigKey string                       `json:"slotConfigKey"`
	Type          TypeUse                      `json:"type"`
	Mode          nodecontract.StateAccessMode `json:"mode" jsonschema:"required,enum=read,enum=write"`
}

type NodeProjection struct {
	NodeRef        nodecontract.NodeRef                  `json:"nodeRef"`
	TitleKey       string                                `json:"titleKey,omitempty"`
	DescriptionKey string                                `json:"descriptionKey,omitempty"`
	Category       string                                `json:"category,omitempty"`
	Tags           []string                              `json:"tags"`
	Icon           string                                `json:"icon,omitempty"`
	EditorAdapter  string                                `json:"editorAdapter,omitempty"`
	Execution      nodecontract.ExecutionSpec            `json:"execution"`
	Instruction    nodecontract.InstructionSpec          `json:"instruction"`
	Availability   Availability                          `json:"availability" jsonschema:"required,enum=portable,enum=host-required,enum=target-required,enum=host-and-target-required"`
	HostFeatures   []nodecontract.HostFeatureRequirement `json:"hostFeatureRequirements" jsonschema:"required,maxItems=256"`
	DataInputs     []PortProjection                      `json:"dataInputs"`
	DataOutputs    []PortProjection                      `json:"dataOutputs"`
	Signals        []SignalProjection                    `json:"signals"`
	ConfigFields   []FieldProjection                     `json:"configFields"`
	Capabilities   []CapabilityProjection                `json:"capabilities"`
	StateAccesses  []StateAccessProjection               `json:"stateAccesses"`
	Errors         []nodecontract.ErrorSpec              `json:"errors"`
	StatusEvents   []nodecontract.StatusEventSpec        `json:"statusEvents"`
}

type body struct {
	CatalogHash      artifact.Digest  `json:"catalogHash"`
	GeneratorVersion string           `json:"generatorVersion"`
	Types            []TypeProjection `json:"types"`
	Nodes            []NodeProjection `json:"nodes"`
}

type document struct {
	Format           string          `json:"format" jsonschema:"required,enum=yotta.node-authoring-projection"`
	Version          string          `json:"version" jsonschema:"required,enum=3.1"`
	ProjectionDigest artifact.Digest `json:"projectionDigest"`
	Body             body            `json:"body"`
}

type state struct {
	document document
	bytes    []byte
	types    map[string]TypeProjection
	nodes    map[string]NodeProjection
}

type Snapshot struct{ state *state }

func Project(input Input) (Snapshot, error) {
	if !input.Catalog.Valid() || !generatorVersionPattern.MatchString(input.GeneratorVersion) {
		return Snapshot{}, errors.New("authoring projection requires a trusted Catalog and vN generator")
	}
	types, typeIndex, err := projectTypes(input)
	if err != nil {
		return Snapshot{}, err
	}
	capabilityIndex, err := indexCapabilities(input)
	if err != nil {
		return Snapshot{}, err
	}
	nodes, nodeIndex, err := projectNodes(input, typeIndex, capabilityIndex)
	if err != nil {
		return Snapshot{}, err
	}
	body := body{CatalogHash: input.Catalog.Hash(), GeneratorVersion: input.GeneratorVersion, Types: types, Nodes: nodes}
	bodyBytes, err := artifact.Marshal(body)
	if err != nil {
		return Snapshot{}, err
	}
	digest, err := artifact.Sum(digestDomain, bodyBytes)
	if err != nil {
		return Snapshot{}, err
	}
	document := document{Format: Format, Version: Version, ProjectionDigest: digest, Body: body}
	raw, err := artifact.Marshal(document)
	if err != nil {
		return Snapshot{}, err
	}
	if len(raw) > MaxProjectionBytes {
		return Snapshot{}, errors.New("authoring projection exceeds byte budget")
	}
	return Snapshot{state: &state{document: document, bytes: raw, types: typeIndex, nodes: nodeIndex}}, nil
}

// Open strict-opens a projection only by regenerating it from the same trusted
// inputs. Projection JSON is never a second source of node or type semantics.
func Open(raw []byte, input Input) (Snapshot, error) {
	if len(raw) == 0 || len(raw) > MaxProjectionBytes {
		return Snapshot{}, errors.New("authoring projection exceeds byte budget")
	}
	expected, err := Project(input)
	if err != nil {
		return Snapshot{}, err
	}
	if !bytes.Equal(raw, expected.Bytes()) {
		return Snapshot{}, errors.New("authoring projection does not match trusted contracts")
	}
	return expected, nil
}

func projectTypes(input Input) ([]TypeProjection, map[string]TypeProjection, error) {
	refs := input.Catalog.TypeRefs()
	if len(input.Types) != len(refs) {
		return nil, nil, errors.New("authoring projection must present every Catalog data type exactly once")
	}
	definitions := make(map[string]datatype.Definition, len(input.Types))
	for _, definition := range input.Types {
		if !definition.Valid() {
			return nil, nil, errors.New("authoring projection contains an invalid data type")
		}
		ref := definition.TypeRef()
		machine, ok := input.Catalog.LookupType(ref.TypeID)
		if !ok || machine.TypeRef() != ref {
			return nil, nil, fmt.Errorf("data type %q does not match the machine Catalog", ref.TypeID)
		}
		if _, duplicate := definitions[ref.TypeID]; duplicate {
			return nil, nil, fmt.Errorf("duplicate data type presentation %q", ref.TypeID)
		}
		definitions[ref.TypeID] = definition
	}
	result := make([]TypeProjection, 0, len(refs))
	index := make(map[string]TypeProjection, len(refs))
	for _, ref := range refs {
		definition, ok := definitions[ref.TypeID]
		if !ok || definition.TypeRef() != ref {
			return nil, nil, fmt.Errorf("missing data type presentation %q", ref.TypeID)
		}
		machine := definition.Machine()
		authoring := definition.Authoring()
		root, err := schemaByID(machine.SchemaBundle, machine.SchemaRoot)
		if err != nil {
			return nil, nil, fmt.Errorf("data type %q: %w", ref.TypeID, err)
		}
		resolved, _, complete, err := resolveSchema(root, machine.SchemaBundle, machine.SchemaRoot, map[string]bool{}, &projectionBudget{}, 0)
		if err != nil {
			return nil, nil, fmt.Errorf("data type %q: %w", ref.TypeID, err)
		}
		if !complete {
			resolved = root
		}
		projection := TypeProjection{
			TypeRef: ref, SchemaRoot: machine.SchemaRoot, TitleKey: authoring.TitleKey, DescriptionKey: authoring.DescriptionKey,
			Color: authoring.Color, Icon: authoring.Icon, EditorAdapter: authoring.EditorAdapter,
			Examples: cloneRawList(authoring.Examples), Representations: append([]datatype.RepresentationSpec(nil), machine.Representations...),
			Lifecycle: lifecycleFor(machine.Representations), Control: controlForSchema(resolved, complete), Constraints: constraintsFor(resolved),
		}
		result = append(result, projection)
		index[ref.TypeID] = projection
	}
	return result, index, nil
}

func indexCapabilities(input Input) (map[string]capability.Definition, error) {
	refs := input.Catalog.CapabilityRefs()
	if len(input.Capabilities) != len(refs) {
		return nil, errors.New("authoring projection must present every Catalog capability exactly once")
	}
	index := make(map[string]capability.Definition, len(refs))
	for _, definition := range input.Capabilities {
		if !definition.Valid() {
			return nil, errors.New("authoring projection contains an invalid capability")
		}
		ref := definition.Ref()
		machine, ok := input.Catalog.LookupCapability(ref.CapabilityID)
		if !ok || machine.Ref() != ref {
			return nil, fmt.Errorf("capability %q does not match the machine Catalog", ref.CapabilityID)
		}
		if _, duplicate := index[ref.CapabilityID]; duplicate {
			return nil, fmt.Errorf("duplicate capability presentation %q", ref.CapabilityID)
		}
		index[ref.CapabilityID] = definition
	}
	for _, ref := range refs {
		if definition, ok := index[ref.CapabilityID]; !ok || definition.Ref() != ref {
			return nil, fmt.Errorf("missing capability presentation %q", ref.CapabilityID)
		}
	}
	return index, nil
}

func projectNodes(input Input, types map[string]TypeProjection, capabilities map[string]capability.Definition) ([]NodeProjection, map[string]NodeProjection, error) {
	refs := input.Catalog.NodeRefs()
	if len(input.Contracts) != len(refs) {
		return nil, nil, errors.New("authoring projection must present every Catalog node exactly once")
	}
	contracts := make(map[string]nodecontract.Contract, len(input.Contracts))
	for _, contract := range input.Contracts {
		if !contract.Valid() {
			return nil, nil, errors.New("authoring projection contains an invalid node contract")
		}
		ref := contract.NodeRef()
		entry, ok := input.Catalog.Lookup(ref.NodeTypeID)
		if !ok || entry.Contract.NodeRef() != ref {
			return nil, nil, fmt.Errorf("node %q does not match the machine Catalog", ref.NodeTypeID)
		}
		if _, duplicate := contracts[ref.NodeTypeID]; duplicate {
			return nil, nil, fmt.Errorf("duplicate node presentation %q", ref.NodeTypeID)
		}
		contracts[ref.NodeTypeID] = contract
	}
	result := make([]NodeProjection, 0, len(refs))
	index := make(map[string]NodeProjection, len(refs))
	for _, ref := range refs {
		contract, ok := contracts[ref.NodeTypeID]
		if !ok || contract.NodeRef() != ref {
			return nil, nil, fmt.Errorf("missing node presentation %q", ref.NodeTypeID)
		}
		projection, err := projectNode(contract, types, capabilities)
		if err != nil {
			return nil, nil, fmt.Errorf("node %q: %w", ref.NodeTypeID, err)
		}
		result = append(result, projection)
		index[ref.NodeTypeID] = projection
	}
	return result, index, nil
}

func projectNode(contract nodecontract.Contract, types map[string]TypeProjection, capabilities map[string]capability.Definition) (NodeProjection, error) {
	machine := contract.Machine()
	authoring := contract.Authoring()
	fields, err := projectConfigFields(machine.ConfigSchemaBundle, machine.ConfigSchemaRoot)
	if err != nil {
		return NodeProjection{}, err
	}
	projection := NodeProjection{
		NodeRef: contract.NodeRef(), TitleKey: authoring.TitleKey, DescriptionKey: authoring.DescriptionKey,
		Category: authoring.Category, Tags: append([]string(nil), authoring.Tags...), Icon: authoring.Icon, EditorAdapter: authoring.EditorAdapter,
		Execution: machine.Execution, Instruction: machine.Instruction, Availability: AvailabilityPortable,
		HostFeatures: append([]nodecontract.HostFeatureRequirement{}, machine.HostFeatureRequirements...), DataInputs: []PortProjection{}, DataOutputs: []PortProjection{},
		Signals: []SignalProjection{}, ConfigFields: fields, Capabilities: []CapabilityProjection{}, StateAccesses: []StateAccessProjection{}, Errors: append([]nodecontract.ErrorSpec{}, machine.Errors...),
		StatusEvents: append([]nodecontract.StatusEventSpec{}, machine.StatusEvents...),
	}
	for _, port := range machine.Ports.DataInputs {
		use, err := projectTypeUse(port.Type, types)
		if err != nil {
			return NodeProjection{}, fmt.Errorf("input %q: %w", port.ID, err)
		}
		binding := BindingOptional
		if port.Default != nil {
			binding = BindingDefault
		} else if port.Required {
			binding = BindingRequired
		}
		carrier := CarrierDurable
		if port.ResourceLease != nil {
			carrier = CarrierRuntime
		}
		projectedPort := PortProjection{ID: port.ID, Binding: binding, Carrier: carrier, Type: use, Lease: cloneLease(port.ResourceLease)}
		if port.Default != nil {
			projectedPort.HasDefault = true
			projectedPort.Default = append(json.RawMessage(nil), (*port.Default)...)
		}
		projection.DataInputs = append(projection.DataInputs, projectedPort)
	}
	for _, port := range machine.Ports.DataOutputs {
		use, err := projectTypeUse(port.Type, types)
		if err != nil {
			return NodeProjection{}, fmt.Errorf("output %q: %w", port.ID, err)
		}
		carrier := CarrierDurable
		if port.ResourceLease != nil {
			carrier = CarrierRuntime
		}
		projection.DataOutputs = append(projection.DataOutputs, PortProjection{ID: port.ID, Binding: BindingOutput, Carrier: carrier, Type: use, Lease: cloneLease(port.ResourceLease)})
	}
	addSignals := func(channel, direction string, ports []nodecontract.SignalPort) {
		for _, port := range ports {
			projection.Signals = append(projection.Signals, SignalProjection{ID: port.ID, Channel: channel, Direction: direction})
		}
	}
	for _, port := range machine.Ports.ExecInputs {
		channel := "exec"
		if machine.Instruction.AcceptsSignalInput("error", port.ID) && !machine.Instruction.AcceptsSignalInput("exec", port.ID) {
			channel = "error"
		}
		projection.Signals = append(projection.Signals, SignalProjection{ID: port.ID, Channel: channel, Direction: "input"})
	}
	addSignals("exec", "output", machine.Ports.ExecOutputs)
	addSignals("error", "output", machine.Ports.ErrorOutputs)
	requirementBindings := make(map[string]nodecontract.RequirementBindingSpec, len(machine.RequirementBindings))
	for _, binding := range machine.RequirementBindings {
		requirementBindings[binding.RequirementID] = binding
	}
	for _, requirement := range machine.CapabilityRequirements {
		definition, ok := capabilities[requirement.Capability.CapabilityID]
		if !ok || definition.Ref() != requirement.Capability {
			return NodeProjection{}, fmt.Errorf("capability requirement %q has no presentation definition", requirement.ID)
		}
		capabilityMachine := definition.Machine()
		binding := requirementBindings[requirement.ID]
		projection.Capabilities = append(projection.Capabilities, CapabilityProjection{
			RequirementID: requirement.ID, Capability: requirement.Capability, Operations: append([]string(nil), requirement.Operations...),
			Scope: append(json.RawMessage(nil), requirement.Scope...), TargetSlot: requirement.TargetSlot, CredentialSlot: requirement.CredentialSlot,
			TargetSlotConfigKey: binding.TargetSlotConfigKey, CredentialSlotConfigKey: binding.CredentialSlotConfigKey,
			TargetKinds: append([]string(nil), capabilityMachine.TargetKinds...), Credential: capabilityMachine.Credential,
			Risk: capabilityMachine.Risk, Consent: capabilityMachine.Consent,
		})
	}
	for _, access := range machine.StateAccesses {
		use, err := projectTypeUse(access.Type, types)
		if err != nil {
			return NodeProjection{}, fmt.Errorf("state access %q: %w", access.ID, err)
		}
		projection.StateAccesses = append(projection.StateAccesses, StateAccessProjection{
			ID: access.ID, SlotConfigKey: access.SlotConfigKey, Type: use, Mode: access.Mode,
		})
	}
	if len(projection.HostFeatures) != 0 && len(projection.Capabilities) != 0 {
		projection.Availability = AvailabilityHostAndTargetRequired
	} else if len(projection.HostFeatures) != 0 {
		projection.Availability = AvailabilityHostRequired
	} else if len(projection.Capabilities) != 0 {
		projection.Availability = AvailabilityTargetRequired
	}
	return projection, nil
}

func projectTypeUse(expression datatype.TypeExpression, types map[string]TypeProjection) (TypeUse, error) {
	use := TypeUse{
		Expression: expression, Label: typeExpressionLabel(expression), Control: ControlJSON,
		TypeIDs: []string{}, Representations: []datatype.RepresentationSpec{}, Lifecycle: LifecycleDependent,
		Constraints: FieldConstraints{Enum: []json.RawMessage{}}, Examples: []json.RawMessage{},
	}
	ids := map[string]struct{}{}
	representations := map[string]datatype.RepresentationSpec{}
	lifecycles := map[Lifecycle]struct{}{}
	var visit func(datatype.TypeExpression) error
	visit = func(current datatype.TypeExpression) error {
		switch current.Kind {
		case datatype.TypeExpressionRef:
			if current.Ref == nil {
				return errors.New("type expression has no ref")
			}
			definition, ok := types[current.Ref.TypeID]
			if !ok || definition.TypeRef != *current.Ref {
				return fmt.Errorf("type %q has no exact presentation", current.Ref.TypeID)
			}
			ids[current.Ref.TypeID] = struct{}{}
			lifecycles[definition.Lifecycle] = struct{}{}
			for _, representation := range definition.Representations {
				representations[string(representation.Kind)+"\x00"+representation.Codec] = representation
			}
			if use.TitleKey == "" {
				use.TitleKey, use.DescriptionKey = definition.TitleKey, definition.DescriptionKey
			}
		case datatype.TypeExpressionList:
			if current.Element == nil {
				return errors.New("list type expression has no element")
			}
			return visit(*current.Element)
		case datatype.TypeExpressionUnion:
			for _, member := range current.Members {
				if err := visit(member); err != nil {
					return err
				}
			}
		case datatype.TypeExpressionVariable:
			lifecycles[LifecycleDependent] = struct{}{}
		default:
			return errors.New("unknown type expression kind")
		}
		return nil
	}
	if err := visit(expression); err != nil {
		return TypeUse{}, err
	}
	for id := range ids {
		use.TypeIDs = append(use.TypeIDs, id)
	}
	sort.Strings(use.TypeIDs)
	if len(use.TypeIDs) == 1 {
		projection := types[use.TypeIDs[0]]
		use.Color, use.Control, use.Constraints = projection.Color, projection.Control, cloneConstraints(projection.Constraints)
		use.Examples, use.EditorAdapter = cloneRawList(projection.Examples), projection.EditorAdapter
	}
	if expression.Kind != datatype.TypeExpressionRef {
		use.Control, use.Constraints, use.Examples, use.EditorAdapter = ControlJSON, FieldConstraints{Enum: []json.RawMessage{}}, []json.RawMessage{}, ""
		if expression.Kind == datatype.TypeExpressionList {
			use.Control = ControlList
		}
	}
	keys := make([]string, 0, len(representations))
	for key := range representations {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	for _, key := range keys {
		use.Representations = append(use.Representations, representations[key])
	}
	if len(lifecycles) == 1 {
		for lifecycle := range lifecycles {
			use.Lifecycle = lifecycle
		}
	} else if len(lifecycles) > 1 {
		use.Lifecycle = LifecycleMixed
	}
	return use, nil
}

func projectConfigFields(bundle []datatype.SchemaResource, rootID string) ([]FieldProjection, error) {
	budget := &projectionBudget{}
	root, err := schemaByID(bundle, rootID)
	if err != nil {
		return nil, err
	}
	resolved, baseID, complete, err := resolveSchema(root, bundle, rootID, map[string]bool{}, budget, 0)
	if err != nil {
		return nil, err
	}
	if !complete {
		return nil, errors.New("config schema root cannot be projected as an object")
	}
	if schemaType(resolved) != "object" {
		return nil, errors.New("config schema root must be an object")
	}
	properties, _ := resolved["properties"].(map[string]any)
	required := stringSet(resolved["required"])
	ids := make([]string, 0, len(properties))
	for id := range properties {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	result := make([]FieldProjection, 0, len(ids))
	for _, id := range ids {
		child, ok := properties[id].(map[string]any)
		if !ok {
			return nil, fmt.Errorf("config property %q is not a schema object", id)
		}
		field, err := projectField(id, child, required[id], bundle, baseID, map[string]bool{}, budget, 0)
		if err != nil {
			return nil, err
		}
		result = append(result, field)
	}
	return result, nil
}

func projectField(id string, schema map[string]any, required bool, bundle []datatype.SchemaResource, baseID string, stack map[string]bool, budget *projectionBudget, depth int) (FieldProjection, error) {
	if err := budget.consumeField(); err != nil {
		return FieldProjection{}, err
	}
	resolved, resolvedBase, complete, err := resolveSchema(schema, bundle, baseID, stack, budget, depth)
	if err != nil {
		return FieldProjection{}, fmt.Errorf("field %q: %w", id, err)
	}
	if !complete {
		resolved = schema
	}
	if boolValue(resolved["writeOnly"]) {
		return FieldProjection{}, fmt.Errorf("field %q is writeOnly; secrets must use a capability credential slot", id)
	}
	field := FieldProjection{
		ID: id, Title: stringValue(resolved["title"]), Description: stringValue(resolved["description"]),
		TitleKey: stringValue(resolved["x-yotta-title-key"]), DescriptionKey: stringValue(resolved["x-yotta-description-key"]), Required: required,
		Examples: rawList(resolved["examples"]), Deprecated: boolValue(resolved["deprecated"]), ReadOnly: boolValue(resolved["readOnly"]),
		Constraints: constraintsFor(resolved), Properties: []FieldProjection{},
	}
	if value, ok := resolved["default"]; ok {
		field.HasDefault = true
		field.Default = rawValue(value)
	}
	if !complete {
		field.Control = ControlJSON
		return field, nil
	}
	if requested := stringValue(resolved["x-yotta-control"]); requested != "" {
		if schemaType(resolved) != "string" {
			return FieldProjection{}, fmt.Errorf("field %q requests an unsupported generated control", id)
		}
		switch Control(requested) {
		case ControlCode:
			field.Control = ControlCode
		case ControlStateVariable:
			field.Control = ControlStateVariable
		default:
			return FieldProjection{}, fmt.Errorf("field %q requests an unsupported generated control", id)
		}
		return field, nil
	}
	if len(field.Constraints.Enum) != 0 {
		field.Control = ControlSelect
		return field, nil
	}
	switch schemaType(resolved) {
	case "string":
		field.Control = ControlText
	case "number":
		field.Control = ControlNumber
	case "integer":
		field.Control = ControlInteger
	case "boolean":
		field.Control = ControlToggle
	case "object":
		field.Control = ControlObject
		properties, _ := resolved["properties"].(map[string]any)
		requiredChildren := stringSet(resolved["required"])
		ids := make([]string, 0, len(properties))
		for childID := range properties {
			ids = append(ids, childID)
		}
		sort.Strings(ids)
		for _, childID := range ids {
			child, ok := properties[childID].(map[string]any)
			if !ok {
				return FieldProjection{}, fmt.Errorf("field %q property %q is not a schema object", id, childID)
			}
			projected, err := projectField(childID, child, requiredChildren[childID], bundle, resolvedBase, stack, budget, depth+1)
			if err != nil {
				return FieldProjection{}, err
			}
			field.Properties = append(field.Properties, projected)
		}
	case "array":
		field.Control = ControlList
		items, ok := resolved["items"].(map[string]any)
		if !ok {
			field.Control = ControlJSON
			break
		}
		projected, err := projectField("item", items, true, bundle, resolvedBase, stack, budget, depth+1)
		if err != nil {
			return FieldProjection{}, err
		}
		field.Items = &projected
	default:
		field.Control = ControlJSON
	}
	return field, nil
}

type projectionBudget struct {
	fields int
	refs   int
}

func (budget *projectionBudget) consumeField() error {
	budget.fields++
	if budget.fields > MaxProjectedFields {
		return errors.New("authoring projection exceeds field expansion budget")
	}
	return nil
}

func (budget *projectionBudget) consumeRef() error {
	budget.refs++
	if budget.refs > MaxProjectionRefs {
		return errors.New("authoring projection exceeds reference expansion budget")
	}
	return nil
}

func resolveSchema(schema map[string]any, bundle []datatype.SchemaResource, baseID string, stack map[string]bool, budget *projectionBudget, depth int) (map[string]any, string, bool, error) {
	if depth > 32 {
		return schema, baseID, false, nil
	}
	base, err := url.Parse(baseID)
	if err != nil {
		return nil, "", false, err
	}
	if declaredID := stringValue(schema["$id"]); declaredID != "" {
		parsed, err := url.Parse(declaredID)
		if err != nil {
			return nil, "", false, err
		}
		base = base.ResolveReference(parsed)
		base.Fragment = ""
	}
	ref, _ := schema["$ref"].(string)
	if ref == "" {
		return schema, base.String(), true, nil
	}
	if err := budget.consumeRef(); err != nil {
		return nil, "", false, err
	}
	parsed, err := url.Parse(ref)
	if err != nil {
		return nil, "", false, err
	}
	resolvedURL := base.ResolveReference(parsed)
	resolvedKey := resolvedURL.String()
	if stack[resolvedKey] {
		return schema, base.String(), false, nil
	}
	fragment := resolvedURL.Fragment
	resolvedURL.Fragment = ""
	targetBase := resolvedURL.String()
	target, err := schemaByID(bundle, targetBase)
	if err != nil {
		return nil, "", false, err
	}
	if fragment != "" {
		if !strings.HasPrefix(fragment, "/") {
			return schema, base.String(), false, nil
		}
		target, err = resolveJSONPointer(target, fragment)
		if err != nil {
			return nil, "", false, err
		}
	}
	stack[resolvedKey] = true
	defer delete(stack, resolvedKey)
	resolved, resolvedBase, complete, err := resolveSchema(target, bundle, targetBase, stack, budget, depth+1)
	if err != nil || !complete {
		return schema, base.String(), complete, err
	}
	merged := make(map[string]any, len(resolved)+len(schema))
	for key, value := range resolved {
		merged[key] = value
	}
	for key, value := range schema {
		if key != "$ref" {
			merged[key] = value
		}
	}
	return merged, resolvedBase, true, nil
}

func resolveJSONPointer(root map[string]any, fragment string) (map[string]any, error) {
	var current any = root
	for _, rawToken := range strings.Split(strings.TrimPrefix(fragment, "/"), "/") {
		token, err := url.PathUnescape(rawToken)
		if err != nil {
			return nil, err
		}
		token = strings.ReplaceAll(strings.ReplaceAll(token, "~1", "/"), "~0", "~")
		object, ok := current.(map[string]any)
		if !ok {
			return nil, errors.New("authoring $ref JSON Pointer does not address an object")
		}
		current, ok = object[token]
		if !ok {
			return nil, errors.New("authoring $ref JSON Pointer target is missing")
		}
	}
	result, ok := current.(map[string]any)
	if !ok {
		return nil, errors.New("authoring $ref target is not a schema object")
	}
	return result, nil
}

func schemaByID(bundle []datatype.SchemaResource, id string) (map[string]any, error) {
	for _, resource := range bundle {
		if resource.ID != id {
			continue
		}
		decoder := json.NewDecoder(bytes.NewReader(resource.Schema))
		decoder.UseNumber()
		var schema map[string]any
		if err := decoder.Decode(&schema); err != nil {
			return nil, err
		}
		return schema, nil
	}
	return nil, fmt.Errorf("schema root %q is not bundled", id)
}

func constraintsFor(schema map[string]any) FieldConstraints {
	return FieldConstraints{
		MinLength: intPointer(schema["minLength"]), MaxLength: intPointer(schema["maxLength"]), Pattern: stringValue(schema["pattern"]),
		Minimum: numberRaw(schema["minimum"]), Maximum: numberRaw(schema["maximum"]),
		MinItems: intPointer(schema["minItems"]), MaxItems: intPointer(schema["maxItems"]), Enum: rawList(schema["enum"]),
	}
}

func controlForSchema(schema map[string]any, complete bool) Control {
	if !complete {
		return ControlJSON
	}
	if values := rawList(schema["enum"]); len(values) != 0 {
		return ControlSelect
	}
	switch schemaType(schema) {
	case "string":
		return ControlText
	case "number":
		return ControlNumber
	case "integer":
		return ControlInteger
	case "boolean":
		return ControlToggle
	case "object":
		return ControlObject
	case "array":
		return ControlList
	default:
		return ControlJSON
	}
}

func cloneConstraints(source FieldConstraints) FieldConstraints {
	clone := source
	clone.Minimum = append(json.RawMessage(nil), source.Minimum...)
	clone.Maximum = append(json.RawMessage(nil), source.Maximum...)
	clone.Enum = cloneRawList(source.Enum)
	return clone
}

func lifecycleFor(representations []datatype.RepresentationSpec) Lifecycle {
	durable, runtime := false, false
	for _, representation := range representations {
		switch representation.Kind {
		case datatype.RepresentationInlineJSON, datatype.RepresentationBlobRef:
			durable = true
		case datatype.RepresentationStreamRef, datatype.RepresentationHandleRef:
			runtime = true
		}
	}
	if durable && runtime {
		return LifecycleMixed
	}
	if runtime {
		return LifecycleRuntime
	}
	return LifecycleDurable
}

func typeExpressionLabel(expression datatype.TypeExpression) string {
	switch expression.Kind {
	case datatype.TypeExpressionRef:
		if expression.Ref != nil {
			return expression.Ref.TypeID
		}
	case datatype.TypeExpressionList:
		if expression.Element != nil {
			return "list<" + typeExpressionLabel(*expression.Element) + ">"
		}
	case datatype.TypeExpressionUnion:
		labels := make([]string, 0, len(expression.Members))
		for _, member := range expression.Members {
			labels = append(labels, typeExpressionLabel(member))
		}
		return strings.Join(labels, " | ")
	case datatype.TypeExpressionVariable:
		return "$" + expression.Variable
	}
	return "invalid"
}

func cloneLease(lease *nodecontract.ResourceLeaseBinding) *nodecontract.ResourceLeaseBinding {
	if lease == nil {
		return nil
	}
	return &nodecontract.ResourceLeaseBinding{RequirementID: lease.RequirementID, Operations: append([]string(nil), lease.Operations...)}
}

func cloneRawList(values []json.RawMessage) []json.RawMessage {
	result := make([]json.RawMessage, len(values))
	for index := range values {
		result[index] = append(json.RawMessage(nil), values[index]...)
	}
	return result
}

func rawList(value any) []json.RawMessage {
	values, ok := value.([]any)
	if !ok {
		return []json.RawMessage{}
	}
	result := make([]json.RawMessage, 0, len(values))
	for _, item := range values {
		result = append(result, rawValue(item))
	}
	return result
}

func rawValue(value any) json.RawMessage {
	raw, err := artifact.Marshal(value)
	if err != nil {
		panic("normalized schema value: " + err.Error())
	}
	return raw
}

func numberRaw(value any) json.RawMessage {
	if _, ok := value.(json.Number); !ok {
		return nil
	}
	return rawValue(value)
}

func intPointer(value any) *int {
	number, ok := value.(json.Number)
	if !ok {
		return nil
	}
	parsed, err := number.Int64()
	if err != nil || parsed < 0 || parsed > int64(^uint(0)>>1) {
		return nil
	}
	result := int(parsed)
	return &result
}

func schemaType(schema map[string]any) string {
	value, _ := schema["type"].(string)
	return value
}

func stringSet(value any) map[string]bool {
	result := map[string]bool{}
	values, _ := value.([]any)
	for _, item := range values {
		if text, ok := item.(string); ok {
			result[text] = true
		}
	}
	return result
}

func stringValue(value any) string {
	result, _ := value.(string)
	return result
}

func boolValue(value any) bool {
	result, _ := value.(bool)
	return result
}

func (s Snapshot) Valid() bool { return s.state != nil && s.state.document.ProjectionDigest.Valid() }

func (s Snapshot) Digest() artifact.Digest {
	if !s.Valid() {
		return ""
	}
	return s.state.document.ProjectionDigest
}

func (s Snapshot) CatalogHash() artifact.Digest {
	if !s.Valid() {
		return ""
	}
	return s.state.document.Body.CatalogHash
}

func (s Snapshot) GeneratorVersion() string {
	if !s.Valid() {
		return ""
	}
	return s.state.document.Body.GeneratorVersion
}

func (s Snapshot) Bytes() []byte {
	if !s.Valid() {
		return nil
	}
	return append([]byte(nil), s.state.bytes...)
}

func (s Snapshot) Type(typeID string) (TypeProjection, bool) {
	if !s.Valid() {
		return TypeProjection{}, false
	}
	projection, ok := s.state.types[typeID]
	return cloneType(projection), ok
}

func (s Snapshot) Node(nodeTypeID string) (NodeProjection, bool) {
	if !s.Valid() {
		return NodeProjection{}, false
	}
	projection, ok := s.state.nodes[nodeTypeID]
	return cloneNode(projection), ok
}

func (s Snapshot) Nodes() []NodeProjection {
	if !s.Valid() {
		return nil
	}
	result := make([]NodeProjection, len(s.state.document.Body.Nodes))
	for index := range s.state.document.Body.Nodes {
		result[index] = cloneNode(s.state.document.Body.Nodes[index])
	}
	return result
}

func cloneType(source TypeProjection) TypeProjection {
	if source.TypeRef.TypeID == "" {
		return TypeProjection{}
	}
	raw, _ := json.Marshal(source)
	var result TypeProjection
	_ = json.Unmarshal(raw, &result)
	return result
}

func cloneNode(source NodeProjection) NodeProjection {
	if source.NodeRef.NodeTypeID == "" {
		return NodeProjection{}
	}
	raw, _ := json.Marshal(source)
	var result NodeProjection
	_ = json.Unmarshal(raw, &result)
	return result
}
