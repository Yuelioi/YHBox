package noderuntime

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/yottaapp/yotta/internal/automation/installed"
	"github.com/yottaapp/yotta/internal/nodeadapter"
	"github.com/yottaapp/yotta/internal/nodes"
	"github.com/yottaapp/yotta/internal/resource"
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
		if invocation.Wait == nil {
			return nodeadapter.AdapterResult{}, observationFailure(installed.CodeContractViolation, errors.New("frame observation scheduler is missing"))
		}
		captureHandle, err := openConfiguredTarget(ctx, invocation, installed.KindCapture, installed.CaptureOperations())
		if err != nil {
			return nodeadapter.AdapterResult{}, mapAutomationFailure(err)
		}
		defer func() {
			runErr = errors.Join(runErr, invocation.Targets.Drop(context.WithoutCancel(ctx), captureHandle))
		}()
		difference, output, captures, captureBytes, err := pollFrameObservationWithClock(
			ctx, invocation.Wait, time.Now, timeout, poll, stableDuration, threshold, int(cellDelta),
			nodeTypeID == nodes.WaitStableNodeID,
			func(observeCtx context.Context) (frameSignature, int64, error) {
				return captureFrameSignature(observeCtx, invocation, captureHandle, region, int(gridSize))
			},
		)
		counters["captures"], counters["capture_bytes"] = captures, captureBytes
		if err != nil {
			if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
				return nodeadapter.AdapterResult{}, err
			}
			return nodeadapter.AdapterResult{}, observationNodeFailure(err)
		}
		return observationResult(builtins, invocation, difference, output)
	}
}

func pollFrameObservationWithClock(
	ctx context.Context,
	wait func(context.Context, time.Duration) error,
	now func() time.Time,
	timeout, poll, stableDuration time.Duration,
	threshold float64,
	cellDelta int,
	stable bool,
	observe func(context.Context) (frameSignature, int64, error),
) (frameDifference, string, int64, int64, error) {
	deadline := now().Add(timeout)
	baseline, captureBytes, err := observe(ctx)
	captures := int64(1)
	if err != nil {
		return frameDifference{}, "", captures, captureBytes, err
	}
	last := frameDifference{}
	if stable && stableDuration == 0 {
		return last, "stable", captures, captureBytes, nil
	}
	stableSince := now()
	for {
		remaining := deadline.Sub(now())
		if remaining <= 0 {
			return last, "timeout", captures, captureBytes, nil
		}
		if err := wait(ctx, min(poll, remaining)); err != nil {
			return frameDifference{}, "", captures, captureBytes, err
		}
		if !now().Before(deadline) {
			return last, "timeout", captures, captureBytes, nil
		}
		current, currentBytes, err := observe(ctx)
		captures++
		captureBytes += currentBytes
		if err != nil {
			return frameDifference{}, "", captures, captureBytes, err
		}
		last = compareFrameSignatures(baseline, current, cellDelta)
		observedAt := now()
		if !stable {
			if last.changedRatio >= threshold {
				return last, "changed", captures, captureBytes, nil
			}
		} else if last.changedRatio <= threshold {
			if observedAt.Sub(stableSince) >= stableDuration {
				return last, "stable", captures, captureBytes, nil
			}
		} else {
			stableSince = observedAt
		}
		baseline = current
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

func captureFrameSignature(ctx context.Context, invocation nodeadapter.Invocation, handle resource.Handle, region visionRegion, gridSize int) (frameSignature, int64, error) {
	captured, captureBytes, err := captureTemplateFrameFromHandle(ctx, invocation, handle, &installed.CaptureRegion{
		X: region.X, Y: region.Y, Width: region.Width, Height: region.Height, Unit: region.Unit,
	})
	if err != nil {
		return frameSignature{}, 0, err
	}
	return frameSignature{grid: visionpkg.Downsample(captured.Image, gridSize)}, captureBytes, nil
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
