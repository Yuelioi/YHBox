package compiler

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sort"

	runtimejsonschema "github.com/santhosh-tekuri/jsonschema/v6"
	"github.com/yottaapp/yotta/internal/artifact"
	"github.com/yottaapp/yotta/internal/blob"
	"github.com/yottaapp/yotta/internal/capability"
	"github.com/yottaapp/yotta/internal/configvalidator"
	"github.com/yottaapp/yotta/internal/datatype"
	"github.com/yottaapp/yotta/internal/nodecatalog"
	"github.com/yottaapp/yotta/internal/nodecontract"
	"github.com/yottaapp/yotta/internal/workflow/schema"
)

const (
	MaxSourceBytes = 16 << 20

	CodeInvalidCatalog           = "INVALID_CATALOG"
	CodeUnknownNodeType          = "UNKNOWN_NODE_TYPE"
	CodeNodeContractMismatch     = "NODE_CONTRACT_MISMATCH"
	CodeUnknownPort              = "UNKNOWN_PORT"
	CodeEdgeChannelMismatch      = "EDGE_CHANNEL_MISMATCH"
	CodeTypeMismatch             = "TYPE_MISMATCH"
	CodeUnresolvedType           = "UNRESOLVED_TYPE"
	CodeResourceLeaseMismatch    = "RESOURCE_LEASE_MISMATCH"
	CodeMissingInputBinding      = "MISSING_INPUT_BINDING"
	CodeDuplicateInputBinding    = "DUPLICATE_INPUT_BINDING"
	CodeDuplicateSignalRoute     = "DUPLICATE_SIGNAL_ROUTE"
	CodeRegionSignalScope        = "REGION_SIGNAL_SCOPE"
	CodeInvalidBinding           = "INVALID_BINDING"
	CodeBlobUnavailable          = "BLOB_UNAVAILABLE"
	CodeInvalidConfig            = "INVALID_CONFIG"
	CodeInvalidStateVariable     = "INVALID_STATE_VARIABLE"
	CodeInvalidStateAccess       = "INVALID_STATE_ACCESS"
	CodeInvalidCapabilityBinding = "INVALID_CAPABILITY_BINDING"
	CodeNoExecutionRoot          = "NO_EXECUTION_ROOT"
	CodeUnreachableExecution     = "UNREACHABLE_EXECUTION"
	CodeDataCycle                = "DATA_CYCLE"
	CodeUnsupportedGraph         = "UNSUPPORTED_GRAPH"
	CodeUnsupportedSourceFeature = "UNSUPPORTED_SOURCE_FEATURE"
)

type Diagnostic = schema.Diagnostic

type BlobVerifier interface {
	Verify(context.Context, blob.BlobRef) error
}

type BlobVerifierFunc func(context.Context, blob.BlobRef) error

func (verify BlobVerifierFunc) Verify(ctx context.Context, ref blob.BlobRef) error {
	return verify(ctx, ref)
}

type CompileRequest struct {
	SourceJSON   []byte
	Catalog      nodecatalog.Snapshot
	BlobVerifier BlobVerifier
}

type CompileResult struct {
	SourceHash  artifact.Digest `json:"sourceHash,omitempty"`
	Diagnostics []Diagnostic    `json:"diagnostics"`
	program     ProgramSnapshot
}

func (r CompileResult) Program() (ProgramSnapshot, bool) { return r.program, r.program.Valid() }

type Compiler struct {
	build      artifact.Digest
	validators configvalidator.Registry
}

func New(build artifact.Digest, validators configvalidator.Registry) *Compiler {
	return &Compiler{build: build, validators: validators}
}

