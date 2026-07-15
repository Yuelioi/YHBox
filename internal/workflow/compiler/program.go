package compiler

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"reflect"
	"slices"

	runtimejsonschema "github.com/santhosh-tekuri/jsonschema/v6"
	"github.com/yottaapp/yotta/internal/artifact"
	"github.com/yottaapp/yotta/internal/capability"
	"github.com/yottaapp/yotta/internal/datatype"
	"github.com/yottaapp/yotta/internal/nodecatalog"
	"github.com/yottaapp/yotta/internal/nodecontract"
	"github.com/yottaapp/yotta/internal/workflow/schema"
)

const (
	ProgramFormat       = "yotta.program"
	ProgramVersion      = "3.1"
	MaxProgramBytes     = 16 << 20
	MaxProgramJSONDepth = 128
	MaxProgramJSONNodes = 1_048_576
	programHashDomain   = "yotta/program/v1"
)

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
	ID             string                         `json:"id"`
	NodeRef        nodecontract.NodeRef           `json:"nodeRef"`
	Config         map[string]any                 `json:"config"`
	Inputs         map[string]inputPlan           `json:"inputs"`
	Ports          nodecontract.PortSet           `json:"ports"`
	Execution      nodecontract.ExecutionSpec     `json:"execution"`
	Implementation nodecatalog.ImplementationLock `json:"implementation"`
}

type programGraph struct {
	ID    string        `json:"id"`
	Nodes []programNode `json:"nodes"`
	Edges []schema.Edge `json:"edges"`
	Order []string      `json:"order"`
}

