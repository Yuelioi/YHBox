package compiler

import (
	"errors"
	"fmt"
	"reflect"

	"github.com/yottaapp/yotta/internal/nodecatalog"
	"github.com/yottaapp/yotta/internal/workflow/schema"
)

const graphCallInputPort = "in"

type sourceNodeLocation struct {
	GraphPath []string
	GraphID   string
	NodeID    string
}

type expandedBoundary struct {
	inputs  map[string]schema.Endpoint
	outputs map[string]schema.Endpoint
	entries []schema.Endpoint
	exits   map[string]schema.GraphExit
}

type graphExpansion struct {
	graph     schema.Graph
	locations map[string]sourceNodeLocation
	boundary  expandedBoundary
}

type graphExpander struct {
	graphs   map[string]schema.Graph
	catalog  nodecatalog.Snapshot
	nextNode int
	usedIDs  map[string]bool
}

func expandWorkflow(source schema.WorkflowSource, catalog nodecatalog.Snapshot) (graphExpansion, []Diagnostic, error) {
	graphs := make(map[string]schema.Graph, len(source.Graphs))
	for _, graph := range source.Graphs {
		graphs[graph.ID] = graph
	}
	expander := graphExpander{graphs: graphs, catalog: catalog, usedIDs: map[string]bool{}}
	for _, node := range graphs[source.EntryGraph].Nodes {
		expander.usedIDs[node.ID] = true
	}
	expanded, diagnostics, err := expander.expand(source.EntryGraph, []string{source.EntryGraph}, 1)
	if err != nil {
		return graphExpansion{}, nil, err
	}
	expanded.graph.ID = source.EntryGraph
	expanded.graph.Kind = schema.GraphKindMain
	expanded.graph.Inputs = nil
	expanded.graph.Outputs = nil
	expanded.graph.Entries = nil
	expanded.graph.Exits = nil
	expanded.graph.Calls = nil
	expanded.graph.Annotations = nil
	diagnostics = append(expander.validateContracts(source), diagnostics...)
	return expanded, diagnostics, nil
}

func (e *graphExpander) validateContracts(source schema.WorkflowSource) []Diagnostic {
	var diagnostics []Diagnostic
	for graphIndex, graph := range source.Graphs {
		for portIndex, port := range graph.Inputs {
			actual, ok := e.boundaryType(graph, schema.Endpoint{NodeID: port.NodeID, PortID: port.PortID}, false)
			if !ok || !reflect.DeepEqual(actual, port.Type) {
				diagnostics = append(diagnostics, diagnostic(schema.CodeCallPinTypeMismatch, []string{"graphs", fmt.Sprint(graphIndex), "inputs", fmt.Sprint(portIndex)}, graph.ID))
			}
		}
		for portIndex, port := range graph.Outputs {
			actual, ok := e.boundaryType(graph, schema.Endpoint{NodeID: port.NodeID, PortID: port.PortID}, true)
			if !ok || !reflect.DeepEqual(actual, port.Type) {
				diagnostics = append(diagnostics, diagnostic(schema.CodeCallPinTypeMismatch, []string{"graphs", fmt.Sprint(graphIndex), "outputs", fmt.Sprint(portIndex)}, graph.ID))
			}
		}
		for entryIndex, entry := range graph.Entries {
			if !e.signalBoundary(graph, entry, false, schema.EdgeExec) {
				diagnostics = append(diagnostics, diagnostic(schema.CodeInvalidGraphEntry, []string{"graphs", fmt.Sprint(graphIndex), "entries", fmt.Sprint(entryIndex)}, graph.ID))
			}
		}
		for exitIndex, exit := range graph.Exits {
			if !e.signalBoundary(graph, exit.Endpoint, true, exit.Channel) {
				diagnostics = append(diagnostics, diagnostic(schema.CodeInvalidGraphBoundaryEdge, []string{"graphs", fmt.Sprint(graphIndex), "exits", fmt.Sprint(exitIndex)}, graph.ID))
			}
		}
		for callIndex, call := range graph.Calls {
			callee, ok := e.graphs[call.GraphID]
			if !ok {
				continue
			}
			for portID := range call.Bindings {
				found := false
				for _, port := range callee.Inputs {
					found = found || port.ID == portID
				}
				if !found {
					diagnostics = append(diagnostics, diagnosticAtNode(CodeUnknownPort, []string{"graphs", fmt.Sprint(graphIndex), "calls", fmt.Sprint(callIndex), "bindings", portID}, graph.ID, call.ID))
				}
			}
		}
	}
	return diagnostics
}

