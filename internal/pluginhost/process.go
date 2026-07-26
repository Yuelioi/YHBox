package pluginhost

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"path/filepath"
	"slices"
	"strings"
	"time"

	"github.com/yottaapp/yotta/internal/artifact"
	"github.com/yottaapp/yotta/internal/datatype"
	"github.com/yottaapp/yotta/internal/nodecatalog"
	"github.com/yottaapp/yotta/internal/nodecontract"
	"github.com/yottaapp/yotta/internal/nodepackage"
	"github.com/yottaapp/yotta/internal/pluginprotocol"
	"github.com/yottaapp/yotta/internal/processsandbox"
	"github.com/yottaapp/yotta/internal/resource"
	run "github.com/yottaapp/yotta/internal/run"
	"github.com/yottaapp/yotta/internal/workflow/compiler"
)

const (
	ProcessIsolationHostFeatureID = pluginprotocol.ProcessIsolationHostFeatureID
	ProcessGuestArgument          = "--yotta-plugin-process-v1"
	DefaultInvocationTimeout      = 30 * time.Second
	MaxInvocationTimeout          = 24 * time.Hour
	processExitOK                 = 0
	cancelGrace                   = 100 * time.Millisecond
)

type ProcessHostOptions struct {
	ProcessMemoryBytes uint64
	JobMemoryBytes     uint64
	InvocationTimeout  time.Duration
	MaxHostCalls       uint32
	MaxStatusEvents    uint32
	MaxOutputBytes     uint64
}

type ProcessHost struct {
	executionHost
	runner *processsandbox.Runner
}

type executionHost struct {
	catalog nodecatalog.Snapshot
	options ProcessHostOptions
}

func NewProcessHost(catalog nodecatalog.Snapshot, options ProcessHostOptions) (*ProcessHost, error) {
	if !catalog.Valid() {
		return nil, errors.New("process plugin host requires a valid Catalog")
	}
	var err error
	options, err = normalizeExecutionOptions(options)
	if err != nil {
		return nil, err
	}
	runner, err := processsandbox.New(processsandbox.Options{
		ProfileName: "Yotta.Plugin.Process.V1", DisplayName: "Yotta Process Plugin",
		Description:        "Isolated third-party Yotta node process",
		ProcessMemoryBytes: options.ProcessMemoryBytes, JobMemoryBytes: options.JobMemoryBytes,
	})
	if err != nil {
		return nil, err
	}
	return &ProcessHost{executionHost: executionHost{catalog: catalog, options: options}, runner: runner}, nil
}

func normalizeExecutionOptions(options ProcessHostOptions) (ProcessHostOptions, error) {
	if options.ProcessMemoryBytes == 0 {
		options.ProcessMemoryBytes = processsandbox.DefaultMemoryBytes
	}
	if options.JobMemoryBytes == 0 {
		options.JobMemoryBytes = options.ProcessMemoryBytes
	}
	if options.InvocationTimeout == 0 {
		options.InvocationTimeout = DefaultInvocationTimeout
	}
	if options.MaxHostCalls == 0 {
		options.MaxHostCalls = pluginprotocol.MaxHostCalls
	}
	if options.MaxStatusEvents == 0 {
		options.MaxStatusEvents = pluginprotocol.MaxStatusEvents
	}
	if options.MaxOutputBytes == 0 {
		options.MaxOutputBytes = pluginprotocol.MaxFrameBytes
	}
	if options.InvocationTimeout <= 0 || options.InvocationTimeout > MaxInvocationTimeout ||
		options.MaxHostCalls > pluginprotocol.MaxHostCalls || options.MaxStatusEvents > pluginprotocol.MaxStatusEvents ||
		options.MaxOutputBytes == 0 || options.MaxOutputBytes > pluginprotocol.MaxFrameBytes {
		return ProcessHostOptions{}, errors.New("plugin execution host budgets are invalid")
	}
	return options, nil
}

func (host *ProcessHost) HostFeatures() []string {
	if host == nil || host.runner == nil || !host.runner.Available() {
		return []string{}
	}
	return []string{ProcessIsolationHostFeatureID}
}

