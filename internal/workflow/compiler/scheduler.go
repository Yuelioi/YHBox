package compiler

import (
	"context"
	"errors"
	"fmt"
	"reflect"
	"sort"
	"sync"

	"github.com/yottaapp/yotta/internal/datatype"
	"github.com/yottaapp/yotta/internal/nodecontract"
	"github.com/yottaapp/yotta/internal/resource"
	run31 "github.com/yottaapp/yotta/internal/run"
	"github.com/yottaapp/yotta/internal/runid"
	"github.com/yottaapp/yotta/internal/workflow/schema"
)

const MaxScheduledInvocations = 16_384

type routeKey struct {
	channel schema.EdgeChannel
	nodeID  string
	portID  string
}

type scheduledInvocation struct {
	nodeID  string
	trigger *SignalTrigger
}

type scheduler struct {
	executor       *Executor
	graph          *programGraph
	owner          *run31.Owner
	journal        *run31.JournalWriter
	state          *runState
	nodes          map[string]programNode
	routes         map[routeKey][]programSignalRoute
	dataConsumers  map[string]int
	volatile       map[string]bool
	queue          []scheduledInvocation
	result         ExecutionResult
	attempts       map[string]int
	outputSessions map[string]map[string]*run31.Session
	evaluating     map[string]bool
	owned          []ownedLease
	retainedBytes  int
	invocations    int
}

func newScheduler(executor *Executor, graph *programGraph, owner *run31.Owner, journal *run31.JournalWriter, state *runState) *scheduler {
	s := &scheduler{
		executor: executor, graph: graph, owner: owner, journal: journal, state: state,
		nodes: make(map[string]programNode, len(graph.Nodes)), routes: make(map[routeKey][]programSignalRoute),
		dataConsumers: make(map[string]int), volatile: make(map[string]bool), attempts: make(map[string]int),
		outputSessions: make(map[string]map[string]*run31.Session), evaluating: make(map[string]bool),
		result: ExecutionResult{
			NodeOutputs: make(map[string]map[string]datatype.ValueEnvelope),
			attempts:    make(map[string]map[string]int),
		},
	}
	for _, node := range graph.Nodes {
		s.nodes[node.ID] = node
		for _, input := range node.Inputs {
			if input.Kind == inputEdge {
				s.dataConsumers[input.From.NodeID]++
			}
		}
	}
	for _, route := range graph.SignalRoutes {
		key := routeKey{channel: route.Channel, nodeID: route.From.NodeID, portID: route.From.PortID}
		s.routes[key] = append(s.routes[key], route)
	}
	for _, nodeID := range graph.DataOrder {
		node := s.nodes[nodeID]
		volatile := node.Execution.Cache == nodecontract.CacheNone
		for _, input := range node.Inputs {
			if input.Kind == inputEdge && s.volatile[input.From.NodeID] {
				volatile = true
			}
		}
		s.volatile[nodeID] = volatile
	}
	return s
}

func (s *scheduler) run(ctx context.Context) (ExecutionResult, error) {
	completed := false
	defer func() {
		if !completed {
			_ = s.cleanup()
		}
	}()
	for _, nodeID := range executionRoots(*s.graph) {
		s.queue = append(s.queue, scheduledInvocation{nodeID: nodeID})
	}
	if len(s.nodes) != 0 && len(s.queue) == 0 {
		return ExecutionResult{}, errors.New("Program has no event or pull-data entry")
	}
	for len(s.queue) != 0 {
		if err := ctx.Err(); err != nil {
			return ExecutionResult{}, err
		}
		next := s.queue[0]
		s.queue = s.queue[1:]
		if err := s.dispatch(ctx, next.nodeID, next.trigger, map[string]bool{}); err != nil {
			return ExecutionResult{}, err
		}
	}
	if err := s.owner.Wait(ctx); err != nil {
		return ExecutionResult{}, err
	}
	if err := s.cleanup(); err != nil {
		return ExecutionResult{}, err
	}
	removeRuntimeOutputs(s.result.NodeOutputs)
	completed = true
	return s.result, nil
}

