package noderuntime

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/yottaapp/yotta/internal/automation/installed"
	"github.com/yottaapp/yotta/internal/nodeadapter"
	"github.com/yottaapp/yotta/internal/nodes"
	"github.com/yottaapp/yotta/internal/services/macro"
)

func playMacro() nodeadapter.Adapter {
	return func(ctx context.Context, invocation nodeadapter.Invocation) (_ nodeadapter.AdapterResult, runErr error) {
		counters := map[string]int64{}
		defer func() {
			runErr = errors.Join(runErr, recordAdapterOutcome(ctx, invocation, nodeadapter.AdapterAction{
				EffectID: nodes.PlayMacroEffectID, Action: "automation.play-macro", SummaryCode: "automation.play-macro", Counters: counters,
			}, installed.CodePlaybackFailed, runErr))
		}()
		carrier, ref, err := readBlobInput(ctx, invocation, "macro", macro.MediaType, macro.MaxEncodedMacroBytes)
		if err != nil {
			return nodeadapter.AdapterResult{}, macroFailure(err)
		}
		document, err := macro.Decode(bytes.NewReader(carrier))
		if err != nil {
			return nodeadapter.AdapterResult{}, macroFailure(err)
		}
		analysis := macro.Analyze(document)
		plan, err := macro.ExecutionPlan(document)
		if err != nil {
			return nodeadapter.AdapterResult{}, macroFailure(err)
		}
		counters["blob_bytes"] = ref.Size
		counters["actions"] = int64(len(document.Actions))
		counters["planned_actions"] = int64(len(plan))
		counters["duration_ms"] = int64(analysis.DurationUs / 1000)
		if err := runPlaybackSession(ctx, invocation, func(commands playbackCommands) error {
			for _, action := range plan {
				if err := playMacroAction(action, commands); err != nil {
					return err
				}
			}
			return nil
		}); err != nil {
			return nodeadapter.AdapterResult{}, err
		}
		return nodeadapter.AdapterResult{ExecOutputs: []string{"completed"}}, nil
	}
}

func macroClickEvent(point *macro.Point, button string, durationUs uint64) installed.PlaybackEvent {
	durationMs := int64((durationUs + uint64(time.Millisecond/time.Microsecond) - 1) / uint64(time.Millisecond/time.Microsecond))
	return installed.PlaybackEvent{
		Kind:   installed.PlaybackClick,
		Point:  &installed.Point{X: point.X, Y: point.Y, Unit: point.Unit},
		Button: button, DurationMilliseconds: durationMs,
	}
}

func macroPoint(point *macro.Point) *installed.Point {
	return &installed.Point{X: point.X, Y: point.Y, Unit: point.Unit}
}

func macroDurationMilliseconds(durationUs uint64) int64 {
	return int64((durationUs + uint64(time.Millisecond/time.Microsecond) - 1) / uint64(time.Millisecond/time.Microsecond))
}

func playMacroAction(action macro.Action, commands playbackCommands) error {
	point := func() *installed.Point {
		return macroPoint(action.Point)
	}
	keyCode := func() (uint32, error) {
		_, code, ok := macro.CanonicalKey(action.Key)
		if !ok {
			return 0, fmt.Errorf("unsupported macro key %q", action.Key)
		}
		return code, nil
	}
	switch action.Kind {
	case macro.ActionSleep:
		return commands.Wait(time.Duration(action.DurationUs) * time.Microsecond)
	case macro.ActionKeyDown, macro.ActionKeyUp:
		code, err := keyCode()
		if err != nil {
			return err
		}
		kind := installed.PlaybackKeyDown
		if action.Kind == macro.ActionKeyUp {
			kind = installed.PlaybackKeyUp
		}
		return commands.Play(installed.PlaybackEvent{Kind: kind, KeyCode: code})
	case macro.ActionMouseDown, macro.ActionMouseUp:
		kind := installed.PlaybackButtonDown
		if action.Kind == macro.ActionMouseUp {
			kind = installed.PlaybackButtonUp
		}
		return commands.Play(installed.PlaybackEvent{Kind: kind, Point: point(), Button: action.Button})
	case macro.ActionClick:
		return commands.Play(macroClickEvent(action.Point, action.Button, action.DurationUs))
	case macro.ActionMove:
		return commands.Play(installed.PlaybackEvent{
			Kind: installed.PlaybackMove, Point: point(), Motion: action.Motion,
			DurationMilliseconds: macroDurationMilliseconds(action.DurationUs),
		})
	case macro.ActionDrag:
		return commands.Play(installed.PlaybackEvent{
			Kind: installed.PlaybackDrag, From: macroPoint(action.From), Point: point(), Button: action.Button,
			Motion: action.Motion, DurationMilliseconds: macroDurationMilliseconds(action.DurationUs),
		})
	case macro.ActionScroll:
		return commands.Play(installed.PlaybackEvent{Kind: installed.PlaybackScroll, Point: point(), Notches: int64(action.Notches)})
	default:
		return fmt.Errorf("unsupported macro action %q", action.Kind)
	}
}

func macroFailure(cause error) error {
	return &nodeadapter.NodeFailure{Code: nodes.MacroInvalidCode, Output: "failed", Cause: fmt.Errorf("macro: %w", cause)}
}
