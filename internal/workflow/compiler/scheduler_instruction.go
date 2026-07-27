package compiler

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/yottaapp/yotta/internal/datatype"
	"github.com/yottaapp/yotta/internal/nodeadapter"
	"github.com/yottaapp/yotta/internal/nodecontract"
	run "github.com/yottaapp/yotta/internal/run"
	"github.com/yottaapp/yotta/internal/workflow/schema"
)

type regionSignal struct {
	nodeID  string
	input   string
	failure *nodeadapter.RoutedFailure
}

func (s *regionSignal) Error() string {
	return fmt.Sprintf("region signal %s.%s escaped its activation", s.nodeID, s.input)
}

func (s *scheduler) dispatch(ctx context.Context, nodeID string, trigger *nodeadapter.SignalTrigger, evaluation map[string]bool) error {
	if s.invocations >= MaxScheduledInvocations {
		return errors.New("scheduler invocation budget exceeded")
	}
	s.invocations++
	node, ok := s.nodes[nodeID]
	if !ok {
		return fmt.Errorf("scheduled node %q is missing from Program", nodeID)
	}
	switch node.Instruction.Kind {
	case nodecontract.InstructionInvoke:
		return s.invoke(ctx, nodeID, trigger, evaluation)
	case nodecontract.InstructionRunRoot:
		return s.executeRunRoot(ctx, node, trigger)
	case nodecontract.InstructionCountedLoop:
		return s.executeCountedLoop(ctx, node, trigger)
	case nodecontract.InstructionForEach:
		return s.executeForEach(ctx, node, trigger)
	case nodecontract.InstructionRetry:
		return s.executeRetry(ctx, node, trigger)
	default:
		return fmt.Errorf("Program node %q has an unsupported instruction", nodeID)
	}
}

func (s *scheduler) executeRunRoot(ctx context.Context, node programNode, trigger *nodeadapter.SignalTrigger) error {
	if trigger != nil || node.Instruction.RunRoot == nil {
		return errors.New("run-root instruction received a signal or invalid payload")
	}
	if err := s.debugCheckpoint(ctx, node, s.attempts[node.ID]+1, nil); err != nil {
		return err
	}
	attempt, summary, err := s.beginInstruction(ctx, node)
	if err != nil {
		return err
	}
	if err := s.finishInstruction(ctx, node, attempt, summary); err != nil {
		return err
	}
	s.enqueueInstructionOutput(node.ID, node.Instruction.RunRoot.Output, nil)
	return nil
}

func (s *scheduler) executeCountedLoop(ctx context.Context, node programNode, trigger *nodeadapter.SignalTrigger) (returnErr error) {
	spec := node.Instruction.CountedLoop
	if spec == nil || trigger == nil {
		return errors.New("counted-loop instruction requires a signal and payload")
	}
	switch trigger.InputPort {
	case spec.BreakInput, spec.ContinueInput:
		return &regionSignal{nodeID: node.ID, input: trigger.InputPort, failure: cloneRoutedFailure(trigger.Failure)}
	case spec.EntryInput:
	default:
		return fmt.Errorf("counted-loop instruction received unknown input %q", trigger.InputPort)
	}
	inputs, err := s.resolveInputs(ctx, node, nil, map[string]bool{})
	if err != nil {
		return err
	}
	if err := s.debugCheckpoint(ctx, node, s.attempts[node.ID]+1, inputs); err != nil {
		return err
	}
	attempt, summary, err := s.beginInstruction(ctx, node)
	if err != nil {
		return err
	}
	closed := false
	defer func() {
		if returnErr != nil && !closed {
			returnErr = errors.Join(returnErr, s.closeInterruptedInstruction(ctx, node, attempt, summary, returnErr))
		}
	}()
	count, err := decodeInstructionInteger(inputs[spec.CountInput])
	if err != nil || count < 0 || count > int64(spec.MaxIterations) {
		return errors.Join(errors.New("counted-loop count exceeds its frozen budget"), err)
	}
	for index := int64(0); index < count; index++ {
		if err := s.setInstructionIntegerOutput(node, spec.IndexOutput, index, attempt); err != nil {
			return err
		}
		err := s.runActivation(ctx, s.instructionRoutes(node.ID, spec.BodyOutput, nil))
		if err == nil {
			continue
		}
		var signal *regionSignal
		if !errors.As(err, &signal) || signal.nodeID != node.ID {
			return err
		}
		if signal.input == spec.BreakInput {
			break
		}
		if signal.input != spec.ContinueInput {
			return err
		}
	}
	if err := s.finishInstruction(ctx, node, attempt, summary); err != nil {
		return err
	}
	closed = true
	s.enqueueInstructionOutput(node.ID, spec.CompletedOutput, nil)
	return nil
}

