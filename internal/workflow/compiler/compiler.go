// Package compiler is the only path from editable Workflow Source to an
// immutable ProgramSnapshot.
package compiler

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"math/big"
	"sort"
	"strconv"
	"strings"

	"github.com/yottaapp/yotta/internal/artifact"
	"github.com/yottaapp/yotta/internal/node"
	"github.com/yottaapp/yotta/internal/workflow/catalog"
	"github.com/yottaapp/yotta/internal/workflow/schema"
)

const (
	CallSubgraphKind = catalog.CompilerIntrinsicPrefix + "call-subgraph"
	callFailurePin   = "Fail"
	callInputPin     = "In"
	sourceHashDomain = "yotta/source/v3"
	MaxSourceBytes   = 8 << 20
	MaxJSONDepth     = 128
	MaxGraphs        = 256
	MaxNodesPerGraph = 4096
	MaxEdgesPerGraph = 16384
	MaxPortsPerGraph = 4096
	MaxTotalNodes    = 20000
	MaxTotalEdges    = 50000
	MaxTotalPorts    = 10000
	MaxCallPlanPorts = 10000
	MaxCallPlanBytes = 4 << 20
	MaxDiagnostics   = 10000
)

var ErrInvalidCatalog = errors.New("invalid catalog snapshot")
var ErrSourceBudgetExceeded = errors.New("workflow source exceeds compiler budget")

type Compiler struct{ build artifact.Digest }

func New(build artifact.Digest) (*Compiler, error) {
	if !build.Valid() {
		return nil, errors.New("compiler build identity must be a content digest")
	}
	return &Compiler{build: build}, nil
}

type CompileRequest struct {
	SourceJSON []byte
	Catalog    catalog.Snapshot
}

type CompileResult struct {
	SourceHash  artifact.Digest
	Diagnostics []schema.Diagnostic
	program     ProgramSnapshot
}

func (r CompileResult) Program() (ProgramSnapshot, bool) {
	return r.program, r.program.Valid()
}

// CompileDraft reports source and catalog failures as stable diagnostics.
// Errors are reserved for cancellation and compiler infrastructure failures.
func (c *Compiler) CompileDraft(ctx context.Context, request CompileRequest) (CompileResult, error) {
	if c == nil || !c.build.Valid() {
		return CompileResult{}, errors.New("compiler is not initialized")
	}
	if err := ctx.Err(); err != nil {
		return CompileResult{}, err
	}
	if len(request.SourceJSON) > MaxSourceBytes || exceedsJSONDepth(request.SourceJSON, MaxJSONDepth) {
		return CompileResult{}, ErrSourceBudgetExceeded
	}
	result := CompileResult{}

	source, diagnostics := schema.ParseSource(request.SourceJSON)
	if len(diagnostics) > 0 {
		result.Diagnostics = cappedDiagnostics(diagnostics)
		return result, nil
	}
	if exceedsSourceCollections(source) {
		return result, ErrSourceBudgetExceeded
	}
	canonicalSource, err := artifact.Canonicalize(request.SourceJSON)
	if err != nil {
		result.Diagnostics = []schema.Diagnostic{{
			Code: schema.CodeInvalidField, Severity: schema.SeverityError,
			Params: map[string]any{"keyword": "rfc8785"},
		}}
		return result, nil
	}
	result.SourceHash, err = artifact.Sum(sourceHashDomain, canonicalSource)
	if err != nil {
		return result, err
	}
	if !request.Catalog.Valid() {
		return result, ErrInvalidCatalog
	}

	binding, err := bindSource(ctx, source, request.Catalog)
	if err != nil {
		return result, err
	}
	if len(binding.diagnostics) > 0 {
		result.Diagnostics = binding.diagnostics
		return result, nil
	}
	declaredCapabilities := capabilitiesFromSource(source.RequestedCapabilities)
	capabilityDiagnostics := validateCapabilityDeclaration(declaredCapabilities, binding.capabilities)
	if len(capabilityDiagnostics) > 0 {
		result.Diagnostics = capabilityDiagnostics
		return result, nil
	}
	if err := ctx.Err(); err != nil {
		return result, err
	}
	body := programBody{
		SourceHash: result.SourceHash, CatalogHash: request.Catalog.Hash(),
		CompilerBuild: c.build, ImplementationSet: request.Catalog.ImplementationSet(),
		WorkflowID: source.Workflow.ID, Revision: source.Revision, EntryGraph: source.EntryGraph,
		NodeLocks: binding.locks, RequestedCapabilities: declaredCapabilities, RequiredCapabilities: binding.capabilities,
		Variables: source.Variables, SecretRefs: source.SecretRefs,
		Graphs: lowerGraphs(source.Graphs, binding),
	}
	result.program, err = sealProgram(body)
	if err != nil {
		return result, fmt.Errorf("seal program: %w", err)
	}
	return result, nil
}

type bindingResult struct {
	locks        []NodeLock
	capabilities []string
	diagnostics  []schema.Diagnostic
	reachable    map[string]bool
	calls        map[string]map[string]*programCall
}

type graphInterface struct {
	entry   schema.GraphPort
	inputs  map[string]schema.GraphPort
	outputs map[string]schema.GraphPort
}

type callReference struct {
	graphID, nodeID, target string
	graphIndex, nodeIndex   int
}