func (c *Compiler) CompileDraft(ctx context.Context, request CompileRequest) (CompileResult, error) {
	if err := ctx.Err(); err != nil {
		return CompileResult{}, err
	}
	if !c.build.Valid() || !c.validators.Valid() {
		return CompileResult{}, errors.New("compiler requires a content-addressed build and trusted config validators")
	}
	if len(request.SourceJSON) == 0 || len(request.SourceJSON) > MaxSourceBytes {
		return CompileResult{}, errors.New("workflow source exceeds byte budget")
	}
	source, _, sourceHash, diagnostics, err := schema.CanonicalSource(request.SourceJSON)
	result := CompileResult{Diagnostics: diagnostics}
	if schema.HasErrors(diagnostics) {
		return result, nil
	}
	if err != nil {
		return result, err
	}
	result.SourceHash = sourceHash
	if !request.Catalog.Valid() {
		result.Diagnostics = []Diagnostic{diagnostic(CodeInvalidCatalog, nil, "")}
		return result, nil
	}

	body := programBody{
		SourceHash: result.SourceHash, CatalogHash: request.Catalog.Hash(), CompilerBuild: c.build,
		WorkflowID: source.Workflow.ID, Revision: source.Revision, EntryGraph: source.EntryGraph,
		State: []programStateSlot{}, Graphs: []programGraph{},
	}
	stateSlots, stateDiagnostics := compileStateVariables(source.Variables, request.Catalog)
	body.State = stateSlots
	result.Diagnostics = append(result.Diagnostics, stateDiagnostics...)
	stateByName := make(map[string]programStateSlot, len(stateSlots))
	for _, slot := range stateSlots {
		stateByName[slot.Name] = slot
	}
	if len(source.SecretRefs) != 0 {
		result.Diagnostics = append(result.Diagnostics, diagnostic(CodeUnsupportedSourceFeature, []string{"secretRefs"}, ""))
	}
	planEntries := make([]capability.PlanEntry, 0)
	for graphIndex, graph := range source.Graphs {
		if err := ctx.Err(); err != nil {
			return CompileResult{}, err
		}
		if graph.Kind != schema.GraphKindMain || graph.ID != source.EntryGraph {
			result.Diagnostics = append(result.Diagnostics, diagnostic(CodeUnsupportedGraph, []string{"graphs", fmt.Sprint(graphIndex)}, ""))
			continue
		}
		if len(graph.Inputs) != 0 || len(graph.Outputs) != 0 {
			result.Diagnostics = append(result.Diagnostics, diagnostic(CodeUnsupportedSourceFeature, []string{"graphs", fmt.Sprint(graphIndex), "inputs"}, graph.ID))
			continue
		}
		compiled, graphDiagnostics, compileErr := compileGraph(ctx, graph, graphIndex, request.Catalog, c.validators, stateByName, request.BlobVerifier)
		if compileErr != nil {
			return CompileResult{}, compileErr
		}
		result.Diagnostics = append(result.Diagnostics, graphDiagnostics...)
		for _, node := range compiled.Nodes {
			for _, requirement := range node.Capabilities {
				planEntries = append(planEntries, capability.PlanEntry{GraphID: graph.ID, NodeID: node.ID, Requirement: requirement})
			}
		}
		body.Graphs = append(body.Graphs, compiled)
	}
	plan, err := capability.SealPlan(planEntries)
	if err != nil {
		return result, fmt.Errorf("seal capability plan: %w", err)
	}
	body.CapabilityPlan = plan.Bytes()
	if schema.HasErrors(result.Diagnostics) {
		if len(result.Diagnostics) > schema.MaxDiagnostics {
			result.Diagnostics = result.Diagnostics[:schema.MaxDiagnostics]
		}
		sortDiagnostics(result.Diagnostics)
		return result, nil
	}
	result.program, err = sealProgram(body)
	if err != nil {
		return result, err
	}
	if len(result.Diagnostics) > schema.MaxDiagnostics {
		result.Diagnostics = result.Diagnostics[:schema.MaxDiagnostics]
	}
	sortDiagnostics(result.Diagnostics)
	return result, nil
}

func compileStateVariables(source []schema.Variable, catalog nodecatalog.Snapshot) ([]programStateSlot, []Diagnostic) {
	result := make([]programStateSlot, 0, len(source))
	diagnostics := make([]Diagnostic, 0)
	for index, variable := range source {
		path := []string{"variables", fmt.Sprint(index)}
		resolved, err := resolveDeclaredStateType(variable.Type, catalog, 0)
		if err != nil {
			diagnostic := diagnostic(CodeInvalidStateVariable, append(path, "type"), "")
			diagnostic.Params["reason"] = err.Error()
			diagnostics = append(diagnostics, diagnostic)
			continue
		}
		canonical, err := artifact.Canonicalize(variable.Default)
		if err != nil {
			diagnostic := diagnostic(CodeInvalidStateVariable, append(path, "default"), "")
			diagnostic.Params["reason"] = err.Error()
			diagnostics = append(diagnostics, diagnostic)
			continue
		}
		envelope, err := datatype.SealInlineJSON(catalog, resolved, canonical)
		if err != nil {
			diagnostic := diagnostic(CodeInvalidStateVariable, append(path, "default"), "")
			diagnostic.Params["reason"] = err.Error()
			diagnostics = append(diagnostics, diagnostic)
			continue
		}
		result = append(result, programStateSlot{Name: variable.Name, Type: resolved, Initial: envelope.Artifact()})
	}
	return result, diagnostics
}

