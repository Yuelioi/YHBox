package compiler

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"reflect"
	"sync"
	"time"

	"github.com/yottaapp/yotta/internal/artifact"
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
	GraphID      string
	NodeID       string
	Config       map[string]any
	Inputs       map[string]datatype.ValueEnvelope
	Sessions     map[string]*run31.Session
	Spawn        func(func(context.Context) error) error
	RecordAction func(context.Context, AdapterAction) error
}

type AdapterAction struct {
	EffectID    string
	Action      string
	Outcome     run31.ActionOutcome
	ErrorCode   string
	SummaryCode string
	Counters    map[string]int64
}

var errAdapterActionFailed = errors.New("adapter recorded a failed action")

type ExecutionResult struct {
	// NodeOutputs contains only durable envelopes. Runtime authority remains an
	// executor-local edge value and is reclaimed before Run returns.
	NodeOutputs map[string]map[string]datatype.ValueEnvelope
}

type Executor struct {
	catalog  nodecatalog.Snapshot
	adapters map[string]InstalledAdapter
	now      func() time.Time
}

type ExecutorOptions struct{ Now func() time.Time }

type ownedLease struct {
	session *run31.Session
	handle  resource.Handle
}

func NewExecutor(catalog nodecatalog.Snapshot, adapters map[string]InstalledAdapter, options ExecutorOptions) *Executor {
	installed := make(map[string]InstalledAdapter, len(adapters))
	for entrypoint, adapter := range adapters {
		installed[entrypoint] = adapter
	}
	if options.Now == nil {
		options.Now = time.Now
	}
	return &Executor{catalog: catalog, adapters: installed, now: options.Now}
}

func (e *Executor) Run(ctx context.Context, program ProgramSnapshot, owner *run31.Owner, journal *run31.JournalWriter) (ExecutionResult, error) {
	if ctx == nil || !program.Valid() || !e.catalog.Valid() || owner == nil || journal == nil {
		return ExecutionResult{}, errors.New("executor requires Program, Catalog, Run Owner, and journal")
	}
	current := journal.Current()
	if current.Status() != run31.StatusRunning || current.Admission().CatalogHash != e.catalog.Hash() ||
		current.Admission().ProgramHash != program.Hash() || current.Admission().CapabilityPlanDigest != program.CapabilityPlan().Digest() {
		return ExecutionResult{}, errors.New("executor journal does not match Program and Catalog")
	}
	if err := owner.ValidateAdmission(current.Admission()); err != nil {
		return ExecutionResult{}, err
	}
	result, executionErr := e.execute(ctx, program, owner, journal)
	finishedAt := e.now().UTC()
	if executionErr != nil {
		if errors.Is(executionErr, context.Canceled) || errors.Is(executionErr, context.DeadlineExceeded) {
			_, terminalErr := journal.Cancel(context.WithoutCancel(ctx), finishedAt)
			return ExecutionResult{}, errors.Join(executionErr, terminalErr)
		}
		_, terminalErr := journal.Fail(context.WithoutCancel(ctx), finishedAt, runErrorForExecution(executionErr, journal.Current().Journal()))
		return ExecutionResult{}, errors.Join(executionErr, terminalErr)
	}
	produced := make([]run31.ProducedValue, 0)
	graphID := program.state.document.Body.EntryGraph
	for nodeID, outputs := range result.NodeOutputs {
		for portID, envelope := range outputs {
			valueID, err := runValueID(current.Admission().RunID, graphID, nodeID, portID, 1)
			if err != nil {
				return ExecutionResult{}, fmt.Errorf("derive Run Value identity: %w", err)
			}
			produced = append(produced, run31.ProducedValue{
				ValueID: valueID, GraphID: graphID,
				NodeID: nodeID, PortID: portID, Attempt: 1, Envelope: envelope,
			})
		}
	}
	if _, err := journal.Succeed(context.WithoutCancel(ctx), finishedAt, e.catalog, produced); err != nil {
		return ExecutionResult{}, fmt.Errorf("persist successful Run: %w", err)
	}
	return result, nil
}

func runValueID(runID, graphID, nodeID, portID string, attempt int) (string, error) {
	identity, err := artifact.Marshal(struct {
		RunID   string `json:"runId"`
		GraphID string `json:"graphId"`
		NodeID  string `json:"nodeId"`
		PortID  string `json:"portId"`
		Attempt int    `json:"attempt"`
	}{RunID: runID, GraphID: graphID, NodeID: nodeID, PortID: portID, Attempt: attempt})
	if err != nil {
		return "", err
	}
	digest, err := artifact.Sum("yotta/run-value-id/v1", identity)
	return string(digest), err
}