func (e *graphExpander) boundaryType(graph schema.Graph, endpoint schema.Endpoint, output bool) (any, bool) {
	for _, node := range graph.Nodes {
		if node.ID != endpoint.NodeID {
			continue
		}
		entry, ok := e.catalog.Lookup(node.NodeRef.NodeTypeID)
		if !ok || entry.Contract.NodeRef() != node.NodeRef {
			return nil, false
		}
		ports := entry.Contract.Machine().Ports.DataInputs
		if output {
			for _, port := range entry.Contract.Machine().Ports.DataOutputs {
				if port.ID == endpoint.PortID {
					return port.Type, true
				}
			}
			return nil, false
		}
		for _, port := range ports {
			if port.ID == endpoint.PortID {
				return port.Type, true
			}
		}
		return nil, false
	}
	for _, call := range graph.Calls {
		if call.ID != endpoint.NodeID {
			continue
		}
		callee, ok := e.graphs[call.GraphID]
		if !ok {
			return nil, false
		}
		ports := callee.Inputs
		if output {
			ports = callee.Outputs
		}
		for _, port := range ports {
			if port.ID == endpoint.PortID {
				return port.Type, true
			}
		}
	}
	return nil, false
}

func (e *graphExpander) signalBoundary(graph schema.Graph, endpoint schema.Endpoint, output bool, channel schema.EdgeChannel) bool {
	for _, node := range graph.Nodes {
		if node.ID != endpoint.NodeID {
			continue
		}
		entry, ok := e.catalog.Lookup(node.NodeRef.NodeTypeID)
		if !ok || entry.Contract.NodeRef() != node.NodeRef {
			return false
		}
		if output {
			_, ok = outputForChannel(entry.Contract.Machine().Ports, channel, endpoint.PortID)
		} else {
			_, ok = inputForChannel(entry.Contract.Machine().Ports, channel, endpoint.PortID)
		}
		return ok
	}
	for _, call := range graph.Calls {
		if call.ID != endpoint.NodeID {
			continue
		}
		if !output {
			return channel == schema.EdgeExec && endpoint.PortID == graphCallInputPort
		}
		callee, ok := e.graphs[call.GraphID]
		if !ok {
			return false
		}
		for _, exit := range callee.Exits {
			if exit.ID == endpoint.PortID && exit.Channel == channel {
				return true
			}
		}
	}
	return false
}