func executionRoots(graph programGraph) []string {
	consumers := make(map[string]int, len(graph.Nodes))
	nodes := make(map[string]programNode, len(graph.Nodes))
	for _, node := range graph.Nodes {
		nodes[node.ID] = node
		for _, input := range node.Inputs {
			if input.Kind == inputEdge {
				consumers[input.From.NodeID]++
			}
		}
	}
	result := make([]string, 0)
	for _, nodeID := range graph.DataOrder {
		node := nodes[nodeID]
		if node.Execution.Class == nodecontract.ExecutionEvent && len(node.Ports.ExecInputs) == 0 ||
			node.Execution.Evaluation == nodecontract.EvaluationPull && len(node.Ports.ExecInputs) == 0 && consumers[nodeID] == 0 {
			result = append(result, nodeID)
		}
	}
	return result
}

func executionReachability(graph programGraph) ([]string, map[string]bool) {
	roots := executionRoots(graph)
	reachable := make(map[string]bool, len(graph.Nodes))
	queue := append([]string(nil), roots...)
	for len(queue) != 0 {
		nodeID := queue[0]
		queue = queue[1:]
		if reachable[nodeID] {
			continue
		}
		reachable[nodeID] = true
		for _, route := range graph.SignalRoutes {
			if route.From.NodeID == nodeID && !reachable[route.To.NodeID] {
				queue = append(queue, route.To.NodeID)
			}
		}
	}
	return roots, reachable
}