func runErrorForExecution(executionErr error, journal []run31.JournalEntry) run31.RunError {
	if errors.Is(executionErr, run31.ErrGrantDenied) {
		return run31.RunError{Code: "policy.grant_denied", Category: run31.ErrorCategoryPolicy}
	}
	for index := len(journal) - 1; index >= 0; index-- {
		entry := journal[index]
		if entry.Kind == run31.JournalNodeAttempt && entry.AttemptOutcome == run31.AttemptFailed {
			category := run31.ErrorCategoryNode
			for actionIndex := index - 1; actionIndex >= 0; actionIndex-- {
				action := journal[actionIndex]
				if action.Kind == run31.JournalNodeAttempt && action.NodeID == entry.NodeID && action.Attempt == entry.Attempt {
					break
				}
				if action.Kind == run31.JournalAdapterAction && action.NodeID == entry.NodeID && action.Attempt == entry.Attempt && action.ActionOutcome == run31.ActionFailed {
					category = run31.ErrorCategoryAdapter
					break
				}
			}
			return run31.RunError{
				Code: entry.ErrorCode, Category: category, GraphID: entry.GraphPath[len(entry.GraphPath)-1],
				NodeID: entry.NodeID, Attempt: entry.Attempt,
			}
		}
	}
	return run31.RunError{Code: "runtime.execution_failed", Category: run31.ErrorCategoryInfrastructure}
}

