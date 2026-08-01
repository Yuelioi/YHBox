package noderuntime

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"image"
	"time"

	"github.com/yottaapp/yotta/internal/artifact"
	"github.com/yottaapp/yotta/internal/automation/installed"
	"github.com/yottaapp/yotta/internal/datatype"
	"github.com/yottaapp/yotta/internal/nodeadapter"
	"github.com/yottaapp/yotta/internal/nodes"
	"github.com/yottaapp/yotta/internal/resource"
	"github.com/yottaapp/yotta/pkg/vision"
)

func controlDualColorBar(builtins nodes.Builtins) nodeadapter.Adapter {
	return func(ctx context.Context, invocation nodeadapter.Invocation) (_ nodeadapter.AdapterResult, runErr error) {
		counters := map[string]int64{}
		defer func() {
			runErr = errors.Join(runErr, recordAdapterOutcome(ctx, invocation, nodeadapter.AdapterAction{
				EffectID: nodes.ControlDualColorBarEffectID, Action: "automation.control-dual-color-bar", SummaryCode: "automation.control-dual-color-bar", Counters: counters,
			}, installed.CodeInputFailed, runErr))
		}()

		options, err := dualColorBarControlInputs(invocation)
		if err != nil {
			return nodeadapter.AdapterResult{}, err
		}
		captureHandle, err := openConfiguredTarget(ctx, invocation, installed.KindCapture, installed.CaptureOperations())
		if err != nil {
			return nodeadapter.AdapterResult{}, mapAutomationFailure(err)
		}
		defer func() {
			runErr = errors.Join(runErr, invocation.Targets.Drop(context.WithoutCancel(ctx), captureHandle))
		}()
		inputHandle, err := openConfiguredTarget(ctx, invocation, installed.KindInput, []string{installed.OperationPressKeys})
		if err != nil {
			return nodeadapter.AdapterResult{}, mapAutomationFailure(err)
		}
		defer func() { runErr = errors.Join(runErr, invocation.Targets.Drop(context.WithoutCancel(ctx), inputHandle)) }()

		startedAt := time.Now()
		var lastActivation time.Time
		observed := false
		for range options.maximumIterations {
			cycleStartedAt := time.Now()
			if !observed && len(options.activationKeys) > 0 &&
				(lastActivation.IsZero() || time.Since(lastActivation) >= time.Duration(options.activationRetryDuration)*time.Millisecond) {
				if err := pressDualColorBarKeys(ctx, invocation, inputHandle, options.activationKeys, options.activationHoldDuration); err != nil {
					return nodeadapter.AdapterResult{}, err
				}
				counters["activation_actions"]++
				lastActivation = time.Now()
				if err := waitDualColorBar(ctx, invocation, options.appearancePollDuration); err != nil {
					return nodeadapter.AdapterResult{}, err
				}
			}
			captureStarted := time.Now()
			frame, captureBytes, err := captureDualColorBarRegion(ctx, invocation, captureHandle, options.region)
			counters["capture_ms"] += time.Since(captureStarted).Milliseconds()
			if err != nil {
				return nodeadapter.AdapterResult{}, err
			}
			counters["frames"]++
			counters["capture_bytes"] += captureBytes
			analysisStarted := time.Now()
			result := vision.AnalyzeDualColorBar(frame, frame.Bounds(), options.innerRange, options.outerRange, options.analysis)
			counters["analysis_ms"] += time.Since(analysisStarted).Milliseconds()
			counters["inner_pixels"] += int64(result.InnerPixels)
			counters["outer_pixels"] += int64(result.OuterPixels)
			if !result.Found {
				if observed || len(options.activationKeys) == 0 && options.appearanceTimeout == 0 {
					break
				}
				if options.appearanceTimeout == 0 || time.Since(startedAt) >= time.Duration(options.appearanceTimeout)*time.Millisecond {
					return nodeadapter.AdapterResult{}, automationFailure(nodes.ControlDualColorBarNotFoundCode, errors.New("dual color bar did not appear before the activation timeout"))
				}
				if err := waitDualColorBar(ctx, invocation, options.appearancePollDuration); err != nil {
					return nodeadapter.AdapterResult{}, err
				}
				continue
			}
			observed = true
			delta := float64(result.OuterX - result.InnerX)
			tolerance := max(float64(result.OuterWidth)*options.toleranceRatio, options.minimumTolerance)
			switch {
			case delta > tolerance:
				if err := pressDualColorBarKeys(ctx, invocation, inputHandle, options.rightKeys, options.holdDuration); err != nil {
					return nodeadapter.AdapterResult{}, err
				}
				counters["right_actions"]++
			case delta < -tolerance:
				if err := pressDualColorBarKeys(ctx, invocation, inputHandle, options.leftKeys, options.holdDuration); err != nil {
					return nodeadapter.AdapterResult{}, err
				}
				counters["left_actions"]++
			default:
				if err := waitDualColorBar(ctx, invocation, options.neutralDuration); err != nil {
					return nodeadapter.AdapterResult{}, err
				}
				counters["neutral_actions"]++
			}
			pacingWait, err := paceDualColorBar(ctx, invocation, cycleStartedAt, time.Duration(options.cycleDuration)*time.Millisecond)
			if err != nil {
				return nodeadapter.AdapterResult{}, err
			}
			if pacingWait > 0 {
				counters["paced_cycles"]++
				counters["pacing_wait_ms"] += pacingWait.Milliseconds()
			}
		}
		outputs := map[string]any{
			"frames": counters["frames"], "left-actions": counters["left_actions"],
			"right-actions": counters["right_actions"], "neutral-actions": counters["neutral_actions"],
			"activation-actions": counters["activation_actions"],
		}
		sealed, err := sealControlDualColorBarOutputs(builtins, invocation, outputs)
		if err != nil {
			return nodeadapter.AdapterResult{}, err
		}
		sealed.ExecOutputs = []string{"completed"}
		return sealed, nil
	}
}

