package nodes31runtime

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/yottaapp/yotta/internal/artifact"
	"github.com/yottaapp/yotta/internal/automation/installed"
	"github.com/yottaapp/yotta/internal/blob"
	"github.com/yottaapp/yotta/internal/datatype"
	"github.com/yottaapp/yotta/internal/nodes31"
	"github.com/yottaapp/yotta/internal/workflow/compiler"
)

const captureChunkBytes = int64(64 << 10)

func captureWindow(builtins nodes31.Builtins) compiler.Adapter {
	return func(ctx context.Context, invocation compiler.Invocation) (_ compiler.AdapterResult, runErr error) {
		counters := map[string]int64{}
		defer func() {
			runErr = errors.Join(runErr, recordAdapterOutcome(ctx, invocation, compiler.AdapterAction{
				EffectID: nodes31.CaptureWindowEffectID, Action: "automation.capture-window", SummaryCode: "automation.capture-window", Counters: counters,
			}, installed.CodeCaptureFailed, runErr))
		}()

		targetSession, blobSession := invocation.Sessions["target"], invocation.Sessions["blob-write"]
		if targetSession == nil || blobSession == nil {
			return compiler.AdapterResult{}, automationFailure(installed.CodeContractViolation, errors.New("capture window capability session is missing"))
		}
		captureHandle, err := targetSession.Open(ctx, installed.CaptureOperations(), []byte(`{}`))
		if err != nil {
			return compiler.AdapterResult{}, mapAutomationFailure(err)
		}
		defer func() { runErr = errors.Join(runErr, targetSession.Drop(context.WithoutCancel(ctx), captureHandle)) }()

		capturePayload, err := artifact.Marshal(struct{}{})
		if err != nil {
			return compiler.AdapterResult{}, automationFailure(installed.CodeContractViolation, err)
		}
		rawResponse, err := targetSession.Invoke(ctx, captureHandle, installed.OperationCapture, capturePayload)
		if err != nil {
			return compiler.AdapterResult{}, mapAutomationFailure(err)
		}
		response, err := installed.OpenCaptureResponse(rawResponse)
		if err != nil {
			return compiler.AdapterResult{}, mapAutomationFailure(err)
		}

		writeConfig, err := artifact.Marshal(blob.WriteConfig{MediaType: response.MediaType})
		if err != nil {
			return compiler.AdapterResult{}, automationFailure(installed.CodeContractViolation, err)
		}
		writer, err := blobSession.Open(ctx, []string{blob.OperationAppend, blob.OperationCancel, blob.OperationCommit}, writeConfig)
		if err != nil {
			return compiler.AdapterResult{}, automationFailure(installed.CodeCaptureFailed, err)
		}
		committed := false
		defer func() {
			if !committed {
				_, _ = blobSession.Invoke(context.Background(), writer, blob.OperationCancel, nil)
			}
			runErr = errors.Join(runErr, blobSession.Drop(context.WithoutCancel(ctx), writer))
		}()

		for offset := int64(0); offset < response.Size; {
			length := min(captureChunkBytes, response.Size-offset)
			rangePayload, marshalErr := artifact.Marshal(installed.CaptureRangeRequest{Offset: offset, Length: length})
			if marshalErr != nil {
				return compiler.AdapterResult{}, automationFailure(installed.CodeContractViolation, marshalErr)
			}
			chunk, invokeErr := targetSession.Invoke(ctx, captureHandle, installed.OperationReadCapture, rangePayload)
			if invokeErr != nil {
				return compiler.AdapterResult{}, mapAutomationFailure(invokeErr)
			}
			if int64(len(chunk)) != length {
				return compiler.AdapterResult{}, automationFailure(installed.CodeContractViolation, errors.New("capture provider returned an invalid chunk length"))
			}
			if _, invokeErr := blobSession.Invoke(ctx, writer, blob.OperationAppend, chunk); invokeErr != nil {
				return compiler.AdapterResult{}, automationFailure(installed.CodeCaptureFailed, invokeErr)
			}
			offset += length
			counters["chunks"]++
		}

		rawRef, err := blobSession.Invoke(ctx, writer, blob.OperationCommit, nil)
		if err != nil {
			return compiler.AdapterResult{}, automationFailure(installed.CodeCaptureFailed, err)
		}
		var ref blob.BlobRef
		if err := json.Unmarshal(rawRef, &ref); err != nil {
			return compiler.AdapterResult{}, automationFailure(installed.CodeContractViolation, fmt.Errorf("decode capture BlobRef: %w", err))
		}
		if err := ref.Validate(); err != nil || ref.MediaType != response.MediaType || ref.Size != response.Size {
			return compiler.AdapterResult{}, automationFailure(installed.CodeContractViolation, errors.New("committed capture BlobRef does not match the capture response"))
		}
		canonicalRef, err := artifact.Marshal(ref)
		if err != nil || !bytes.Equal(canonicalRef, rawRef) {
			return compiler.AdapterResult{}, automationFailure(installed.CodeContractViolation, errors.New("committed capture BlobRef is not canonical"))
		}
		outputType, ok := invocation.OutputTypes["image"]
		if !ok {
			return compiler.AdapterResult{}, automationFailure(installed.CodeContractViolation, errors.New("capture image output type is missing"))
		}
		envelope, err := datatype.SealBlobRef(builtins.Catalog, outputType, ref)
		if err != nil {
			return compiler.AdapterResult{}, automationFailure(installed.CodeContractViolation, err)
		}
		committed = true
		counters["bytes"] = ref.Size
		return compiler.AdapterResult{Outputs: map[string]datatype.ValueEnvelope{"image": envelope}, ExecOutputs: []string{"completed"}}, nil
	}
}
