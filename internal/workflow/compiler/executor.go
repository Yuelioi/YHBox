package compiler

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"reflect"

	"github.com/yottaapp/yotta/internal/datatype"
	"github.com/yottaapp/yotta/internal/nodecatalog"
	"github.com/yottaapp/yotta/internal/nodecontract"
	"github.com/yottaapp/yotta/internal/resource"
	run31 "github.com/yottaapp/yotta/internal/run"
	"github.com/yottaapp/yotta/internal/runid"
)

type Adapter func(context.Context, Invocation) (map[string]datatype.ValueEnvelope, error)

type InstalledAdapter struct {
	Implementation nodecatalog.ImplementationLock
	Run            Adapter
}

type Invocation struct {
	GraphID  string
	NodeID   string
	Config   map[string]any
	Inputs   map[string]datatype.ValueEnvelope
	Sessions map[string]*run31.Session
	Spawn    func(func(context.Context) error) error
}

type ExecutionResult struct {
	// NodeOutputs contains only durable envelopes. Runtime authority remains an
	// executor-local edge value and is reclaimed before Run returns.
	NodeOutputs map[string]map[string]datatype.ValueEnvelope
}

type Executor struct {
	catalog  nodecatalog.Snapshot
	adapters map[string]InstalledAdapter
}

type ownedLease struct {
	session *run31.Session
	handle  resource.Handle
}

func NewExecutor(catalog nodecatalog.Snapshot, adapters map[string]InstalledAdapter) *Executor {
	installed := make(map[string]InstalledAdapter, len(adapters))
	for entrypoint, adapter := range adapters {
		installed[entrypoint] = adapter
	}
	return &Executor{catalog: catalog, adapters: installed}
}

