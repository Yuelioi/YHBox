// Package noderuntime contains installed adapters for Yotta's built-in nodes.
package noderuntime

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"time"

	"github.com/yottaapp/yotta/internal/artifact"
	automationinstalled "github.com/yottaapp/yotta/internal/automation/installed"
	"github.com/yottaapp/yotta/internal/blob"
	"github.com/yottaapp/yotta/internal/datatype"
	"github.com/yottaapp/yotta/internal/nodecontract"
	"github.com/yottaapp/yotta/internal/nodes"
	run "github.com/yottaapp/yotta/internal/run"
	"github.com/yottaapp/yotta/internal/scriptengine"
	"github.com/yottaapp/yotta/internal/stream"
	"github.com/yottaapp/yotta/internal/workflow/compiler"
)

const conversionChunkBytes = 64 << 10

type Dependencies struct {
	Script ScriptExecutor
	Log    LogEmitter
	Now    func() time.Time
}

type ScriptExecutor interface {
	Execute(context.Context, scriptengine.Request) (scriptengine.Response, error)
}

func Installed(builtins nodes.Builtins, dependencies Dependencies) (map[string]compiler.InstalledAdapter, error) {
	if dependencies.Script == nil || dependencies.Log == nil {
		return nil, errors.New("installed built-ins require isolated script and workflow log runtimes")
	}
	if dependencies.Now == nil {
		dependencies.Now = time.Now
	}
	installed := make(map[string]compiler.InstalledAdapter, len(builtins.Definitions()))
	specialized := map[string]compiler.Adapter{
		nodes.BlobToStreamNodeID:         blobToStream(builtins),
		nodes.StreamToBlobNodeID:         streamToBlob(builtins),
		nodes.RandomIntegerNodeID:        randomInteger(builtins),
		nodes.RandomNumberNodeID:         randomNumber(builtins),
		nodes.RandomBooleanNodeID:        randomBoolean(builtins),
		nodes.RandomChoiceNodeID:         randomChoice(builtins),
		nodes.ObserveTimeNodeID:          observeTime(builtins),
		nodes.StateReadNodeID:            stateRead(builtins),
		nodes.StateWriteNodeID:           stateWrite(builtins),
		nodes.StateMetadataNodeID:        stateMetadata(builtins),
		nodes.BranchNodeID:               branch(),
		nodes.DelayNodeID:                delay(),
		nodes.EndBranchNodeID:            endBranch(),
		nodes.AIGenerateNodeID:           aiGenerate(builtins, false),
		nodes.AIExtractNodeID:            aiGenerate(builtins, true),
		nodes.AIAgentNodeID:              aiAgent(builtins, dependencies.Now),
		nodes.ScriptExecuteNodeID:        scriptExecute(builtins, dependencies.Script),
		nodes.FileReadTextNodeID:         fileRead(builtins, false),
		nodes.FileReadJSONNodeID:         fileRead(builtins, true),
		nodes.FileStatNodeID:             fileStat(builtins),
		nodes.HTTPGetNodeID:              httpGet(builtins),
		nodes.LaunchApplicationNodeID:    launchApplication(),
		nodes.TerminateApplicationNodeID: terminateApplication(builtins),
		nodes.ClickPointerNodeID:         automationInput(nodes.ClickPointerNodeID, automationinstalled.OperationClick),
		nodes.MovePointerNodeID:          automationInput(nodes.MovePointerNodeID, automationinstalled.OperationMove),
		nodes.ScrollPointerNodeID:        automationInput(nodes.ScrollPointerNodeID, automationinstalled.OperationScroll),
		nodes.DragPointerNodeID:          automationInput(nodes.DragPointerNodeID, automationinstalled.OperationDrag),
		nodes.MovePointerRelativeNodeID:  automationInput(nodes.MovePointerRelativeNodeID, automationinstalled.OperationMoveRelative),
		nodes.PressKeysNodeID:            automationInput(nodes.PressKeysNodeID, automationinstalled.OperationPressKeys),
		nodes.TypeTextNodeID:             automationInput(nodes.TypeTextNodeID, automationinstalled.OperationTypeText),
		nodes.ActivateWindowNodeID:       activateWindow(),
		nodes.StopTargetAppNodeID:        stopTargetApp(),
		nodes.CaptureWindowNodeID:        captureWindow(builtins),
		nodes.WaitTemplateNodeID:         automationTemplate(builtins, nodes.WaitTemplateNodeID),
		nodes.WaitTemplateGoneNodeID:     automationTemplate(builtins, nodes.WaitTemplateGoneNodeID),
		nodes.ClickTemplateNodeID:        automationTemplate(builtins, nodes.ClickTemplateNodeID),
		nodes.PlayInputClipNodeID:        playInputClip(),
		nodes.MatchTemplateNodeID:        matchTemplate(builtins),
		nodes.FindTemplateMatchesNodeID:  findTemplateMatches(builtins),
		nodes.CompareImagesNodeID:        compareImages(builtins),
		nodes.DecodeQRNodeID:             decodeQR(builtins),
		nodes.AnalyzeColorNodeID:         analyzeColor(builtins),
		nodes.FindColorBlobsNodeID:       findColorBlobs(builtins),
		nodes.LogNodeID:                  writeLog(dependencies.Log),
		nodes.ThrowNodeID:                throwFailure(),
	}
	for _, definition := range builtins.Definitions() {
		trusted, err := trustedDefinition(builtins, definition.Contract.NodeRef().NodeTypeID)
		if err != nil {
			return nil, err
		}
		if trusted.Contract.Machine().Instruction.Kind != nodecontract.InstructionInvoke {
			continue
		}
		adapter := specialized[trusted.Contract.NodeRef().NodeTypeID]
		if trusted.EvaluateInline != nil {
			adapter = inlineAdapter(builtins, trusted)
		}
		if adapter == nil {
			return nil, fmt.Errorf("built-in %q has no runtime adapter", trusted.Contract.NodeRef().NodeTypeID)
		}
		entrypoint := trusted.Implementation.Entrypoint
		if _, duplicate := installed[entrypoint]; duplicate {
			return nil, fmt.Errorf("duplicate built-in entrypoint %q", entrypoint)
		}
		installed[entrypoint] = compiler.InstalledAdapter{Implementation: trusted.Implementation, Run: adapter}
	}
	return installed, nil
}

