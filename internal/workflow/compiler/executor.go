package compiler

import (
	"bytes"
	"context"
	cryptorand "crypto/rand"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"reflect"
	"sync"
	"time"

	"github.com/yottaapp/yotta/internal/artifact"
	"github.com/yottaapp/yotta/internal/datatype"
	"github.com/yottaapp/yotta/internal/nodecatalog"
	"github.com/yottaapp/yotta/internal/nodecontract"
	"github.com/yottaapp/yotta/internal/resource"
	run31 "github.com/yottaapp/yotta/internal/run"
	"github.com/yottaapp/yotta/internal/workflow/schema"
)

type Adapter func(context.Context, Invocation) (AdapterResult, error)

type AdapterResult struct {
	Outputs     map[string]datatype.ValueEnvelope
	ExecOutputs []string
}

type NodeFailure struct {
	Code   string
	Output string
	Cause  error
}

func (f *NodeFailure) Error() string {
	if f == nil {
		return "node failure"
	}
	if f.Cause != nil {
		return f.Cause.Error()
	}
	return f.Code
}

func (f *NodeFailure) Unwrap() error {
	if f == nil {
		return nil
	}
	return f.Cause
}

type RoutedFailure struct {
	Code         string
	Category     string
	RetryHint    bool
	SourceNodeID string
	SourcePortID string
	Attempt      int
}

type SignalTrigger struct {
	Channel   schema.EdgeChannel
	InputPort string
	From      schema.Endpoint
	Failure   *RoutedFailure
}

type InstalledAdapter struct {
	Implementation nodecatalog.ImplementationLock
	Run            Adapter
}

type Invocation struct {
	GraphID      string
	NodeID       string
	Config       map[string]any
	Inputs       map[string]datatype.ValueEnvelope
	InputTypes   map[string]datatype.ResolvedType
	OutputTypes  map[string]datatype.ResolvedType
	Sessions     map[string]*run31.Session
	Trigger      *SignalTrigger
	ObservedAt   time.Time
	ReadEntropy  func([]byte) error
	Spawn        func(func(context.Context) error) error
	RecordAction func(context.Context, AdapterAction) error
	EmitStatus   func(context.Context, string, map[string]int64) error
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
	attempts    map[string]map[string]int
}

type Executor struct {
	catalog   nodecatalog.Snapshot
	adapters  map[string]InstalledAdapter
	now       func() time.Time
	entropy   io.Reader
	entropyMu sync.Mutex
}

type ExecutorOptions struct {
	Now     func() time.Time
	Entropy io.Reader
}

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
	if options.Entropy == nil {
		options.Entropy = cryptorand.Reader
	}
	return &Executor{catalog: catalog, adapters: installed, now: options.Now, entropy: options.Entropy}
}

