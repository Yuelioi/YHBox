package noderuntime

import (
	"context"
	"errors"
	"fmt"
	"image"
	"time"

	"github.com/yottaapp/yotta/internal/artifact"
	"github.com/yottaapp/yotta/internal/automation/installed"
	"github.com/yottaapp/yotta/internal/nodeadapter"
	"github.com/yottaapp/yotta/internal/nodes"
	"github.com/yottaapp/yotta/internal/resource"
)

const (
	minTemplatePoll = 10 * time.Millisecond
	maxTemplatePoll = time.Minute
	maxTemplateWait = time.Hour
)

func automationTemplate(builtins nodes.Builtins, nodeTypeID string) nodeadapter.Adapter {
	return func(ctx context.Context, invocation nodeadapter.Invocation) (_ nodeadapter.AdapterResult, runErr error) {
		effectID, action := templateEffect(nodeTypeID)
		counters := map[string]int64{}
		defer func() {
			runErr = errors.Join(runErr, recordAdapterOutcome(ctx, invocation, nodeadapter.AdapterAction{
				EffectID: effectID, Action: action, SummaryCode: action, Counters: counters,
			}, nodes.VisionMatchFailedCode, runErr))
		}()

		threshold, err := numberInput(invocation, "threshold")
		if err != nil || threshold < 0 || threshold > 1 {
			return nodeadapter.AdapterResult{}, templateFailure(nodes.VisionMatchFailedCode, errors.Join(err, errors.New("threshold must be between 0 and 1")))
		}
		region, err := visionRegionInput(invocation)
		if err != nil {
			return nodeadapter.AdapterResult{}, templateFailure(nodes.VisionRegionInvalidCode, err)
		}
		timeout, poll, settle, err := templateDurations(invocation, nodeTypeID)
		if err != nil {
			return nodeadapter.AdapterResult{}, templateFailure(installed.CodeInvalidRequest, err)
		}
		templateRef, err := visionBlobInput(invocation, "template")
		if err != nil {
			return nodeadapter.AdapterResult{}, templateFailure(nodes.VisionTemplateInvalidCode, err)
		}
		templateBytes, err := readVisionBlob(ctx, invocation, templateRef)
		if err != nil {
			return nodeadapter.AdapterResult{}, templateFailure(nodes.VisionMatchFailedCode, fmt.Errorf("read template: %w", err))
		}
		counters["template_bytes"] = templateRef.Size
		prepareStarted := time.Now()
		preparedTemplate, err := prepareVisionTemplate(templateBytes)
		counters["template_prepare_ms"] = time.Since(prepareStarted).Milliseconds()
		if err != nil {
			return nodeadapter.AdapterResult{}, templateNodeFailure(err)
		}
		var sourceFrame *image.RGBA
		if nodeTypeID == nodes.ClickTemplateNodeID {
			if _, supplied := invocation.Inputs["image"]; supplied {
				if settle != 0 {
					return nodeadapter.AdapterResult{}, templateFailure(installed.CodeInvalidRequest, errors.New("settle duration must be zero when a source image is supplied"))
				}
				sourceStarted := time.Now()
				frame, ref, loadErr := loadVisionImage(ctx, invocation, "image")
				counters["image_read_ms"] = time.Since(sourceStarted).Milliseconds()
				if loadErr != nil {
					return nodeadapter.AdapterResult{}, templateFailure(nodes.VisionImageInvalidCode, fmt.Errorf("read source image: %w", loadErr))
				}
				sourceFrame = frame
				counters["image_bytes"] = ref.Size
				counters["source_images"] = 1
				counters["capture_bytes"] = 0
				counters["capture_ms"] = 0
			}
		}
		var captureHandle resource.Handle
		if sourceFrame == nil {
			captureHandle, err = openConfiguredTarget(ctx, invocation, installed.KindCapture, installed.CaptureOperations())
			if err != nil {
				return nodeadapter.AdapterResult{}, mapAutomationFailure(err)
			}
			defer func() {
				runErr = errors.Join(runErr, invocation.Targets.Drop(context.WithoutCancel(ctx), captureHandle))
			}()
		}
		if err := emitTemplateStatus(ctx, invocation, nodes.AutomationTemplateWaitingStatus, map[string]int64{
			"timeout_ms": timeout.Milliseconds(), "poll_ms": poll.Milliseconds(),
		}); err != nil {
			return nodeadapter.AdapterResult{}, err
		}

		wantPresent := nodeTypeID != nodes.WaitTemplateGoneNodeID
		var (
			match    visionMatchResult
			captures int
		)
		if sourceFrame != nil {
			matchStarted := time.Now()
			match, err = matchPreparedTemplateFrame(sourceFrame, preparedTemplate, region, threshold)
			counters["match_ms"] = time.Since(matchStarted).Milliseconds()
		} else {
			match, captures, err = waitForTemplateState(ctx, invocation, captureHandle, preparedTemplate, region, threshold, timeout, poll, wantPresent, counters)
		}
		counters["captures"] = int64(captures)
		if err != nil {
			return nodeadapter.AdapterResult{}, templateNodeFailure(err)
		}

		switch nodeTypeID {
		case nodes.WaitTemplateNodeID:
			if match.Matched && settle > 0 {
				relocated, relocateErr := settleTemplateMatch(ctx, settle, invocation.Wait, func(relocateCtx context.Context) (visionMatchResult, error) {
					value, _, err := captureAndMatch(relocateCtx, invocation, captureHandle, preparedTemplate, region, threshold, counters)
					return value, err
				})
				counters["captures"]++
				if relocateErr != nil {
					return nodeadapter.AdapterResult{}, templateNodeFailure(relocateErr)
				}
				match = relocated
			}
			if err := emitTemplateMatchStatus(ctx, invocation, match.Matched, counters); err != nil {
				return nodeadapter.AdapterResult{}, err
			}
			return templateResult(builtins, invocation, match, choose(match.Matched, "found", "timeout"))
		case nodes.WaitTemplateGoneNodeID:
			if err := emitTemplateMatchStatus(ctx, invocation, !match.Matched, counters); err != nil {
				return nodeadapter.AdapterResult{}, err
			}
			return templateResult(builtins, invocation, match, choose(!match.Matched, "gone", "timeout"))
		case nodes.ClickTemplateNodeID:
			if !match.Matched {
				if err := emitTemplateMatchStatus(ctx, invocation, false, counters); err != nil {
					return nodeadapter.AdapterResult{}, err
				}
				return templateResult(builtins, invocation, match, "timeout")
			}
			if settle > 0 {
				if err := invocation.Wait(ctx, settle); err != nil {
					return nodeadapter.AdapterResult{}, err
				}
				relocated, _, relocateErr := captureAndMatch(ctx, invocation, captureHandle, preparedTemplate, region, threshold, counters)
				if relocateErr != nil {
					return nodeadapter.AdapterResult{}, templateNodeFailure(relocateErr)
				}
				counters["captures"]++
				if !relocated.Matched {
					if err := emitTemplateMatchStatus(ctx, invocation, false, counters); err != nil {
						return nodeadapter.AdapterResult{}, err
					}
					return templateResult(builtins, invocation, relocated, "timeout")
				}
				match = relocated
			}
			if err := clickTemplateMatch(ctx, invocation, match); err != nil {
				return nodeadapter.AdapterResult{}, err
			}
			counters["clicks"] = 1
			if err := emitTemplateMatchStatus(ctx, invocation, true, counters); err != nil {
				return nodeadapter.AdapterResult{}, err
			}
			return templateResult(builtins, invocation, match, "completed")
		default:
			return nodeadapter.AdapterResult{}, templateFailure(installed.CodeContractViolation, errors.New("template automation node is not installed"))
		}
	}
}

