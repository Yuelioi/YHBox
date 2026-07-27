package noderuntime

import (
	"context"
	"errors"
	"fmt"
	"image"
	"time"

	"github.com/yottaapp/yotta/internal/automation/installed"
	"github.com/yottaapp/yotta/internal/nodeadapter"
	"github.com/yottaapp/yotta/internal/nodes"
	visionpkg "github.com/yottaapp/yotta/pkg/vision"
)

type frameDifference struct {
	changedRatio   float64
	meanDifference float64
}

type frameSignature struct {
	grid []uint8
}

func automationObservation(builtins nodes.Builtins, nodeTypeID string) nodeadapter.Adapter {
	return func(ctx context.Context, invocation nodeadapter.Invocation) (_ nodeadapter.AdapterResult, runErr error) {
		counters := map[string]int64{}
		effectID, action := nodes.WaitChangeEffectID, "automation.wait-change"
		if nodeTypeID == nodes.WaitStableNodeID {
			effectID, action = nodes.WaitStableEffectID, "automation.wait-stable"
		}
		defer func() {
			runErr = errors.Join(runErr, recordAdapterOutcome(ctx, invocation, nodeadapter.AdapterAction{
				EffectID: effectID, Action: action, SummaryCode: action, Counters: counters,
			}, nodes.ObservationFailedCode, runErr))
		}()

		threshold, err := numberInput(invocation, "threshold")
		if err != nil || threshold < 0 || threshold > 1 {
			return nodeadapter.AdapterResult{}, observationFailure(installed.CodeInvalidRequest, errors.Join(err, errors.New("observation threshold must be between 0 and 1")))
		}
		timeout, poll, stableDuration, err := observationDurations(invocation, nodeTypeID)
		if err != nil {
			return nodeadapter.AdapterResult{}, observationFailure(installed.CodeInvalidRequest, err)
		}
		gridSize, err := integerInput(invocation, "grid-size")
		if err != nil || gridSize < 1 || gridSize > 256 {
			return nodeadapter.AdapterResult{}, observationFailure(installed.CodeInvalidRequest, errors.Join(err, errors.New("grid size must be between 1 and 256")))
		}
		cellDelta, err := integerInput(invocation, "cell-delta")
		if err != nil || cellDelta < 0 || cellDelta > 255 {
			return nodeadapter.AdapterResult{}, observationFailure(installed.CodeInvalidRequest, errors.Join(err, errors.New("cell delta must be between 0 and 255")))
		}
		region, err := visionRegionInput(invocation)
		if err != nil {
			return nodeadapter.AdapterResult{}, observationFailure(nodes.VisionRegionInvalidCode, err)
		}
		baseline, bytesRead, err := captureFrameSignature(ctx, invocation, region, int(gridSize))
		counters["captures"], counters["capture_bytes"] = 1, bytesRead
		if err != nil {
			return nodeadapter.AdapterResult{}, observationNodeFailure(err)
		}
		last := frameDifference{}
		if nodeTypeID == nodes.WaitStableNodeID && stableDuration == 0 {
			return observationResult(builtins, invocation, last, "stable")
		}
		elapsed, stableElapsed := time.Duration(0), time.Duration(0)
		for elapsed < timeout {
			delay := min(poll, timeout-elapsed)
			if err := invocation.Wait(ctx, delay); err != nil {
				return nodeadapter.AdapterResult{}, err
			}
			elapsed += delay
			current, capturedBytes, err := captureFrameSignature(ctx, invocation, region, int(gridSize))
			counters["captures"]++
			counters["capture_bytes"] += capturedBytes
			if err != nil {
				return nodeadapter.AdapterResult{}, observationNodeFailure(err)
			}
			last = compareFrameSignatures(baseline, current, int(cellDelta))
			if nodeTypeID == nodes.WaitChangeNodeID {
				if last.changedRatio >= threshold {
					return observationResult(builtins, invocation, last, "changed")
				}
				continue
			}
			if last.changedRatio <= threshold {
				stableElapsed += delay
				if stableElapsed >= stableDuration {
					return observationResult(builtins, invocation, last, "stable")
				}
			} else {
				stableElapsed = 0
			}
			baseline = current
		}
		return observationResult(builtins, invocation, last, "timeout")
	}
}