func trustedDefinition(builtins nodes.Builtins, nodeTypeID string) (nodes.BuiltinDefinition, error) {
	definition, ok := builtins.Definition(nodeTypeID)
	if !ok {
		return nodes.BuiltinDefinition{}, fmt.Errorf("built-in definition for %q is missing", nodeTypeID)
	}
	entry, ok := builtins.Catalog.Lookup(nodeTypeID)
	if !ok {
		return nodes.BuiltinDefinition{}, fmt.Errorf("built-in Catalog entry for %q is missing", nodeTypeID)
	}
	if entry.Contract.NodeRef() != definition.Contract.NodeRef() || entry.Implementation != definition.Implementation {
		return nodes.BuiltinDefinition{}, fmt.Errorf("built-in definition for %q does not match the Catalog", nodeTypeID)
	}
	return definition, nil
}

func inlineAdapter(builtins nodes.Builtins, definition nodes.BuiltinDefinition) compiler.Adapter {
	machine := definition.Contract.Machine()
	return func(ctx context.Context, invocation compiler.Invocation) (compiler.AdapterResult, error) {
		inputs := make(map[string]json.RawMessage, len(machine.Ports.DataInputs))
		for _, port := range machine.Ports.DataInputs {
			envelope, ok := invocation.Inputs[port.ID]
			if !ok || len(envelope.InlineJSON()) == 0 {
				return compiler.AdapterResult{}, fmt.Errorf("inline input %q is missing", port.ID)
			}
			inputs[port.ID] = envelope.InlineJSON()
		}
		rawOutputs, err := definition.EvaluateInline(ctx, inputs, invocation.Config)
		if err != nil {
			var failure *nodes.InlineFailure
			if errors.As(err, &failure) && err == failure {
				return compiler.AdapterResult{}, &compiler.NodeFailure{Code: failure.Code, Output: failure.Output, Cause: failure.Cause}
			}
			return compiler.AdapterResult{}, err
		}
		if len(rawOutputs) != len(machine.Ports.DataOutputs) {
			return compiler.AdapterResult{}, errors.New("inline evaluator returned the wrong output count")
		}
		outputs := make(map[string]datatype.ValueEnvelope, len(machine.Ports.DataOutputs))
		for _, port := range machine.Ports.DataOutputs {
			raw, ok := rawOutputs[port.ID]
			resolved, resolvedOK := invocation.OutputTypes[port.ID]
			if !ok || !resolvedOK {
				return compiler.AdapterResult{}, fmt.Errorf("inline output %q is missing or unresolved", port.ID)
			}
			sealed, err := datatype.SealInlineJSON(builtins.Catalog, resolved, raw)
			if err != nil {
				return compiler.AdapterResult{}, fmt.Errorf("seal inline output %q: %w", port.ID, err)
			}
			outputs[port.ID] = sealed
		}
		return compiler.AdapterResult{Outputs: outputs}, nil
	}
}

