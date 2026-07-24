package compiler

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"reflect"
	"regexp"
	"slices"

	runtimejsonschema "github.com/santhosh-tekuri/jsonschema/v6"
	"github.com/yottaapp/yotta/internal/artifact"
	"github.com/yottaapp/yotta/internal/blob"
	"github.com/yottaapp/yotta/internal/capability"
	"github.com/yottaapp/yotta/internal/configvalidator"
	"github.com/yottaapp/yotta/internal/datatype"
	"github.com/yottaapp/yotta/internal/nodecatalog"
	"github.com/yottaapp/yotta/internal/nodecontract"
	"github.com/yottaapp/yotta/internal/nodeinstance"
	"github.com/yottaapp/yotta/internal/workflow/schema"
)

const (
	ProgramFormat       = "yotta.program"
	ProgramVersion      = "1"
	MaxProgramBytes     = 16 << 20
	MaxProgramJSONDepth = 128
	MaxProgramJSONNodes = 1_048_576
	programHashDomain   = "yotta/program/v1"
)

var programStateNamePattern = regexp.MustCompile(`^[A-Za-z0-9_][A-Za-z0-9._-]*$`)

const (
	inputLiteral       = "literal"
	inputEdge          = "edge"
	inputSourceLiteral = "literal"
	inputSourceDefault = "default"
	inputSourceBlob    = "blob"
)

type inputPlan struct {
	Kind       string          `json:"kind"`
	Provenance string          `json:"provenance,omitempty"`
	Value      json.RawMessage `json:"value,omitempty"`
	From       schema.Endpoint `json:"from,omitempty"`
}

type programNode struct {
	ID             string                                `json:"id"`
	GraphPath      []string                              `json:"graphPath"`
	SourceNodeID   string                                `json:"sourceNodeId"`
	NodeRef        nodecontract.NodeRef                  `json:"nodeRef"`
	Config         map[string]any                        `json:"config"`
	Inputs         map[string]inputPlan                  `json:"inputs"`
	InputTypes     map[string]datatype.ResolvedType      `json:"inputTypes"`
	OutputTypes    map[string]datatype.ResolvedType      `json:"outputTypes"`
	Ports          nodecontract.PortSet                  `json:"ports"`
	Execution      nodecontract.ExecutionSpec            `json:"execution"`
	Instruction    nodecontract.InstructionSpec          `json:"instruction"`
	HostFeatures   []nodecontract.HostFeatureRequirement `json:"hostFeatureRequirements"`
	Capabilities   []capability.Requirement              `json:"capabilityRequirements"`
	Implementation nodecatalog.ImplementationLock        `json:"implementation"`
}

// programSignalRoute is an ordered control instruction. Data dependencies are
// already frozen into programNode.Inputs and are intentionally not duplicated
// as routes.
type programSignalRoute struct {
	Channel schema.EdgeChannel `json:"channel"`
	From    schema.Endpoint    `json:"from"`
	To      schema.Endpoint    `json:"to"`
}

type programGraph struct {
	ID           string               `json:"id"`
	Nodes        []programNode        `json:"nodes"`
	SignalRoutes []programSignalRoute `json:"signalRoutes"`
	DataOrder    []string             `json:"dataOrder"`
}

type programStateSlot struct {
	Name    string                `json:"name"`
	Type    datatype.ResolvedType `json:"type"`
	Initial json.RawMessage       `json:"initial"`
}

type programBody struct {
	SourceHash     artifact.Digest    `json:"sourceHash"`
	CatalogHash    artifact.Digest    `json:"catalogHash"`
	CompilerBuild  artifact.Digest    `json:"compilerBuild"`
	WorkflowID     string             `json:"workflowId"`
	Revision       int64              `json:"revision"`
	EntryGraph     string             `json:"entryGraph"`
	CapabilityPlan json.RawMessage    `json:"capabilityPlan"`
	State          []programStateSlot `json:"state"`
	Graphs         []programGraph     `json:"graphs"`
}

type programDocument struct {
	Format      string          `json:"format"`
	Version     string          `json:"version"`
	ProgramHash artifact.Digest `json:"programHash"`
	Body        programBody     `json:"body"`
}

type programState struct {
	document programDocument
	artifact []byte
}

type ProgramSnapshot struct{ state *programState }