func (s *scheduler) executeForEach(ctx context.Context, node programNode, trigger *nodeadapter.SignalTrigger) (returnErr error) {
	spec := node.Instruction.ForEach
	if spec == nil || trigger == nil {
		return errors.New("for-each instruction requires a signal and payload")
	}
	switch trigger.InputPort {
	case spec.BreakInput, spec.ContinueInput:
		return &regionSignal{nodeID: node.ID, input: trigger.InputPort, failure: cloneRoutedFailure(trigger.Failure)}
	case spec.EntryInput:
	default:
		return fmt.Errorf("for-each instruction received unknown input %q", trigger.InputPort)
	}
	inputs, err := s.resolveInputs(ctx, node, nil, map[string]bool{})
	if err != nil {
		return err
	}
	if err := s.debugCheckpoint(ctx, node, s.attempts[node.ID]+1, inputs); err != nil {
		return err
	}
	attempt, summary, err := s.beginInstruction(ctx, node)
	if err != nil {
		return err
	}
	closed := false
	defer func() {
		if returnErr != nil && !closed {
			returnErr = errors.Join(returnErr, s.closeInterruptedInstruction(ctx, node, attempt, summary, returnErr))
		}
	}()
	var items []json.RawMessage
	if err := json.Unmarshal(inputs[spec.ItemsInput].InlineJSON(), &items); err != nil || len(items) > spec.MaxItems {
		return errors.Join(errors.New("for-each items exceed the frozen budget"), err)
	}
	itemType := node.OutputTypes[spec.ItemOutput]
	for index, item := range items {
		if err := s.setInstructionIntegerOutput(node, spec.IndexOutput, int64(index), attempt); err != nil {
			return err
		}
		sealed, err := datatype.SealInlineJSON(s.executor.catalog, itemType, item)
		if err != nil {
			return fmt.Errorf("seal for-each item: %w", err)
		}
		if err := s.setInstructionOutput(node, spec.ItemOutput, sealed, attempt); err != nil {
			return err
		}
		err = s.runActivation(ctx, s.instructionRoutes(node.ID, spec.BodyOutput, nil))
		if err == nil {
			continue
		}
		var signal *regionSignal
		if !errors.As(err, &signal) || signal.nodeID != node.ID {
			return err
		}
		if signal.input == spec.BreakInput {
			break
		}
		if signal.input != spec.ContinueInput {
			return err
		}
	}
	if err := s.finishInstruction(ctx, node, attempt, summary); err != nil {
		return err
	}
	closed = true
	s.enqueueInstructionOutput(node.ID, spec.CompletedOutput, nil)
	return nil
}