func resolveDeclaredStateType(expression datatype.TypeExpression, catalog nodecatalog.Snapshot, depth int) (datatype.ResolvedType, error) {
	if depth > datatype.MaxTypeDepth {
		return datatype.ResolvedType{}, errors.New("state type exceeds depth budget")
	}
	switch expression.Kind {
	case datatype.TypeExpressionRef:
		if expression.Ref == nil {
			return datatype.ResolvedType{}, errors.New("state type reference is missing")
		}
		definition, ok := catalog.LookupType(expression.Ref.TypeID)
		if !ok || definition.TypeRef() != *expression.Ref {
			return datatype.ResolvedType{}, errors.New("state type is not pinned by the trusted Catalog")
		}
		return datatype.RefResolvedType(*expression.Ref), nil
	case datatype.TypeExpressionList:
		if expression.Element == nil {
			return datatype.ResolvedType{}, errors.New("state list element type is missing")
		}
		element, err := resolveDeclaredStateType(*expression.Element, catalog, depth+1)
		if err != nil {
			return datatype.ResolvedType{}, err
		}
		return datatype.ListResolvedType(element), nil
	case datatype.TypeExpressionVariable, datatype.TypeExpressionUnion:
		return datatype.ResolvedType{}, errors.New("state variables require one concrete resolved type")
	default:
		return datatype.ResolvedType{}, errors.New("state variable uses an unknown type expression")
	}
}