type NodeView struct {
	GraphID        string
	GraphPath      []string
	ID             string
	NodeRef        nodecontract.NodeRef
	Ports          nodecontract.PortSet
	InputTypes     map[string]datatype.ResolvedType
	OutputTypes    map[string]datatype.ResolvedType
	Execution      nodecontract.ExecutionSpec
	Instruction    nodecontract.InstructionSpec
	HostFeatures   []nodecontract.HostFeatureRequirement
	Capabilities   []capability.Requirement
	Implementation nodecatalog.ImplementationLock
}

type StateView struct {
	Name            string
	Type            datatype.ResolvedType
	InitialArtifact []byte
}

func sealProgram(body programBody) (ProgramSnapshot, error) {
	bodyBytes, err := artifact.Marshal(body)
	if err != nil {
		return ProgramSnapshot{}, err
	}
	hash, err := artifact.Sum(programHashDomain, bodyBytes)
	if err != nil {
		return ProgramSnapshot{}, err
	}
	document := programDocument{Format: ProgramFormat, Version: ProgramVersion, ProgramHash: hash, Body: body}
	raw, err := artifact.Marshal(document)
	if err != nil {
		return ProgramSnapshot{}, err
	}
	if len(raw) > MaxProgramBytes {
		return ProgramSnapshot{}, errors.New("program exceeds byte budget")
	}
	return ProgramSnapshot{state: &programState{document: document, artifact: raw}}, nil
}

func OpenProgram(raw []byte, trustedCatalog nodecatalog.Snapshot, validators configvalidator.Registry, expectedCompilerBuild artifact.Digest) (ProgramSnapshot, error) {
	if len(raw) == 0 || len(raw) > MaxProgramBytes {
		return ProgramSnapshot{}, errors.New("program exceeds byte budget")
	}
	if err := artifact.InspectJSONBudget(raw, MaxProgramJSONDepth, MaxProgramJSONNodes, 1<<20); err != nil {
		return ProgramSnapshot{}, fmt.Errorf("program exceeds structural budget: %w", err)
	}
	canonical, err := artifact.Canonicalize(raw)
	if err != nil {
		return ProgramSnapshot{}, fmt.Errorf("canonicalize program: %w", err)
	}
	if !bytes.Equal(raw, canonical) {
		return ProgramSnapshot{}, errors.New("program is not canonical")
	}
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.DisallowUnknownFields()
	decoder.UseNumber()
	var document programDocument
	if err := decoder.Decode(&document); err != nil {
		return ProgramSnapshot{}, fmt.Errorf("decode program: %w", err)
	}
	var trailing any
	if err := decoder.Decode(&trailing); !errors.Is(err, io.EOF) {
		return ProgramSnapshot{}, errors.New("program contains trailing JSON values")
	}
	if document.Format != ProgramFormat || document.Version != ProgramVersion {
		return ProgramSnapshot{}, errors.New("unsupported program format")
	}
	sealed, err := sealProgram(document.Body)
	if err != nil {
		return ProgramSnapshot{}, err
	}
	if sealed.Hash() != document.ProgramHash || !bytes.Equal(sealed.Artifact(), raw) {
		return ProgramSnapshot{}, errors.New("program hash mismatch")
	}
	if !trustedCatalog.Valid() || !validators.Valid() || document.Body.CatalogHash != trustedCatalog.Hash() {
		return ProgramSnapshot{}, errors.New("program catalog hash mismatch")
	}
	if !expectedCompilerBuild.Valid() || document.Body.CompilerBuild != expectedCompilerBuild {
		return ProgramSnapshot{}, errors.New("program compiler build mismatch")
	}
	if !document.Body.SourceHash.Valid() || document.Body.WorkflowID == "" || document.Body.Revision < 0 || document.Body.State == nil ||
		document.Body.EntryGraph == "" || len(document.Body.Graphs) != 1 || document.Body.Graphs[0].ID != document.Body.EntryGraph {
		return ProgramSnapshot{}, errors.New("program identity or entry graph is invalid")
	}
	if err := validateProgramState(document.Body.State, trustedCatalog); err != nil {
		return ProgramSnapshot{}, err
	}
	plan, err := capability.OpenPlan(document.Body.CapabilityPlan)
	if err != nil {
		return ProgramSnapshot{}, fmt.Errorf("program capability plan: %w", err)
	}
	wantPlanEntries := make([]capability.PlanEntry, 0)
	stateByName := make(map[string]programStateSlot, len(document.Body.State))
	for _, slot := range document.Body.State {
		stateByName[slot.Name] = slot
	}
	for _, graph := range document.Body.Graphs {
		if err := validateProgramGraph(graph, trustedCatalog, validators, stateByName); err != nil {
			return ProgramSnapshot{}, err
		}
		for _, node := range graph.Nodes {
			entry, ok := trustedCatalog.Lookup(node.NodeRef.NodeTypeID)
			if !ok || entry.Contract.NodeRef() != node.NodeRef || entry.Implementation != node.Implementation {
				return ProgramSnapshot{}, errors.New("program node lock mismatch")
			}
			machine := entry.Contract.Machine()
			if !reflect.DeepEqual(machine.Ports, node.Ports) || !reflect.DeepEqual(machine.Execution, node.Execution) ||
				!reflect.DeepEqual(machine.Instruction, node.Instruction) ||
				!reflect.DeepEqual(machine.HostFeatureRequirements, node.HostFeatures) {
				return ProgramSnapshot{}, errors.New("program effective contract mismatch")
			}
			effectiveRequirements, err := nodecontract.ResolveCapabilityRequirements(machine, node.Config)
			if err != nil || !reflect.DeepEqual(effectiveRequirements, node.Capabilities) {
				return ProgramSnapshot{}, errors.New("program effective capability requirements mismatch")
			}
			if !programExecutableClass(machine.Execution.Class) {
				return ProgramSnapshot{}, errors.New("program contains an unsupported execution class")
			}
			for _, requirement := range node.Capabilities {
				definition, ok := trustedCatalog.LookupCapability(requirement.Capability.CapabilityID)
				if !ok {
					return ProgramSnapshot{}, errors.New("program capability definition is missing")
				}
				normalized, err := definition.NormalizeRequirement(requirement)
				if err != nil || !reflect.DeepEqual(normalized, requirement) {
					return ProgramSnapshot{}, errors.New("program capability requirement is invalid")
				}
				wantPlanEntries = append(wantPlanEntries, capability.PlanEntry{GraphID: node.GraphPath[len(node.GraphPath)-1], NodeID: node.SourceNodeID, Requirement: requirement})
			}
		}
	}
	wantPlanEntries, err = deduplicatePlanEntries(wantPlanEntries)
	if err != nil {
		return ProgramSnapshot{}, err
	}
	wantPlan, err := capability.SealPlan(wantPlanEntries)
	if err != nil || wantPlan.Digest() != plan.Digest() || !bytes.Equal(wantPlan.Bytes(), plan.Bytes()) {
		return ProgramSnapshot{}, errors.New("program capability manifest mismatch")
	}
	return sealed, nil
}

