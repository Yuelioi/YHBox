package noderuntime

import (
	"context"
	"errors"

	"github.com/yottaapp/yotta/internal/artifact"
	"github.com/yottaapp/yotta/internal/automation/installed"
	"github.com/yottaapp/yotta/internal/nodeadapter"
	"github.com/yottaapp/yotta/internal/resource"
)

func readAutomationCaptureBytes(
	ctx context.Context,
	invocation nodeadapter.Invocation,
	handle resource.Handle,
	size int64,
	contractFailure func(error) error,
) ([]byte, error) {
	var assembled []byte
	if size > installed.MaxCaptureChunkBytes {
		assembled = make([]byte, size)
	}
	for offset := int64(0); offset < size; {
		length := min(installed.MaxCaptureChunkBytes, size-offset)
		payload, err := artifact.Marshal(installed.CaptureRangeRequest{Offset: offset, Length: length})
		if err != nil {
			return nil, contractFailure(err)
		}
		chunk, err := invocation.Targets.Invoke(ctx, handle, installed.OperationReadCapture, payload)
		if err != nil {
			return nil, mapAutomationFailure(err)
		}
		if int64(len(chunk)) != length {
			return nil, contractFailure(errors.New("capture provider returned an invalid chunk length"))
		}
		if assembled == nil {
			return chunk, nil
		}
		copy(assembled[offset:offset+length], chunk)
		offset += length
	}
	return assembled, nil
}