func bindSource(ctx context.Context, source schema.WorkflowSource, snapshot catalog.Snapshot) (bindingResult, error) {
	result := bindingResult{reachable: map[string]bool{}, calls: map[string]map[string]*programCall{}}
	locksByKind := map[string]NodeLock{}
	capabilities := map[string]bool{}
	contractCache := map[string]catalog.NodeContract{}
	validatedContractDefaults := map[string]bool{}
	pinIndexesByKind := map[string]nodePinIndex{}
	configIndexesByKind := map[string]nodeConfigContract{}
	missingKinds := map[string]bool{}
	graphsByID := make(map[string]schema.Graph, len(source.Graphs))
	interfaces := make(map[string]graphInterface, len(source.Graphs))
	totalCallPlanPorts, totalCallPlanBytes := 0, 0
	for graphIndex, graph := range source.Graphs {
		graphsByID[graph.ID] = graph
		interfaces[graph.ID] = validateGraphInterface(&result.diagnostics, graph, graphIndex)
	}

	adjacency := map[string][]callReference{}
	for graphIndex, graph := range source.Graphs {
		for nodeIndex, sourceNode := range graph.Nodes {
			if sourceNode.Kind != CallSubgraphKind {
				continue
			}
			path := []string{"graphs", strconv.Itoa(graphIndex), "nodes", strconv.Itoa(nodeIndex), "config", "graphId"}
			target, ok := sourceNode.Config["graphId"].(string)
			if !ok || target == "" {
				appendDiagnostic(&result.diagnostics, bindingDiagnostic(graph.ID, sourceNode.ID, path, "calleeGraphId", map[string]any{}))
				continue
			}
			targetGraph, exists := graphsByID[target]
			if !exists {
				appendDiagnostic(&result.diagnostics, schema.Diagnostic{Code: schema.CodeUnknownCalleeGraph, Severity: schema.SeverityError, GraphPath: []string{graph.ID}, NodeID: sourceNode.ID, FieldPath: path, Params: map[string]any{"graphId": target}})
				continue
			}
			if targetGraph.Kind != schema.GraphKindSubgraph {
				appendDiagnostic(&result.diagnostics, schema.Diagnostic{Code: schema.CodeInvalidCalleeGraphKind, Severity: schema.SeverityError, GraphPath: []string{graph.ID}, NodeID: sourceNode.ID, FieldPath: path, Params: map[string]any{"graphId": target, "kind": targetGraph.Kind}})
				continue
			}
			ref := callReference{graphID: graph.ID, nodeID: sourceNode.ID, target: target, graphIndex: graphIndex, nodeIndex: nodeIndex}
			adjacency[graph.ID] = append(adjacency[graph.ID], ref)
			callPorts := len(targetGraph.Inputs) + len(targetGraph.Outputs)
			callBytes := callPlanEncodedSize(target, targetGraph)
			if totalCallPlanPorts > MaxCallPlanPorts-callPorts || totalCallPlanBytes > MaxCallPlanBytes-callBytes {
				return result, ErrSourceBudgetExceeded
			}
			totalCallPlanPorts += callPorts
			totalCallPlanBytes += callBytes
			iface := interfaces[target]
			plan := &programCall{GraphID: target, Entry: programCallPort{ID: iface.entry.ID, Type: iface.entry.Type, NodeID: iface.entry.NodeID}, Inputs: []programCallPort{}, Outputs: []programCallPort{}, FailurePin: callFailurePin}
			for _, port := range targetGraph.Inputs {
				if port.Type != node.TypeExec {
					plan.Inputs = append(plan.Inputs, programCallPort{ID: port.ID, Type: port.Type, NodeID: port.NodeID})
				}
			}
			for _, port := range targetGraph.Outputs {
				plan.Outputs = append(plan.Outputs, programCallPort{ID: port.ID, Type: port.Type, NodeID: port.NodeID})
			}
			if result.calls[graph.ID] == nil {
				result.calls[graph.ID] = map[string]*programCall{}
			}
			result.calls[graph.ID][sourceNode.ID] = plan
		}
	}
	appendCallCycleDiagnostics(&result.diagnostics, source.Graphs, adjacency)
	queue := []string{source.EntryGraph}
	for len(queue) > 0 {
		graphID := queue[0]
		queue = queue[1:]
		if result.reachable[graphID] {
			continue
		}
		result.reachable[graphID] = true
		for _, ref := range adjacency[graphID] {
			queue = append(queue, ref.target)
		}
	}

	for graphIndex, graph := range source.Graphs {
		if err := ctx.Err(); err != nil {
			return result, err
		}
		contractsByNode := make(map[string]catalog.NodeContract, len(graph.Nodes))
		nodeExists := make(map[string]bool, len(graph.Nodes))
		callNodeIDs := make(map[string]bool)
		for nodeIndex, sourceNode := range graph.Nodes {
			if nodeIndex&255 == 0 {
				if err := ctx.Err(); err != nil {
					return result, err
				}
			}
			nodeExists[sourceNode.ID] = true
			if sourceNode.Kind == CallSubgraphKind {
				callNodeIDs[sourceNode.ID] = true
				if plan := result.calls[graph.ID][sourceNode.ID]; plan != nil {
					contract := callContract(plan)
					contractsByNode[sourceNode.ID] = contract
				}
				continue
			}
			contract, cached := contractCache[sourceNode.Kind]
			ok := cached
			if !cached && !missingKinds[sourceNode.Kind] {
				contract, ok = snapshot.Lookup(sourceNode.Kind)
				if ok {
					contractCache[sourceNode.Kind] = contract
				} else {
					missingKinds[sourceNode.Kind] = true
				}
			}
			if !ok {
				appendDiagnostic(&result.diagnostics, schema.Diagnostic{
					Code: schema.CodeUnknownNodeKind, Severity: schema.SeverityError,
					GraphPath: []string{graph.ID}, NodeID: sourceNode.ID,
					FieldPath: []string{"graphs", strconv.Itoa(graphIndex), "nodes", strconv.Itoa(nodeIndex), "kind"},
					Params:    map[string]any{"kind": sourceNode.Kind},
				})
				continue
			}
			contractsByNode[sourceNode.ID] = contract
			if !validatedContractDefaults[contract.Kind] {
				for _, input := range contract.Inputs {
					if input.Default != nil && (!valueMatchesType(input.Default, input.Type) || input.Schema != nil && !valueMatchesSchema(input.Default, input.Schema)) {
						return result, fmt.Errorf("%w: node %q input %q has invalid default", ErrInvalidCatalog, contract.Kind, input.Name)
					}
				}
				validatedContractDefaults[contract.Kind] = true
			}
			if contract.DynamicInputs || contract.DynamicOutputs || contract.DynamicDataFields || contract.HasCustomValidation || contract.HasDependencies {
				appendDiagnostic(&result.diagnostics, schema.Diagnostic{
					Code: schema.CodeUnsupportedNodeContract, Severity: schema.SeverityError,
					GraphPath: []string{graph.ID}, NodeID: sourceNode.ID,
					FieldPath: []string{"graphs", strconv.Itoa(graphIndex), "nodes", strconv.Itoa(nodeIndex), "kind"},
					Params:    map[string]any{"kind": sourceNode.Kind},
				})
				continue
			}
			if result.reachable[graph.ID] {
				locksByKind[contract.Kind] = NodeLock{Kind: contract.Kind, ContractHash: contract.ContractHash}
				for _, capability := range contract.RuntimeCapabilities {
					capabilities["runtime:"+string(capability)] = true
				}
				for _, capability := range contract.TargetCapabilities {
					capabilities["target:"+string(capability)] = true
				}
				if contract.NeedsTarget {
					capabilities["target:required"] = true
				}
				if contract.NeedsWindow {
					capabilities["platform:win32-window"] = true
				}
			}
		}
		incoming := map[string]map[string]bool{}
		incomingCounts := map[string]map[string]int{}
		usedBoundaryInputs, usedBoundaryOutputs := map[string]bool{}, map[string]bool{}
		boundaryOutputCounts := map[string]int{}
		iface := interfaces[graph.ID]
		pinsByNode := indexNodePins(contractsByNode, pinIndexesByKind)
		configByNode := indexNodeConfigContracts(contractsByNode, configIndexesByKind)
		for edgeIndex, edge := range graph.Edges {
			if edgeIndex&255 == 0 {
				if err := ctx.Err(); err != nil {
					return result, err
				}
			}
			from := resolveEndpoint(edge.From, true, nodeExists, pinsByNode, iface)
			to := resolveEndpoint(edge.To, false, nodeExists, pinsByNode, iface)
			for _, endpoint := range []struct {
				field string
				value string
				from  bool
				bound endpointResolution
			}{{"from", edge.From, true, from}, {"to", edge.To, false, to}} {
				path := []string{"graphs", strconv.Itoa(graphIndex), "edges", strconv.Itoa(edgeIndex), endpoint.field}
				if endpoint.bound.wrongDirection {
					appendDiagnostic(&result.diagnostics, schema.Diagnostic{Code: schema.CodeInvalidGraphBoundaryEdge, Severity: schema.SeverityError, GraphPath: []string{graph.ID}, FieldPath: path, Params: map[string]any{"endpoint": endpoint.value}})
					continue
				}
				if !endpoint.bound.valid {
					appendDiagnostic(&result.diagnostics, bindingDiagnostic(graph.ID, "", path, "edgeEndpoint", map[string]any{"endpoint": endpoint.value}))
					continue
				}
				if endpoint.bound.nodeID == "" {
					continue
				}
				contract, bound := contractsByNode[endpoint.bound.nodeID]
				if !bound {
					continue
				}
				if !endpoint.bound.pinValid {
					appendDiagnostic(&result.diagnostics, bindingDiagnostic(graph.ID, endpoint.bound.nodeID, path, "edgePin", map[string]any{"pin": endpoint.bound.pin, "kind": contract.Kind}))
				}
			}
			if from.valid && to.valid && from.pinValid && to.pinValid && from.typ != "" && to.typ != "" {
				if !compatibleTypes(from.typ, to.typ) {
					path := []string{"graphs", strconv.Itoa(graphIndex), "edges", strconv.Itoa(edgeIndex)}
					if from.boundary || to.boundary || callNodeIDs[from.nodeID] || callNodeIDs[to.nodeID] {
						appendDiagnostic(&result.diagnostics, schema.Diagnostic{Code: schema.CodeCallPinTypeMismatch, Severity: schema.SeverityError, GraphPath: []string{graph.ID}, NodeID: to.nodeID, FieldPath: path, Params: map[string]any{"fromType": from.typ, "toType": to.typ}})
					} else {
						appendDiagnostic(&result.diagnostics, bindingDiagnostic(graph.ID, to.nodeID, path, "pinType", map[string]any{"fromType": from.typ, "toType": to.typ}))
					}
				} else {
					if from.boundary {
						usedBoundaryInputs[edge.From] = true
					}
					if to.boundary {
						usedBoundaryOutputs[edge.To] = true
						boundaryOutputCounts[edge.To]++
						if to.typ != node.TypeExec && boundaryOutputCounts[edge.To] == 2 {
							path := []string{"graphs", strconv.Itoa(graphIndex), "edges", strconv.Itoa(edgeIndex), "to"}
							appendDiagnostic(&result.diagnostics, bindingDiagnostic(graph.ID, "", path, "multipleGraphOutputSources", map[string]any{"endpoint": edge.To}))
						}
					}
					if to.nodeID != "" {
						if incoming[to.nodeID] == nil {
							incoming[to.nodeID] = map[string]bool{}
							incomingCounts[to.nodeID] = map[string]int{}
						}
						incoming[to.nodeID][to.pin] = true
						incomingCounts[to.nodeID][to.pin]++
						if to.typ != node.TypeExec && incomingCounts[to.nodeID][to.pin] == 2 {
							path := []string{"graphs", strconv.Itoa(graphIndex), "edges", strconv.Itoa(edgeIndex), "to"}
							appendDiagnostic(&result.diagnostics, bindingDiagnostic(graph.ID, to.nodeID, path, "multipleInputSources", map[string]any{"pin": to.pin}))
						}
					}
				}
			}
		}
		validateBoundaryCoverage(&result.diagnostics, graph, graphIndex, usedBoundaryInputs, usedBoundaryOutputs)
		for nodeIndex, sourceNode := range graph.Nodes {
			contract, ok := contractsByNode[sourceNode.ID]
			if !ok || contract.DynamicInputs || contract.DynamicOutputs || contract.DynamicDataFields || contract.HasCustomValidation || contract.HasDependencies {
				continue
			}
			if sourceNode.Kind == CallSubgraphKind {
				copyNode := sourceNode
				copyNode.Config = make(map[string]any, len(sourceNode.Config))
				for key, value := range sourceNode.Config {
					if key != "graphId" {
						copyNode.Config[key] = value
					}
				}
				validateNodeConfig(&result.diagnostics, graph.ID, graphIndex, nodeIndex, copyNode, configByNode[sourceNode.ID], incoming[sourceNode.ID])
			} else {
				validateNodeConfig(&result.diagnostics, graph.ID, graphIndex, nodeIndex, sourceNode, configByNode[sourceNode.ID], incoming[sourceNode.ID])
			}
		}
	}
	result.locks = make([]NodeLock, 0, len(locksByKind))
	for _, lock := range locksByKind {
		result.locks = append(result.locks, lock)
	}
	sort.Slice(result.locks, func(i, j int) bool { return result.locks[i].Kind < result.locks[j].Kind })
	result.capabilities = sortedSet(capabilities)
	sortDiagnostics(result.diagnostics)
	return result, nil
}