func deduplicatePlanEntries(entries []capability.PlanEntry) ([]capability.PlanEntry, error) {
	type key struct{ graphID, nodeID, requirementID string }
	seen := make(map[key]capability.Requirement, len(entries))
	result := make([]capability.PlanEntry, 0, len(entries))
	for _, entry := range entries {
		identity := key{entry.GraphID, entry.NodeID, entry.Requirement.ID}
		if previous, exists := seen[identity]; exists {
			if !reflect.DeepEqual(previous, entry.Requirement) {
				return nil, errors.New("expanded source node has inconsistent capability requirements")
			}
			continue
		}
		seen[identity] = entry.Requirement
		result = append(result, entry)
	}
	return result, nil
}

func validateProgramState(slots []programStateSlot, catalog nodecatalog.Snapshot) error {
	if len(slots) > schema.MaxVariables {
		return errors.New("program state exceeds slot budget")
	}
	seen := make(map[string]struct{}, len(slots))
	for _, slot := range slots {
		if len(slot.Name) > 128 || !programStateNamePattern.MatchString(slot.Name) {
			return errors.New("program state contains an invalid name")
		}
		if _, duplicate := seen[slot.Name]; duplicate {
			return errors.New("program state contains duplicate names")
		}
		seen[slot.Name] = struct{}{}
		if err := slot.Type.Validate(); err != nil {
			return fmt.Errorf("program state %q has an invalid type: %w", slot.Name, err)
		}
		envelope, err := datatype.OpenValueEnvelope(catalog, slot.Initial)
		if err != nil || !envelope.Durable() || envelope.Representation() != datatype.RepresentationInlineJSON || !reflect.DeepEqual(envelope.Type(), slot.Type) {
			return fmt.Errorf("program state %q has an invalid initial value", slot.Name)
		}
	}
	return nil
}