// Adapters projects only process-ABI nodes and pins each closure to its exact
// runtime payload and implementation lock.
func (host *ProcessHost) Adapters(packages []nodepackage.RuntimePackage) (map[string]compiler.InstalledAdapter, error) {
	if host == nil || host.runner == nil || !host.catalog.Valid() {
		return nil, errors.New("process plugin host is not initialized")
	}
	result := map[string]compiler.InstalledAdapter{}
	for _, runtimePackage := range packages {
		for _, node := range runtimePackage.Nodes {
			if node.Implementation.ABI.Kind != nodecontract.ABIProcess {
				continue
			}
			if err := host.validateRuntimeNode(runtimePackage, node, nodecontract.ABIProcess); err != nil {
				return nil, err
			}
			metadata := node.Payload.Metadata()
			if metadata.MediaType != "application/vnd.microsoft.portable-executable" || !strings.EqualFold(filepath.Ext(metadata.Path), ".exe") {
				return nil, fmt.Errorf("process plugin %q does not contain a portable executable payload", node.Lock.Entrypoint)
			}
			if _, duplicate := result[node.Lock.Entrypoint]; duplicate {
				return nil, fmt.Errorf("duplicate process plugin entrypoint %q", node.Lock.Entrypoint)
			}
			pinned := node
			result[node.Lock.Entrypoint] = compiler.InstalledAdapter{
				Implementation: node.Lock,
				Run: func(ctx context.Context, invocation compiler.Invocation) (compiler.AdapterResult, error) {
					return host.invoke(ctx, pinned, invocation)
				},
			}
		}
	}
	return result, nil
}

func (host *executionHost) validateRuntimeNode(runtimePackage nodepackage.RuntimePackage, node nodepackage.RuntimeNode, kind nodecontract.ABIKind) error {
	if node.Implementation.ABI.Kind != kind || node.Implementation.ABI.Version != "v1" || node.Lock != (nodecatalog.ImplementationLock{
		PackageID: runtimePackage.PackageID, ArtifactDigest: runtimePackage.ManifestDigest,
		ABI: node.Implementation.ABI, Entrypoint: node.Implementation.Entrypoint,
	}) {
		return fmt.Errorf("plugin %q has an unsupported or mismatched implementation lock", node.Lock.Entrypoint)
	}
	entry, ok := host.catalog.Lookup(node.Contract.NodeRef().NodeTypeID)
	if !ok || entry.Contract.NodeRef() != node.Contract.NodeRef() || entry.Implementation != node.Lock {
		return fmt.Errorf("plugin %q is not pinned by the execution Catalog", node.Lock.Entrypoint)
	}
	featureID := ProcessIsolationHostFeatureID
	if kind == nodecontract.ABIWIT {
		featureID = WasmIsolationHostFeatureID
	}
	if !slices.ContainsFunc(node.Contract.Machine().HostFeatureRequirements, func(requirement nodecontract.HostFeatureRequirement) bool {
		return requirement.FeatureID == featureID
	}) {
		return fmt.Errorf("plugin %q does not declare its required isolation host feature", node.Lock.Entrypoint)
	}
	return nil
}

func (host *ProcessHost) invoke(parent context.Context, node nodepackage.RuntimeNode, invocation compiler.Invocation) (compiler.AdapterResult, error) {
	initial, deadline, err := host.invocationFrame(parent, node, invocation)
	if err != nil {
		return compiler.AdapterResult{}, err
	}
	payload, err := node.Payload.Read(parent, processsandbox.MaxImageBytes)
	if err != nil {
		return compiler.AdapterResult{}, fmt.Errorf("read process plugin payload: %w", err)
	}
	image, err := processsandbox.NewImage(filepath.Base(node.Payload.Metadata().Path), payload)
	if err != nil {
		return compiler.AdapterResult{}, fmt.Errorf("seal process plugin image: %w", err)
	}
	executionContext, cancelExecution := context.WithDeadline(parent, deadline)
	defer cancelExecution()
	session := &processSession{
		catalog: host.catalog, invocation: invocation,
		nextSequence: 2, maxHostCalls: host.options.MaxHostCalls, maxStatusEvents: host.options.MaxStatusEvents,
	}
	result, err := executeSandboxed(executionContext, host.runner, processsandbox.Request{
		Image: image, Args: []string{ProcessGuestArgument}, Timeout: time.Until(deadline),
	}, session, processExitOK, "process plugin", func(writer io.Writer) error {
		return pluginprotocol.WriteFrame(writer, initial)
	})
	if err != nil {
		return compiler.AdapterResult{}, err
	}
	return host.openResult(result)
}