func settleTemplateMatch(
	ctx context.Context,
	settle time.Duration,
	wait func(context.Context, time.Duration) error,
	relocate func(context.Context) (visionMatchResult, error),
) (visionMatchResult, error) {
	if wait == nil || relocate == nil {
		return visionMatchResult{}, errors.New("template settle host functions are missing")
	}
	if err := wait(ctx, settle); err != nil {
		return visionMatchResult{}, err
	}
	return relocate(ctx)
}

func emitTemplateMatchStatus(ctx context.Context, invocation nodeadapter.Invocation, matched bool, counters map[string]int64) error {
	code := nodes.AutomationTemplateTimeoutStatus
	if matched {
		code = nodes.AutomationTemplateMatchedStatus
	}
	statusCounters := make(map[string]int64, len(counters))
	for name, value := range counters {
		statusCounters[name] = value
	}
	return emitTemplateStatus(ctx, invocation, code, statusCounters)
}

func emitTemplateStatus(ctx context.Context, invocation nodeadapter.Invocation, code string, counters map[string]int64) error {
	if invocation.EmitStatus == nil {
		return errors.New("template automation status emitter is missing")
	}
	return invocation.EmitStatus(ctx, code, counters)
}

func templateEffect(nodeTypeID string) (string, string) {
	switch nodeTypeID {
	case nodes.WaitTemplateNodeID:
		return nodes.WaitTemplateEffectID, "automation.wait-template"
	case nodes.WaitTemplateGoneNodeID:
		return nodes.WaitTemplateGoneEffectID, "automation.wait-template-gone"
	default:
		return nodes.ClickTemplateEffectID, "automation.click-template"
	}
}