func validateProgramGraph(graph programGraph, catalog nodecatalog.Snapshot, configValidators configvalidator.Registry, state map[string]programStateSlot) error {
	if graph.ID == "" || len(graph.Nodes) > 4096 || len(graph.SignalRoutes) > 16384 {
		return errors.New("program graph exceeds structural budget")
	}
	if graph.Nodes == nil || graph.SignalRoutes == nil || graph.DataOrder == nil {
		return errors.New("program graph instructions must use explicit arrays")
	}
	nodes := make(map[string]programNode, len(graph.Nodes))
	adjacency := map[string][]string{}
	indegree := map[string]int{}
	schemaValidators := map[string]*runtimejsonschema.Schema{}
	for _, node := range graph.Nodes {
		if node.ID == "" || node.SourceNodeID == "" || len(node.GraphPath) == 0 || len(node.GraphPath) > schema.MaxGraphPath {
			return errors.New("program contains an invalid source node location")
		}
		for _, graphID := range node.GraphPath {
			if !programStateNamePattern.MatchString(graphID) {
				return errors.New("program contains an invalid graph path")
			}
		}
		if _, duplicate := nodes[node.ID]; duplicate {
			return errors.New("program contains duplicate node ids")
		}
		nodes[node.ID] = node
		indegree[node.ID] = 0
	}
	seenRoutes := make(map[programSignalRoute]struct{}, len(graph.SignalRoutes))
	for _, route := range graph.SignalRoutes {
		if route.Channel != schema.EdgeExec && route.Channel != schema.EdgeError {
			return errors.New("program signal route has an invalid channel")
		}
		fromNode, fromOK := nodes[route.From.NodeID]
		toNode, toOK := nodes[route.To.NodeID]
		if !fromOK || !toOK {
			return errors.New("program signal route references an unknown node")
		}
		if _, exists := seenRoutes[route]; exists {
			return errors.New("program contains a duplicate signal route")
		}
		seenRoutes[route] = struct{}{}
		if _, exists := outputForChannel(fromNode.Ports, route.Channel, route.From.PortID); !exists {
			return errors.New("program signal route references an unknown output")
		}
		if _, exists := inputForChannel(toNode.Ports, route.Channel, route.To.PortID); !exists {
			return errors.New("program signal route references an unknown input")
		}
		if !toNode.Instruction.AcceptsSignalInput(string(route.Channel), route.To.PortID) {
			return errors.New("program signal route violates its target instruction")
		}
	}
	violations, err := regionSignalScopeViolations(context.Background(), graph)
	if err != nil || len(violations) != 0 {
		return errors.New("program region signal escapes its activation scope")
	}
	for _, node := range graph.Nodes {
		entry, ok := catalog.Lookup(node.NodeRef.NodeTypeID)
		if !ok {
			return errors.New("program references an unknown node type")
		}
		machine := entry.Contract.Machine()
		machine, err = nodeinstance.Resolve(machine, node.Config)
		if err != nil {
			return fmt.Errorf("program node %q has an invalid effective contract: %w", node.ID, err)
		}
		if err := validateEffectivePortTypes(node, machine, state); err != nil {
			return fmt.Errorf("program node %q has invalid effective types: %w", node.ID, err)
		}
		if err := validateJSONSchemaBundleCached(schemaValidators, "config:"+node.NodeRef.SemanticDigest.String(), machine.ConfigSchemaRoot, machine.ConfigSchemaBundle, node.Config); err != nil {
			return fmt.Errorf("program node %q has invalid config: %w", node.ID, err)
		}
		if err := configValidators.Validate(machine, node.Config); err != nil {
			return fmt.Errorf("program node %q has invalid config semantics: %w", node.ID, err)
		}
		ports := make(map[string]nodecontract.DataInputPort, len(machine.Ports.DataInputs))
		for _, port := range machine.Ports.DataInputs {
			ports[port.ID] = port
		}
		for portID, plan := range node.Inputs {
			port, exists := ports[portID]
			if !exists {
				return fmt.Errorf("program node %q has an unknown input %q", node.ID, portID)
			}
			if plan.Kind == inputLiteral && port.ResourceLease != nil {
				return fmt.Errorf("program node %q has a literal for runtime authority input %q", node.ID, portID)
			}
			switch plan.Kind {
			case inputLiteral:
				if len(plan.Value) == 0 || plan.From.NodeID != "" || plan.From.PortID != "" ||
					(plan.Provenance != inputSourceLiteral && plan.Provenance != inputSourceDefault && plan.Provenance != inputSourceBlob) {
					return fmt.Errorf("program node %q has a malformed literal input %q", node.ID, portID)
				}
				envelope, err := datatype.OpenValueEnvelope(catalog, plan.Value)
				if err != nil {
					return fmt.Errorf("program node %q has an invalid value envelope for input %q: %w", node.ID, portID, err)
				}
				expected, ok := node.InputTypes[portID]
				if !ok || !reflect.DeepEqual(envelope.Type(), expected) {
					return fmt.Errorf("program node %q has a forged value type for input %q", node.ID, portID)
				}
				if plan.Provenance == inputSourceBlob {
					if _, ok := envelope.BlobRef(); !ok {
						return fmt.Errorf("program node %q has a forged blob input %q", node.ID, portID)
					}
				}
			case inputEdge:
				if len(plan.Value) != 0 || plan.Provenance != "" {
					return fmt.Errorf("program node %q has a forged edge input %q", node.ID, portID)
				}
				fromNode, exists := nodes[plan.From.NodeID]
				if !exists {
					return fmt.Errorf("program node %q has an edge from an unknown node", node.ID)
				}
				_, fromExists := outputForChannel(fromNode.Ports, schema.EdgeData, plan.From.PortID)
				if !fromExists {
					return fmt.Errorf("program node %q has an edge from an unknown output", node.ID)
				}
				fromType, fromOK := fromNode.OutputTypes[plan.From.PortID]
				toType, toOK := node.InputTypes[portID]
				if !fromOK || !toOK || !reflect.DeepEqual(fromType, toType) {
					return fmt.Errorf("program node %q has an incompatible data edge", node.ID)
				}
				fromPort, _ := dataOutputPort(fromNode.Ports, plan.From.PortID)
				if !resourceLeaseAssignable(fromPort.ResourceLease, port.ResourceLease) {
					return fmt.Errorf("program node %q has incompatible resource authority", node.ID)
				}
				adjacency[plan.From.NodeID] = append(adjacency[plan.From.NodeID], node.ID)
				indegree[node.ID]++
			default:
				return fmt.Errorf("program node %q has an unknown input plan", node.ID)
			}
		}
		for _, port := range machine.Ports.DataInputs {
			if port.Required {
				if _, present := node.Inputs[port.ID]; !present {
					return fmt.Errorf("program node %q is missing required input %q", node.ID, port.ID)
				}
			}
		}
	}
	if expected := topologicalOrder(graph.Nodes, adjacency, indegree); len(expected) != len(graph.Nodes) || !slices.Equal(expected, graph.DataOrder) {
		return errors.New("program data order does not match its data graph")
	}
	if len(graph.Nodes) != 0 {
		roots, _ := executionReachability(graph)
		if len(roots) == 0 {
			return errors.New("program graph has no execution root")
		}
	}
	return nil
}