func (e *graphExpander) expand(graphID string, path []string, depth int) (graphExpansion, []Diagnostic, error) {
	if depth > schema.MaxGraphDepth {
		return graphExpansion{}, nil, errors.New("subgraph expansion exceeds runtime depth budget")
	}
	graph, ok := e.graphs[graphID]
	if !ok {
		return graphExpansion{}, nil, fmt.Errorf("expand unknown graph %q", graphID)
	}
	result := graphExpansion{
		graph:     schema.Graph{ID: graph.ID, Kind: graph.Kind, Nodes: []schema.Node{}, Edges: []schema.Edge{}},
		locations: map[string]sourceNodeLocation{},
		boundary:  expandedBoundary{inputs: map[string]schema.Endpoint{}, outputs: map[string]schema.Endpoint{}, exits: map[string]schema.GraphExit{}},
	}
	localNodes := make(map[string]string, len(graph.Nodes))
	for _, node := range graph.Nodes {
		runtimeID := node.ID
		if depth > 1 {
			for e.usedIDs[runtimeID] {
				e.nextNode++
				runtimeID = fmt.Sprintf("n%d", e.nextNode)
			}
		}
		e.usedIDs[runtimeID] = true
		localNodes[node.ID] = runtimeID
		copyNode := node
		copyNode.ID = runtimeID
		result.graph.Nodes = append(result.graph.Nodes, copyNode)
		result.locations[runtimeID] = sourceNodeLocation{GraphPath: append([]string(nil), path...), GraphID: graph.ID, NodeID: node.ID}
	}

	callBoundaries := make(map[string]expandedBoundary, len(graph.Calls))
	var diagnostics []Diagnostic
	for callIndex, call := range graph.Calls {
		callee, exists := e.graphs[call.GraphID]
		if !exists || callee.Kind != schema.GraphKindSubgraph {
			continue
		}
		// Preserve the call site as part of runtime provenance. Graph IDs alone
		// cannot distinguish two invocations of the same reusable subgraph.
		childPath := append(append([]string(nil), path...), call.ID, call.GraphID)
		child, childDiagnostics, err := e.expand(call.GraphID, childPath, depth+1)
		if err != nil {
			return graphExpansion{}, diagnostics, err
		}
		diagnostics = append(diagnostics, childDiagnostics...)
		result.graph.Nodes = append(result.graph.Nodes, child.graph.Nodes...)
		result.graph.Edges = append(result.graph.Edges, child.graph.Edges...)
		for runtimeID, location := range child.locations {
			result.locations[runtimeID] = location
		}
		callBoundaries[call.ID] = child.boundary
		for portID, binding := range call.Bindings {
			endpoint, present := child.boundary.inputs[portID]
			if !present {
				diagnostics = append(diagnostics, diagnosticAtNode(CodeUnknownPort, []string{"graphs", graph.ID, "calls", fmt.Sprint(callIndex), "bindings", portID}, graph.ID, call.ID))
				continue
			}
			for nodeIndex := range result.graph.Nodes {
				if result.graph.Nodes[nodeIndex].ID == endpoint.NodeID {
					if result.graph.Nodes[nodeIndex].Bindings == nil {
						result.graph.Nodes[nodeIndex].Bindings = map[string]schema.InputBinding{}
					}
					result.graph.Nodes[nodeIndex].Bindings[endpoint.PortID] = binding
					break
				}
			}
		}
	}

	resolve := func(endpoint schema.Endpoint, from bool, channel schema.EdgeChannel) ([]schema.Endpoint, bool) {
		if runtimeID, present := localNodes[endpoint.NodeID]; present {
			return []schema.Endpoint{{NodeID: runtimeID, PortID: endpoint.PortID}}, true
		}
		boundary, present := callBoundaries[endpoint.NodeID]
		if !present {
			return nil, false
		}
		if from {
			if channel == schema.EdgeData {
				mapped, ok := boundary.outputs[endpoint.PortID]
				return []schema.Endpoint{mapped}, ok
			}
			exit, ok := boundary.exits[endpoint.PortID]
			if !ok || exit.Channel != channel {
				return nil, false
			}
			return []schema.Endpoint{exit.Endpoint}, true
		}
		if channel == schema.EdgeData {
			mapped, ok := boundary.inputs[endpoint.PortID]
			return []schema.Endpoint{mapped}, ok
		}
		if endpoint.PortID != graphCallInputPort || channel != schema.EdgeExec || len(boundary.entries) == 0 {
			return nil, false
		}
		return append([]schema.Endpoint(nil), boundary.entries...), true
	}

	for edgeIndex, edge := range graph.Edges {
		from, fromOK := resolve(edge.From, true, edge.Channel)
		to, toOK := resolve(edge.To, false, edge.Channel)
		if !fromOK || !toOK {
			diagnostics = append(diagnostics, diagnostic(CodeUnknownPort, []string{"graphs", graph.ID, "edges", fmt.Sprint(edgeIndex)}, graph.ID))
			continue
		}
		for _, left := range from {
			for _, right := range to {
				copyEdge := edge
				copyEdge.From = left
				copyEdge.To = right
				result.graph.Edges = append(result.graph.Edges, copyEdge)
			}
		}
	}

	for _, port := range graph.Inputs {
		mapped, ok := resolve(schema.Endpoint{NodeID: port.NodeID, PortID: port.PortID}, false, schema.EdgeData)
		if ok && len(mapped) == 1 {
			result.boundary.inputs[port.ID] = mapped[0]
		}
	}
	for _, port := range graph.Outputs {
		mapped, ok := resolve(schema.Endpoint{NodeID: port.NodeID, PortID: port.PortID}, true, schema.EdgeData)
		if ok && len(mapped) == 1 {
			result.boundary.outputs[port.ID] = mapped[0]
		}
	}
	for _, entry := range graph.Entries {
		mapped, ok := resolve(entry, false, schema.EdgeExec)
		if ok {
			result.boundary.entries = append(result.boundary.entries, mapped...)
		}
	}
	for _, exit := range graph.Exits {
		mapped, ok := resolve(exit.Endpoint, true, exit.Channel)
		if ok && len(mapped) == 1 {
			copyExit := exit
			copyExit.Endpoint = mapped[0]
			result.boundary.exits[exit.ID] = copyExit
		}
	}
	return result, diagnostics, nil
}