func validateGraphInterface(out *[]schema.Diagnostic, graph schema.Graph, graphIndex int) graphInterface {
	iface := graphInterface{inputs: map[string]schema.GraphPort{}, outputs: map[string]schema.GraphPort{}}
	if graph.Kind == schema.GraphKindMain {
		if len(graph.Inputs) != 0 || len(graph.Outputs) != 0 {
			appendDiagnostic(out, schema.Diagnostic{Code: schema.CodeUnsupportedGraphContract, Severity: schema.SeverityError, GraphPath: []string{graph.ID}, FieldPath: []string{"graphs", strconv.Itoa(graphIndex)}, Params: map[string]any{"kind": graph.Kind}})
		}
		return iface
	}
	nodes := map[string]bool{}
	for _, sourceNode := range graph.Nodes {
		nodes[sourceNode.ID] = true
	}
	ids, boundaryNodes := map[string]bool{}, map[string]bool{}
	execInputs, execOutputs := 0, 0
	for _, group := range []struct {
		name  string
		ports []schema.GraphPort
		input bool
	}{{"inputs", graph.Inputs, true}, {"outputs", graph.Outputs, false}} {
		for index, port := range group.ports {
			path := []string{"graphs", strconv.Itoa(graphIndex), group.name, strconv.Itoa(index)}
			if ids[port.ID] || boundaryNodes[port.NodeID] || nodes[port.NodeID] || strings.Contains(port.ID, ".") || strings.Contains(port.NodeID, ".") || port.ID == callInputPin || port.ID == callFailurePin || port.ID == "graphId" {
				appendDiagnostic(out, bindingDiagnostic(graph.ID, "", path, "graphInterfaceIdentity", map[string]any{"id": port.ID, "nodeId": port.NodeID}))
			}
			ids[port.ID], boundaryNodes[port.NodeID] = true, true
			endpoint := port.NodeID + "." + port.ID
			if group.input {
				iface.inputs[endpoint] = port
				if port.Type == node.TypeExec {
					execInputs++
					iface.entry = port
				}
			} else {
				iface.outputs[endpoint] = port
				if port.Type == node.TypeExec {
					execOutputs++
				}
			}
		}
	}
	if execInputs != 1 {
		appendDiagnostic(out, schema.Diagnostic{Code: schema.CodeInvalidGraphEntry, Severity: schema.SeverityError, GraphPath: []string{graph.ID}, FieldPath: []string{"graphs", strconv.Itoa(graphIndex), "inputs"}, Params: map[string]any{"execInputs": execInputs}})
	}
	if execOutputs == 0 {
		appendDiagnostic(out, schema.Diagnostic{Code: schema.CodeMissingGraphOutput, Severity: schema.SeverityError, GraphPath: []string{graph.ID}, FieldPath: []string{"graphs", strconv.Itoa(graphIndex), "outputs"}})
	}
	return iface
}