func validateEffectivePortTypes(node programNode, machine nodecontract.MachineContract, state map[string]programStateSlot) error {
	if node.InputTypes == nil || node.OutputTypes == nil ||
		len(node.InputTypes) != len(machine.Ports.DataInputs) || len(node.OutputTypes) != len(machine.Ports.DataOutputs) {
		return errors.New("effective port type maps do not match the contract")
	}
	variables := map[string]datatype.ResolvedType{}
	for _, port := range machine.Ports.DataInputs {
		resolved, ok := node.InputTypes[port.ID]
		if !ok {
			return fmt.Errorf("input %q has no effective type", port.ID)
		}
		matched, err := datatype.MatchResolved(port.Type, resolved, variables)
		if err != nil || !matched {
			return fmt.Errorf("input %q does not satisfy its contract type", port.ID)
		}
	}
	for _, port := range machine.Ports.DataOutputs {
		resolved, ok := node.OutputTypes[port.ID]
		if !ok {
			return fmt.Errorf("output %q has no effective type", port.ID)
		}
		matched, err := datatype.MatchResolved(port.Type, resolved, variables)
		if err != nil || !matched {
			return fmt.Errorf("output %q does not satisfy its contract type", port.ID)
		}
	}
	for _, access := range machine.StateAccesses {
		slotName, ok := node.Config[access.SlotConfigKey].(string)
		slot, exists := state[slotName]
		if !ok || !exists {
			return fmt.Errorf("state access %q references an unknown slot", access.ID)
		}
		matched, err := datatype.MatchResolved(access.Type, slot.Type, variables)
		if err != nil || !matched {
			return fmt.Errorf("state access %q does not satisfy its contract type", access.ID)
		}
	}
	return nil
}