func templateDurations(invocation nodeadapter.Invocation, nodeTypeID string) (time.Duration, time.Duration, time.Duration, error) {
	timeoutMillis, err := integerInput(invocation, "timeout")
	if err != nil {
		return 0, 0, 0, err
	}
	pollMillis, err := integerInput(invocation, "poll-interval")
	if err != nil {
		return 0, 0, 0, err
	}
	if timeoutMillis < 0 || time.Duration(timeoutMillis)*time.Millisecond > maxTemplateWait {
		return 0, 0, 0, errors.New("template timeout must be between 0 and 3600000 milliseconds")
	}
	poll := time.Duration(pollMillis) * time.Millisecond
	if poll < minTemplatePoll || poll > maxTemplatePoll {
		return 0, 0, 0, errors.New("template poll interval must be between 10 and 60000 milliseconds")
	}
	settle := time.Duration(0)
	if nodeTypeID != nodes.WaitTemplateGoneNodeID {
		settleMillis, settleErr := integerInput(invocation, "settle-duration")
		if settleErr != nil {
			return 0, 0, 0, settleErr
		}
		settle = time.Duration(settleMillis) * time.Millisecond
		if settle < 0 || settle > maxTemplatePoll {
			return 0, 0, 0, errors.New("template settle duration must be between 0 and 60000 milliseconds")
		}
	}
	return time.Duration(timeoutMillis) * time.Millisecond, poll, settle, nil
}

func waitForTemplateState(ctx context.Context, invocation nodeadapter.Invocation, captureHandle resource.Handle, template *preparedVisionTemplate, region visionRegion, threshold float64, timeout, poll time.Duration, wantPresent bool, counters map[string]int64) (visionMatchResult, int, error) {
	return pollTemplateState(ctx, invocation.Wait, timeout, poll, wantPresent, func(observeCtx context.Context) (visionMatchResult, error) {
		match, _, err := captureAndMatch(observeCtx, invocation, captureHandle, template, region, threshold, counters)
		return match, err
	})
}

func pollTemplateState(ctx context.Context, wait func(context.Context, time.Duration) error, timeout, poll time.Duration, wantPresent bool, observe func(context.Context) (visionMatchResult, error)) (visionMatchResult, int, error) {
	return pollTemplateStateWithClock(ctx, wait, time.Now, timeout, poll, wantPresent, observe)
}