type processWait struct {
	result processsandbox.Result
	err    error
}

func executeSandboxed(
	ctx context.Context,
	runner *processsandbox.Runner,
	request processsandbox.Request,
	session *processSession,
	exitOK uint32,
	label string,
	initialize func(io.Writer) error,
) (*pluginprotocol.Result, error) {
	lifetimeContext, cancelLifetime := context.WithCancel(context.Background())
	defer cancelLifetime()
	process, err := runner.Start(lifetimeContext, request)
	if err != nil {
		return nil, fmt.Errorf("start %s: %w", label, err)
	}
	defer process.Close()
	stderrDone := make(chan struct{}, 1)
	go func() {
		_, _ = io.Copy(io.Discard, process.Stderr())
		stderrDone <- struct{}{}
	}()
	if err := initialize(process.Stdin()); err != nil {
		return nil, fmt.Errorf("initialize %s: %w", label, err)
	}
	session.reader, session.writer = process.Stdout(), process.Stdin()
	result, err := session.serve(ctx, process)
	if err != nil {
		return nil, err
	}
	_ = process.Stdin().Close()
	_ = process.Stdout().Close()
	waitDone := make(chan processWait, 1)
	go func() {
		outcome, waitErr := process.Wait()
		waitDone <- processWait{result: outcome, err: waitErr}
	}()
	select {
	case waited := <-waitDone:
		if waited.err != nil {
			return nil, fmt.Errorf("%s exited abnormally: exit=%d: %w", label, waited.result.ExitCode, waited.err)
		}
		if waited.result.ExitCode != exitOK {
			return nil, fmt.Errorf("%s exited abnormally: exit=%d", label, waited.result.ExitCode)
		}
	case <-ctx.Done():
		_ = process.Terminate()
		return nil, ctx.Err()
	}
	select {
	case <-stderrDone:
	case <-time.After(cancelGrace):
	}
	return result, nil
}

type processControl interface {
	Terminate() error
}

type processSession struct {
	ctx             context.Context
	catalog         nodecatalog.Snapshot
	invocation      compiler.Invocation
	reader          io.Reader
	writer          io.Writer
	nextSequence    uint64
	hostCalls       uint32
	statusEvents    uint32
	requestIDs      map[string]struct{}
	maxHostCalls    uint32
	maxStatusEvents uint32
}