func (e *Executor) execute(ctx context.Context, program ProgramSnapshot, owner *run31.Owner, journal *run31.JournalWriter) (ExecutionResult, error) {
	if ctx == nil || !program.Valid() || !e.catalog.Valid() || owner == nil || journal == nil {
		return ExecutionResult{}, errors.New("executor requires Program, Catalog, Run Owner, and journal")
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
		machine := entry.Contract.Machine()
		installed, ok := e.adapters[node.Implementation.Entrypoint]
		if !ok || installed.Run == nil || installed.Implementation != node.Implementation {
			return ExecutionResult{}, fmt.Errorf("adapter %q does not match the Program lock", node.Implementation.Entrypoint)
		}
		invocationID, err := runid.New()
		if err != nil {
			return ExecutionResult{}, err
		}
		nodeSessions := make(map[string]*run31.Session, len(machine.CapabilityRequirements))
		for _, requirement := range machine.CapabilityRequirements {
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
		summary, err := run31.NewRedactedSummary("node.execute", nil)
		if err != nil {
			return ExecutionResult{}, err
		}
		started, err := run31.NewNodeAttemptFact(run31.NodeAttemptInput{
			GraphPath: []string{graph.ID}, NodeID: node.ID, Attempt: 1, Outcome: run31.AttemptStarted,
			OccurredAt: e.now().UTC(), Summary: summary,
		})
		if err != nil {
			return ExecutionResult{}, err
		}
		if _, err := journal.Append(ctx, started); err != nil {
			return ExecutionResult{}, fmt.Errorf("journal node %q start: %w", node.ID, err)
		}
		recorder := newAdapterActionRecorder(e, journal, graph.ID, node.ID, machine)
		outputs, runErr := installed.Run(ctx, Invocation{
			GraphID: graph.ID, NodeID: node.ID, Config: config, Inputs: inputs,
			Sessions: nodeSessions, Spawn: owner.Go, RecordAction: recorder.Record,
		})
		actionErr := recorder.Close()
		if runErr != nil {
			if errors.Is(runErr, context.Canceled) || errors.Is(runErr, context.DeadlineExceeded) || ctx.Err() != nil {
				journalErr := e.cancelAttempt(context.WithoutCancel(ctx), journal, graph.ID, node.ID, summary)
				return ExecutionResult{}, errors.Join(fmt.Errorf("run node %q: %w", node.ID, runErr), actionErr, journalErr)
			}
			code := declaredErrorCode(machine, "runtime.adapter_failed")
			journalErr := e.failAttempt(context.WithoutCancel(ctx), journal, graph.ID, node.ID, code, summary)
			return ExecutionResult{}, errors.Join(fmt.Errorf("run node %q: %w", node.ID, runErr), actionErr, journalErr)
		}
		if actionErr != nil {
			if errors.Is(actionErr, context.Canceled) || errors.Is(actionErr, context.DeadlineExceeded) {
				journalErr := e.cancelAttempt(context.WithoutCancel(ctx), journal, graph.ID, node.ID, summary)
				return ExecutionResult{}, errors.Join(actionErr, journalErr)
			}
			code := "runtime.journal_failed"
			if errors.Is(actionErr, errAdapterActionFailed) {
				code = declaredErrorCode(machine, "runtime.adapter_failed")
			}
			journalErr := e.failAttempt(context.WithoutCancel(ctx), journal, graph.ID, node.ID, code, summary)
			return ExecutionResult{}, errors.Join(actionErr, journalErr)
		}
		sealed, leases, err := e.validateOutputs(node, outputs, nodeSessions)
		if err != nil {
			journalErr := e.failAttempt(context.WithoutCancel(ctx), journal, graph.ID, node.ID, "runtime.output_invalid", summary)
			return ExecutionResult{}, errors.Join(err, journalErr)
		}
		for _, envelope := range sealed {
			size := len(envelope.RuntimeArtifact())
			if size > MaxRunRetainedValueBytes-retainedBytes {
				journalErr := e.failAttempt(context.WithoutCancel(ctx), journal, graph.ID, node.ID, "runtime.value_budget_exceeded", summary)
				return ExecutionResult{}, errors.Join(errors.New("run retained value budget exceeded"), journalErr)
			}
			retainedBytes += size
		}
		owned = append(owned, leases...)
		result.NodeOutputs[node.ID] = sealed
		finished, err := run31.NewNodeAttemptFact(run31.NodeAttemptInput{
			GraphPath: []string{graph.ID}, NodeID: node.ID, Attempt: 1, Outcome: run31.AttemptSucceeded,
			OccurredAt: e.now().UTC(), Summary: summary,
		})
		if err != nil {
			return ExecutionResult{}, err
		}
		if _, err := journal.Append(context.WithoutCancel(ctx), finished); err != nil {
			return ExecutionResult{}, fmt.Errorf("journal node %q success: %w", node.ID, err)
		}
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

func declaredErrorCode(machine nodecontract.MachineContract, fallback string) string {
	if len(machine.Errors) > 0 {
		return machine.Errors[0].Code
	}
	return fallback
}

type adapterActionRecorder struct {
	mu         sync.Mutex
	executor   *Executor
	journal    *run31.JournalWriter
	graphID    string
	nodeID     string
	expected   map[string]struct{}
	recorded   map[string]struct{}
	violation  error
	outcomeErr error
	closed     bool
}

func newAdapterActionRecorder(executor *Executor, journal *run31.JournalWriter, graphID, nodeID string, machine nodecontract.MachineContract) *adapterActionRecorder {
	expected := make(map[string]struct{}, len(machine.Execution.Effects))
	for _, effect := range machine.Execution.Effects {
		expected[string(effect)] = struct{}{}
	}
	return &adapterActionRecorder{executor: executor, journal: journal, graphID: graphID, nodeID: nodeID, expected: expected, recorded: map[string]struct{}{}}
}

func (r *adapterActionRecorder) Record(ctx context.Context, action AdapterAction) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	reject := func(err error) error {
		if r.violation == nil {
			r.violation = err
		}
		return err
	}
	if ctx == nil {
		return reject(errors.New("adapter action context is required"))
	}
	if r.closed {
		return errors.New("adapter action was recorded after invocation returned")
	}
	if _, expected := r.expected[action.EffectID]; !expected {
		return reject(errors.New("adapter action does not match a declared effect"))
	}
	if _, duplicate := r.recorded[action.EffectID]; duplicate {
		return reject(errors.New("adapter recorded a declared effect more than once"))
	}
	summary, err := run31.NewRedactedSummary(action.SummaryCode, action.Counters)
	if err != nil {
		return reject(err)
	}
	fact, err := run31.NewAdapterActionFact(run31.AdapterActionInput{
		GraphPath: []string{r.graphID}, NodeID: r.nodeID, EffectID: action.EffectID, Attempt: 1,
		Action: action.Action, Outcome: action.Outcome, OccurredAt: r.executor.now().UTC(), ErrorCode: action.ErrorCode, Summary: summary,
	})
	if err != nil {
		return reject(err)
	}
	if _, err := r.journal.Append(ctx, fact); err != nil {
		return reject(err)
	}
	r.recorded[action.EffectID] = struct{}{}
	switch action.Outcome {
	case run31.ActionFailed:
		r.outcomeErr = errAdapterActionFailed
	case run31.ActionCancelled:
		if r.outcomeErr == nil {
			r.outcomeErr = context.Canceled
		}
	}
	return nil
}

func (r *adapterActionRecorder) Close() error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.closed = true
	if r.violation != nil {
		return r.violation
	}
	for effectID := range r.expected {
		if _, recorded := r.recorded[effectID]; !recorded {
			return errors.New("adapter did not record every declared effect")
		}
	}
	return r.outcomeErr
}

func (e *Executor) failAttempt(ctx context.Context, journal *run31.JournalWriter, graphID, nodeID, code string, summary run31.RedactedSummary) error {
	fact, err := run31.NewNodeAttemptFact(run31.NodeAttemptInput{
		GraphPath: []string{graphID}, NodeID: nodeID, Attempt: 1, Outcome: run31.AttemptFailed,
		OccurredAt: e.now().UTC(), ErrorCode: code, Summary: summary,
	})
	if err != nil {
		return err
	}
	_, err = journal.Append(ctx, fact)
	return err
}

func (e *Executor) cancelAttempt(ctx context.Context, journal *run31.JournalWriter, graphID, nodeID string, summary run31.RedactedSummary) error {
	fact, err := run31.NewNodeAttemptFact(run31.NodeAttemptInput{
		GraphPath: []string{graphID}, NodeID: nodeID, Attempt: 1, Outcome: run31.AttemptCancelled,
		OccurredAt: e.now().UTC(), Summary: summary,
	})
	if err != nil {
		return err
	}
	_, err = journal.Append(ctx, fact)
	return err
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
