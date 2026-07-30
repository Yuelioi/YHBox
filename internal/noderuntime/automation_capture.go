package noderuntime

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
	"github.com/yottaapp/yotta/internal/nodeadapter"
	"github.com/yottaapp/yotta/internal/nodes"
)

const captureChunkBytes = int64(64 << 10)

func captureWindow(builtins nodes.Builtins) nodeadapter.Adapter {
	return func(ctx context.Context, invocation nodeadapter.Invocation) (_ nodeadapter.AdapterResult, runErr error) {
		counters := map[string]int64{}
		defer func() {
			runErr = errors.Join(runErr, recordAdapterOutcome(ctx, invocation, nodeadapter.AdapterAction{
				EffectID: nodes.CaptureWindowEffectID, Action: "automation.capture-window", SummaryCode: "automation.capture-window", Counters: counters,
			}, installed.CodeCaptureFailed, runErr))
		}()

		blobSession := invocation.Sessions["blob-write"]
		if blobSession == nil {
			return nodeadapter.AdapterResult{}, automationFailure(installed.CodeContractViolation, errors.New("capture blob session is missing"))
		}
		captureHandle, err := openConfiguredTarget(ctx, invocation, installed.KindCapture, installed.CaptureOperations())
		if err != nil {
			return nodeadapter.AdapterResult{}, mapAutomationFailure(err)
		}
		defer func() {
			runErr = errors.Join(runErr, invocation.Targets.Drop(context.WithoutCancel(ctx), captureHandle))
		}()

		capturePayload, err := artifact.Marshal(struct{}{})
		if err != nil {
			return nodeadapter.AdapterResult{}, automationFailure(installed.CodeContractViolation, err)
		}
		rawResponse, err := invocation.Targets.Invoke(ctx, captureHandle, installed.OperationCapture, capturePayload)
		if err != nil {
			return nodeadapter.AdapterResult{}, mapAutomationFailure(err)
		}
		response, err := installed.OpenCaptureResponse(rawResponse)
		if err != nil {
			return nodeadapter.AdapterResult{}, mapAutomationFailure(err)
		}

		writeConfig, err := artifact.Marshal(blob.WriteConfig{MediaType: response.MediaType})
		if err != nil {
			return nodeadapter.AdapterResult{}, automationFailure(installed.CodeContractViolation, err)
		}
		writer, err := blobSession.Open(ctx, []string{blob.OperationAppend, blob.OperationCancel, blob.OperationCommit}, writeConfig)
		if err != nil {
			return nodeadapter.AdapterResult{}, automationFailure(installed.CodeCaptureFailed, err)
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
				return nodeadapter.AdapterResult{}, automationFailure(installed.CodeContractViolation, marshalErr)
			}
			chunk, invokeErr := invocation.Targets.Invoke(ctx, captureHandle, installed.OperationReadCapture, rangePayload)
			if invokeErr != nil {
				return nodeadapter.AdapterResult{}, mapAutomationFailure(invokeErr)
			}
			if int64(len(chunk)) != length {
				return nodeadapter.AdapterResult{}, automationFailure(installed.CodeContractViolation, errors.New("capture provider returned an invalid chunk length"))
			}
			if _, invokeErr := blobSession.Invoke(ctx, writer, blob.OperationAppend, chunk); invokeErr != nil {
				return nodeadapter.AdapterResult{}, automationFailure(installed.CodeCaptureFailed, invokeErr)
			}
			offset += length
			counters["chunks"]++
		}

		rawRef, err := blobSession.Invoke(ctx, writer, blob.OperationCommit, nil)
		if err != nil {
			return nodeadapter.AdapterResult{}, automationFailure(installed.CodeCaptureFailed, err)
		}
		var ref blob.BlobRef
		if err := json.Unmarshal(rawRef, &ref); err != nil {
			return nodeadapter.AdapterResult{}, automationFailure(installed.CodeContractViolation, fmt.Errorf("decode capture BlobRef: %w", err))
		}
		if err := ref.Validate(); err != nil || ref.MediaType != response.MediaType || ref.Size != response.Size {
			return nodeadapter.AdapterResult{}, automationFailure(installed.CodeContractViolation, errors.New("committed capture BlobRef does not match the capture response"))
		}
		canonicalRef, err := artifact.Marshal(ref)
		if err != nil || !bytes.Equal(canonicalRef, rawRef) {
			return nodeadapter.AdapterResult{}, automationFailure(installed.CodeContractViolation, errors.New("committed capture BlobRef is not canonical"))
		}
		outputType, ok := invocation.OutputTypes["image"]
		if !ok {
			return nodeadapter.AdapterResult{}, automationFailure(installed.CodeContractViolation, errors.New("capture image output type is missing"))
		}
		envelope, err := datatype.SealBlobRef(builtins.Catalog, outputType, ref)
		if err != nil {
			return nodeadapter.AdapterResult{}, automationFailure(installed.CodeContractViolation, err)
		}
		committed = true
		counters["bytes"] = ref.Size
		return nodeadapter.AdapterResult{Outputs: map[string]datatype.ValueEnvelope{"image": envelope}, ExecOutputs: []string{"completed"}}, nil
	}
}