func (e *Executor) Run(ctx context.Context, program ProgramSnapshot, owner *run31.Owner) (ExecutionResult, error) {
	if ctx == nil || !program.Valid() || !e.catalog.Valid() || owner == nil {
		return ExecutionResult{}, errors.New("executor requires Program, Catalog, and Run Owner")
	}
	runCtx, cancelRun := context.WithCancel(ctx)
	stopOwnerWatch := context.AfterFunc(owner.Context(), cancelRun)
	defer func() {
		stopOwnerWatch()
		cancelRun()
	}()
	ctx = runCtx
	if program.state.document.Body.CatalogHash != e.catalog.Hash() {
		return ExecutionResult{}, errors.New("executor catalog does not match Program")
	}
	plan := program.CapabilityPlan()
	if err := owner.ValidateProgram(program.Hash(), plan.Digest()); err != nil {
		return ExecutionResult{}, err
	}
	body := program.state.document.Body
	var graph *programGraph
	for index := range body.Graphs {
		if body.Graphs[index].ID == body.EntryGraph {
			graph = &body.Graphs[index]
			break
		}
	}
	if graph == nil {
		return ExecutionResult{}, errors.New("Program entry graph is missing")
	}
	nodes := make(map[string]programNode, len(graph.Nodes))
	for _, node := range graph.Nodes {
		nodes[node.ID] = node
	}
	result := ExecutionResult{NodeOutputs: make(map[string]map[string]datatype.ValueEnvelope)}
	retainedBytes := 0
	sessions := make(map[string]map[string]*run31.Session, len(graph.Nodes))
	owned := make([]ownedLease, 0)
	cleanup := func() error {
		var cleanupErrors []error
		for index := len(owned) - 1; index >= 0; index-- {
			if err := owned[index].session.Drop(context.Background(), owned[index].handle); err != nil && !errors.Is(err, resource.ErrUnknownHandle) {
				cleanupErrors = append(cleanupErrors, err)
			}
		}
		return errors.Join(cleanupErrors...)
	}
	completed := false
	defer func() {
		if !completed {
			_ = cleanup()
		}
	}()
	for _, nodeID := range graph.Order {
		if err := ctx.Err(); err != nil {
			return ExecutionResult{}, err
		}
		node, ok := nodes[nodeID]
		if !ok {
			return ExecutionResult{}, fmt.Errorf("Program order references missing node %q", nodeID)
		}
		entry, ok := e.catalog.Lookup(node.NodeRef.NodeTypeID)
		if !ok {
			return ExecutionResult{}, fmt.Errorf("node type %q is not installed", node.NodeRef.NodeTypeID)
		}
		installed, ok := e.adapters[node.Implementation.Entrypoint]
		if !ok || installed.Run == nil || installed.Implementation != node.Implementation {
			return ExecutionResult{}, fmt.Errorf("adapter %q does not match the Program lock", node.Implementation.Entrypoint)
		}
		invocationID, err := runid.New()
		if err != nil {
			return ExecutionResult{}, err
		}
		nodeSessions := make(map[string]*run31.Session, len(entry.Contract.Machine().CapabilityRequirements))
		for _, requirement := range entry.Contract.Machine().CapabilityRequirements {
			session, err := owner.Session(graph.ID, node.ID, requirement.ID, invocationID)
			if err != nil {
				return ExecutionResult{}, err
			}
			nodeSessions[requirement.ID] = session
		}
		sessions[node.ID] = nodeSessions
		inputs := make(map[string]datatype.ValueEnvelope, len(node.Inputs))
		for portID, input := range node.Inputs {
			envelope, sourceNode, sourcePort, err := e.resolveInput(result, input)
			if err != nil {
				return ExecutionResult{}, fmt.Errorf("resolve input %s.%s: %w", node.ID, portID, err)
			}
			inputPort, exists := dataInputPort(node.Ports, portID)
			if !exists {
				return ExecutionResult{}, errors.New("Program input port is missing from its effective contract")
			}
			if handle, runtimeKind, ok := runtimeHandle(envelope); ok {
				if !exists || inputPort.ResourceLease == nil || sourceNode == "" {
					return ExecutionResult{}, errors.New("runtime authority crossed a data edge without an explicit resource lease")
				}
				sourcePlan := nodes[sourceNode]
				outputPort, exists := dataOutputPort(sourcePlan.Ports, sourcePort)
				if !exists || outputPort.ResourceLease == nil {
					return ExecutionResult{}, errors.New("runtime authority source has no explicit resource lease")
				}
				if !resourceLeaseAssignable(outputPort.ResourceLease, inputPort.ResourceLease) {
					return ExecutionResult{}, errors.New("runtime authority edge widens its source resource lease")
				}
				lender := sessions[sourceNode][outputPort.ResourceLease.RequirementID]
				borrower := nodeSessions[inputPort.ResourceLease.RequirementID]
				borrowedHandle, err := owner.Borrow(ctx, lender, handle, borrower, inputPort.ResourceLease.Operations)
				if err != nil {
					return ExecutionResult{}, err
				}
				owned = append(owned, ownedLease{session: borrower, handle: borrowedHandle})
				switch runtimeKind {
				case datatype.RepresentationStreamRef:
					envelope, err = datatype.SealStreamRef(e.catalog, envelope.Type(), borrowedHandle)
				case datatype.RepresentationHandleRef:
					envelope, err = datatype.SealHandleRef(e.catalog, envelope.Type(), borrowedHandle)
				}
				if err != nil {
					return ExecutionResult{}, err
				}
			} else if inputPort.ResourceLease != nil {
				return ExecutionResult{}, errors.New("resource-leased input received a durable value")
			}
			inputs[portID] = envelope
		}
		config, err := cloneConfig(node.Config)
		if err != nil {
			return ExecutionResult{}, fmt.Errorf("clone config for node %q: %w", node.ID, err)
		}
		outputs, runErr := installed.Run(ctx, Invocation{
			GraphID: graph.ID, NodeID: node.ID, Config: config, Inputs: inputs,
			Sessions: nodeSessions, Spawn: owner.Go,
		})
		if runErr != nil {
			return ExecutionResult{}, fmt.Errorf("run node %q: %w", node.ID, runErr)
		}
		sealed, leases, err := e.validateOutputs(node, outputs, nodeSessions)
		if err != nil {
			return ExecutionResult{}, err
		}
		for _, envelope := range sealed {
			size := len(envelope.RuntimeArtifact())
			if size > MaxRunRetainedValueBytes-retainedBytes {
				return ExecutionResult{}, errors.New("run retained value budget exceeded")
			}
			retainedBytes += size
		}
		owned = append(owned, leases...)
		result.NodeOutputs[node.ID] = sealed
	}
	if err := owner.Wait(ctx); err != nil {
		return ExecutionResult{}, err
	}
	if err := cleanup(); err != nil {
		return ExecutionResult{}, err
	}
	removeRuntimeOutputs(result.NodeOutputs)
	completed = true
	return result, nil
}