func (s *scheduler) executeRetry(ctx context.Context, node programNode, trigger *nodeadapter.SignalTrigger) (returnErr error) {
	spec := node.Instruction.Retry
	if spec == nil || trigger == nil {
		return errors.New("retry instruction requires a signal and payload")
	}
	if trigger.InputPort == spec.RetryInput {
		if trigger.Failure == nil {
			return errors.New("retry input requires a routed failure")
		}
		return &regionSignal{nodeID: node.ID, input: spec.RetryInput, failure: cloneRoutedFailure(trigger.Failure)}
	}
	if trigger.InputPort != spec.EntryInput {
		return fmt.Errorf("retry instruction received unknown input %q", trigger.InputPort)
	}
	inputs, err := s.resolveInputs(ctx, node, nil, map[string]bool{})
	if err != nil {
		return err
	}
	if err := s.debugCheckpoint(ctx, node, s.attempts[node.ID]+1, inputs); err != nil {
		return err
	}
	attempt, summary, err := s.beginInstruction(ctx, node)
	if err != nil {
		return err
	}
	closed := false
	defer func() {
		if returnErr != nil && !closed {
			returnErr = errors.Join(returnErr, s.closeInterruptedInstruction(ctx, node, attempt, summary, returnErr))
		}
	}()
	limit, err := decodeInstructionInteger(inputs[spec.AttemptsInput])
	if err != nil || limit < 1 || limit > int64(spec.MaxAttempts) {
		return errors.Join(errors.New("retry attempts exceed the frozen budget"), err)
	}
	var lastFailure *nodeadapter.RoutedFailure
	for current := int64(1); current <= limit; current++ {
		if err := s.setInstructionIntegerOutput(node, spec.AttemptOutput, current, attempt); err != nil {
			return err
		}
		err := s.runActivation(ctx, s.instructionRoutes(node.ID, spec.BodyOutput, nil))
		if err == nil {
			if err := s.finishInstruction(ctx, node, attempt, summary); err != nil {
				return err
			}
			closed = true
			s.enqueueInstructionOutput(node.ID, spec.CompletedOutput, nil)
			return nil
		}
		var signal *regionSignal
		if !errors.As(err, &signal) || signal.nodeID != node.ID || signal.input != spec.RetryInput || signal.failure == nil {
			return err
		}
		lastFailure = signal.failure
	}
	if err := s.finishInstruction(ctx, node, attempt, summary); err != nil {
		return err
	}
	closed = true
	s.enqueueInstructionOutput(node.ID, spec.ExhaustedOutput, lastFailure)
	return nil
}

func (s *scheduler) runActivation(ctx context.Context, queue []scheduledInvocation) error {
	parent := s.queue
	s.queue = append([]scheduledInvocation(nil), queue...)
	defer func() { s.queue = parent }()
	for len(s.queue) != 0 {
		if err := ctx.Err(); err != nil {
			return err
		}
		next := s.queue[0]
		s.queue = s.queue[1:]
		if err := s.dispatch(ctx, next.nodeID, next.trigger, map[string]bool{}); err != nil {
			return err
		}
	}
	return nil
}

func (s *scheduler) instructionRoutes(nodeID, output string, failure *nodeadapter.RoutedFailure) []scheduledInvocation {
	routes := s.routes[routeKey{channel: schema.EdgeExec, nodeID: nodeID, portID: output}]
	result := make([]scheduledInvocation, 0, len(routes))
	for _, route := range routes {
		result = append(result, scheduledInvocation{nodeID: route.To.NodeID, trigger: &nodeadapter.SignalTrigger{
			Channel: schema.EdgeExec, InputPort: route.To.PortID, From: route.From, Failure: cloneRoutedFailure(failure),
		}})
	}
	return result
}

func (s *scheduler) enqueueInstructionOutput(nodeID, output string, failure *nodeadapter.RoutedFailure) {
	s.queue = append(s.queue, s.instructionRoutes(nodeID, output, failure)...)
}

