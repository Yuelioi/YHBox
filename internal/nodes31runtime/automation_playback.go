package nodes31runtime

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/yottaapp/yotta/internal/artifact"
	"github.com/yottaapp/yotta/internal/automation/installed"
	"github.com/yottaapp/yotta/internal/blob"
	"github.com/yottaapp/yotta/internal/nodes31"
	"github.com/yottaapp/yotta/internal/services/inputclip"
	"github.com/yottaapp/yotta/internal/workflow/compiler"
)

const inputClipReadChunkBytes = int64(64 << 10)

func playInputClip() compiler.Adapter {
	return func(ctx context.Context, invocation compiler.Invocation) (_ compiler.AdapterResult, runErr error) {
		counters := map[string]int64{}
		defer func() {
			runErr = errors.Join(runErr, recordAdapterOutcome(ctx, invocation, compiler.AdapterAction{
				EffectID: nodes31.PlayInputClipEffectID, Action: "automation.play-input-clip", SummaryCode: "automation.play-input-clip", Counters: counters,
			}, installed.CodePlaybackFailed, runErr))
		}()

		clip, ref, err := readInputClip(ctx, invocation)
		if err != nil {
			return compiler.AdapterResult{}, inputClipFailure(err)
		}
		counters["blob_bytes"] = ref.Size
		counters["events"] = int64(len(clip.Events))
		counters["duration_ms"] = int64(clip.DurationUs / 1000)

		targetSession := invocation.Sessions["target"]
		if targetSession == nil {
			return compiler.AdapterResult{}, automationFailure(installed.CodeContractViolation, errors.New("automation playback capability session is missing"))
		}
		handle, err := targetSession.Open(ctx, installed.PlaybackOperations(), []byte(`{}`))
		if err != nil {
			return compiler.AdapterResult{}, mapAutomationFailure(err)
		}
		released := false
		defer func() {
			cleanupCtx := context.WithoutCancel(ctx)
			if !released {
				payload, marshalErr := artifact.Marshal(struct{}{})
				if marshalErr == nil {
					_, _ = targetSession.Invoke(cleanupCtx, handle, installed.OperationReleaseHeld, payload)
				}
			}
			runErr = errors.Join(runErr, targetSession.Drop(cleanupCtx, handle))
		}()

		var previousUs uint64
		for _, event := range clip.Events {
			if event.TUs > previousUs {
				if invocation.Wait == nil {
					return compiler.AdapterResult{}, automationFailure(installed.CodeContractViolation, errors.New("playback scheduler is missing"))
				}
				if err := invocation.Wait(ctx, time.Duration(event.TUs-previousUs)*time.Microsecond); err != nil {
					return compiler.AdapterResult{}, err
				}
			}
			previousUs = event.TUs
			payload, err := artifact.Marshal(playbackEvent(event, clip.Meta))
			if err != nil {
				return compiler.AdapterResult{}, automationFailure(installed.CodeContractViolation, err)
			}
			raw, err := targetSession.Invoke(ctx, handle, installed.OperationPlayEvent, payload)
			if err != nil {
				return compiler.AdapterResult{}, mapAutomationFailure(err)
			}
			if err := installed.OpenEffectResponse(raw); err != nil {
				return compiler.AdapterResult{}, mapAutomationFailure(err)
			}
		}

		releasePayload, err := artifact.Marshal(struct{}{})
		if err != nil {
			return compiler.AdapterResult{}, automationFailure(installed.CodeContractViolation, err)
		}
		raw, err := targetSession.Invoke(ctx, handle, installed.OperationReleaseHeld, releasePayload)
		if err != nil {
			return compiler.AdapterResult{}, mapAutomationFailure(err)
		}
		if err := installed.OpenEffectResponse(raw); err != nil {
			return compiler.AdapterResult{}, mapAutomationFailure(err)
		}
		released = true
		return compiler.AdapterResult{ExecOutputs: []string{"completed"}}, nil
	}
}