func (s *scheduler) invoke(ctx context.Context, nodeID string, trigger *SignalTrigger, evaluation map[string]bool) error {
	node, ok := s.nodes[nodeID]
	if !ok {
		return fmt.Errorf("scheduled node %q is missing from Program", nodeID)
	}
	entry, ok := s.executor.catalog.Lookup(node.NodeRef.NodeTypeID)
	if !ok {
		return fmt.Errorf("node type %q is not installed", node.NodeRef.NodeTypeID)
	}
	machine := entry.Contract.Machine()
	installed, ok := s.executor.adapters[node.Implementation.Entrypoint]
	if !ok || installed.Run == nil || installed.Implementation != node.Implementation {
		return fmt.Errorf("adapter %q does not match the Program lock", node.Implementation.Entrypoint)
	}
	s.attempts[nodeID]++
	attempt := s.attempts[nodeID]
	invocationID, err := runid.New()
	if err != nil {
		return err
	}
	nodeSessions := make(map[string]*run31.Session, len(node.Capabilities))
	for _, requirement := range node.Capabilities {
		session, err := s.owner.Session(s.graph.ID, node.ID, requirement.ID, invocationID)
		if err != nil {
			return err
		}
		nodeSessions[requirement.ID] = session
	}
	inputs, err := s.resolveInputs(ctx, node, nodeSessions, evaluation)
	if err != nil {
		return fmt.Errorf("resolve inputs for %s: %w", node.ID, err)
	}
	config, err := cloneConfig(node.Config)
	if err != nil {
		return fmt.Errorf("clone config for node %q: %w", node.ID, err)
	}
	stateBindings, err := s.state.bindings(machine, config)
	if err != nil {
		return fmt.Errorf("bind state for node %q: %w", node.ID, err)
	}
	summary, err := run31.NewRedactedSummary("node.execute", nil)
	if err != nil {
		return err
	}
	observedAt := s.executor.now().UTC()
	started, err := run31.NewNodeAttemptFact(run31.NodeAttemptInput{
		GraphPath: []string{s.graph.ID}, NodeID: node.ID, Attempt: attempt, Outcome: run31.AttemptStarted,
		OccurredAt: observedAt, Summary: summary,
	})
	if err != nil {
		return err
	}
	if _, err := s.journal.Append(ctx, started); err != nil {
		return fmt.Errorf("journal node %q start: %w", node.ID, err)
	}
	s.clearNodeResult(node.ID)
	actions := newAdapterActionRecorder(s.executor, s.journal, s.graph.ID, node.ID, attempt, machine)
	statuses := newStatusEmitter(s.executor, s.journal, s.graph.ID, node.ID, attempt, machine.StatusEvents)
	outcome, runErr := installed.Run(ctx, Invocation{
		GraphID: s.graph.ID, NodeID: node.ID, Config: config, Inputs: inputs,
		InputTypes: cloneResolvedTypes(node.InputTypes), OutputTypes: cloneResolvedTypes(node.OutputTypes), Sessions: nodeSessions, State: stateBindings,
		Trigger: cloneTrigger(trigger), ObservedAt: observedAt, ReadEntropy: s.executor.readEntropy,
		Wait: s.executor.wait, Spawn: s.owner.Go, RecordAction: actions.Record, EmitStatus: statuses.Emit,
	})
	actionErr := actions.Close()
	statusErr := statuses.Close()
	if runErr != nil && (errors.Is(runErr, context.Canceled) || errors.Is(runErr, context.DeadlineExceeded) || ctx.Err() != nil) ||
		errors.Is(actionErr, context.Canceled) || errors.Is(statusErr, context.Canceled) {
		journalErr := s.executor.cancelAttempt(context.WithoutCancel(ctx), s.journal, s.graph.ID, node.ID, attempt, summary)
		return errors.Join(wrapNodeRunError(node.ID, runErr), actionErr, statusErr, journalErr)
	}
	var failure *NodeFailure
	if errors.As(runErr, &failure) && onlyNodeFailure(runErr, failure) {
		return s.routeFailure(ctx, node, machine, attempt, outcome, failure, actions, actionErr, statusErr, summary)
	}
	if runErr != nil || actionErr != nil || statusErr != nil {
		code := "runtime.adapter_failed"
		if statusErr != nil {
			code = "runtime.status_invalid"
		} else if actionErr != nil && !errors.Is(actionErr, errAdapterActionFailed) {
			code = "runtime.journal_failed"
		} else if actions.FailureCode() != "" {
			code = actions.FailureCode()
		}
		journalErr := s.executor.failAttempt(context.WithoutCancel(ctx), s.journal, s.graph.ID, node.ID, attempt, code, summary)
		return errors.Join(wrapNodeRunError(node.ID, runErr), actionErr, statusErr, journalErr)
	}
	selected, err := validateExecSelection(node.Ports.ExecOutputs, outcome.ExecOutputs)
	if err != nil {
		journalErr := s.executor.failAttempt(context.WithoutCancel(ctx), s.journal, s.graph.ID, node.ID, attempt, "runtime.signal_invalid", summary)
		return errors.Join(err, journalErr)
	}
	sealed, leases, err := s.executor.validateOutputs(node, outcome.Outputs, nodeSessions)
	if err != nil {
		journalErr := s.executor.failAttempt(context.WithoutCancel(ctx), s.journal, s.graph.ID, node.ID, attempt, "runtime.output_invalid", summary)
		return errors.Join(err, journalErr)
	}
	nextBytes := 0
	for _, port := range node.Ports.DataOutputs {
		nextBytes += len(sealed[port.ID].RuntimeArtifact())
	}
	if nextBytes > MaxRunRetainedValueBytes-s.retainedBytes {
		journalErr := s.executor.failAttempt(context.WithoutCancel(ctx), s.journal, s.graph.ID, node.ID, attempt, "runtime.value_budget_exceeded", summary)
		return errors.Join(errors.New("run retained value budget exceeded"), journalErr)
	}
	s.retainedBytes += nextBytes
	s.owned = append(s.owned, leases...)
	s.result.NodeOutputs[node.ID] = sealed
	s.result.attempts[node.ID] = make(map[string]int, len(sealed))
	for _, port := range node.Ports.DataOutputs {
		envelope := sealed[port.ID]
		s.result.attempts[node.ID][port.ID] = attempt
		if _, _, runtimeValue := runtimeHandle(envelope); runtimeValue {
			if s.outputSessions[node.ID] == nil {
				s.outputSessions[node.ID] = make(map[string]*run31.Session)
			}
			s.outputSessions[node.ID][port.ID] = nodeSessions[port.ResourceLease.RequirementID]
		}
	}
	finished, err := run31.NewNodeAttemptFact(run31.NodeAttemptInput{
		GraphPath: []string{s.graph.ID}, NodeID: node.ID, Attempt: attempt, Outcome: run31.AttemptSucceeded,
		OccurredAt: s.executor.now().UTC(), Summary: summary,
	})
	if err != nil {
		return err
	}
	if _, err := s.journal.Append(context.WithoutCancel(ctx), finished); err != nil {
		return fmt.Errorf("journal node %q success: %w", node.ID, err)
	}
	s.enqueueSelected(node.ID, selected, nil)
	return nil
}