func blobToStream(builtins nodes.Builtins) compiler.Adapter {
	return func(ctx context.Context, invocation compiler.Invocation) (_ compiler.AdapterResult, runErr error) {
		counters := map[string]int64{}
		defer func() {
			runErr = errors.Join(runErr, recordAdapterOutcome(ctx, invocation, compiler.AdapterAction{
				EffectID: nodes.BlobToStreamEffectID, Action: "conversion.stream-opened",
				SummaryCode: "conversion.blob-to-stream", Counters: counters,
			}, "conversion.blob_to_stream_failed", runErr))
		}()
		input, ok := invocation.Inputs["blob"]
		if !ok {
			return compiler.AdapterResult{}, errors.New("blob-to-stream input is missing")
		}
		ref, ok := input.BlobRef()
		if !ok {
			return compiler.AdapterResult{}, errors.New("blob-to-stream input is not a BlobRef")
		}
		counters["bytes"] = ref.Size
		blobSession, streamSession := invocation.Sessions["blob-read"], invocation.Sessions["stream"]
		if blobSession == nil || streamSession == nil {
			return compiler.AdapterResult{}, errors.New("blob-to-stream capability session is missing")
		}
		readConfig, err := artifact.Marshal(blob.ReadConfig{Blob: ref})
		if err != nil {
			return compiler.AdapterResult{}, err
		}
		reader, err := blobSession.Open(ctx, []string{blob.OperationReadRange}, readConfig)
		if err != nil {
			return compiler.AdapterResult{}, err
		}
		streamConfig, err := artifact.Marshal(stream.Config{Capacity: 4, MaxChunkBytes: conversionChunkBytes})
		if err != nil {
			_ = blobSession.Drop(context.Background(), reader)
			return compiler.AdapterResult{}, err
		}
		streamHandle, err := streamSession.Open(ctx, []string{
			stream.OperationCancel, stream.OperationFinish, stream.OperationReceive, stream.OperationSend,
		}, streamConfig)
		if err != nil {
			_ = blobSession.Drop(context.Background(), reader)
			return compiler.AdapterResult{}, err
		}
		if err := invocation.Spawn(func(taskCtx context.Context) (taskErr error) {
			defer func() { taskErr = errors.Join(taskErr, blobSession.Drop(context.Background(), reader)) }()
			defer func() {
				if taskErr != nil {
					_, _ = streamSession.Invoke(context.Background(), streamHandle, stream.OperationCancel, []byte("blob producer failed"))
				}
			}()
			for offset := int64(0); offset < ref.Size; {
				length := min(int64(conversionChunkBytes), ref.Size-offset)
				request, err := artifact.Marshal(blob.RangeRequest{Offset: offset, Length: length})
				if err != nil {
					return err
				}
				chunk, err := blobSession.Invoke(taskCtx, reader, blob.OperationReadRange, request)
				if err != nil {
					return err
				}
				if _, err := streamSession.Invoke(taskCtx, streamHandle, stream.OperationSend, chunk); err != nil {
					return err
				}
				offset += int64(len(chunk))
			}
			_, err := streamSession.Invoke(taskCtx, streamHandle, stream.OperationFinish, nil)
			return err
		}); err != nil {
			_ = blobSession.Drop(context.Background(), reader)
			_ = streamSession.Drop(context.Background(), streamHandle)
			return compiler.AdapterResult{}, err
		}
		envelope, err := datatype.SealStreamRef(builtins.Catalog, input.Type(), streamHandle)
		if err != nil {
			return compiler.AdapterResult{}, err
		}
		return compiler.AdapterResult{Outputs: map[string]datatype.ValueEnvelope{"stream": envelope}}, nil
	}
}