func pollTemplateStateWithClock(ctx context.Context, wait func(context.Context, time.Duration) error, now func() time.Time, timeout, poll time.Duration, wantPresent bool, observe func(context.Context) (visionMatchResult, error)) (visionMatchResult, int, error) {
	deadline := now().Add(timeout)
	match, err := observe(ctx)
	if err != nil || match.Matched == wantPresent || timeout == 0 {
		return match, 1, err
	}
	captures := 1
	for {
		remaining := deadline.Sub(now())
		if remaining <= 0 {
			return match, captures, nil
		}
		delay := min(poll, remaining)
		if err := wait(ctx, delay); err != nil {
			return visionMatchResult{}, captures, err
		}
		if !now().Before(deadline) {
			return match, captures, nil
		}
		match, err = observe(ctx)
		captures++
		if err != nil || match.Matched == wantPresent {
			return match, captures, err
		}
	}
}

func captureAndMatch(ctx context.Context, invocation nodeadapter.Invocation, captureHandle resource.Handle, template *preparedVisionTemplate, region visionRegion, threshold float64, counters map[string]int64) (visionMatchResult, int64, error) {
	captureStarted := time.Now()
	captured, captureBytes, err := captureTemplateFrameFromHandle(ctx, invocation, captureHandle, &installed.CaptureRegion{
		X: region.X, Y: region.Y, Width: region.Width, Height: region.Height, Unit: region.Unit,
	})
	counters["capture_ms"] += time.Since(captureStarted).Milliseconds()
	if err != nil {
		return visionMatchResult{}, 0, err
	}
	counters["capture_bytes"] += captureBytes
	matchStarted := time.Now()
	match, err := matchPreparedTemplateFrame(captured.Image, template, visionRegion{X: 0, Y: 0, Width: 1, Height: 1, Unit: "ratio"}, threshold)
	counters["match_ms"] += time.Since(matchStarted).Milliseconds()
	if err == nil {
		match.FrameWidth, match.FrameHeight = captured.FrameWidth, captured.FrameHeight
		match.Center.X += float64(captured.OriginX)
		match.Center.Y += float64(captured.OriginY)
		match.Bounds.X += float64(captured.OriginX)
		match.Bounds.Y += float64(captured.OriginY)
	}
	return match, captureBytes, err
}

type capturedTemplateFrame struct {
	Image                   *image.RGBA
	OriginX, OriginY        int
	FrameWidth, FrameHeight int
}

func captureTemplateFrameFromHandle(ctx context.Context, invocation nodeadapter.Invocation, handle resource.Handle, region *installed.CaptureRegion) (_ capturedTemplateFrame, captureBytes int64, runErr error) {
	request, err := artifact.Marshal(installed.CaptureRequest{Format: installed.CaptureFormatRGBA, Region: region})
	if err != nil {
		return capturedTemplateFrame{}, 0, templateFailure(installed.CodeContractViolation, err)
	}
	rawResponse, err := invocation.Targets.Invoke(ctx, handle, installed.OperationCapture, request)
	if err != nil {
		return capturedTemplateFrame{}, 0, mapAutomationFailure(err)
	}
	response, err := installed.OpenCaptureResponse(rawResponse)
	if err != nil {
		return capturedTemplateFrame{}, 0, mapAutomationFailure(err)
	}
	if response.Size <= 0 || response.Size > installed.MaxCaptureBytes {
		return capturedTemplateFrame{}, 0, templateFailure(installed.CodeContractViolation, errors.New("template capture exceeds its byte budget"))
	}
	content, err := readAutomationCaptureBytes(ctx, invocation, handle, response.Size, func(err error) error {
		return templateFailure(installed.CodeContractViolation, err)
	})
	if err != nil {
		return capturedTemplateFrame{}, 0, err
	}
	switch response.MediaType {
	case installed.CaptureMediaTypeRGBA:
		frameWidth, frameHeight := response.FrameWidth, response.FrameHeight
		if frameWidth == 0 && frameHeight == 0 {
			frameWidth, frameHeight = response.Width, response.Height
		}
		return capturedTemplateFrame{Image: &image.RGBA{
			Pix: content, Stride: int(response.Width * 4),
			Rect: image.Rect(0, 0, int(response.Width), int(response.Height)),
		}, OriginX: int(response.OriginX), OriginY: int(response.OriginY), FrameWidth: int(frameWidth), FrameHeight: int(frameHeight)}, response.Size, nil
	case "image/png":
		frame, err := decodeVisionPNG(content)
		if err != nil {
			return capturedTemplateFrame{}, 0, templateFailure(nodes.VisionImageInvalidCode, err)
		}
		return capturedTemplateFrame{Image: frame, FrameWidth: frame.Bounds().Dx(), FrameHeight: frame.Bounds().Dy()}, response.Size, nil
	default:
		return capturedTemplateFrame{}, 0, templateFailure(installed.CodeContractViolation, errors.New("template capture returned an unsupported media type"))
	}
}