func (p ProgramSnapshot) Valid() bool { return p.state != nil && p.state.document.ProgramHash.Valid() }

func (p ProgramSnapshot) Hash() artifact.Digest {
	if !p.Valid() {
		return ""
	}
	return p.state.document.ProgramHash
}

func (p ProgramSnapshot) CatalogHash() artifact.Digest {
	if !p.Valid() {
		return ""
	}
	return p.state.document.Body.CatalogHash
}

func (p ProgramSnapshot) Artifact() []byte {
	if !p.Valid() {
		return nil
	}
	return append([]byte(nil), p.state.artifact...)
}

func (p ProgramSnapshot) CapabilityPlan() capability.Plan {
	if !p.Valid() {
		return capability.Plan{}
	}
	plan, err := capability.OpenPlan(p.state.document.Body.CapabilityPlan)
	if err != nil {
		panic("program snapshot invariant: " + err.Error())
	}
	return plan
}

func (p ProgramSnapshot) Nodes() []NodeView {
	if !p.Valid() {
		return nil
	}
	var result []NodeView
	for _, graph := range p.state.document.Body.Graphs {
		for _, node := range graph.Nodes {
			view := NodeView{
				GraphID: node.GraphPath[len(node.GraphPath)-1], GraphPath: node.GraphPath, ID: node.SourceNodeID, NodeRef: node.NodeRef, Ports: node.Ports,
				InputTypes: node.InputTypes, OutputTypes: node.OutputTypes,
				Execution: node.Execution, Instruction: node.Instruction, HostFeatures: node.HostFeatures, Capabilities: node.Capabilities, Implementation: node.Implementation,
			}
			raw, err := json.Marshal(view)
			if err != nil {
				panic("program snapshot invariant: " + err.Error())
			}
			var clone NodeView
			if err := json.Unmarshal(raw, &clone); err != nil {
				panic("program snapshot invariant: " + err.Error())
			}
			result = append(result, clone)
		}
	}
	return result
}

func (p ProgramSnapshot) State() []StateView {
	if !p.Valid() {
		return nil
	}
	result := make([]StateView, 0, len(p.state.document.Body.State))
	for _, slot := range p.state.document.Body.State {
		rawType, err := json.Marshal(slot.Type)
		if err != nil {
			panic("program snapshot invariant: " + err.Error())
		}
		var clonedType datatype.ResolvedType
		if err := json.Unmarshal(rawType, &clonedType); err != nil {
			panic("program snapshot invariant: " + err.Error())
		}
		result = append(result, StateView{Name: slot.Name, Type: clonedType, InitialArtifact: append([]byte(nil), slot.Initial...)})
	}
	return result
}

func (p ProgramSnapshot) BlobReferences(catalog datatype.ValueTypeCatalog) ([]blob.BlobRef, error) {
	if !p.Valid() || catalog == nil {
		return nil, errors.New("program blob inventory requires Program and type catalog")
	}
	refs := make([]blob.BlobRef, 0)
	seen := make(map[blob.BlobRef]struct{})
	for _, graph := range p.state.document.Body.Graphs {
		for _, node := range graph.Nodes {
			for _, input := range node.Inputs {
				if input.Kind != inputLiteral {
					continue
				}
				envelope, err := datatype.OpenValueEnvelope(catalog, input.Value)
				if err != nil {
					return nil, fmt.Errorf("open Program input value: %w", err)
				}
				if ref, ok := envelope.BlobRef(); ok {
					if _, duplicate := seen[ref]; !duplicate {
						seen[ref] = struct{}{}
						refs = append(refs, ref)
					}
				}
			}
		}
	}
	return refs, nil
}