func (s *scheduler) beginInstruction(ctx context.Context, node programNode) (int, run.RedactedSummary, error) {
	s.attempts[node.ID]++
	attempt := s.attempts[node.ID]
	summary, err := run.NewRedactedSummary("node.execute", nil, nil)
	if err != nil {
		return 0, run.RedactedSummary{}, err
	}
	fact, err := run.NewNodeAttemptFact(run.NodeAttemptInput{
		GraphPath: append([]string(nil), node.GraphPath...), NodeID: node.SourceNodeID, Attempt: attempt, Outcome: run.AttemptStarted,
		OccurredAt: s.executor.now().UTC(), Summary: summary,
	})
	if err != nil {
		return 0, run.RedactedSummary{}, err
	}
	if _, err := s.journal.Append(ctx, fact); err != nil {
		return 0, run.RedactedSummary{}, err
	}
	s.clearNodeResult(node.ID)
	return attempt, summary, nil
}

func (s *scheduler) finishInstruction(ctx context.Context, node programNode, attempt int, summary run.RedactedSummary) error {
	fact, err := run.NewNodeAttemptFact(run.NodeAttemptInput{
		GraphPath: append([]string(nil), node.GraphPath...), NodeID: node.SourceNodeID, Attempt: attempt, Outcome: run.AttemptSucceeded,
		OccurredAt: s.executor.now().UTC(), Summary: summary,
	})
	if err != nil {
		return err
	}
	_, err = s.journal.Append(context.WithoutCancel(ctx), fact)
	return err
}

func (s *scheduler) closeInterruptedInstruction(ctx context.Context, node programNode, attempt int, summary run.RedactedSummary, cause error) error {
	var signal *regionSignal
	if errors.As(cause, &signal) {
		return s.finishInstruction(ctx, node, attempt, summary)
	}
	if errors.Is(cause, context.Canceled) || errors.Is(cause, context.DeadlineExceeded) || ctx.Err() != nil {
		return s.executor.cancelAttempt(context.WithoutCancel(ctx), s.journal, node.GraphPath, node.SourceNodeID, attempt, summary)
	}
	fact, err := run.NewNodeAttemptFact(run.NodeAttemptInput{
		GraphPath: append([]string(nil), node.GraphPath...), NodeID: node.SourceNodeID, Attempt: attempt, Outcome: run.AttemptRouted,
		OccurredAt: s.executor.now().UTC(), ErrorCode: "control.region_failed", Summary: summary,
	})
	if err != nil {
		return err
	}
	_, err = s.journal.Append(context.WithoutCancel(ctx), fact)
	return err
}

func (s *scheduler) setInstructionIntegerOutput(node programNode, portID string, value int64, attempt int) error {
	raw, err := json.Marshal(value)
	if err != nil {
		return err
	}
	resolved, ok := node.OutputTypes[portID]
	if !ok {
		return fmt.Errorf("instruction output %q has no frozen type", portID)
	}
	sealed, err := datatype.SealInlineJSON(s.executor.catalog, resolved, raw)
	if err != nil {
		return err
	}
	return s.setInstructionOutput(node, portID, sealed, attempt)
}

func (s *scheduler) setInstructionOutput(node programNode, portID string, value datatype.ValueEnvelope, attempt int) error {
	previousBytes := 0
	if previous, ok := s.result.NodeOutputs[node.ID][portID]; ok {
		previousBytes = len(previous.RuntimeArtifact())
	}
	nextBytes := len(value.RuntimeArtifact())
	if s.retainedBytes-previousBytes+nextBytes > MaxRunRetainedValueBytes {
		return errors.New("run retained value budget exceeded")
	}
	s.retainedBytes = s.retainedBytes - previousBytes + nextBytes
	if s.result.NodeOutputs[node.ID] == nil {
		s.result.NodeOutputs[node.ID] = make(map[string]datatype.ValueEnvelope)
		s.result.attempts[node.ID] = make(map[string]int)
	}
	s.result.NodeOutputs[node.ID][portID] = value
	s.result.attempts[node.ID][portID] = attempt
	return nil
}

func decodeInstructionInteger(value datatype.ValueEnvelope) (int64, error) {
	var result int64
	if !value.Valid() || json.Unmarshal(value.InlineJSON(), &result) != nil {
		return 0, errors.New("instruction integer input is invalid")
	}
	return result, nil
}