func observationDurations(invocation nodeadapter.Invocation, nodeTypeID string) (time.Duration, time.Duration, time.Duration, error) {
	timeoutMillis, err := integerInput(invocation, "timeout")
	if err != nil {
		return 0, 0, 0, err
	}
	pollMillis, err := integerInput(invocation, "poll-interval")
	if err != nil {
		return 0, 0, 0, err
	}
	if timeoutMillis < 0 || time.Duration(timeoutMillis)*time.Millisecond > maxTemplateWait {
		return 0, 0, 0, errors.New("observation timeout must be between 0 and 3600000 milliseconds")
	}
	poll := time.Duration(pollMillis) * time.Millisecond
	if poll < minTemplatePoll || poll > maxTemplatePoll {
		return 0, 0, 0, errors.New("observation poll interval must be between 10 and 60000 milliseconds")
	}
	stable := time.Duration(0)
	if nodeTypeID == nodes.WaitStableNodeID {
		stableMillis, err := integerInput(invocation, "stable-duration")
		if err != nil || stableMillis < 0 || time.Duration(stableMillis)*time.Millisecond > maxTemplateWait {
			return 0, 0, 0, errors.Join(err, errors.New("stable duration must be between 0 and 3600000 milliseconds"))
		}
		stable = time.Duration(stableMillis) * time.Millisecond
	}
	return time.Duration(timeoutMillis) * time.Millisecond, poll, stable, nil
}

func captureFrameSignature(ctx context.Context, invocation nodeadapter.Invocation, region visionRegion, gridSize int) (frameSignature, int64, error) {
	raw, err := captureTemplateFrame(ctx, invocation)
	if err != nil {
		return frameSignature{}, 0, err
	}
	frame, err := decodeVisionPNG(raw)
	if err != nil {
		return frameSignature{}, int64(len(raw)), observationFailure(nodes.VisionImageInvalidCode, err)
	}
	search, err := resolveVisionRegion(frame.Bounds(), region)
	if err != nil {
		return frameSignature{}, int64(len(raw)), observationFailure(nodes.VisionRegionInvalidCode, err)
	}
	return frameSignature{grid: visionpkg.Downsample(frame.SubImage(search).(*image.RGBA), gridSize)}, int64(len(raw)), nil
}

func compareFrameSignatures(before, after frameSignature, cellDelta int) frameDifference {
	return frameDifference{
		changedRatio:   visionpkg.GridChangedRatio(before.grid, after.grid, cellDelta),
		meanDifference: visionpkg.GridMeanDiff(before.grid, after.grid),
	}
}

func observationResult(builtins nodes.Builtins, invocation nodeadapter.Invocation, difference frameDifference, output string) (nodeadapter.AdapterResult, error) {
	result, err := sealVisionOutputs(builtins, invocation, map[string]any{
		"changed-ratio": difference.changedRatio, "mean-difference": difference.meanDifference,
	})
	if err != nil {
		return nodeadapter.AdapterResult{}, observationFailure(nodes.ObservationFailedCode, err)
	}
	result.ExecOutputs = []string{output}
	return result, nil
}

func observationNodeFailure(err error) error {
	var failure *nodeadapter.NodeFailure
	if errors.As(err, &failure) {
		return &nodeadapter.NodeFailure{Code: failure.Code, Output: "failed", Cause: failure.Cause}
	}
	return observationFailure(nodes.ObservationFailedCode, err)
}

func observationFailure(code string, cause error) error {
	return &nodeadapter.NodeFailure{Code: code, Output: "failed", Cause: fmt.Errorf("frame observation: %w", cause)}
}