func (e *Executor) readEntropy(target []byte) error {
	if len(target) == 0 {
		return nil
	}
	e.entropyMu.Lock()
	defer e.entropyMu.Unlock()
	_, err := io.ReadFull(e.entropy, target)
	return err
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
			attempt := result.attempts[nodeID][portID]
			if attempt < 1 {
				return ExecutionResult{}, errors.New("executor result is missing output attempt provenance")
			}
			valueID, err := runValueID(current.Admission().RunID, graphID, nodeID, portID, attempt)
			if err != nil {
				return ExecutionResult{}, fmt.Errorf("derive Run Value identity: %w", err)
			}
			produced = append(produced, run31.ProducedValue{
				ValueID: valueID, GraphID: graphID,
				NodeID: nodeID, PortID: portID, Attempt: attempt, Envelope: envelope,
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
	scheduler := newScheduler(e, graph, owner, journal)
	return scheduler.run(ctx)
}

type adapterActionRecorder struct {
	mu             sync.Mutex
	executor       *Executor
	journal        *run31.JournalWriter
	graphID        string
	nodeID         string
	attempt        int
	expected       map[string]struct{}
	declaredErrors map[string]struct{}
	recorded       map[string]struct{}
	violation      error
	outcomeErr     error
	failureCode    string
	closed         bool
}

func newAdapterActionRecorder(executor *Executor, journal *run31.JournalWriter, graphID, nodeID string, attempt int, machine nodecontract.MachineContract) *adapterActionRecorder {
	expected := make(map[string]struct{}, len(machine.Execution.Effects))
	for _, effect := range machine.Execution.Effects {
		expected[string(effect)] = struct{}{}
	}
	declaredErrors := make(map[string]struct{}, len(machine.Errors))
	for _, declaration := range machine.Errors {
		declaredErrors[declaration.Code] = struct{}{}
	}
	return &adapterActionRecorder{
		executor: executor, journal: journal, graphID: graphID, nodeID: nodeID, attempt: attempt,
		expected: expected, declaredErrors: declaredErrors, recorded: map[string]struct{}{},
	}
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
	if action.Outcome == run31.ActionFailed {
		if _, declared := r.declaredErrors[action.ErrorCode]; !declared {
			return reject(errors.New("adapter action used an undeclared node error code"))
		}
	}
	summary, err := run31.NewRedactedSummary(action.SummaryCode, action.Counters)
	if err != nil {
		return reject(err)
	}
	fact, err := run31.NewAdapterActionFact(run31.AdapterActionInput{
		GraphPath: []string{r.graphID}, NodeID: r.nodeID, EffectID: action.EffectID, Attempt: r.attempt,
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
		if r.failureCode == "" {
			r.failureCode = action.ErrorCode
		}
	case run31.ActionCancelled:
		if r.outcomeErr == nil {
			r.outcomeErr = context.Canceled
		}
	}
	return nil
}

func (r *adapterActionRecorder) FailureCode() string {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.failureCode
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

func (e *Executor) failAttempt(ctx context.Context, journal *run31.JournalWriter, graphID, nodeID string, attempt int, code string, summary run31.RedactedSummary) error {
	fact, err := run31.NewNodeAttemptFact(run31.NodeAttemptInput{
		GraphPath: []string{graphID}, NodeID: nodeID, Attempt: attempt, Outcome: run31.AttemptFailed,
		OccurredAt: e.now().UTC(), ErrorCode: code, Summary: summary,
	})
	if err != nil {
		return err
	}
	_, err = journal.Append(ctx, fact)
	return err
}

func (e *Executor) cancelAttempt(ctx context.Context, journal *run31.JournalWriter, graphID, nodeID string, attempt int, summary run31.RedactedSummary) error {
	fact, err := run31.NewNodeAttemptFact(run31.NodeAttemptInput{
		GraphPath: []string{graphID}, NodeID: nodeID, Attempt: attempt, Outcome: run31.AttemptCancelled,
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
	if len(outputs) != len(node.Ports.DataOutputs) {
		return nil, nil, errors.New("adapter output count does not match Node Contract")
	}
	sealed := make(map[string]datatype.ValueEnvelope, len(outputs))
	leases := make([]ownedLease, 0)
	for _, port := range node.Ports.DataOutputs {
		envelope, ok := outputs[port.ID]
		if !ok || !envelope.Valid() {
			return nil, nil, fmt.Errorf("adapter omitted or returned an invalid output %q", port.ID)
		}
		expected, ok := node.OutputTypes[port.ID]
		if !ok || !reflect.DeepEqual(envelope.Type(), expected) {
			return nil, nil, fmt.Errorf("adapter output %q violates its pinned type", port.ID)
		}
		if handle, _, runtime := runtimeHandle(envelope); runtime {
			if port.ResourceLease == nil || sessions[port.ResourceLease.RequirementID] == nil {
				return nil, nil, fmt.Errorf("adapter output %q has unbound runtime authority", port.ID)
			}
			leases = append(leases, ownedLease{session: sessions[port.ResourceLease.RequirementID], handle: handle})
		} else if port.ResourceLease != nil {
			return nil, nil, fmt.Errorf("adapter output %q omitted declared runtime authority", port.ID)
		}
		sealed[port.ID] = envelope
	}
	return sealed, leases, nil
}

func cloneResolvedTypes(source map[string]datatype.ResolvedType) map[string]datatype.ResolvedType {
	result := make(map[string]datatype.ResolvedType, len(source))
	for portID, resolved := range source {
		result[portID] = resolved
	}
	return result
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