type dualColorBarControlOptions struct {
	region                           visionRegion
	innerRange, outerRange           vision.ColorRange
	analysis                         vision.DualColorBarOptions
	toleranceRatio, minimumTolerance float64
	leftKeys, rightKeys              []string
	holdDuration, neutralDuration    int64
	cycleDuration                    int64
	activationKeys                   []string
	activationHoldDuration           int64
	appearancePollDuration           int64
	activationRetryDuration          int64
	appearanceTimeout                int64
	maximumIterations                int
}

func dualColorBarControlInputs(invocation nodeadapter.Invocation) (dualColorBarControlOptions, error) {
	inner, err := visionColorRangeNamedInput(invocation, "inner-range")
	if err != nil {
		return dualColorBarControlOptions{}, visionFailure(nodes.VisionColorRangeInvalidCode, fmt.Errorf("inner range: %w", err))
	}
	outer, err := visionColorRangeNamedInput(invocation, "outer-range")
	if err != nil {
		return dualColorBarControlOptions{}, visionFailure(nodes.VisionColorRangeInvalidCode, fmt.Errorf("outer range: %w", err))
	}
	region, err := visionRegionInput(invocation)
	if err != nil {
		return dualColorBarControlOptions{}, visionFailure(nodes.VisionRegionInvalidCode, err)
	}
	readInteger := func(id string) (int64, error) { return integerInput(invocation, id) }
	innerMinimumWidth, err := readInteger("inner-minimum-width")
	if err != nil {
		return dualColorBarControlOptions{}, visionFailure(nodes.VisionAnalysisFailedCode, err)
	}
	innerMaximumWidth, err := readInteger("inner-maximum-width")
	if err != nil {
		return dualColorBarControlOptions{}, visionFailure(nodes.VisionAnalysisFailedCode, err)
	}
	outerMinimumWidth, err := readInteger("outer-minimum-width")
	if err != nil {
		return dualColorBarControlOptions{}, visionFailure(nodes.VisionAnalysisFailedCode, err)
	}
	bandHeightRatio, err := numberInput(invocation, "band-height-ratio")
	if err != nil {
		return dualColorBarControlOptions{}, visionFailure(nodes.VisionAnalysisFailedCode, err)
	}
	bandInnerHeightRatio, err := numberInput(invocation, "band-inner-height-ratio")
	if err != nil {
		return dualColorBarControlOptions{}, visionFailure(nodes.VisionAnalysisFailedCode, err)
	}
	innerWeight, err := numberInput(invocation, "inner-confidence-weight")
	if err != nil {
		return dualColorBarControlOptions{}, visionFailure(nodes.VisionAnalysisFailedCode, err)
	}
	outerWeight, err := numberInput(invocation, "outer-confidence-weight")
	if err != nil {
		return dualColorBarControlOptions{}, visionFailure(nodes.VisionAnalysisFailedCode, err)
	}
	toleranceRatio, err := numberInput(invocation, "tolerance-ratio")
	if err != nil {
		return dualColorBarControlOptions{}, visionFailure(nodes.VisionAnalysisFailedCode, err)
	}
	minimumTolerance, err := numberInput(invocation, "minimum-tolerance")
	if err != nil {
		return dualColorBarControlOptions{}, visionFailure(nodes.VisionAnalysisFailedCode, err)
	}
	holdDuration, err := readInteger("hold-duration")
	if err != nil {
		return dualColorBarControlOptions{}, automationFailure(installed.CodeInvalidRequest, err)
	}
	neutralDuration, err := readInteger("neutral-duration")
	if err != nil {
		return dualColorBarControlOptions{}, automationFailure(installed.CodeInvalidRequest, err)
	}
	cycleDuration, err := readInteger("cycle-duration")
	if err != nil {
		return dualColorBarControlOptions{}, automationFailure(installed.CodeInvalidRequest, err)
	}
	maximumIterations, err := readInteger("maximum-iterations")
	if err != nil {
		return dualColorBarControlOptions{}, automationFailure(installed.CodeInvalidRequest, err)
	}
	var leftKeys, rightKeys []string
	if err := decodeAutomationInput(invocation, "left-keys", &leftKeys); err != nil {
		return dualColorBarControlOptions{}, automationFailure(installed.CodeInvalidRequest, err)
	}
	if err := decodeAutomationInput(invocation, "right-keys", &rightKeys); err != nil {
		return dualColorBarControlOptions{}, automationFailure(installed.CodeInvalidRequest, err)
	}
	var activationKeys []string
	if err := decodeAutomationInput(invocation, "activation-keys", &activationKeys); err != nil {
		return dualColorBarControlOptions{}, automationFailure(installed.CodeInvalidRequest, err)
	}
	activationHoldDuration, err := readInteger("activation-hold-duration")
	if err != nil {
		return dualColorBarControlOptions{}, automationFailure(installed.CodeInvalidRequest, err)
	}
	appearancePollDuration, err := readInteger("appearance-poll-duration")
	if err != nil {
		return dualColorBarControlOptions{}, automationFailure(installed.CodeInvalidRequest, err)
	}
	activationRetryDuration, err := readInteger("activation-retry-duration")
	if err != nil {
		return dualColorBarControlOptions{}, automationFailure(installed.CodeInvalidRequest, err)
	}
	appearanceTimeout, err := readInteger("appearance-timeout")
	if err != nil {
		return dualColorBarControlOptions{}, automationFailure(installed.CodeInvalidRequest, err)
	}
	if innerMinimumWidth < 1 || innerMaximumWidth < 0 || outerMinimumWidth < 0 || bandHeightRatio <= 0 || bandInnerHeightRatio <= 0 ||
		innerWeight < 0 || outerWeight < 0 || innerWeight+outerWeight <= 0 || toleranceRatio < 0 || minimumTolerance < 0 {
		return dualColorBarControlOptions{}, visionFailure(nodes.VisionAnalysisFailedCode, errors.New("dual color bar control options are outside their supported ranges"))
	}
	if len(leftKeys) == 0 || len(rightKeys) == 0 || holdDuration < 0 || neutralDuration < 0 || cycleDuration < 0 || cycleDuration > 60_000 || maximumIterations < 1 || maximumIterations > 10_000 ||
		activationHoldDuration < 0 || appearancePollDuration < 0 || activationRetryDuration < 1 || appearanceTimeout < 0 || len(activationKeys) == 0 && appearanceTimeout > 0 {
		return dualColorBarControlOptions{}, automationFailure(installed.CodeInvalidRequest, errors.New("dual color bar input options are outside their supported ranges"))
	}
	return dualColorBarControlOptions{
		region:         region,
		innerRange:     vision.ColorRange{Space: inner.Space, Minimum: inner.Minimum, Maximum: inner.Maximum},
		outerRange:     vision.ColorRange{Space: outer.Space, Minimum: outer.Minimum, Maximum: outer.Maximum},
		analysis:       vision.DualColorBarOptions{InnerMinimumWidth: int(innerMinimumWidth), InnerMaximumWidth: int(innerMaximumWidth), OuterMinimumWidth: int(outerMinimumWidth), BandHeightRatio: bandHeightRatio, BandInnerHeightRatio: bandInnerHeightRatio, InnerConfidenceWeight: innerWeight, OuterConfidenceWeight: outerWeight},
		toleranceRatio: toleranceRatio, minimumTolerance: minimumTolerance, leftKeys: leftKeys, rightKeys: rightKeys,
		holdDuration: holdDuration, neutralDuration: neutralDuration, cycleDuration: cycleDuration, maximumIterations: int(maximumIterations),
		activationKeys: activationKeys, activationHoldDuration: activationHoldDuration,
		appearancePollDuration: appearancePollDuration, activationRetryDuration: activationRetryDuration, appearanceTimeout: appearanceTimeout,
	}, nil
}

