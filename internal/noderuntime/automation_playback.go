package noderuntime

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/yottaapp/yotta/internal/artifact"
	"github.com/yottaapp/yotta/internal/automation/installed"
	"github.com/yottaapp/yotta/internal/blob"
	"github.com/yottaapp/yotta/internal/nodes"
	"github.com/yottaapp/yotta/internal/services/inputclip"
	"github.com/yottaapp/yotta/internal/workflow/compiler"
)

const playbackReadChunkBytes = int64(64 << 10)

type playbackCommands struct {
	Now  func() time.Time
	Wait func(time.Duration) error
	Play func(installed.PlaybackEvent) error
}

func playInputClip() compiler.Adapter {
	return func(ctx context.Context, invocation compiler.Invocation) (_ compiler.AdapterResult, runErr error) {
		counters := map[string]int64{}
		defer func() {
			runErr = errors.Join(runErr, recordAdapterOutcome(ctx, invocation, compiler.AdapterAction{
				EffectID: nodes.PlayInputClipEffectID, Action: "automation.play-input-clip", SummaryCode: "automation.play-input-clip", Counters: counters,
			}, installed.CodePlaybackFailed, runErr))
		}()

		clip, ref, err := readInputClip(ctx, invocation)
		if err != nil {
			return compiler.AdapterResult{}, inputClipFailure(err)
		}
		counters["blob_bytes"] = ref.Size
		counters["events"] = int64(len(clip.Events))
		counters["duration_ms"] = int64(clip.DurationUs / 1000)
		err = runPlaybackSession(ctx, invocation, func(commands playbackCommands) error {
			return playInputClipTimeline(clip.Events, clip.Meta, commands)
		})
		if err != nil {
			return compiler.AdapterResult{}, err
		}
		return compiler.AdapterResult{ExecOutputs: []string{"completed"}}, nil
	}
}

func runPlaybackSession(ctx context.Context, invocation compiler.Invocation, sequence func(playbackCommands) error) (runErr error) {
	targetSession := invocation.Sessions["target"]
	if targetSession == nil {
		return automationFailure(installed.CodeContractViolation, errors.New("automation playback capability session is missing"))
	}
	handle, err := targetSession.Open(ctx, installed.PlaybackOperations(), []byte(`{}`))
	if err != nil {
		return mapAutomationFailure(err)
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
	commands := playbackCommands{
		Now: invocation.MonotonicNow,
		Wait: func(duration time.Duration) error {
			if duration <= 0 {
				return nil
			}
			if invocation.Wait == nil {
				return automationFailure(installed.CodeContractViolation, errors.New("playback scheduler is missing"))
			}
			return invocation.Wait(ctx, duration)
		},
		Play: func(event installed.PlaybackEvent) error {
			payload, err := artifact.Marshal(event)
			if err != nil {
				return automationFailure(installed.CodeContractViolation, err)
			}
			raw, err := targetSession.Invoke(ctx, handle, installed.OperationPlayEvent, payload)
			if err != nil {
				return mapAutomationFailure(err)
			}
			if err := installed.OpenEffectResponse(raw); err != nil {
				return mapAutomationFailure(err)
			}
			return nil
		},
	}
	if err := sequence(commands); err != nil {
		return err
	}
	releasePayload, err := artifact.Marshal(struct{}{})
	if err != nil {
		return automationFailure(installed.CodeContractViolation, err)
	}
	raw, err := targetSession.Invoke(ctx, handle, installed.OperationReleaseHeld, releasePayload)
	if err != nil {
		return mapAutomationFailure(err)
	}
	if err := installed.OpenEffectResponse(raw); err != nil {
		return mapAutomationFailure(err)
	}
	released = true
	return nil
}

func playInputClipTimeline(events []inputclip.Event, meta inputclip.ClipMeta, commands playbackCommands) error {
	if commands.Now == nil || commands.Wait == nil || commands.Play == nil {
		return automationFailure(installed.CodeContractViolation, errors.New("playback clock or command is missing"))
	}
	started := commands.Now()
	for _, event := range events {
		deadline := started.Add(time.Duration(event.TUs) * time.Microsecond)
		if remaining := deadline.Sub(commands.Now()); remaining > 0 {
			if err := commands.Wait(remaining); err != nil {
				return err
			}
		}
		if err := commands.Play(playbackEvent(event, meta)); err != nil {
			return err
		}
	}
	return nil
}

func readInputClip(ctx context.Context, invocation compiler.Invocation) (*inputclip.InputClip, blob.BlobRef, error) {
	carrier, ref, err := readPlaybackBlob(ctx, invocation, "clip", inputclip.MediaType, inputclip.MaxEncodedInputClipBytes)
	if err != nil {
		return nil, blob.BlobRef{}, err
	}
	clip, err := inputclip.Decode(bytes.NewReader(carrier))
	if err != nil {
		return nil, blob.BlobRef{}, fmt.Errorf("decode input clip: %w", err)
	}
	return clip, ref, nil
}

func readPlaybackBlob(ctx context.Context, invocation compiler.Invocation, inputID, mediaType string, maxBytes int) ([]byte, blob.BlobRef, error) {
	input, ok := invocation.Inputs[inputID]
	if !ok {
		return nil, blob.BlobRef{}, fmt.Errorf("%s is missing", inputID)
	}
	ref, ok := input.BlobRef()
	if !ok || ref.Validate() != nil || ref.MediaType != mediaType || ref.Size <= 0 || ref.Size > int64(maxBytes) {
		return nil, blob.BlobRef{}, fmt.Errorf("%s BlobRef is invalid", inputID)
	}
	session := invocation.Sessions["blob-read"]
	if session == nil {
		return nil, blob.BlobRef{}, fmt.Errorf("%s blob-read capability session is missing", inputID)
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
		length := min(playbackReadChunkBytes, ref.Size-offset)
		payload, err := artifact.Marshal(blob.RangeRequest{Offset: offset, Length: length})
		if err != nil {
			return nil, blob.BlobRef{}, err
		}
		chunk, err := session.Invoke(ctx, handle, blob.OperationReadRange, payload)
		if err != nil {
			return nil, blob.BlobRef{}, err
		}
		if int64(len(chunk)) != length {
			return nil, blob.BlobRef{}, fmt.Errorf("blob provider returned an invalid %s chunk length", inputID)
		}
		_, _ = carrier.Write(chunk)
		offset += length
	}
	return carrier.Bytes(), ref, nil
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
	return &compiler.NodeFailure{Code: nodes.InputClipInvalidCode, Output: "failed", Cause: fmt.Errorf("input clip: %w", cause)}
}