func (session *processSession) serve(ctx context.Context, process processControl) (*pluginprotocol.Result, error) {
	session.ctx = ctx
	if session.requestIDs == nil {
		session.requestIDs = map[string]struct{}{}
	}
	for {
		frameDone := make(chan frameRead, 1)
		go func() {
			frame, err := pluginprotocol.ReadFrame(session.reader)
			frameDone <- frameRead{frame: frame, err: err}
		}()
		var frame *pluginprotocol.Frame
		select {
		case opened := <-frameDone:
			if opened.err != nil {
				return nil, fmt.Errorf("read process plugin frame: %w", opened.err)
			}
			frame = opened.frame
		case <-ctx.Done():
			session.cancel(process, cancelReason(ctx.Err()))
			return nil, ctx.Err()
		}
		if frame.Sequence != session.nextSequence {
			return nil, fmt.Errorf("process plugin sequence %d, want %d", frame.Sequence, session.nextSequence)
		}
		session.nextSequence++
		switch payload := frame.Payload.(type) {
		case *pluginprotocol.Frame_HostOpenRequest:
			if err := session.hostCall(payload.HostOpenRequest.RequestId, func() *pluginprotocol.Frame { return session.open(payload.HostOpenRequest) }); err != nil {
				return nil, err
			}
		case *pluginprotocol.Frame_HostInvokeRequest:
			if err := session.hostCall(payload.HostInvokeRequest.RequestId, func() *pluginprotocol.Frame { return session.invoke(payload.HostInvokeRequest) }); err != nil {
				return nil, err
			}
		case *pluginprotocol.Frame_HostDropRequest:
			if err := session.hostCall(payload.HostDropRequest.RequestId, func() *pluginprotocol.Frame { return session.drop(payload.HostDropRequest) }); err != nil {
				return nil, err
			}
		case *pluginprotocol.Frame_HostEntropyRequest:
			if err := session.hostCall(payload.HostEntropyRequest.RequestId, func() *pluginprotocol.Frame { return session.entropy(payload.HostEntropyRequest) }); err != nil {
				return nil, err
			}
		case *pluginprotocol.Frame_HostWaitRequest:
			if err := session.hostCall(payload.HostWaitRequest.RequestId, func() *pluginprotocol.Frame { return session.wait(ctx, payload.HostWaitRequest) }); err != nil {
				return nil, err
			}
		case *pluginprotocol.Frame_StateReadRequest:
			if err := session.hostCall(payload.StateReadRequest.RequestId, func() *pluginprotocol.Frame { return session.stateRead(payload.StateReadRequest) }); err != nil {
				return nil, err
			}
		case *pluginprotocol.Frame_StateWriteRequest:
			if err := session.hostCall(payload.StateWriteRequest.RequestId, func() *pluginprotocol.Frame { return session.stateWrite(payload.StateWriteRequest) }); err != nil {
				return nil, err
			}
		case *pluginprotocol.Frame_Status:
			session.statusEvents++
			if session.statusEvents > session.maxStatusEvents {
				return nil, errors.New("process plugin exceeded its status-event budget")
			}
			if session.invocation.EmitStatus == nil {
				return nil, errors.New("process plugin invocation has no status emitter")
			}
			if err := session.invocation.EmitStatus(ctx, payload.Status.Code, counters(payload.Status.Counters)); err != nil {
				return nil, fmt.Errorf("emit process plugin status: %w", err)
			}
		case *pluginprotocol.Frame_Action:
			if session.invocation.RecordAction == nil {
				return nil, errors.New("process plugin invocation has no action recorder")
			}
			if err := session.invocation.RecordAction(ctx, adapterAction(payload.Action)); err != nil {
				return nil, fmt.Errorf("record process plugin action: %w", err)
			}
		case *pluginprotocol.Frame_Result:
			return payload.Result, nil
		default:
			return nil, errors.New("process plugin sent a host-only or unsupported frame")
		}
	}
}

type frameRead struct {
	frame *pluginprotocol.Frame
	err   error
}

func (session *processSession) hostCall(requestID string, buildResponse func() *pluginprotocol.Frame) error {
	if _, duplicate := session.requestIDs[requestID]; duplicate {
		return errors.New("process plugin reused a host request ID")
	}
	session.requestIDs[requestID] = struct{}{}
	session.hostCalls++
	if session.hostCalls > session.maxHostCalls {
		return errors.New("process plugin exceeded its host-call budget")
	}
	response := buildResponse()
	response.Protocol = pluginprotocol.Protocol
	response.Sequence = session.nextSequence
	if err := pluginprotocol.WriteFrame(session.writer, response); err != nil {
		return fmt.Errorf("write process plugin host response: %w", err)
	}
	session.nextSequence++
	return nil
}