func onlyNodeFailure(err error, failure *NodeFailure) bool {
	if err == nil || failure == nil {
		return false
	}
	if err == failure {
		return true
	}
	if joined, ok := err.(interface{ Unwrap() []error }); ok {
		children := joined.Unwrap()
		if len(children) == 0 {
			return false
		}
		for _, child := range children {
			if child != nil && !onlyNodeFailure(child, failure) {
				return false
			}
		}
		return true
	}
	if wrapped := errors.Unwrap(err); wrapped != nil {
		return onlyNodeFailure(wrapped, failure)
	}
	return false
}

func wrapNodeRunError(nodeID string, err error) error {
	if err == nil {
		return nil
	}
	return fmt.Errorf("run node %q: %w", nodeID, err)
}

func (s *scheduler) clearNodeResult(nodeID string) {
	delete(s.result.NodeOutputs, nodeID)
	delete(s.result.attempts, nodeID)
	delete(s.outputSessions, nodeID)
	s.retainedBytes = 0
	for _, outputs := range s.result.NodeOutputs {
		for _, value := range outputs {
			s.retainedBytes += len(value.RuntimeArtifact())
		}
	}
}

func (s *scheduler) resolveInputs(ctx context.Context, node programNode, nodeSessions map[string]*run31.Session, evaluation map[string]bool) (map[string]datatype.ValueEnvelope, error) {
	inputs := make(map[string]datatype.ValueEnvelope, len(node.Inputs))
	portIDs := make([]string, 0, len(node.Inputs))
	for portID := range node.Inputs {
		portIDs = append(portIDs, portID)
	}
	sort.Strings(portIDs)
	for _, portID := range portIDs {
		input := node.Inputs[portID]
		if input.Kind == inputEdge {
			_, available := s.result.NodeOutputs[input.From.NodeID][input.From.PortID]
			source := s.nodes[input.From.NodeID]
			if (!available || s.volatile[input.From.NodeID] && source.Execution.Evaluation == nodecontract.EvaluationPull) && !evaluation[input.From.NodeID] {
				if err := s.evaluatePull(ctx, input.From.NodeID, evaluation); err != nil {
					return nil, err
				}
			}
		}
		envelope, sourceNode, sourcePort, err := s.executor.resolveInput(s.result, input)
		if err != nil {
			return nil, err
		}
		inputPort, exists := dataInputPort(node.Ports, portID)
		if !exists {
			return nil, errors.New("Program input port is missing from its effective contract")
		}
		if handle, runtimeKind, runtimeValue := runtimeHandle(envelope); runtimeValue {
			if inputPort.ResourceLease == nil || sourceNode == "" {
				return nil, errors.New("runtime authority crossed a data edge without an explicit resource lease")
			}
			sourcePlan := s.nodes[sourceNode]
			outputPort, exists := dataOutputPort(sourcePlan.Ports, sourcePort)
			if !exists || outputPort.ResourceLease == nil || !resourceLeaseAssignable(outputPort.ResourceLease, inputPort.ResourceLease) {
				return nil, errors.New("runtime authority edge has an invalid resource lease")
			}
			lender := s.outputSessions[sourceNode][sourcePort]
			borrower := nodeSessions[inputPort.ResourceLease.RequirementID]
			if lender == nil || borrower == nil {
				return nil, errors.New("runtime authority edge has no invocation session")
			}
			borrowedHandle, err := s.owner.Borrow(ctx, lender, handle, borrower, inputPort.ResourceLease.Operations)
			if err != nil {
				return nil, err
			}
			s.owned = append(s.owned, ownedLease{session: borrower, handle: borrowedHandle})
			switch runtimeKind {
			case datatype.RepresentationStreamRef:
				envelope, err = datatype.SealStreamRef(s.executor.catalog, envelope.Type(), borrowedHandle)
			case datatype.RepresentationHandleRef:
				envelope, err = datatype.SealHandleRef(s.executor.catalog, envelope.Type(), borrowedHandle)
			}
			if err != nil {
				return nil, err
			}
		} else if inputPort.ResourceLease != nil {
			return nil, errors.New("resource-leased input received a durable value")
		}
		expected, ok := node.InputTypes[portID]
		if !ok || !reflect.DeepEqual(envelope.Type(), expected) {
			return nil, fmt.Errorf("input %q violates its Program-resolved type", portID)
		}
		inputs[portID] = envelope
	}
	return inputs, nil
}