func clickTemplateMatch(ctx context.Context, invocation nodeadapter.Invocation, match visionMatchResult) (runErr error) {
	button, err := stringInput(invocation, "button")
	if err != nil || (button != "left" && button != "right" && button != "middle") {
		return templateFailure(installed.CodeInvalidRequest, errors.Join(err, errors.New("template click button must be left, right, or middle")))
	}
	hold, err := integerInput(invocation, "hold-duration")
	if err != nil || hold < 0 || hold > 60000 {
		return templateFailure(installed.CodeInvalidRequest, errors.Join(err, errors.New("template click hold duration must be between 0 and 60000 milliseconds")))
	}
	if match.FrameWidth <= 0 || match.FrameHeight <= 0 {
		return templateFailure(installed.CodeContractViolation, errors.New("template match omitted capture dimensions"))
	}
	request := installed.ClickRequest{
		Point:  installed.Point{X: match.Center.X / float64(match.FrameWidth), Y: match.Center.Y / float64(match.FrameHeight), Unit: "ratio"},
		Button: button, DurationMilliseconds: hold,
	}
	payload, err := artifact.Marshal(request)
	if err != nil {
		return templateFailure(installed.CodeContractViolation, err)
	}
	handle, err := openConfiguredTarget(ctx, invocation, installed.KindInput, []string{installed.OperationClick})
	if err != nil {
		return mapAutomationFailure(err)
	}
	defer func() { runErr = errors.Join(runErr, invocation.Targets.Drop(context.WithoutCancel(ctx), handle)) }()
	raw, err := invocation.Targets.Invoke(ctx, handle, installed.OperationClick, payload)
	if err != nil {
		return mapAutomationFailure(err)
	}
	if err := installed.OpenEffectResponse(raw); err != nil {
		return mapAutomationFailure(err)
	}
	return nil
}

func templateResult(builtins nodes.Builtins, invocation nodeadapter.Invocation, match visionMatchResult, output string) (nodeadapter.AdapterResult, error) {
	result, err := sealVisionOutputs(builtins, invocation, map[string]any{
		"matched": match.Matched, "score": match.Score, "center": match.Center, "bounds": match.Bounds,
	})
	if err != nil {
		return nodeadapter.AdapterResult{}, templateFailure(installed.CodeContractViolation, err)
	}
	result.ExecOutputs = []string{output}
	return result, nil
}

func templateNodeFailure(err error) error {
	var failure *nodeadapter.NodeFailure
	if errors.As(err, &failure) {
		return &nodeadapter.NodeFailure{Code: failure.Code, Output: "failed", Cause: failure.Cause}
	}
	return templateFailure(nodes.VisionMatchFailedCode, err)
}

func templateFailure(code string, cause error) error {
	return &nodeadapter.NodeFailure{Code: code, Output: "failed", Cause: fmt.Errorf("template automation: %w", cause)}
}

func choose(condition bool, yes, no string) string {
	if condition {
		return yes
	}
	return no
}