func (session *processSession) open(request *pluginprotocol.HostOpenRequest) *pluginprotocol.Frame {
	response := &pluginprotocol.HostOpenResponse{RequestId: request.RequestId}
	binding := session.invocation.Sessions[request.RequirementId]
	if binding == nil {
		response.Failure = hostFailure("resource requirement is unavailable")
	} else if handle, err := binding.Open(session.ctx, request.Operations, request.ConfigJson); err != nil {
		response.Failure = hostFailure("resource open was denied")
	} else if response.HandleJson, err = artifact.Marshal(handle); err != nil {
		response.Failure = hostFailure("resource handle could not be encoded")
	}
	return &pluginprotocol.Frame{Payload: &pluginprotocol.Frame_HostOpenResponse{HostOpenResponse: response}}
}

func (session *processSession) invoke(request *pluginprotocol.HostInvokeRequest) *pluginprotocol.Frame {
	response := &pluginprotocol.HostInvokeResponse{RequestId: request.RequestId}
	binding := session.invocation.Sessions[request.RequirementId]
	var handle resource.Handle
	if binding == nil || decodeCanonical(request.HandleJson, &handle) != nil || handle.Validate() != nil {
		response.Failure = hostFailure("resource invocation authority is invalid")
	} else if payload, err := binding.Invoke(session.ctx, handle, request.Operation, request.Payload); err != nil {
		response.Failure = hostFailure("resource invocation failed")
	} else {
		response.Payload = payload
	}
	return &pluginprotocol.Frame{Payload: &pluginprotocol.Frame_HostInvokeResponse{HostInvokeResponse: response}}
}

func (session *processSession) drop(request *pluginprotocol.HostDropRequest) *pluginprotocol.Frame {
	response := &pluginprotocol.HostDropResponse{RequestId: request.RequestId}
	binding := session.invocation.Sessions[request.RequirementId]
	var handle resource.Handle
	if binding == nil || decodeCanonical(request.HandleJson, &handle) != nil || handle.Validate() != nil {
		response.Failure = hostFailure("resource drop authority is invalid")
	} else if err := binding.Drop(session.ctx, handle); err != nil {
		response.Failure = hostFailure("resource drop failed")
	}
	return &pluginprotocol.Frame{Payload: &pluginprotocol.Frame_HostDropResponse{HostDropResponse: response}}
}

func (session *processSession) entropy(request *pluginprotocol.HostEntropyRequest) *pluginprotocol.Frame {
	response := &pluginprotocol.HostEntropyResponse{RequestId: request.RequestId}
	if session.invocation.ReadEntropy == nil {
		response.Failure = hostFailure("entropy is unavailable")
	} else {
		response.Entropy = make([]byte, request.ByteCount)
		if err := session.invocation.ReadEntropy(response.Entropy); err != nil {
			response.Entropy = nil
			response.Failure = hostFailure("entropy request failed")
		}
	}
	return &pluginprotocol.Frame{Payload: &pluginprotocol.Frame_HostEntropyResponse{HostEntropyResponse: response}}
}

func (session *processSession) wait(ctx context.Context, request *pluginprotocol.HostWaitRequest) *pluginprotocol.Frame {
	response := &pluginprotocol.HostWaitResponse{RequestId: request.RequestId}
	if session.invocation.Wait == nil || session.invocation.Wait(ctx, time.Duration(request.DurationMillis)*time.Millisecond) != nil {
		response.Failure = hostFailure("wait request failed")
	}
	return &pluginprotocol.Frame{Payload: &pluginprotocol.Frame_HostWaitResponse{HostWaitResponse: response}}
}

func (session *processSession) stateRead(request *pluginprotocol.StateReadRequest) *pluginprotocol.Frame {
	response := &pluginprotocol.StateReadResponse{RequestId: request.RequestId}
	binding, ok := session.invocation.State[request.AccessId]
	if !ok {
		response.Failure = hostFailure("state read authority is unavailable")
	} else if snapshot, err := binding.Read(); err != nil {
		response.Failure = hostFailure("state read was denied")
	} else {
		response.ValueEnvelope = snapshot.Value.RuntimeArtifact()
		response.Revision = snapshot.Revision
		response.ChangedUnixMillis = snapshot.ChangedAt.UnixMilli()
	}
	return &pluginprotocol.Frame{Payload: &pluginprotocol.Frame_StateReadResponse{StateReadResponse: response}}
}