func callContract(plan *programCall) catalog.NodeContract {
	contract := catalog.NodeContract{Kind: CallSubgraphKind, Inputs: []catalog.InputContract{{Name: callInputPin, Type: node.TypeExec}}, Outputs: []catalog.OutputContract{}}
	for _, port := range plan.Inputs {
		contract.Inputs = append(contract.Inputs, catalog.InputContract{Name: port.ID, Type: port.Type, Required: true})
	}
	for _, port := range plan.Outputs {
		contract.Outputs = append(contract.Outputs, catalog.OutputContract{Name: port.ID, Type: port.Type})
	}
	contract.Outputs = append(contract.Outputs, catalog.OutputContract{Name: callFailurePin, Type: node.TypeExec, Semantic: "error"})
	return contract
}

type endpointResolution struct {
	nodeID, pin, typ                string
	valid, pinValid, wrongDirection bool
	boundary                        bool
}

type nodePinIndex struct {
	inputs, outputs map[string]string
}

type nodeConfigContract struct {
	inputs   map[string]catalog.InputContract
	required []catalog.InputContract
}

func indexNodeConfigContracts(contracts map[string]catalog.NodeContract, byKind map[string]nodeConfigContract) map[string]nodeConfigContract {
	out := make(map[string]nodeConfigContract, len(contracts))
	for nodeID, contract := range contracts {
		if contract.Kind != CallSubgraphKind {
			if cached, ok := byKind[contract.Kind]; ok {
				out[nodeID] = cached
				continue
			}
		}
		indexed := nodeConfigContract{inputs: make(map[string]catalog.InputContract, len(contract.Inputs))}
		for _, input := range contract.Inputs {
			indexed.inputs[input.Name] = input
			if input.Required {
				indexed.required = append(indexed.required, input)
			}
		}
		out[nodeID] = indexed
		if contract.Kind != CallSubgraphKind {
			byKind[contract.Kind] = indexed
		}
	}
	return out
}