func compileGraph(ctx context.Context, graph schema.Graph, graphIndex int, catalog nodecatalog.Snapshot, configValidators configvalidator.Registry, state map[string]programStateSlot, blobVerifier BlobVerifier) (programGraph, []Diagnostic, error) {
	compiled := programGraph{ID: graph.ID, Nodes: []programNode{}, SignalRoutes: []programSignalRoute{}, DataOrder: []string{}}
	var diagnostics []Diagnostic
	nodes := make(map[string]int, len(graph.Nodes))
	contracts := make(map[string]nodecontract.MachineContract, len(graph.Nodes))
	pendingBindings := make(map[string]map[string]schema.InputBinding, len(graph.Nodes))
	types := newTypeSolver()
	validators := map[string]*runtimejsonschema.Schema{}
	for nodeIndex, sourceNode := range graph.Nodes {
		if err := ctx.Err(); err != nil {
			return compiled, diagnostics, err
		}
		path := []string{"graphs", fmt.Sprint(graphIndex), "nodes", fmt.Sprint(nodeIndex)}
		if sourceNode.Disabled {
			diagnostics = append(diagnostics, diagnosticAtNode(CodeUnsupportedSourceFeature, append(path, "disabled"), graph.ID, sourceNode.ID))
			continue
		}
		entry, ok := catalog.Lookup(sourceNode.NodeRef.NodeTypeID)
		if !ok {
			diagnostics = append(diagnostics, diagnosticAtNode(CodeUnknownNodeType, path, graph.ID, sourceNode.ID))
			continue
		}
		if entry.Contract.NodeRef() != sourceNode.NodeRef {
			diagnostics = append(diagnostics, diagnosticAtNode(CodeNodeContractMismatch, append(path, "nodeRef"), graph.ID, sourceNode.ID))
			continue
		}
		machine := entry.Contract.Machine()
		if !programExecutableClass(machine.Execution.Class) {
			diagnostics = append(diagnostics, diagnosticAtNode(CodeUnsupportedSourceFeature, append(path, "nodeRef"), graph.ID, sourceNode.ID))
			continue
		}
		if err := validateJSONSchemaBundleCached(validators, "config:"+sourceNode.NodeRef.SemanticDigest.String(), machine.ConfigSchemaRoot, machine.ConfigSchemaBundle, sourceNode.Config); err != nil {
			diagnostic := diagnosticAtNode(CodeInvalidConfig, append(path, "config"), graph.ID, sourceNode.ID)
			diagnostic.Params["reason"] = err.Error()
			diagnostics = append(diagnostics, diagnostic)
		} else if err := configValidators.Validate(machine, sourceNode.Config); err != nil {
			diagnostic := diagnosticAtNode(CodeInvalidConfig, append(path, "config"), graph.ID, sourceNode.ID)
			diagnostic.Params["reason"] = err.Error()
			diagnostics = append(diagnostics, diagnostic)
		}
		effectiveRequirements, err := nodecontract.ResolveCapabilityRequirements(machine, sourceNode.Config)
		if err != nil {
			diagnostic := diagnosticAtNode(CodeInvalidCapabilityBinding, append(path, "config"), graph.ID, sourceNode.ID)
			diagnostic.Params["reason"] = err.Error()
			diagnostics = append(diagnostics, diagnostic)
			effectiveRequirements = append([]capability.Requirement(nil), machine.CapabilityRequirements...)
		}
		for _, access := range machine.StateAccesses {
			slotName, ok := sourceNode.Config[access.SlotConfigKey].(string)
			slot, exists := state[slotName]
			if !ok || !exists {
				diagnostic := diagnosticAtNode(CodeInvalidStateAccess, append(path, "config", access.SlotConfigKey), graph.ID, sourceNode.ID)
				diagnostic.Params["accessId"] = access.ID
				diagnostics = append(diagnostics, diagnostic)
				continue
			}
			concrete, err := datatype.ResolvedExpression(slot.Type)
			if err != nil || types.unify(
				scopedTypeExpression{scope: sourceNode.ID, expression: access.Type},
				scopedTypeExpression{scope: sourceNode.ID, expression: concrete},
			) != nil {
				diagnostic := diagnosticAtNode(CodeInvalidStateAccess, append(path, "config", access.SlotConfigKey), graph.ID, sourceNode.ID)
				diagnostic.Params["accessId"] = access.ID
				diagnostic.Params["reason"] = "state slot type does not satisfy the node access contract"
				diagnostics = append(diagnostics, diagnostic)
			}
		}
		plan := programNode{
			ID: sourceNode.ID, NodeRef: sourceNode.NodeRef, Config: sourceNode.Config,
			Inputs: map[string]inputPlan{}, InputTypes: map[string]datatype.ResolvedType{}, OutputTypes: map[string]datatype.ResolvedType{},
			Ports: machine.Ports, Execution: machine.Execution, Instruction: machine.Instruction,
			HostFeatures:   append([]nodecontract.HostFeatureRequirement{}, machine.HostFeatureRequirements...),
			Capabilities:   effectiveRequirements,
			Implementation: entry.Implementation,
		}
		inputs := make(map[string]nodecontract.DataInputPort, len(machine.Ports.DataInputs))
		for _, port := range machine.Ports.DataInputs {
			inputs[port.ID] = port
		}
		pendingBindings[sourceNode.ID] = make(map[string]schema.InputBinding, len(sourceNode.Bindings))
		for portID, binding := range sourceNode.Bindings {
			port, exists := inputs[portID]
			if !exists {
				diagnostics = append(diagnostics, diagnosticAtNode(CodeUnknownPort, append(path, "bindings", portID), graph.ID, sourceNode.ID))
				continue
			}
			if port.ResourceLease != nil {
				diagnostic := diagnosticAtNode(CodeInvalidBinding, append(path, "bindings", portID), graph.ID, sourceNode.ID)
				diagnostic.Params["reason"] = "runtime authority inputs must come from an explicitly leased data edge"
				diagnostics = append(diagnostics, diagnostic)
				continue
			}
			switch binding.Kind {
			case schema.BindingValue:
			case schema.BindingDefault:
				if port.Default == nil {
					diagnostics = append(diagnostics, diagnosticAtNode(CodeInvalidBinding, append(path, "bindings", portID), graph.ID, sourceNode.ID))
					continue
				}
			case schema.BindingBlob:
				if binding.Blob == nil {
					diagnostics = append(diagnostics, diagnosticAtNode(CodeInvalidBinding, append(path, "bindings", portID), graph.ID, sourceNode.ID))
					continue
				}
				if blobVerifier != nil {
					if err := blobVerifier.Verify(ctx, *binding.Blob); err != nil {
						diagnostic := diagnosticAtNode(CodeBlobUnavailable, append(path, "bindings", portID), graph.ID, sourceNode.ID)
						diagnostic.Params["reason"] = err.Error()
						diagnostics = append(diagnostics, diagnostic)
						continue
					}
				}
			default:
				diagnostics = append(diagnostics, diagnosticAtNode(CodeInvalidBinding, append(path, "bindings", portID), graph.ID, sourceNode.ID))
				continue
			}
			pendingBindings[sourceNode.ID][portID] = binding
		}
		compiled.Nodes = append(compiled.Nodes, plan)
		nodes[plan.ID] = len(compiled.Nodes) - 1
		contracts[plan.ID] = machine
	}

	incoming := map[string]bool{}
	signalRoutes := map[programSignalRoute]bool{}
	adjacency := map[string][]string{}
	indegree := map[string]int{}
	for _, node := range compiled.Nodes {
		indegree[node.ID] = 0
	}
	for edgeIndex, edge := range graph.Edges {
		if err := ctx.Err(); err != nil {
			return compiled, diagnostics, err
		}
		path := []string{"graphs", fmt.Sprint(graphIndex), "edges", fmt.Sprint(edgeIndex)}
		fromIndex, fromOK := nodes[edge.From.NodeID]
		toIndex, toOK := nodes[edge.To.NodeID]
		if !fromOK || !toOK {
			diagnostics = append(diagnostics, diagnostic(CodeUnknownPort, path, graph.ID))
			continue
		}
		fromNode, toNode := &compiled.Nodes[fromIndex], &compiled.Nodes[toIndex]
		fromMachine, toMachine := contracts[fromNode.ID], contracts[toNode.ID]
		fromType, fromExists := outputForChannel(fromMachine.Ports, edge.Channel, edge.From.PortID)
		toType, toExists := inputForChannel(toMachine.Ports, edge.Channel, edge.To.PortID)
		if !fromExists || !toExists {
			code := CodeUnknownPort
			if (!fromExists && outputPortExists(fromMachine.Ports, edge.From.PortID)) ||
				(!toExists && inputPortExists(toMachine.Ports, edge.To.PortID)) {
				code = CodeEdgeChannelMismatch
			}
			diagnostics = append(diagnostics, diagnostic(code, path, graph.ID))
			continue
		}
		if edge.Channel != schema.EdgeData && !toMachine.Instruction.AcceptsSignalInput(string(edge.Channel), edge.To.PortID) {
			diagnostics = append(diagnostics, diagnostic(CodeEdgeChannelMismatch, path, graph.ID))
			continue
		}
		if edge.Channel == schema.EdgeData {
			key := edge.To.NodeID + "\x00" + edge.To.PortID
			_, sourceBound := pendingBindings[toNode.ID][edge.To.PortID]
			if incoming[key] || sourceBound {
				diagnostics = append(diagnostics, diagnostic(CodeDuplicateInputBinding, path, graph.ID))
				continue
			}
			incoming[key] = true
			if err := types.unify(
				scopedTypeExpression{scope: fromNode.ID, expression: *fromType},
				scopedTypeExpression{scope: toNode.ID, expression: *toType},
			); err != nil {
				diagnostics = append(diagnostics, diagnostic(CodeTypeMismatch, path, graph.ID))
				continue
			}
			fromPort, _ := dataOutputPort(fromMachine.Ports, edge.From.PortID)
			toPort, _ := dataInputPort(toMachine.Ports, edge.To.PortID)
			if !resourceLeaseAssignable(fromPort.ResourceLease, toPort.ResourceLease) {
				diagnostics = append(diagnostics, diagnostic(CodeResourceLeaseMismatch, path, graph.ID))
				continue
			}
			if _, already := toNode.Inputs[edge.To.PortID]; already {
				diagnostics = append(diagnostics, diagnostic(CodeDuplicateInputBinding, path, graph.ID))
				continue
			}
			toNode.Inputs[edge.To.PortID] = inputPlan{Kind: inputEdge, From: edge.From}
			adjacency[fromNode.ID] = append(adjacency[fromNode.ID], toNode.ID)
			indegree[toNode.ID]++
		} else {
			route := programSignalRoute{Channel: edge.Channel, From: edge.From, To: edge.To}
			if signalRoutes[route] {
				diagnostics = append(diagnostics, diagnostic(CodeDuplicateSignalRoute, path, graph.ID))
				continue
			}
			signalRoutes[route] = true
			compiled.SignalRoutes = append(compiled.SignalRoutes, route)
		}
	}
	violations, err := regionSignalScopeViolations(ctx, compiled)
	if err != nil {
		return compiled, diagnostics, err
	}
	for _, violation := range violations {
		diagnostic := diagnosticAtNode(CodeRegionSignalScope, []string{"graphs", fmt.Sprint(graphIndex), "edges"}, graph.ID, violation.regionNodeID)
		diagnostic.Params["inputPort"] = violation.inputPort
		diagnostic.Params["sourceNodeId"] = violation.sourceNodeID
		diagnostics = append(diagnostics, diagnostic)
	}
	for nodeIndex := range compiled.Nodes {
		node := &compiled.Nodes[nodeIndex]
		machine := contracts[node.ID]
		for _, port := range machine.Ports.DataInputs {
			resolved, err := types.resolve(scopedTypeExpression{scope: node.ID, expression: port.Type})
			if err != nil {
				diagnostic := diagnosticAtNode(CodeUnresolvedType, []string{"ports", "dataInputs", port.ID}, graph.ID, node.ID)
				diagnostic.Params["reason"] = err.Error()
				diagnostics = append(diagnostics, diagnostic)
				continue
			}
			node.InputTypes[port.ID] = resolved
		}
		for _, port := range machine.Ports.DataOutputs {
			resolved, err := types.resolve(scopedTypeExpression{scope: node.ID, expression: port.Type})
			if err != nil {
				diagnostic := diagnosticAtNode(CodeUnresolvedType, []string{"ports", "dataOutputs", port.ID}, graph.ID, node.ID)
				diagnostic.Params["reason"] = err.Error()
				diagnostics = append(diagnostics, diagnostic)
				continue
			}
			node.OutputTypes[port.ID] = resolved
		}
		for portID, binding := range pendingBindings[node.ID] {
			port, _ := dataInputPort(machine.Ports, portID)
			resolved, ok := node.InputTypes[portID]
			if !ok {
				continue
			}
			plan, err := compileInputBinding(binding, port, resolved, catalog)
			if err != nil {
				diagnostic := diagnosticAtNode(CodeInvalidBinding, []string{"bindings", portID}, graph.ID, node.ID)
				diagnostic.Params["reason"] = err.Error()
				diagnostics = append(diagnostics, diagnostic)
				continue
			}
			node.Inputs[portID] = plan
		}
	}
	for _, node := range compiled.Nodes {
		for _, port := range node.Ports.DataInputs {
			if port.Required {
				if _, present := node.Inputs[port.ID]; !present {
					diagnostics = append(diagnostics, diagnosticAtNode(CodeMissingInputBinding, []string{"bindings", port.ID}, graph.ID, node.ID))
				}
			}
		}
	}
	compiled.DataOrder = topologicalOrder(compiled.Nodes, adjacency, indegree)
	if len(compiled.DataOrder) != len(compiled.Nodes) {
		diagnostics = append(diagnostics, diagnostic(CodeDataCycle, []string{"graphs", fmt.Sprint(graphIndex), "edges"}, graph.ID))
	} else if len(compiled.Nodes) != 0 {
		roots, reachable := executionReachability(compiled)
		if len(roots) == 0 {
			diagnostics = append(diagnostics, diagnostic(CodeNoExecutionRoot, []string{"graphs", fmt.Sprint(graphIndex)}, graph.ID))
		}
		for nodeIndex, node := range compiled.Nodes {
			if node.Execution.Evaluation == nodecontract.EvaluationPush && !reachable[node.ID] {
				diagnostics = append(diagnostics, warningAtNode(CodeUnreachableExecution,
					[]string{"graphs", fmt.Sprint(graphIndex), "nodes", fmt.Sprint(nodeIndex)}, graph.ID, node.ID))
			}
		}
	}
	return compiled, diagnostics, nil
}