func readInputClip(ctx context.Context, invocation compiler.Invocation) (*inputclip.InputClip, blob.BlobRef, error) {
	input, ok := invocation.Inputs["clip"]
	if !ok {
		return nil, blob.BlobRef{}, errors.New("input clip is missing")
	}
	ref, ok := input.BlobRef()
	if !ok || ref.Validate() != nil || ref.MediaType != inputclip.MediaType || ref.Size <= 0 || ref.Size > inputclip.MaxEncodedInputClipBytes {
		return nil, blob.BlobRef{}, errors.New("input clip BlobRef is invalid")
	}
	session := invocation.Sessions["blob-read"]
	if session == nil {
		return nil, blob.BlobRef{}, errors.New("input clip blob-read capability session is missing")
	}
	config, err := artifact.Marshal(blob.ReadConfig{Blob: ref})
	if err != nil {
		return nil, blob.BlobRef{}, err
	}
	handle, err := session.Open(ctx, []string{blob.OperationReadRange}, config)
	if err != nil {
		return nil, blob.BlobRef{}, err
	}
	defer func() { _ = session.Drop(context.WithoutCancel(ctx), handle) }()

	var carrier bytes.Buffer
	carrier.Grow(int(ref.Size))
	for offset := int64(0); offset < ref.Size; {
		length := min(inputClipReadChunkBytes, ref.Size-offset)
		payload, err := artifact.Marshal(blob.RangeRequest{Offset: offset, Length: length})
		if err != nil {
			return nil, blob.BlobRef{}, err
		}
		chunk, err := session.Invoke(ctx, handle, blob.OperationReadRange, payload)
		if err != nil {
			return nil, blob.BlobRef{}, err
		}
		if int64(len(chunk)) != length {
			return nil, blob.BlobRef{}, errors.New("blob provider returned an invalid input clip chunk length")
		}
		_, _ = carrier.Write(chunk)
		offset += length
	}
	clip, err := inputclip.Decode(bytes.NewReader(carrier.Bytes()))
	if err != nil {
		return nil, blob.BlobRef{}, fmt.Errorf("decode input clip: %w", err)
	}
	return clip, ref, nil
}

func playbackEvent(event inputclip.Event, meta inputclip.ClipMeta) installed.PlaybackEvent {
	point := func() *installed.Point {
		return &installed.Point{X: float64(event.B) / float64(meta.BaseResolution[0]), Y: float64(event.C) / float64(meta.BaseResolution[1]), Unit: "ratio"}
	}
	button := func() string {
		switch event.A {
		case 1:
			return "middle"
		case 2:
			return "right"
		default:
			return "left"
		}
	}
	switch event.Type {
	case inputclip.EventTypeKeyDown:
		return installed.PlaybackEvent{Kind: installed.PlaybackKeyDown, KeyCode: uint32(event.A)}
	case inputclip.EventTypeKeyUp:
		return installed.PlaybackEvent{Kind: installed.PlaybackKeyUp, KeyCode: uint32(event.A)}
	case inputclip.EventTypeMouseBtnDown:
		return installed.PlaybackEvent{Kind: installed.PlaybackButtonDown, Point: point(), Button: button()}
	case inputclip.EventTypeMouseBtnUp:
		return installed.PlaybackEvent{Kind: installed.PlaybackButtonUp, Point: point(), Button: button()}
	case inputclip.EventTypeMouseMove:
		return installed.PlaybackEvent{Kind: installed.PlaybackMove, Point: point()}
	case inputclip.EventTypeRawDelta:
		return installed.PlaybackEvent{Kind: installed.PlaybackMoveRelative, DeltaX: int64(event.B), DeltaY: int64(event.C), SourceCounts360: int64(meta.MouseCounts360)}
	case inputclip.EventTypeScroll:
		return installed.PlaybackEvent{Kind: installed.PlaybackScroll, Point: point(), Notches: int64(event.A)}
	default:
		panic("validated input clip contains an unsupported event type")
	}
}

func inputClipFailure(cause error) error {
	return &compiler.NodeFailure{Code: nodes31.InputClipInvalidCode, Output: "failed", Cause: fmt.Errorf("input clip: %w", cause)}
}
