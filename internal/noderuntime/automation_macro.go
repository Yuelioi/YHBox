package noderuntime

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/yottaapp/yotta/internal/automation/installed"
	"github.com/yottaapp/yotta/internal/nodes"
	"github.com/yottaapp/yotta/internal/services/macro"
	"github.com/yottaapp/yotta/internal/workflow/compiler"
)

func playMacro() compiler.Adapter {
	return func(ctx context.Context, invocation compiler.Invocation) (_ compiler.AdapterResult, runErr error) {
		counters := map[string]int64{}
		defer func() {
			runErr = errors.Join(runErr, recordAdapterOutcome(ctx, invocation, compiler.AdapterAction{
				EffectID: nodes.PlayMacroEffectID, Action: "automation.play-macro", SummaryCode: "automation.play-macro", Counters: counters,
			}, installed.CodePlaybackFailed, runErr))
		}()
		carrier, ref, err := readPlaybackBlob(ctx, invocation, "macro", macro.MediaType, macro.MaxEncodedMacroBytes)
		if err != nil {
			return compiler.AdapterResult{}, macroFailure(err)
		}
		document, err := macro.Decode(bytes.NewReader(carrier))
		if err != nil {
			return compiler.AdapterResult{}, macroFailure(err)
		}
		analysis := macro.Analyze(document)
		counters["blob_bytes"] = ref.Size
		counters["actions"] = int64(len(document.Actions))
		counters["duration_ms"] = int64(analysis.DurationUs / 1000)
		if err := runPlaybackSession(ctx, invocation, func(commands playbackCommands) error {
			for _, action := range document.Actions {
				if err := playMacroAction(action, commands); err != nil {
					return err
				}
			}
			return nil
		}); err != nil {
			return compiler.AdapterResult{}, err
		}
		return compiler.AdapterResult{ExecOutputs: []string{"completed"}}, nil
	}
}

func playMacroAction(action macro.Action, commands playbackCommands) error {
	point := func() *installed.Point {
		return &installed.Point{X: action.Point.X, Y: action.Point.Y, Unit: action.Point.Unit}
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
		if err := commands.Play(installed.PlaybackEvent{Kind: installed.PlaybackButtonDown, Point: point(), Button: action.Button}); err != nil {
			return err
		}
		if err := commands.Wait(time.Duration(action.DurationUs) * time.Microsecond); err != nil {
			return err
		}
		return commands.Play(installed.PlaybackEvent{Kind: installed.PlaybackButtonUp, Point: point(), Button: action.Button})
	case macro.ActionScroll:
		return commands.Play(installed.PlaybackEvent{Kind: installed.PlaybackScroll, Point: point(), Notches: int64(action.Notches)})
	default:
		return fmt.Errorf("unsupported macro action %q", action.Kind)
	}
}

func macroFailure(cause error) error {
	return &compiler.NodeFailure{Code: nodes.MacroInvalidCode, Output: "failed", Cause: fmt.Errorf("macro: %w", cause)}
}