type regionSignalScopeViolation struct {
	regionNodeID string
	inputPort    string
	sourceNodeID string
}

func regionSignalScopeViolations(ctx context.Context, graph programGraph) ([]regionSignalScopeViolation, error) {
	outgoing := make(map[string][]programSignalRoute)
	for _, route := range graph.SignalRoutes {
		outgoing[route.From.NodeID] = append(outgoing[route.From.NodeID], route)
	}
	var violations []regionSignalScopeViolation
	for _, region := range graph.Nodes {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		bodyOutput, controlInputs := regionInstructionPorts(region.Instruction)
		if bodyOutput == "" {
			continue
		}
		reachable := map[string]bool{}
		queue := []string{}
		for _, route := range outgoing[region.ID] {
			if route.Channel == schema.EdgeExec && route.From.PortID == bodyOutput && route.To.NodeID != region.ID {
				queue = append(queue, route.To.NodeID)
			}
		}
		for len(queue) != 0 {
			if err := ctx.Err(); err != nil {
				return nil, err
			}
			nodeID := queue[0]
			queue = queue[1:]
			if reachable[nodeID] {
				continue
			}
			reachable[nodeID] = true
			for _, route := range outgoing[nodeID] {
				if route.To.NodeID != region.ID && !reachable[route.To.NodeID] {
					queue = append(queue, route.To.NodeID)
				}
			}
		}
		external := map[string]bool{}
		queue = signalExecutionRoots(graph)
		for len(queue) != 0 {
			if err := ctx.Err(); err != nil {
				return nil, err
			}
			nodeID := queue[0]
			queue = queue[1:]
			if external[nodeID] {
				continue
			}
			external[nodeID] = true
			for _, route := range outgoing[nodeID] {
				if route.From.NodeID == region.ID && route.From.PortID == bodyOutput {
					continue
				}
				if !external[route.To.NodeID] {
					queue = append(queue, route.To.NodeID)
				}
			}
		}
		for _, route := range graph.SignalRoutes {
			if route.To.NodeID != region.ID || !controlInputs[route.To.PortID] {
				continue
			}
			directBodySignal := route.From.NodeID == region.ID && route.From.PortID == bodyOutput
			inside := directBodySignal || reachable[route.From.NodeID] && !external[route.From.NodeID]
			if !inside {
				violations = append(violations, regionSignalScopeViolation{
					regionNodeID: region.ID, inputPort: route.To.PortID, sourceNodeID: route.From.NodeID,
				})
			}
		}
	}
	return violations, nil
}