func (session *processSession) stateWrite(request *pluginprotocol.StateWriteRequest) *pluginprotocol.Frame {
	response := &pluginprotocol.StateWriteResponse{RequestId: request.RequestId}
	binding, ok := session.invocation.State[request.AccessId]
	envelope, err := datatype.OpenValueEnvelope(session.catalog, request.ValueEnvelope)
	if !ok || err != nil {
		response.Failure = hostFailure("state write authority or value is invalid")
	} else if snapshot, err := binding.Write(envelope); err != nil {
		response.Failure = hostFailure("state write was denied")
	} else {
		response.ValueEnvelope = snapshot.Value.RuntimeArtifact()
		response.Revision = snapshot.Revision
		response.ChangedUnixMillis = snapshot.ChangedAt.UnixMilli()
	}
	return &pluginprotocol.Frame{Payload: &pluginprotocol.Frame_StateWriteResponse{StateWriteResponse: response}}
}

func (session *processSession) cancel(process processControl, reason string) {
	frame := &pluginprotocol.Frame{Protocol: pluginprotocol.Protocol, Sequence: session.nextSequence,
		Payload: &pluginprotocol.Frame_Cancel{Cancel: &pluginprotocol.Cancel{Reason: reason}}}
	done := make(chan struct{}, 1)
	go func() {
		_ = pluginprotocol.WriteFrame(session.writer, frame)
		done <- struct{}{}
	}()
	select {
	case <-done:
	case <-time.After(cancelGrace):
	}
	_ = process.Terminate()
}

func (host *executionHost) invocationFrame(parent context.Context, node nodepackage.RuntimeNode, invocation compiler.Invocation) (*pluginprotocol.Frame, time.Time, error) {
	if parent == nil || invocation.InvocationID == "" || invocation.Attempt <= 0 || invocation.ObservedAt.IsZero() {
		return nil, time.Time{}, errors.New("process plugin invocation identity is invalid")
	}
	deadline := invocation.ObservedAt.Add(host.options.InvocationTimeout)
	if parentDeadline, ok := parent.Deadline(); ok && parentDeadline.Before(deadline) {
		deadline = parentDeadline
	}
	if !deadline.After(invocation.ObservedAt) || time.Until(deadline) <= 0 {
		return nil, time.Time{}, context.DeadlineExceeded
	}
	nodeRefJSON, err := artifact.Marshal(node.Contract.NodeRef())
	if err != nil {
		return nil, time.Time{}, err
	}
	lockJSON, err := artifact.Marshal(node.Lock)
	if err != nil {
		return nil, time.Time{}, err
	}
	configJSON, err := artifact.Marshal(invocation.Config)
	if err != nil {
		return nil, time.Time{}, err
	}
	inputs := make([]*pluginprotocol.PortValue, 0, len(invocation.Inputs))
	for portID, envelope := range invocation.Inputs {
		if !envelope.Valid() {
			return nil, time.Time{}, fmt.Errorf("process plugin input %q is invalid", portID)
		}
		inputs = append(inputs, &pluginprotocol.PortValue{PortId: portID, ValueEnvelope: envelope.RuntimeArtifact()})
	}
	slices.SortFunc(inputs, func(left, right *pluginprotocol.PortValue) int { return strings.Compare(left.PortId, right.PortId) })
	trigger, err := marshalTrigger(invocation.Trigger)
	if err != nil {
		return nil, time.Time{}, err
	}
	request := &pluginprotocol.Invocation{
		RequestId: invocation.InvocationID, InvocationId: invocation.InvocationID, GraphId: invocation.GraphID, NodeId: invocation.NodeID,
		Attempt: uint32(invocation.Attempt), ObservedUnixMillis: invocation.ObservedAt.UnixMilli(), DeadlineUnixMillis: deadline.UnixMilli(),
		NodeRefJson: nodeRefJSON, ImplementationLockJson: lockJSON, ConfigJson: configJSON, Inputs: inputs, Trigger: trigger,
		Budget: &pluginprotocol.Budget{
			MaxFrameBytes: pluginprotocol.MaxFrameBytes, MaxOutputBytes: host.options.MaxOutputBytes,
			MaxHostCalls: host.options.MaxHostCalls, MaxStatusEvents: host.options.MaxStatusEvents,
		},
	}
	frame := &pluginprotocol.Frame{Protocol: pluginprotocol.Protocol, Sequence: 1,
		Payload: &pluginprotocol.Frame_Invocation{Invocation: request}}
	if err := pluginprotocol.ValidateFrame(frame); err != nil {
		return nil, time.Time{}, err
	}
	return frame, deadline, nil
}