func streamToBlob(builtins nodes.Builtins) compiler.Adapter {
	return func(ctx context.Context, invocation compiler.Invocation) (_ compiler.AdapterResult, runErr error) {
		counters := map[string]int64{}
		defer func() {
			runErr = errors.Join(runErr, recordAdapterOutcome(ctx, invocation, compiler.AdapterAction{
				EffectID: nodes.StreamToBlobEffectID, Action: "conversion.blob-committed",
				SummaryCode: "conversion.stream-to-blob", Counters: counters,
			}, "conversion.stream_to_blob_failed", runErr))
		}()
		input, ok := invocation.Inputs["stream"]
		if !ok {
			return compiler.AdapterResult{}, errors.New("stream-to-blob input is missing")
		}
		streamHandle, ok := input.StreamRef()
		if !ok {
			return compiler.AdapterResult{}, errors.New("stream-to-blob input is not a StreamRef")
		}
		mediaType, ok := invocation.Config["mediaType"].(string)
		if !ok {
			return compiler.AdapterResult{}, errors.New("stream-to-blob media type is missing")
		}
		streamSession, blobSession := invocation.Sessions["stream"], invocation.Sessions["blob-write"]
		if streamSession == nil || blobSession == nil {
			return compiler.AdapterResult{}, errors.New("stream-to-blob capability session is missing")
		}
		writeConfig, err := artifact.Marshal(blob.WriteConfig{MediaType: mediaType})
		if err != nil {
			return compiler.AdapterResult{}, err
		}
		writer, err := blobSession.Open(ctx, []string{blob.OperationAppend, blob.OperationCancel, blob.OperationCommit}, writeConfig)
		if err != nil {
			return compiler.AdapterResult{}, err
		}
		committed := false
		defer func() {
			if !committed {
				_, _ = blobSession.Invoke(context.Background(), writer, blob.OperationCancel, nil)
				_ = blobSession.Drop(context.Background(), writer)
			}
		}()
		for {
			chunk, err := streamSession.Invoke(ctx, streamHandle, stream.OperationReceive, nil)
			if errors.Is(err, io.EOF) {
				break
			}
			if err != nil {
				return compiler.AdapterResult{}, err
			}
			if len(chunk) == 0 {
				continue
			}
			if _, err := blobSession.Invoke(ctx, writer, blob.OperationAppend, chunk); err != nil {
				return compiler.AdapterResult{}, err
			}
		}
		rawRef, err := blobSession.Invoke(ctx, writer, blob.OperationCommit, nil)
		if err != nil {
			return compiler.AdapterResult{}, err
		}
		var ref blob.BlobRef
		if err := json.Unmarshal(rawRef, &ref); err != nil {
			return compiler.AdapterResult{}, fmt.Errorf("decode committed BlobRef: %w", err)
		}
		if err := ref.Validate(); err != nil {
			return compiler.AdapterResult{}, fmt.Errorf("validate committed BlobRef: %w", err)
		}
		counters["bytes"] = ref.Size
		committed = true
		envelope, err := datatype.SealBlobRef(builtins.Catalog, input.Type(), ref)
		if err != nil {
			return compiler.AdapterResult{}, err
		}
		return compiler.AdapterResult{Outputs: map[string]datatype.ValueEnvelope{"blob": envelope}}, nil
	}
}

func recordAdapterOutcome(ctx context.Context, invocation compiler.Invocation, action compiler.AdapterAction, failureCode string, runErr error) error {
	if invocation.RecordAction == nil {
		return errors.New("adapter action recorder is required")
	}
	switch {
	case errors.Is(runErr, context.Canceled), errors.Is(runErr, context.DeadlineExceeded), ctx.Err() != nil:
		action.Outcome = run.ActionCancelled
	case runErr != nil:
		action.Outcome = run.ActionFailed
		action.ErrorCode = failureCode
		var failure *compiler.NodeFailure
		if errors.As(runErr, &failure) && failure.Code != "" {
			action.ErrorCode = failure.Code
		}
	default:
		action.Outcome = run.ActionSucceeded
	}
	return invocation.RecordAction(context.WithoutCancel(ctx), action)
}