func signalExecutionRoots(graph programGraph) []string {
	roots := make([]string, 0)
	for _, node := range graph.Nodes {
		if node.Execution.Class == nodecontract.ExecutionEvent && len(node.Ports.ExecInputs) == 0 {
			roots = append(roots, node.ID)
		}
	}
	return roots
}

func regionInstructionPorts(instruction nodecontract.InstructionSpec) (string, map[string]bool) {
	switch instruction.Kind {
	case nodecontract.InstructionCountedLoop:
		spec := instruction.CountedLoop
		if spec == nil {
			return "", nil
		}
		return spec.BodyOutput, map[string]bool{spec.BreakInput: true, spec.ContinueInput: true}
	case nodecontract.InstructionForEach:
		spec := instruction.ForEach
		if spec == nil {
			return "", nil
		}
		return spec.BodyOutput, map[string]bool{spec.BreakInput: true, spec.ContinueInput: true}
	case nodecontract.InstructionRetry:
		spec := instruction.Retry
		if spec == nil {
			return "", nil
		}
		return spec.BodyOutput, map[string]bool{spec.RetryInput: true}
	default:
		return "", nil
	}
}

func compileInputBinding(binding schema.InputBinding, port nodecontract.DataInputPort, resolved datatype.ResolvedType, catalog nodecatalog.Snapshot) (inputPlan, error) {
	switch binding.Kind {
	case schema.BindingBlob:
		if binding.Blob == nil {
			return inputPlan{}, errors.New("blob binding is missing its reference")
		}
		envelope, err := datatype.SealBlobRef(catalog, resolved, *binding.Blob)
		if err != nil {
			return inputPlan{}, err
		}
		return inputPlan{Kind: inputLiteral, Provenance: inputSourceBlob, Value: envelope.Artifact()}, nil
	case schema.BindingValue, schema.BindingDefault:
		value := binding.Value
		provenance := inputSourceLiteral
		if binding.Kind == schema.BindingDefault {
			if port.Default == nil {
				return inputPlan{}, errors.New("input has no declared default")
			}
			value = *port.Default
			provenance = inputSourceDefault
		}
		canonical, err := artifact.Canonicalize(value)
		if err != nil {
			return inputPlan{}, err
		}
		envelope, err := datatype.SealInlineJSON(catalog, resolved, canonical)
		if err != nil {
			return inputPlan{}, err
		}
		return inputPlan{Kind: inputLiteral, Provenance: provenance, Value: envelope.Artifact()}, nil
	default:
		return inputPlan{}, errors.New("unknown binding kind")
	}
}