func (host *executionHost) openResult(result *pluginprotocol.Result) (compiler.AdapterResult, error) {
	if result == nil {
		return compiler.AdapterResult{}, errors.New("process plugin omitted its result")
	}
	if result.Outcome == pluginprotocol.Outcome_OUTCOME_FAILED {
		return compiler.AdapterResult{}, &compiler.NodeFailure{
			Code: result.Failure.Code, Output: result.Failure.Output, Cause: errors.New(result.Failure.Message),
		}
	}
	if result.TerminationStrength != "cooperative" {
		return compiler.AdapterResult{}, errors.New("process plugin claimed a host-only termination strength")
	}
	opened := make(map[string]datatype.ValueEnvelope, len(result.Outputs))
	var total uint64
	for _, output := range result.Outputs {
		total += uint64(len(output.ValueEnvelope))
		if total > host.options.MaxOutputBytes {
			return compiler.AdapterResult{}, errors.New("process plugin exceeded its output byte budget")
		}
		envelope, err := datatype.OpenValueEnvelope(host.catalog, output.ValueEnvelope)
		if err != nil {
			return compiler.AdapterResult{}, fmt.Errorf("open process plugin output %q: %w", output.PortId, err)
		}
		opened[output.PortId] = envelope
	}
	return compiler.AdapterResult{Outputs: opened, ExecOutputs: append([]string(nil), result.ExecOutputs...)}, nil
}

func marshalTrigger(trigger *compiler.SignalTrigger) (*pluginprotocol.Trigger, error) {
	if trigger == nil {
		return nil, nil
	}
	result := &pluginprotocol.Trigger{
		Channel: string(trigger.Channel), InputPort: trigger.InputPort,
		SourceNodeId: trigger.From.NodeID, SourcePortId: trigger.From.PortID,
	}
	if trigger.Failure != nil {
		failure, err := artifact.Marshal(trigger.Failure)
		if err != nil {
			return nil, err
		}
		result.FailureJson = failure
	}
	return result, nil
}

func decodeCanonical(raw []byte, destination any) error {
	canonical, err := artifact.Canonicalize(raw)
	if err != nil || !bytes.Equal(canonical, raw) {
		return errors.New("contract JSON is not canonical")
	}
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(destination); err != nil {
		return err
	}
	if decoder.Decode(&struct{}{}) != io.EOF {
		return errors.New("contract JSON contains trailing data")
	}
	return nil
}

func hostFailure(message string) *pluginprotocol.Failure {
	return &pluginprotocol.Failure{Code: "plugin.host_call_failed", Message: message}
}

func counters(source []*pluginprotocol.Counter) map[string]int64 {
	result := make(map[string]int64, len(source))
	for _, counter := range source {
		result[counter.Key] = counter.Value
	}
	return result
}

func adapterAction(source *pluginprotocol.ActionEvent) compiler.AdapterAction {
	facts := make(map[string]string, len(source.Facts))
	for _, fact := range source.Facts {
		facts[fact.Key] = fact.Value
	}
	return compiler.AdapterAction{
		EffectID: source.EffectId, Action: source.Action, Outcome: run.ActionOutcome(source.Outcome), ErrorCode: source.ErrorCode,
		SummaryCode: source.SummaryCode, Counters: counters(source.Counters), Facts: facts,
	}
}

func cancelReason(err error) string {
	if errors.Is(err, context.DeadlineExceeded) {
		return "deadline_exceeded"
	}
	return "user_cancelled"
}