func indexNodePins(contracts map[string]catalog.NodeContract, byKind map[string]nodePinIndex) map[string]nodePinIndex {
	out := make(map[string]nodePinIndex, len(contracts))
	for nodeID, contract := range contracts {
		if contract.Kind != CallSubgraphKind {
			if pins, ok := byKind[contract.Kind]; ok {
				out[nodeID] = pins
				continue
			}
		}
		pins := nodePinIndex{inputs: make(map[string]string, len(contract.Inputs)), outputs: make(map[string]string, len(contract.Outputs))}
		for _, input := range contract.Inputs {
			pins.inputs[input.Name] = input.Type
		}
		for _, output := range contract.Outputs {
			pins.outputs[output.Name] = output.Type
		}
		out[nodeID] = pins
		if contract.Kind != CallSubgraphKind {
			byKind[contract.Kind] = pins
		}
	}
	return out
}

func resolveEndpoint(value string, from bool, nodeIDs map[string]bool, pinsByNode map[string]nodePinIndex, iface graphInterface) endpointResolution {
	if port, ok := iface.inputs[value]; ok {
		return endpointResolution{pin: port.ID, typ: port.Type, valid: from, pinValid: from, wrongDirection: !from, boundary: true}
	}
	if port, ok := iface.outputs[value]; ok {
		return endpointResolution{pin: port.ID, typ: port.Type, valid: !from, pinValid: !from, wrongDirection: from, boundary: true}
	}
	nodeID, pin, ok := splitEndpoint(value, nodeIDs)
	if !ok {
		return endpointResolution{}
	}
	pins, bound := pinsByNode[nodeID]
	if !bound {
		return endpointResolution{nodeID: nodeID, pin: pin, valid: true, pinValid: true}
	}
	if from {
		typ, allowed := pins.outputs[pin]
		return endpointResolution{nodeID: nodeID, pin: pin, typ: typ, valid: true, pinValid: allowed}
	}
	typ, allowed := pins.inputs[pin]
	return endpointResolution{nodeID: nodeID, pin: pin, typ: typ, valid: true, pinValid: allowed}
}

func callPlanEncodedSize(target string, graph schema.Graph) int {
	size := 128 + len(target)
	for _, ports := range [][]schema.GraphPort{graph.Inputs, graph.Outputs} {
		for _, port := range ports {
			size += 64 + len(port.ID) + len(port.Type) + len(port.NodeID)
		}
	}
	return size
}