func outputPortExists(ports nodecontract.PortSet, id string) bool {
	for _, port := range ports.DataOutputs {
		if port.ID == id {
			return true
		}
	}
	for _, group := range [][]nodecontract.SignalPort{ports.ExecOutputs, ports.ErrorOutputs} {
		for _, port := range group {
			if port.ID == id {
				return true
			}
		}
	}
	return false
}

func inputPortExists(ports nodecontract.PortSet, id string) bool {
	for _, port := range ports.DataInputs {
		if port.ID == id {
			return true
		}
	}
	for _, port := range ports.ExecInputs {
		if port.ID == id {
			return true
		}
	}
	return false
}

func resourceLeaseAssignable(source, target *nodecontract.ResourceLeaseBinding) bool {
	if source == nil || target == nil {
		return source == nil && target == nil
	}
	allowed := make(map[string]struct{}, len(source.Operations))
	for _, operation := range source.Operations {
		allowed[operation] = struct{}{}
	}
	for _, operation := range target.Operations {
		if _, ok := allowed[operation]; !ok {
			return false
		}
	}
	return true
}

func outputForChannel(ports nodecontract.PortSet, channel schema.EdgeChannel, id string) (*datatype.TypeExpression, bool) {
	if channel == schema.EdgeData {
		for _, port := range ports.DataOutputs {
			if port.ID == id {
				return &port.Type, true
			}
		}
		return nil, false
	}
	for _, port := range signalOutputs(ports, channel) {
		if port.ID == id {
			return nil, true
		}
	}
	return nil, false
}