func (s *scheduler) evaluatePull(ctx context.Context, nodeID string, evaluation map[string]bool) error {
	node, ok := s.nodes[nodeID]
	if !ok {
		return errors.New("data dependency references a missing node")
	}
	if node.Execution.Evaluation != nodecontract.EvaluationPull || len(node.Ports.ExecInputs) != 0 {
		return fmt.Errorf("data dependency %q is unavailable before its signal invocation", nodeID)
	}
	if s.evaluating[nodeID] {
		return errors.New("data dependency recursion escaped Program validation")
	}
	s.evaluating[nodeID] = true
	defer delete(s.evaluating, nodeID)
	evaluation[nodeID] = true
	return s.dispatch(ctx, nodeID, nil, evaluation)
}

func (s *scheduler) routeFailure(ctx context.Context, node programNode, machine nodecontract.MachineContract, attempt int, outcome AdapterResult, failure *NodeFailure, actions *adapterActionRecorder, actionErr, statusErr error, summary run31.RedactedSummary) error {
	if len(outcome.Outputs) != 0 || len(outcome.ExecOutputs) != 0 || statusErr != nil {
		journalErr := s.executor.failAttempt(context.WithoutCancel(ctx), s.journal, s.graph.ID, node.ID, attempt, "runtime.failure_invalid", summary)
		return errors.Join(errors.New("node failure returned outputs, exec signals, or invalid status"), statusErr, journalErr)
	}
	spec, declared := declaredNodeError(machine.Errors, failure.Code)
	if !declared || (failure.Output != "" && !signalPortExists(machine.Ports.ErrorOutputs, failure.Output)) {
		journalErr := s.executor.failAttempt(context.WithoutCancel(ctx), s.journal, s.graph.ID, node.ID, attempt, "runtime.failure_invalid", summary)
		return errors.Join(errors.New("adapter returned an undeclared node failure"), journalErr)
	}
	if actionErr != nil && (!errors.Is(actionErr, errAdapterActionFailed) || actions.FailureCode() != failure.Code) {
		journalErr := s.executor.failAttempt(context.WithoutCancel(ctx), s.journal, s.graph.ID, node.ID, attempt, "runtime.failure_invalid", summary)
		return errors.Join(errors.New("node failure does not match its adapter action"), actionErr, journalErr)
	}
	routes := s.routes[routeKey{channel: schema.EdgeError, nodeID: node.ID, portID: failure.Output}]
	terminal := run31.AttemptFailed
	if len(routes) != 0 {
		terminal = run31.AttemptRouted
	}
	fact, err := run31.NewNodeAttemptFact(run31.NodeAttemptInput{
		GraphPath: []string{s.graph.ID}, NodeID: node.ID, Attempt: attempt, Outcome: terminal,
		OccurredAt: s.executor.now().UTC(), ErrorCode: failure.Code, Summary: summary,
	})
	if err != nil {
		return err
	}
	if _, err := s.journal.Append(context.WithoutCancel(ctx), fact); err != nil {
		return err
	}
	if len(routes) == 0 {
		return failure
	}
	routed := &RoutedFailure{
		Code: failure.Code, Category: spec.Category, RetryHint: spec.RetryHint,
		SourceNodeID: node.ID, SourcePortID: failure.Output, Attempt: attempt,
	}
	for _, route := range routes {
		s.queue = append(s.queue, scheduledInvocation{nodeID: route.To.NodeID, trigger: &SignalTrigger{
			Channel: schema.EdgeError, InputPort: route.To.PortID, From: route.From, Failure: cloneRoutedFailure(routed),
		}})
	}
	return nil
}