func validateBoundaryCoverage(out *[]schema.Diagnostic, graph schema.Graph, graphIndex int, usedInputs, usedOutputs map[string]bool) {
	for _, group := range []struct {
		name  string
		ports []schema.GraphPort
		used  map[string]bool
	}{{"inputs", graph.Inputs, usedInputs}, {"outputs", graph.Outputs, usedOutputs}} {
		for portIndex, port := range group.ports {
			endpoint := port.NodeID + "." + port.ID
			if !group.used[endpoint] {
				path := []string{"graphs", strconv.Itoa(graphIndex), group.name, strconv.Itoa(portIndex)}
				appendDiagnostic(out, schema.Diagnostic{Code: schema.CodeInvalidGraphBoundaryEdge, Severity: schema.SeverityError, GraphPath: []string{graph.ID}, FieldPath: path, Params: map[string]any{"endpoint": endpoint, "reason": "unbound"}})
			}
		}
	}
}

func appendCallCycleDiagnostics(out *[]schema.Diagnostic, graphs []schema.Graph, adjacency map[string][]callReference) {
	state := map[string]uint8{}
	var visit func(string)
	visit = func(graphID string) {
		state[graphID] = 1
		for _, ref := range adjacency[graphID] {
			if state[ref.target] == 1 {
				appendDiagnostic(out, schema.Diagnostic{Code: schema.CodeSubgraphCallCycle, Severity: schema.SeverityError, GraphPath: []string{ref.graphID}, NodeID: ref.nodeID, FieldPath: []string{"graphs", strconv.Itoa(ref.graphIndex), "nodes", strconv.Itoa(ref.nodeIndex), "config", "graphId"}, Params: map[string]any{"graphId": ref.target}})
			} else if state[ref.target] == 0 {
				visit(ref.target)
			}
		}
		state[graphID] = 2
	}
	for _, graph := range graphs {
		if state[graph.ID] == 0 {
			visit(graph.ID)
		}
	}
}

func splitEndpoint(endpoint string, nodeIDs map[string]bool) (string, string, bool) {
	for dot := strings.LastIndexByte(endpoint, '.'); dot > 0; dot = strings.LastIndexByte(endpoint[:dot], '.') {
		nodeID, pin := endpoint[:dot], endpoint[dot+1:]
		if nodeIDs[nodeID] && pin != "" {
			return nodeID, pin, true
		}
	}
	return "", "", false
}

func compatibleTypes(from, to string) bool {
	left, right := node.CanonicalPinType(from), node.CanonicalPinType(to)
	return left == right || left == "any" || right == "any"
}

