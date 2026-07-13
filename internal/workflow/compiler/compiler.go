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
		result.Diagnostics = cloneDiagnostics(diagnostics)
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
		Graphs: lowerGraphs(source.Graphs),
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
}

func bindSource(ctx context.Context, source schema.WorkflowSource, snapshot catalog.Snapshot) (bindingResult, error) {
	result := bindingResult{}
	locksByKind := map[string]NodeLock{}
	capabilities := map[string]bool{}
	contractCache := map[string]catalog.NodeContract{}
	missingKinds := map[string]bool{}
	for graphIndex, graph := range source.Graphs {
		if err := ctx.Err(); err != nil {
			return result, err
		}
		if graph.Kind == schema.GraphKindSubgraph || len(graph.Inputs) != 0 || len(graph.Outputs) != 0 {
			result.diagnostics = append(result.diagnostics, schema.Diagnostic{
				Code: schema.CodeUnsupportedGraphContract, Severity: schema.SeverityError,
				GraphPath: []string{graph.ID}, FieldPath: []string{"graphs", strconv.Itoa(graphIndex), "kind"},
				Params: map[string]any{"kind": graph.Kind},
			})
			continue
		}
		contractsByNode := make(map[string]catalog.NodeContract, len(graph.Nodes))
		nodeExists := make(map[string]bool, len(graph.Nodes))
		for nodeIndex, sourceNode := range graph.Nodes {
			if nodeIndex&255 == 0 {
				if err := ctx.Err(); err != nil {
					return result, err
				}
			}
			nodeExists[sourceNode.ID] = true
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
				result.diagnostics = append(result.diagnostics, schema.Diagnostic{
					Code: schema.CodeUnknownNodeKind, Severity: schema.SeverityError,
					GraphPath: []string{graph.ID}, NodeID: sourceNode.ID,
					FieldPath: []string{"graphs", strconv.Itoa(graphIndex), "nodes", strconv.Itoa(nodeIndex), "kind"},
					Params:    map[string]any{"kind": sourceNode.Kind},
				})
				continue
			}
			contractsByNode[sourceNode.ID] = contract
			for _, input := range contract.Inputs {
				if input.Default != nil && (!valueMatchesType(input.Default, input.Type) || input.Schema != nil && !valueMatchesSchema(input.Default, input.Schema)) {
					return result, fmt.Errorf("%w: node %q input %q has invalid default", ErrInvalidCatalog, contract.Kind, input.Name)
				}
			}
			if contract.DynamicInputs || contract.DynamicOutputs || contract.DynamicDataFields || contract.HasCustomValidation || contract.HasDependencies {
				result.diagnostics = append(result.diagnostics, schema.Diagnostic{
					Code: schema.CodeUnsupportedNodeContract, Severity: schema.SeverityError,
					GraphPath: []string{graph.ID}, NodeID: sourceNode.ID,
					FieldPath: []string{"graphs", strconv.Itoa(graphIndex), "nodes", strconv.Itoa(nodeIndex), "kind"},
					Params:    map[string]any{"kind": sourceNode.Kind},
				})
				continue
			}
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
		incoming := map[string]map[string]bool{}
		for edgeIndex, edge := range graph.Edges {
			if edgeIndex&255 == 0 {
				if err := ctx.Err(); err != nil {
					return result, err
				}
			}
			for _, endpoint := range []struct {
				field string
				value string
				from  bool
			}{{"from", edge.From, true}, {"to", edge.To, false}} {
				nodeID, pin, ok := splitEndpoint(endpoint.value, nodeExists)
				path := []string{"graphs", strconv.Itoa(graphIndex), "edges", strconv.Itoa(edgeIndex), endpoint.field}
				if !ok {
					result.diagnostics = append(result.diagnostics, bindingDiagnostic(graph.ID, "", path, "edgeEndpoint", map[string]any{"endpoint": endpoint.value}))
					continue
				}
				contract, bound := contractsByNode[nodeID]
				if !bound {
					continue
				}
				if !pinAllowed(contract, pin, endpoint.from) {
					result.diagnostics = append(result.diagnostics, bindingDiagnostic(graph.ID, nodeID, path, "edgePin", map[string]any{"pin": pin, "kind": contract.Kind}))
				}
			}
			fromNode, fromPin, fromOK := splitEndpoint(edge.From, nodeExists)
			toNode, toPin, toOK := splitEndpoint(edge.To, nodeExists)
			fromContract, fromBound := contractsByNode[fromNode]
			toContract, toBound := contractsByNode[toNode]
			if fromOK && toOK && fromBound && toBound {
				fromType, fromPinOK := outputType(fromContract, fromPin)
				toType, toPinOK := inputType(toContract, toPin)
				if fromPinOK && toPinOK {
					if !compatibleTypes(fromType, toType) {
						path := []string{"graphs", strconv.Itoa(graphIndex), "edges", strconv.Itoa(edgeIndex)}
						result.diagnostics = append(result.diagnostics, bindingDiagnostic(graph.ID, toNode, path, "pinType", map[string]any{"fromType": fromType, "toType": toType}))
					} else {
						if incoming[toNode] == nil {
							incoming[toNode] = map[string]bool{}
						}
						incoming[toNode][toPin] = true
					}
				}
			}
		}
		for nodeIndex, sourceNode := range graph.Nodes {
			contract, ok := contractsByNode[sourceNode.ID]
			if !ok || contract.DynamicInputs || contract.DynamicOutputs || contract.DynamicDataFields || contract.HasCustomValidation || contract.HasDependencies {
				continue
			}
			validateNodeConfig(&result.diagnostics, graph.ID, graphIndex, nodeIndex, sourceNode, contract, incoming[sourceNode.ID])
		}
		for _, ports := range []struct {
			name   string
			values []schema.GraphPort
		}{{"inputs", graph.Inputs}, {"outputs", graph.Outputs}} {
			for portIndex, port := range ports.values {
				if !nodeExists[port.NodeID] {
					path := []string{"graphs", strconv.Itoa(graphIndex), ports.name, strconv.Itoa(portIndex), "nodeId"}
					result.diagnostics = append(result.diagnostics, bindingDiagnostic(graph.ID, port.NodeID, path, "graphPortNode", map[string]any{"nodeId": port.NodeID}))
				}
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

func splitEndpoint(endpoint string, nodeIDs map[string]bool) (string, string, bool) {
	for dot := strings.LastIndexByte(endpoint, '.'); dot > 0; dot = strings.LastIndexByte(endpoint[:dot], '.') {
		nodeID, pin := endpoint[:dot], endpoint[dot+1:]
		if nodeIDs[nodeID] && pin != "" {
			return nodeID, pin, true
		}
	}
	return "", "", false
}

func pinAllowed(contract catalog.NodeContract, pin string, from bool) bool {
	if from {
		if contract.DynamicOutputs {
			return true
		}
		for _, output := range contract.Outputs {
			if output.Name == pin {
				return true
			}
		}
		return false
	}
	if contract.DynamicInputs {
		return true
	}
	for _, input := range contract.Inputs {
		if input.Name == pin {
			return true
		}
	}
	return false
}

func outputType(contract catalog.NodeContract, pin string) (string, bool) {
	for _, output := range contract.Outputs {
		if output.Name == pin {
			return output.Type, true
		}
	}
	return "", false
}

func inputType(contract catalog.NodeContract, pin string) (string, bool) {
	for _, input := range contract.Inputs {
		if input.Name == pin {
			return input.Type, true
		}
	}
	return "", false
}

func compatibleTypes(from, to string) bool {
	left, right := node.CanonicalPinType(from), node.CanonicalPinType(to)
	return left == right || left == "any" || right == "any"
}

func validateNodeConfig(out *[]schema.Diagnostic, graphID string, graphIndex, nodeIndex int, sourceNode schema.Node, contract catalog.NodeContract, incoming map[string]bool) {
	inputs := make(map[string]catalog.InputContract, len(contract.Inputs))
	for _, input := range contract.Inputs {
		inputs[input.Name] = input
	}
	base := []string{"graphs", strconv.Itoa(graphIndex), "nodes", strconv.Itoa(nodeIndex), "config"}
	for key, value := range sourceNode.Config {
		input, ok := inputs[key]
		path := appendPath(base, key)
		if !ok || input.Type == "Exec" {
			*out = append(*out, bindingDiagnostic(graphID, sourceNode.ID, path, "configField", map[string]any{"field": key}))
			continue
		}
		if !valueMatchesType(value, input.Type) || input.Schema != nil && !valueMatchesSchema(value, input.Schema) {
			*out = append(*out, bindingDiagnostic(graphID, sourceNode.ID, path, "configType", map[string]any{"field": key, "type": input.Type}))
		}
	}
	for _, input := range contract.Inputs {
		if !input.Required {
			continue
		}
		_, configured := sourceNode.Config[input.Name]
		if !configured && input.Default == nil && !incoming[input.Name] {
			*out = append(*out, bindingDiagnostic(graphID, sourceNode.ID, appendPath(base, input.Name), "requiredInput", map[string]any{"field": input.Name}))
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
			diagnostics = append(diagnostics, schema.Diagnostic{Code: schema.CodeMissingCapabilityDeclaration, Severity: schema.SeverityError, FieldPath: []string{"requestedCapabilities"}, Params: map[string]any{"capability": value}})
		}
	}
	for _, value := range declared {
		if !requiredSet[value] {
			diagnostics = append(diagnostics, schema.Diagnostic{Code: schema.CodeUnusedCapabilityDeclaration, Severity: schema.SeverityError, FieldPath: []string{"requestedCapabilities"}, Params: map[string]any{"capability": value}})
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

func lowerGraphs(graphs []schema.Graph) []programGraph {
	out := make([]programGraph, len(graphs))
	for graphIndex, graph := range graphs {
		nodes := make([]programNode, len(graph.Nodes))
		for nodeIndex, node := range graph.Nodes {
			nodes[nodeIndex] = programNode{ID: node.ID, Kind: node.Kind, Config: node.Config, Disabled: node.Disabled}
		}
		edges := make([]schema.Edge, len(graph.Edges))
		copy(edges, graph.Edges)
		inputs := make([]schema.GraphPort, len(graph.Inputs))
		copy(inputs, graph.Inputs)
		outputs := make([]schema.GraphPort, len(graph.Outputs))
		copy(outputs, graph.Outputs)
		out[graphIndex] = programGraph{
			ID: graph.ID, Kind: graph.Kind, Nodes: nodes,
			Edges: edges, Inputs: inputs, Outputs: outputs,
		}
	}
	return out
}

func cloneDiagnostics(source []schema.Diagnostic) []schema.Diagnostic {
	out := make([]schema.Diagnostic, len(source))
	copy(out, source)
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