func (s *scheduler) enqueueSelected(nodeID string, selected map[string]struct{}, failure *RoutedFailure) {
	for _, route := range s.graph.SignalRoutes {
		if route.Channel != schema.EdgeExec || route.From.NodeID != nodeID {
			continue
		}
		if _, ok := selected[route.From.PortID]; !ok {
			continue
		}
		s.queue = append(s.queue, scheduledInvocation{nodeID: route.To.NodeID, trigger: &SignalTrigger{
			Channel: schema.EdgeExec, InputPort: route.To.PortID, From: route.From, Failure: cloneRoutedFailure(failure),
		}})
	}
}

func (s *scheduler) cleanup() error {
	var cleanupErrors []error
	for index := len(s.owned) - 1; index >= 0; index-- {
		if err := s.owned[index].session.Drop(context.Background(), s.owned[index].handle); err != nil && !errors.Is(err, resource.ErrUnknownHandle) {
			cleanupErrors = append(cleanupErrors, err)
		}
	}
	s.owned = nil
	return errors.Join(cleanupErrors...)
}

func validateExecSelection(ports []nodecontract.SignalPort, selected []string) (map[string]struct{}, error) {
	declared := make(map[string]struct{}, len(ports))
	for _, port := range ports {
		declared[port.ID] = struct{}{}
	}
	result := make(map[string]struct{}, len(selected))
	for _, portID := range selected {
		if _, ok := declared[portID]; !ok {
			return nil, fmt.Errorf("adapter selected undeclared exec output %q", portID)
		}
		if _, duplicate := result[portID]; duplicate {
			return nil, fmt.Errorf("adapter selected exec output %q more than once", portID)
		}
		result[portID] = struct{}{}
	}
	return result, nil
}

func declaredNodeError(values []nodecontract.ErrorSpec, code string) (nodecontract.ErrorSpec, bool) {
	for _, value := range values {
		if value.Code == code {
			return value, true
		}
	}
	return nodecontract.ErrorSpec{}, false
}

func signalPortExists(values []nodecontract.SignalPort, id string) bool {
	for _, value := range values {
		if value.ID == id {
			return true
		}
	}
	return false
}

func cloneTrigger(source *SignalTrigger) *SignalTrigger {
	if source == nil {
		return nil
	}
	clone := *source
	clone.Failure = cloneRoutedFailure(source.Failure)
	return &clone
}

func cloneRoutedFailure(source *RoutedFailure) *RoutedFailure {
	if source == nil {
		return nil
	}
	clone := *source
	return &clone
}

type statusEmitter struct {
	mu           sync.Mutex
	executor     *Executor
	journal      *run31.JournalWriter
	graphID      string
	nodeID       string
	attempt      int
	declarations map[string]nodecontract.StatusCategory
	violation    error
	closed       bool
}

func newStatusEmitter(executor *Executor, journal *run31.JournalWriter, graphID, nodeID string, attempt int, declarations []nodecontract.StatusEventSpec) *statusEmitter {
	byCode := make(map[string]nodecontract.StatusCategory, len(declarations))
	for _, declaration := range declarations {
		byCode[declaration.Code] = declaration.Category
	}
	return &statusEmitter{executor: executor, journal: journal, graphID: graphID, nodeID: nodeID, attempt: attempt, declarations: byCode}
}

func (e *statusEmitter) Emit(ctx context.Context, code string, counters map[string]int64) error {
	e.mu.Lock()
	defer e.mu.Unlock()
	reject := func(err error) error {
		if e.violation == nil {
			e.violation = err
		}
		return err
	}
	if ctx == nil {
		return reject(errors.New("status event context is required"))
	}
	if e.closed {
		return errors.New("status event was emitted after invocation returned")
	}
	category, ok := e.declarations[code]
	if !ok {
		return reject(errors.New("adapter emitted an undeclared status event"))
	}
	summary, err := run31.NewRedactedSummary(code, counters)
	if err != nil {
		return reject(err)
	}
	fact, err := run31.NewNodeStatusFact(run31.NodeStatusInput{
		GraphPath: []string{e.graphID}, NodeID: e.nodeID, Attempt: e.attempt, Code: code, Category: category,
		OccurredAt: e.executor.now().UTC(), Summary: summary,
	})
	if err != nil {
		return reject(err)
	}
	if _, err := e.journal.Append(ctx, fact); err != nil {
		return reject(err)
	}
	return nil
}

func (e *statusEmitter) Close() error {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.closed = true
	return e.violation
}