func waitDualColorBar(ctx context.Context, invocation nodeadapter.Invocation, duration int64) error {
	if invocation.Wait == nil {
		return automationFailure(installed.CodeContractViolation, errors.New("dual color bar wait host function is missing"))
	}
	return invocation.Wait(ctx, time.Duration(duration)*time.Millisecond)
}

func paceDualColorBar(ctx context.Context, invocation nodeadapter.Invocation, startedAt time.Time, minimumDuration time.Duration) (time.Duration, error) {
	remaining := dualColorBarPacingDelay(startedAt, time.Now(), minimumDuration)
	if remaining <= 0 {
		return 0, nil
	}
	if err := waitDualColorBar(ctx, invocation, remaining.Milliseconds()); err != nil {
		return 0, err
	}
	return remaining, nil
}

func dualColorBarPacingDelay(startedAt, now time.Time, minimumDuration time.Duration) time.Duration {
	return max(minimumDuration-now.Sub(startedAt), 0)
}

func captureDualColorBarRegion(ctx context.Context, invocation nodeadapter.Invocation, handle resource.Handle, region visionRegion) (*image.RGBA, int64, error) {
	request, err := artifact.Marshal(installed.CaptureRequest{Format: installed.CaptureFormatRGBA, Region: &installed.CaptureRegion{X: region.X, Y: region.Y, Width: region.Width, Height: region.Height, Unit: region.Unit}})
	if err != nil {
		return nil, 0, automationFailure(installed.CodeContractViolation, err)
	}
	raw, err := invocation.Targets.Invoke(ctx, handle, installed.OperationCapture, request)
	if err != nil {
		return nil, 0, mapAutomationFailure(err)
	}
	response, err := installed.OpenCaptureResponse(raw)
	if err != nil {
		return nil, 0, mapAutomationFailure(err)
	}
	if response.MediaType != installed.CaptureMediaTypeRGBA || response.Width <= 0 || response.Height <= 0 || response.Size != response.Width*response.Height*4 {
		return nil, 0, automationFailure(installed.CodeContractViolation, errors.New("dual color bar capture returned invalid RGBA metadata"))
	}
	content, err := readAutomationCaptureBytes(ctx, invocation, handle, response.Size, func(err error) error {
		return automationFailure(installed.CodeContractViolation, err)
	})
	if err != nil {
		return nil, 0, err
	}
	return &image.RGBA{Pix: content, Stride: int(response.Width * 4), Rect: image.Rect(0, 0, int(response.Width), int(response.Height))}, response.Size, nil
}