func validateNodeConfig(out *[]schema.Diagnostic, graphID string, graphIndex, nodeIndex int, sourceNode schema.Node, contract nodeConfigContract, incoming map[string]bool) {
	if len(*out) >= MaxDiagnostics {
		return
	}
	base := []string{"graphs", strconv.Itoa(graphIndex), "nodes", strconv.Itoa(nodeIndex), "config"}
	keys := make([]string, 0, len(sourceNode.Config))
	for key := range sourceNode.Config {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	for _, key := range keys {
		if len(*out) >= MaxDiagnostics {
			return
		}
		value := sourceNode.Config[key]
		input, ok := contract.inputs[key]
		path := appendPath(base, key)
		if !ok || input.Type == "Exec" {
			appendDiagnostic(out, bindingDiagnostic(graphID, sourceNode.ID, path, "configField", map[string]any{"field": key}))
			continue
		}
		if !valueMatchesType(value, input.Type) || input.Schema != nil && !valueMatchesSchema(value, input.Schema) {
			appendDiagnostic(out, bindingDiagnostic(graphID, sourceNode.ID, path, "configType", map[string]any{"field": key, "type": input.Type}))
		}
	}
	for _, input := range contract.required {
		if len(*out) >= MaxDiagnostics {
			return
		}
		_, configured := sourceNode.Config[input.Name]
		if !configured && input.Default == nil && !incoming[input.Name] {
			appendDiagnostic(out, bindingDiagnostic(graphID, sourceNode.ID, appendPath(base, input.Name), "requiredInput", map[string]any{"field": input.Name}))
		}
	}
}

func valueMatchesType(value any, expected string) bool {
	if value == nil {
		return expected == "Any" || expected == "JSON" || expected == "*"
	}
	switch expected {
	case "*", "Any", "JSON":
		return true
	case "String":
		_, ok := value.(string)
		return ok
	case "Number":
		switch value.(type) {
		case float64, json.Number:
			return true
		default:
			return false
		}
	case "Integer", "Int", "Duration":
		return isIntegerValue(value)
	case "Bool", "Boolean":
		_, ok := value.(bool)
		return ok
	case "List", "Array":
		_, ok := value.([]any)
		return ok
	case "Object":
		_, ok := value.(map[string]any)
		return ok
	case "Point":
		return matchesPoint(value)
	case "Rect":
		return matchesRect(value, false)
	case "Geometry":
		return matchesGeometry(value)
	case "Color":
		return matchesColor(value)
	case "File":
		return matchesFile(value)
	case "Window", "Image":
		return false
	default:
		return false
	}
}

func isIntegerValue(value any) bool {
	switch typed := value.(type) {
	case float64:
		return math.Trunc(typed) == typed
	case json.Number:
		rational, ok := new(big.Rat).SetString(typed.String())
		return ok && rational.IsInt()
	default:
		return false
	}
}

func isNumberValue(value any) bool {
	switch value.(type) {
	case float64, json.Number:
		return true
	default:
		return false
	}
}

func matchesPoint(value any) bool {
	object, ok := value.(map[string]any)
	if !ok || !hasOnlyKeys(object, "x", "y", "unit") || !isNumberValue(object["x"]) || !isNumberValue(object["y"]) {
		return false
	}
	if unit, exists := object["unit"]; exists {
		text, ok := unit.(string)
		return ok && (text == "" || text == "px")
	}
	return true
}

func matchesRect(value any, integer bool) bool {
	object, ok := value.(map[string]any)
	if !ok || !hasOnlyKeys(object, "x", "y", "w", "h") {
		return false
	}
	for _, key := range []string{"x", "y", "w", "h"} {
		if integer {
			if !isIntegerValue(object[key]) {
				return false
			}
		} else if !isNumberValue(object[key]) {
			return false
		}
	}
	return true
}

func matchesGeometry(value any) bool {
	object, ok := value.(map[string]any)
	if !ok || !hasOnlyKeys(object, "pct", "overrides") || !matchesRect(object["pct"], false) {
		return false
	}
	rawOverrides, exists := object["overrides"]
	if !exists {
		return true
	}
	overrides, ok := rawOverrides.([]any)
	if !ok {
		return false
	}
	for _, raw := range overrides {
		override, ok := raw.(map[string]any)
		if !ok || !hasOnlyKeys(override, "resolution", "px") || !matchesResolution(override["resolution"]) || !matchesRect(override["px"], true) {
			return false
		}
	}
	return true
}

func matchesResolution(value any) bool {
	object, ok := value.(map[string]any)
	return ok && hasOnlyKeys(object, "w", "h") && isIntegerValue(object["w"]) && isIntegerValue(object["h"])
}

func matchesColor(value any) bool {
	object, ok := value.(map[string]any)
	return ok && hasOnlyKeys(object, "h", "s", "v") && isIntegerValue(object["h"]) && isIntegerValue(object["s"]) && isIntegerValue(object["v"])
}

func matchesFile(value any) bool {
	object, ok := value.(map[string]any)
	if !ok || !hasOnlyKeys(object, "path", "name", "ext", "mime", "size", "modTimeMs", "isDir") {
		return false
	}
	if _, ok := object["path"].(string); !ok {
		return false
	}
	if name, exists := object["name"]; exists {
		if _, ok := name.(string); !ok {
			return false
		}
	}
	for _, key := range []string{"ext", "mime"} {
		if item, exists := object[key]; exists {
			if _, ok := item.(string); !ok {
				return false
			}
		}
	}
	for _, key := range []string{"size", "modTimeMs"} {
		if item, exists := object[key]; exists && !isIntegerValue(item) {
			return false
		}
	}
	if item, exists := object["isDir"]; exists {
		if _, ok := item.(bool); !ok {
			return false
		}
	}
	return true
}

func hasOnlyKeys(object map[string]any, allowed ...string) bool {
	set := map[string]bool{}
	for _, key := range allowed {
		set[key] = true
	}
	for key := range object {
		if !set[key] {
			return false
		}
	}
	return true
}

func valueMatchesSchema(value any, field *catalog.FieldContract) bool {
	if field == nil {
		return true
	}
	switch field.Type {
	case "number":
		switch value.(type) {
		case float64, json.Number:
			return true
		default:
			return false
		}
	case "string":
		_, ok := value.(string)
		return ok
	case "enum":
		left, leftErr := artifact.Marshal(map[string]any{"value": value})
		if leftErr != nil {
			return false
		}
		for _, option := range field.Options {
			right, rightErr := artifact.Marshal(map[string]any{"value": option})
			if leftErr == nil && rightErr == nil && string(left) == string(right) {
				return true
			}
		}
		return false
	case "bool":
		_, ok := value.(bool)
		return ok
	case "array":
		items, ok := value.([]any)
		if !ok {
			return false
		}
		for _, item := range items {
			if !valueMatchesSchema(item, field.Items) {
				return false
			}
		}
		return true
	case "tuple":
		items, ok := value.([]any)
		if !ok || len(items) != len(field.Fields) {
			return false
		}
		for index, entry := range field.Fields {
			if !valueMatchesSchema(items[index], entry.Schema) {
				return false
			}
		}
		return true
	case "object":
		if field.Shape == "point" {
			return matchesPoint(value)
		}
		if field.Shape == "geometry" {
			return matchesGeometry(value)
		}
		object, ok := value.(map[string]any)
		if !ok {
			return false
		}
		known := map[string]bool{}
		for _, entry := range field.Fields {
			known[entry.Key] = true
			child, exists := object[entry.Key]
			if entry.Required && !exists {
				return false
			}
			if exists && !valueMatchesSchema(child, entry.Schema) {
				return false
			}
		}
		for key := range object {
			if !known[key] {
				return false
			}
		}
		return true
	default:
		return false
	}
}

func appendPath(path []string, value string) []string {
	return append(append([]string(nil), path...), value)
}

func capabilitiesFromSource(values []schema.Capability) []string {
	set := map[string]bool{}
	for _, value := range values {
		set[string(value)] = true
	}
	return sortedSet(set)
}

func validateCapabilityDeclaration(declared, required []string) []schema.Diagnostic {
	declaredSet, requiredSet := map[string]bool{}, map[string]bool{}
	for _, value := range declared {
		declaredSet[value] = true
	}
	for _, value := range required {
		requiredSet[value] = true
	}
	var diagnostics []schema.Diagnostic
	for _, value := range required {
		if !declaredSet[value] {
			appendDiagnostic(&diagnostics, schema.Diagnostic{Code: schema.CodeMissingCapabilityDeclaration, Severity: schema.SeverityError, FieldPath: []string{"requestedCapabilities"}, Params: map[string]any{"capability": value}})
		}
	}
	for _, value := range declared {
		if !requiredSet[value] {
			appendDiagnostic(&diagnostics, schema.Diagnostic{Code: schema.CodeUnusedCapabilityDeclaration, Severity: schema.SeverityError, FieldPath: []string{"requestedCapabilities"}, Params: map[string]any{"capability": value}})
		}
	}
	sortDiagnostics(diagnostics)
	return diagnostics
}

func exceedsSourceCollections(source schema.WorkflowSource) bool {
	if len(source.Graphs) > MaxGraphs {
		return true
	}
	totalNodes, totalEdges, totalPorts := 0, 0, 0
	for _, graph := range source.Graphs {
		if len(graph.Nodes) > MaxNodesPerGraph || len(graph.Edges) > MaxEdgesPerGraph || len(graph.Inputs)+len(graph.Outputs) > MaxPortsPerGraph {
			return true
		}
		totalNodes += len(graph.Nodes)
		totalEdges += len(graph.Edges)
		totalPorts += len(graph.Inputs) + len(graph.Outputs)
	}
	return totalNodes > MaxTotalNodes || totalEdges > MaxTotalEdges || totalPorts > MaxTotalPorts
}

func exceedsJSONDepth(raw []byte, maximum int) bool {
	depth := 0
	inString, escaped := false, false
	for _, value := range raw {
		if inString {
			if escaped {
				escaped = false
				continue
			}
			if value == '\\' {
				escaped = true
				continue
			}
			if value == '"' {
				inString = false
			}
			continue
		}
		switch value {
		case '"':
			inString = true
		case '{', '[':
			depth++
			if depth > maximum {
				return true
			}
		case '}', ']':
			if depth > 0 {
				depth--
			}
		}
	}
	return false
}

func bindingDiagnostic(graphID, nodeID string, fieldPath []string, keyword string, params map[string]any) schema.Diagnostic {
	params["keyword"] = keyword
	return schema.Diagnostic{
		Code: schema.CodeInvalidField, Severity: schema.SeverityError,
		GraphPath: []string{graphID}, NodeID: nodeID, FieldPath: fieldPath, Params: params,
	}
}

func appendDiagnostic(out *[]schema.Diagnostic, diagnostic schema.Diagnostic) {
	if len(*out) < MaxDiagnostics-1 {
		*out = append(*out, diagnostic)
		return
	}
	if len(*out) == MaxDiagnostics-1 {
		*out = append(*out, diagnosticBudgetExceeded())
	}
}

func diagnosticBudgetExceeded() schema.Diagnostic {
	return schema.Diagnostic{Code: schema.CodeDiagnosticBudgetExceeded, Severity: schema.SeverityError, Params: map[string]any{"maximum": MaxDiagnostics}}
}

func lowerGraphs(graphs []schema.Graph, binding bindingResult) []programGraph {
	out := make([]programGraph, 0, len(graphs))
	for _, graph := range graphs {
		if !binding.reachable[graph.ID] {
			continue
		}
		nodes := make([]programNode, len(graph.Nodes))
		for nodeIndex, node := range graph.Nodes {
			nodes[nodeIndex] = programNode{ID: node.ID, Kind: node.Kind, Config: node.Config, Disabled: node.Disabled, Call: binding.calls[graph.ID][node.ID]}
		}
		edges := make([]schema.Edge, len(graph.Edges))
		copy(edges, graph.Edges)
		inputs := make([]schema.GraphPort, len(graph.Inputs))
		copy(inputs, graph.Inputs)
		outputs := make([]schema.GraphPort, len(graph.Outputs))
		copy(outputs, graph.Outputs)
		out = append(out, programGraph{
			ID: graph.ID, Kind: graph.Kind, Nodes: nodes,
			Edges: edges, Inputs: inputs, Outputs: outputs,
		})
	}
	return out
}

func cloneDiagnostics(source []schema.Diagnostic) []schema.Diagnostic {
	out := make([]schema.Diagnostic, len(source))
	copy(out, source)
	return out
}

func cappedDiagnostics(source []schema.Diagnostic) []schema.Diagnostic {
	if len(source) < MaxDiagnostics {
		return cloneDiagnostics(source)
	}
	out := cloneDiagnostics(source[:MaxDiagnostics-1])
	out = append(out, diagnosticBudgetExceeded())
	return out
}

func sortDiagnostics(diagnostics []schema.Diagnostic) {
	sort.SliceStable(diagnostics, func(i, j int) bool {
		left, right := diagnostics[i], diagnostics[j]
		for _, pair := range [][2]string{
			{strings.Join(left.GraphPath, "/"), strings.Join(right.GraphPath, "/")},
			{left.NodeID, right.NodeID}, {strings.Join(left.FieldPath, "/"), strings.Join(right.FieldPath, "/")}, {left.Code, right.Code},
		} {
			if pair[0] != pair[1] {
				return pair[0] < pair[1]
			}
		}
		return false
	})
}