func inputForChannel(ports nodecontract.PortSet, channel schema.EdgeChannel, id string) (*datatype.TypeExpression, bool) {
	if channel == schema.EdgeData {
		for _, port := range ports.DataInputs {
			if port.ID == id {
				return &port.Type, true
			}
		}
		return nil, false
	}
	if channel != schema.EdgeExec && channel != schema.EdgeError {
		return nil, false
	}
	for _, port := range ports.ExecInputs {
		if port.ID == id {
			return nil, true
		}
	}
	return nil, false
}

func programExecutableClass(class nodecontract.ExecutionClass) bool {
	switch class {
	case nodecontract.ExecutionPureData, nodecontract.ExecutionEffect, nodecontract.ExecutionControl, nodecontract.ExecutionEvent, nodecontract.ExecutionRegion:
		return true
	default:
		return false
	}
}

func signalOutputs(ports nodecontract.PortSet, channel schema.EdgeChannel) []nodecontract.SignalPort {
	switch channel {
	case schema.EdgeExec:
		return ports.ExecOutputs
	case schema.EdgeError:
		return ports.ErrorOutputs
	default:
		return nil
	}
}

func validateJSONSchemaBundleCached(cache map[string]*runtimejsonschema.Schema, key, root string, resources []datatype.SchemaResource, value any) error {
	if cache != nil {
		if validator := cache[key]; validator != nil {
			return validator.Validate(value)
		}
	}
	compiler := runtimejsonschema.NewCompiler()
	for _, resource := range resources {
		var schemaDocument any
		decoder := json.NewDecoder(bytes.NewReader(resource.Schema))
		decoder.UseNumber()
		if err := decoder.Decode(&schemaDocument); err != nil {
			return err
		}
		if err := compiler.AddResource(resource.ID, schemaDocument); err != nil {
			return err
		}
	}
	validator, err := compiler.Compile(root)
	if err != nil {
		return err
	}
	if cache != nil {
		cache[key] = validator
	}
	return validator.Validate(value)
}

func topologicalOrder(nodes []programNode, adjacency map[string][]string, indegree map[string]int) []string {
	queue := make([]string, 0, len(nodes))
	for id, degree := range indegree {
		if degree == 0 {
			queue = append(queue, id)
		}
	}
	sort.Strings(queue)
	order := make([]string, 0, len(nodes))
	for len(queue) > 0 {
		id := queue[0]
		queue = queue[1:]
		order = append(order, id)
		next := append([]string(nil), adjacency[id]...)
		sort.Strings(next)
		for _, target := range next {
			indegree[target]--
			if indegree[target] == 0 {
				queue = append(queue, target)
				sort.Strings(queue)
			}
		}
	}
	return order
}

func diagnostic(code string, path []string, graphID string) Diagnostic {
	diagnostic := Diagnostic{Code: code, Severity: schema.SeverityError, FieldPath: append([]string(nil), path...), Params: map[string]any{}}
	if graphID != "" {
		diagnostic.GraphPath = []string{graphID}
	}
	return diagnostic
}

func diagnosticAtNode(code string, path []string, graphID, nodeID string) Diagnostic {
	diagnostic := diagnostic(code, path, graphID)
	diagnostic.NodeID = nodeID
	return diagnostic
}

func warningAtNode(code string, path []string, graphID, nodeID string) Diagnostic {
	diagnostic := diagnosticAtNode(code, path, graphID, nodeID)
	diagnostic.Severity = schema.SeverityWarning
	return diagnostic
}

func sortDiagnostics(values []Diagnostic) {
	sort.SliceStable(values, func(i, j int) bool {
		left, _ := json.Marshal(values[i].FieldPath)
		right, _ := json.Marshal(values[j].FieldPath)
		if string(left) == string(right) {
			return values[i].Code < values[j].Code
		}
		return string(left) < string(right)
	})
}