func removeRuntimeOutputs(nodes map[string]map[string]datatype.ValueEnvelope) {
	for nodeID, outputs := range nodes {
		for portID, envelope := range outputs {
			if !envelope.Durable() {
				delete(outputs, portID)
			}
		}
		if len(outputs) == 0 {
			delete(nodes, nodeID)
		}
	}
}

func (e *Executor) resolveInput(result ExecutionResult, input inputPlan) (datatype.ValueEnvelope, string, string, error) {
	switch input.Kind {
	case inputLiteral:
		envelope, err := datatype.OpenValueEnvelope(e.catalog, input.Value)
		return envelope, "", "", err
	case inputEdge:
		envelope, ok := result.NodeOutputs[input.From.NodeID][input.From.PortID]
		if !ok {
			return datatype.ValueEnvelope{}, "", "", errors.New("upstream value is unavailable")
		}
		return envelope, input.From.NodeID, input.From.PortID, nil
	default:
		return datatype.ValueEnvelope{}, "", "", errors.New("unknown input plan")
	}
}

func (e *Executor) validateOutputs(node programNode, outputs map[string]datatype.ValueEnvelope, sessions map[string]*run31.Session) (map[string]datatype.ValueEnvelope, []ownedLease, error) {
	ports := make(map[string]nodecontract.DataOutputPort, len(node.Ports.DataOutputs))
	for _, port := range node.Ports.DataOutputs {
		ports[port.ID] = port
	}
	if len(outputs) != len(ports) {
		return nil, nil, errors.New("adapter output count does not match Node Contract")
	}
	sealed := make(map[string]datatype.ValueEnvelope, len(outputs))
	leases := make([]ownedLease, 0)
	for portID, envelope := range outputs {
		port, ok := ports[portID]
		if !ok || !envelope.Valid() {
			return nil, nil, fmt.Errorf("adapter returned undeclared or invalid output %q", portID)
		}
		expected, err := resolvedTypeForExactRef(port.Type, e.catalog)
		if err != nil || !reflect.DeepEqual(envelope.Type(), expected) {
			return nil, nil, fmt.Errorf("adapter output %q violates its pinned type", portID)
		}
		if handle, _, runtime := runtimeHandle(envelope); runtime {
			if port.ResourceLease == nil || sessions[port.ResourceLease.RequirementID] == nil {
				return nil, nil, fmt.Errorf("adapter output %q has unbound runtime authority", portID)
			}
			leases = append(leases, ownedLease{session: sessions[port.ResourceLease.RequirementID], handle: handle})
		} else if port.ResourceLease != nil {
			return nil, nil, fmt.Errorf("adapter output %q omitted declared runtime authority", portID)
		}
		sealed[portID] = envelope
	}
	return sealed, leases, nil
}

func runtimeHandle(envelope datatype.ValueEnvelope) (resource.Handle, datatype.RepresentationKind, bool) {
	if handle, ok := envelope.StreamRef(); ok {
		return handle, datatype.RepresentationStreamRef, true
	}
	if handle, ok := envelope.HandleRef(); ok {
		return handle, datatype.RepresentationHandleRef, true
	}
	return resource.Handle{}, "", false
}

func dataInputPort(ports nodecontract.PortSet, id string) (nodecontract.DataInputPort, bool) {
	for _, port := range ports.DataInputs {
		if port.ID == id {
			return port, true
		}
	}
	return nodecontract.DataInputPort{}, false
}

func dataOutputPort(ports nodecontract.PortSet, id string) (nodecontract.DataOutputPort, bool) {
	for _, port := range ports.DataOutputs {
		if port.ID == id {
			return port, true
		}
	}
	return nodecontract.DataOutputPort{}, false
}

func cloneConfig(source map[string]any) (map[string]any, error) {
	raw, err := json.Marshal(source)
	if err != nil {
		return nil, err
	}
	var result map[string]any
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.UseNumber()
	if err := decoder.Decode(&result); err != nil {
		return nil, err
	}
	return result, nil
}