type programBody struct {
	SourceHash     artifact.Digest `json:"sourceHash"`
	CatalogHash    artifact.Digest `json:"catalogHash"`
	CompilerBuild  artifact.Digest `json:"compilerBuild"`
	WorkflowID     string          `json:"workflowId"`
	Revision       int64           `json:"revision"`
	EntryGraph     string          `json:"entryGraph"`
	CapabilityPlan json.RawMessage `json:"capabilityPlan"`
	Graphs         []programGraph  `json:"graphs"`
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
	ID             string
	NodeRef        nodecontract.NodeRef
	Ports          nodecontract.PortSet
	Execution      nodecontract.ExecutionSpec
	Implementation nodecatalog.ImplementationLock
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

func OpenProgram(raw []byte, trustedCatalog nodecatalog.Snapshot, expectedCompilerBuild artifact.Digest) (ProgramSnapshot, error) {
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
	if !trustedCatalog.Valid() || document.Body.CatalogHash != trustedCatalog.Hash() {
		return ProgramSnapshot{}, errors.New("program catalog hash mismatch")
	}
	if !expectedCompilerBuild.Valid() || document.Body.CompilerBuild != expectedCompilerBuild {
		return ProgramSnapshot{}, errors.New("program compiler build mismatch")
	}
	if !document.Body.SourceHash.Valid() || document.Body.WorkflowID == "" || document.Body.Revision < 0 ||
		document.Body.EntryGraph == "" || len(document.Body.Graphs) != 1 || document.Body.Graphs[0].ID != document.Body.EntryGraph {
		return ProgramSnapshot{}, errors.New("program identity or entry graph is invalid")
	}
	plan, err := capability.OpenPlan(document.Body.CapabilityPlan)
	if err != nil {
		return ProgramSnapshot{}, fmt.Errorf("program capability plan: %w", err)
	}
	wantPlanEntries := make([]capability.PlanEntry, 0)
	for _, graph := range document.Body.Graphs {
		if err := validateProgramGraph(graph, trustedCatalog); err != nil {
			return ProgramSnapshot{}, err
		}
		for _, node := range graph.Nodes {
			entry, ok := trustedCatalog.Lookup(node.NodeRef.NodeTypeID)
			if !ok || entry.Contract.NodeRef() != node.NodeRef || entry.Implementation != node.Implementation {
				return ProgramSnapshot{}, errors.New("program node lock mismatch")
			}
			machine := entry.Contract.Machine()
			if !reflect.DeepEqual(machine.Ports, node.Ports) || !reflect.DeepEqual(machine.Execution, node.Execution) {
				return ProgramSnapshot{}, errors.New("program effective contract mismatch")
			}
			if machine.Execution.Class != nodecontract.ExecutionPureData && machine.Execution.Class != nodecontract.ExecutionEffect {
				return ProgramSnapshot{}, errors.New("program contains an unsupported execution class")
			}
			for _, requirement := range machine.CapabilityRequirements {
				definition, ok := trustedCatalog.LookupCapability(requirement.Capability.CapabilityID)
				if !ok {
					return ProgramSnapshot{}, errors.New("program capability definition is missing")
				}
				normalized, err := definition.NormalizeRequirement(requirement)
				if err != nil || !reflect.DeepEqual(normalized, requirement) {
					return ProgramSnapshot{}, errors.New("program capability requirement is invalid")
				}
				wantPlanEntries = append(wantPlanEntries, capability.PlanEntry{GraphID: graph.ID, NodeID: node.ID, Requirement: requirement})
			}
		}
	}
	wantPlan, err := capability.SealPlan(wantPlanEntries)
	if err != nil || wantPlan.Digest() != plan.Digest() || !bytes.Equal(wantPlan.Bytes(), plan.Bytes()) {
		return ProgramSnapshot{}, errors.New("program capability manifest mismatch")
	}
	return sealed, nil
}

func validateProgramGraph(graph programGraph, catalog nodecatalog.Snapshot) error {
	if graph.ID == "" || len(graph.Nodes) > 4096 || len(graph.Edges) > 16384 {
		return errors.New("program graph exceeds structural budget")
	}
	nodes := make(map[string]programNode, len(graph.Nodes))
	adjacency := map[string][]string{}
	indegree := map[string]int{}
	edgesByInput := map[string]schema.Endpoint{}
	validators := map[string]*runtimejsonschema.Schema{}
	for _, node := range graph.Nodes {
		if node.ID == "" {
			return errors.New("program contains an empty node id")
		}
		if _, duplicate := nodes[node.ID]; duplicate {
			return errors.New("program contains duplicate node ids")
		}
		nodes[node.ID] = node
		indegree[node.ID] = 0
	}
	for _, edge := range graph.Edges {
		if edge.Channel != schema.EdgeData {
			return errors.New("program contains an unsupported edge channel")
		}
		if _, ok := nodes[edge.From.NodeID]; !ok {
			return errors.New("program data edge has an unknown source")
		}
		if _, ok := nodes[edge.To.NodeID]; !ok {
			return errors.New("program data edge has an unknown target")
		}
		fromNode, toNode := nodes[edge.From.NodeID], nodes[edge.To.NodeID]
		fromType, fromExists := outputForChannel(fromNode.Ports, schema.EdgeData, edge.From.PortID)
		toType, toExists := inputForChannel(toNode.Ports, schema.EdgeData, edge.To.PortID)
		if !fromExists || !toExists {
			return errors.New("program data edge references an unknown port")
		}
		assignable, err := datatype.Assignable(*fromType, *toType)
		if err != nil || !assignable {
			return errors.New("program data edge has incompatible types")
		}
		fromPort, _ := dataOutputPort(fromNode.Ports, edge.From.PortID)
		toPort, _ := dataInputPort(toNode.Ports, edge.To.PortID)
		if !resourceLeaseAssignable(fromPort.ResourceLease, toPort.ResourceLease) {
			return errors.New("program data edge has incompatible resource authority")
		}
		key := edge.To.NodeID + "\x00" + edge.To.PortID
		if _, duplicate := edgesByInput[key]; duplicate {
			return errors.New("program contains duplicate data input edges")
		}
		edgesByInput[key] = edge.From
		adjacency[edge.From.NodeID] = append(adjacency[edge.From.NodeID], edge.To.NodeID)
		indegree[edge.To.NodeID]++
	}
	if expected := topologicalOrder(graph.Nodes, adjacency, indegree); !slices.Equal(expected, graph.Order) {
		return errors.New("program execution order does not match its data graph")
	}
	for _, node := range graph.Nodes {
		entry, ok := catalog.Lookup(node.NodeRef.NodeTypeID)
		if !ok {
			return errors.New("program references an unknown node type")
		}
		machine := entry.Contract.Machine()
		if err := validateJSONSchemaBundleCached(validators, "config:"+node.NodeRef.SemanticDigest.String(), machine.ConfigSchemaRoot, machine.ConfigSchemaBundle, node.Config); err != nil {
			return fmt.Errorf("program node %q has invalid config: %w", node.ID, err)
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
				expected, err := resolvedTypeForExactRef(port.Type, catalog)
				if err != nil || !reflect.DeepEqual(envelope.Type(), expected) {
					return fmt.Errorf("program node %q has a forged value type for input %q", node.ID, portID)
				}
				if plan.Provenance == inputSourceBlob {
					if _, ok := envelope.BlobRef(); !ok {
						return fmt.Errorf("program node %q has a forged blob input %q", node.ID, portID)
					}
				} else if err := validateLiteralCached(port.Type, envelope.InlineJSON(), catalog, validators); err != nil {
					return fmt.Errorf("program node %q has an invalid literal input %q: %w", node.ID, portID, err)
				}
			case inputEdge:
				if len(plan.Value) != 0 || plan.Provenance != "" || edgesByInput[node.ID+"\x00"+portID] != plan.From {
					return fmt.Errorf("program node %q has a forged edge input %q", node.ID, portID)
				}
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
	for target, from := range edgesByInput {
		matched := false
		for _, node := range graph.Nodes {
			for portID, plan := range node.Inputs {
				if node.ID+"\x00"+portID == target && plan.Kind == inputEdge && plan.From == from {
					matched = true
				}
			}
		}
		if !matched {
			return errors.New("program data edge is missing its input plan")
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
			view := NodeView{ID: node.ID, NodeRef: node.NodeRef, Ports: node.Ports, Execution: node.Execution, Implementation: node.Implementation}
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
