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
	"github.com/yottaapp/yotta/internal/capability"
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
	CodeMissingInputBinding      = "MISSING_INPUT_BINDING"
	CodeDuplicateInputBinding    = "DUPLICATE_INPUT_BINDING"
	CodeInvalidBinding           = "INVALID_BINDING"
	CodeInvalidConfig            = "INVALID_CONFIG"
	CodeDataCycle                = "DATA_CYCLE"
	CodeUnsupportedGraph         = "UNSUPPORTED_GRAPH"
	CodeUnsupportedSourceFeature = "UNSUPPORTED_SOURCE_FEATURE"

	sourceDigestDomain = "yotta/workflow-source/v1"
)

type Diagnostic = schema.Diagnostic

type CompileRequest struct {
	SourceJSON []byte
	Catalog    nodecatalog.Snapshot
}

type CompileResult struct {
	SourceHash  artifact.Digest `json:"sourceHash,omitempty"`
	Diagnostics []Diagnostic    `json:"diagnostics"`
	program     ProgramSnapshot
}

func (r CompileResult) Program() (ProgramSnapshot, bool) { return r.program, r.program.Valid() }

type Compiler struct{ build artifact.Digest }

func New(build artifact.Digest) *Compiler { return &Compiler{build: build} }

func (c *Compiler) CompileDraft(ctx context.Context, request CompileRequest) (CompileResult, error) {
	if err := ctx.Err(); err != nil {
		return CompileResult{}, err
	}
	if !c.build.Valid() {
		return CompileResult{}, errors.New("compiler build must be a content digest")
	}
	if len(request.SourceJSON) == 0 || len(request.SourceJSON) > MaxSourceBytes {
		return CompileResult{}, errors.New("workflow source exceeds byte budget")
	}
	source, diagnostics := schema.ParseSource(request.SourceJSON)
	result := CompileResult{Diagnostics: diagnostics}
	if len(diagnostics) != 0 {
		return result, nil
	}
	canonicalSource, err := artifact.Canonicalize(request.SourceJSON)
	if err != nil {
		return result, fmt.Errorf("canonicalize workflow source: %w", err)
	}
	result.SourceHash, err = artifact.Sum(sourceDigestDomain, canonicalSource)
	if err != nil {
		return result, err
	}
	if !request.Catalog.Valid() {
		result.Diagnostics = []Diagnostic{diagnostic(CodeInvalidCatalog, nil, "")}
		return result, nil
	}

	body := programBody{
		SourceHash: result.SourceHash, CatalogHash: request.Catalog.Hash(), CompilerBuild: c.build,
		WorkflowID: source.Workflow.ID, Revision: source.Revision, EntryGraph: source.EntryGraph,
		Graphs: []programGraph{},
	}
	if len(source.Variables) != 0 {
		result.Diagnostics = append(result.Diagnostics, diagnostic(CodeUnsupportedSourceFeature, []string{"variables"}, ""))
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
		compiled, graphDiagnostics, compileErr := compileGraph(ctx, graph, graphIndex, request.Catalog)
		if compileErr != nil {
			return CompileResult{}, compileErr
		}
		result.Diagnostics = append(result.Diagnostics, graphDiagnostics...)
		for _, node := range compiled.Nodes {
			entry, _ := request.Catalog.Lookup(node.NodeRef.NodeTypeID)
			for _, requirement := range entry.Contract.Machine().CapabilityRequirements {
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
	if len(result.Diagnostics) != 0 {
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
	return result, nil
}

func compileGraph(ctx context.Context, graph schema.Graph, graphIndex int, catalog nodecatalog.Snapshot) (programGraph, []Diagnostic, error) {
	compiled := programGraph{ID: graph.ID, Nodes: []programNode{}, Edges: append([]schema.Edge(nil), graph.Edges...), Order: []string{}}
	var diagnostics []Diagnostic
	nodes := make(map[string]*programNode, len(graph.Nodes))
	contracts := make(map[string]nodecontract.MachineContract, len(graph.Nodes))
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
		if machine.Execution.Class != nodecontract.ExecutionPureData {
			diagnostics = append(diagnostics, diagnosticAtNode(CodeUnsupportedSourceFeature, append(path, "nodeRef"), graph.ID, sourceNode.ID))
			continue
		}
		if err := validateJSONSchemaBundleCached(validators, "config:"+sourceNode.NodeRef.SemanticDigest.String(), machine.ConfigSchemaRoot, machine.ConfigSchemaBundle, sourceNode.Config); err != nil {
			diagnostics = append(diagnostics, diagnosticAtNode(CodeInvalidConfig, append(path, "config"), graph.ID, sourceNode.ID))
		}
		plan := programNode{
			ID: sourceNode.ID, NodeRef: sourceNode.NodeRef, Config: sourceNode.Config,
			Inputs: map[string]inputPlan{}, Ports: machine.Ports, Execution: machine.Execution,
			Implementation: entry.Implementation,
		}
		inputs := make(map[string]nodecontract.DataInputPort, len(machine.Ports.DataInputs))
		for _, port := range machine.Ports.DataInputs {
			inputs[port.ID] = port
		}
		for portID, binding := range sourceNode.Bindings {
			port, exists := inputs[portID]
			if !exists {
				diagnostics = append(diagnostics, diagnosticAtNode(CodeUnknownPort, append(path, "bindings", portID), graph.ID, sourceNode.ID))
				continue
			}
			var value json.RawMessage
			provenance := inputSourceLiteral
			switch binding.Kind {
			case schema.BindingValue:
				value = binding.Value
			case schema.BindingDefault:
				provenance = inputSourceDefault
				if port.Default == nil {
					diagnostics = append(diagnostics, diagnosticAtNode(CodeInvalidBinding, append(path, "bindings", portID), graph.ID, sourceNode.ID))
					continue
				}
				value = *port.Default
			default:
				continue
			}
			canonical, err := artifact.Canonicalize(value)
			if err == nil {
				err = validateLiteralCached(port.Type, canonical, catalog, validators)
			}
			if err != nil {
				diagnostic := diagnosticAtNode(CodeInvalidBinding, append(path, "bindings", portID), graph.ID, sourceNode.ID)
				diagnostic.Params["reason"] = err.Error()
				diagnostics = append(diagnostics, diagnostic)
				continue
			}
			resolved, resolveErr := resolvedTypeForExactRef(port.Type, catalog)
			if resolveErr != nil {
				diagnostic := diagnosticAtNode(CodeInvalidBinding, append(path, "bindings", portID), graph.ID, sourceNode.ID)
				diagnostic.Params["reason"] = resolveErr.Error()
				diagnostics = append(diagnostics, diagnostic)
				continue
			}
			envelope, envelopeErr := datatype.SealInlineJSON(catalog, resolved, canonical)
			if envelopeErr != nil {
				diagnostic := diagnosticAtNode(CodeInvalidBinding, append(path, "bindings", portID), graph.ID, sourceNode.ID)
				diagnostic.Params["reason"] = envelopeErr.Error()
				diagnostics = append(diagnostics, diagnostic)
				continue
			}
			plan.Inputs[portID] = inputPlan{Kind: inputLiteral, Provenance: provenance, Value: envelope.Artifact()}
		}
		compiled.Nodes = append(compiled.Nodes, plan)
		nodes[plan.ID] = &compiled.Nodes[len(compiled.Nodes)-1]
		contracts[plan.ID] = machine
	}

	incoming := map[string]bool{}
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
		if edge.Channel != schema.EdgeData {
			diagnostics = append(diagnostics, diagnostic(CodeUnsupportedSourceFeature, append(path, "channel"), graph.ID))
			continue
		}
		fromNode, fromOK := nodes[edge.From.NodeID]
		toNode, toOK := nodes[edge.To.NodeID]
		if !fromOK || !toOK {
			diagnostics = append(diagnostics, diagnostic(CodeUnknownPort, path, graph.ID))
			continue
		}
		fromMachine, toMachine := contracts[fromNode.ID], contracts[toNode.ID]
		fromType, fromExists := outputForChannel(fromMachine.Ports, edge.Channel, edge.From.PortID)
		toType, toExists := inputForChannel(toMachine.Ports, edge.Channel, edge.To.PortID)
		if !fromExists || !toExists {
			diagnostics = append(diagnostics, diagnostic(CodeUnknownPort, path, graph.ID))
			continue
		}
		key := edge.To.NodeID + "\x00" + string(edge.Channel) + "\x00" + edge.To.PortID
		if incoming[key] {
			diagnostics = append(diagnostics, diagnostic(CodeDuplicateInputBinding, path, graph.ID))
			continue
		}
		incoming[key] = true
		if edge.Channel == schema.EdgeData {
			assignable, err := datatype.Assignable(*fromType, *toType)
			if err != nil || !assignable {
				diagnostics = append(diagnostics, diagnostic(CodeTypeMismatch, path, graph.ID))
				continue
			}
			if _, already := toNode.Inputs[edge.To.PortID]; already {
				diagnostics = append(diagnostics, diagnostic(CodeDuplicateInputBinding, path, graph.ID))
				continue
			}
			toNode.Inputs[edge.To.PortID] = inputPlan{Kind: inputEdge, From: edge.From}
			adjacency[fromNode.ID] = append(adjacency[fromNode.ID], toNode.ID)
			indegree[toNode.ID]++
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
	compiled.Order = topologicalOrder(compiled.Nodes, adjacency, indegree)
	if len(compiled.Order) != len(compiled.Nodes) {
		diagnostics = append(diagnostics, diagnostic(CodeDataCycle, []string{"graphs", fmt.Sprint(graphIndex), "edges"}, graph.ID))
	}
	return compiled, diagnostics, nil
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
	if channel != schema.EdgeExec {
		return nil, false
	}
	for _, port := range ports.ExecInputs {
		if port.ID == id {
			return nil, true
		}
	}
	return nil, false
}

func signalOutputs(ports nodecontract.PortSet, channel schema.EdgeChannel) []nodecontract.SignalPort {
	switch channel {
	case schema.EdgeExec:
		return ports.ExecOutputs
	case schema.EdgeError:
		return ports.ErrorOutputs
	case schema.EdgeStatus:
		return ports.StatusOutputs
	default:
		return nil
	}
}

func validateLiteralCached(expression datatype.TypeExpression, raw []byte, catalog nodecatalog.Snapshot, validators map[string]*runtimejsonschema.Schema) error {
	if expression.Kind != datatype.TypeExpressionRef || expression.Ref == nil {
		return errors.New("literal binding currently requires an exact ref type")
	}
	definition, ok := catalog.LookupType(expression.Ref.TypeID)
	if !ok || definition.TypeRef() != *expression.Ref {
		return errors.New("literal type is not in catalog")
	}
	machine := definition.Machine()
	if len(machine.SchemaBundle) != 1 {
		return errors.New("literal schema requires one explicit root in this compiler generation")
	}
	var value any
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.UseNumber()
	if err := decoder.Decode(&value); err != nil {
		return err
	}
	return validateJSONSchemaBundleCached(validators, "type:"+expression.Ref.SemanticDigest.String(), machine.SchemaBundle[0].ID, machine.SchemaBundle, value)
}

func resolvedTypeForExactRef(expression datatype.TypeExpression, catalog nodecatalog.Snapshot) (datatype.ResolvedType, error) {
	if expression.Kind != datatype.TypeExpressionRef || expression.Ref == nil {
		return datatype.ResolvedType{}, errors.New("preview runtime requires an exact ref type")
	}
	definition, ok := catalog.LookupType(expression.Ref.TypeID)
	if !ok || definition.TypeRef() != *expression.Ref {
		return datatype.ResolvedType{}, errors.New("value type is not in catalog")
	}
	return datatype.RefResolvedType(*expression.Ref), nil
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