func pressDualColorBarKeys(ctx context.Context, invocation nodeadapter.Invocation, handle resource.Handle, keys []string, duration int64) error {
	payload, err := artifact.Marshal(installed.PressKeysRequest{Keys: keys, DurationMilliseconds: duration})
	if err != nil {
		return automationFailure(installed.CodeContractViolation, err)
	}
	raw, err := invocation.Targets.Invoke(ctx, handle, installed.OperationPressKeys, payload)
	if err != nil {
		return mapAutomationFailure(err)
	}
	if !bytes.Equal(raw, []byte(`{}`)) {
		return automationFailure(installed.CodeContractViolation, errors.New("dual color bar key response is invalid"))
	}
	return nil
}

func sealControlDualColorBarOutputs(builtins nodes.Builtins, invocation nodeadapter.Invocation, values map[string]any) (nodeadapter.AdapterResult, error) {
	outputs := make(map[string]datatype.ValueEnvelope, len(values))
	for id, value := range values {
		resolved, ok := invocation.OutputTypes[id]
		if !ok {
			return nodeadapter.AdapterResult{}, automationFailure(installed.CodeContractViolation, fmt.Errorf("output type %q is unresolved", id))
		}
		raw, err := json.Marshal(value)
		if err != nil {
			return nodeadapter.AdapterResult{}, automationFailure(installed.CodeContractViolation, err)
		}
		envelope, err := datatype.SealInlineJSON(builtins.Catalog, resolved, raw)
		if err != nil {
			return nodeadapter.AdapterResult{}, automationFailure(installed.CodeContractViolation, err)
		}
		outputs[id] = envelope
	}
	return nodeadapter.AdapterResult{Outputs: outputs}, nil
}
