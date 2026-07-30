package noderuntime

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"image"
	"time"

	"github.com/yottaapp/yotta/internal/artifact"
	"github.com/yottaapp/yotta/internal/automation/installed"
	"github.com/yottaapp/yotta/internal/nodeadapter"
	"github.com/yottaapp/yotta/internal/nodes"
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
			match, err = matchTemplateFrame(sourceFrame, templateBytes, region, threshold)
			counters["match_ms"] = time.Since(matchStarted).Milliseconds()
		} else {
			match, captures, err = waitForTemplateState(ctx, invocation, templateBytes, region, threshold, timeout, poll, wantPresent, counters)
		}
		counters["captures"] = int64(captures)
		if err != nil {
			return nodeadapter.AdapterResult{}, templateNodeFailure(err)
		}

		switch nodeTypeID {
		case nodes.WaitTemplateNodeID:
			if match.Matched && settle > 0 {
				if err := invocation.Wait(ctx, settle); err != nil {
					return nodeadapter.AdapterResult{}, err
				}
				if relocated, _, relocateErr := captureAndMatch(ctx, invocation, templateBytes, region, threshold, counters); relocateErr == nil && relocated.Matched {
					match = relocated
					counters["captures"]++
				}
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
				relocated, _, relocateErr := captureAndMatch(ctx, invocation, templateBytes, region, threshold, counters)
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

func waitForTemplateState(ctx context.Context, invocation nodeadapter.Invocation, templateBytes []byte, region visionRegion, threshold float64, timeout, poll time.Duration, wantPresent bool, counters map[string]int64) (visionMatchResult, int, error) {
	return pollTemplateState(ctx, invocation.Wait, timeout, poll, wantPresent, func(observeCtx context.Context) (visionMatchResult, error) {
		match, _, err := captureAndMatch(observeCtx, invocation, templateBytes, region, threshold, counters)
		return match, err
	})
}

func pollTemplateState(ctx context.Context, wait func(context.Context, time.Duration) error, timeout, poll time.Duration, wantPresent bool, observe func(context.Context) (visionMatchResult, error)) (visionMatchResult, int, error) {
	match, err := observe(ctx)
	if err != nil || match.Matched == wantPresent || timeout == 0 {
		return match, 1, err
	}
	elapsed, captures := time.Duration(0), 1
	for elapsed < timeout {
		delay := min(poll, timeout-elapsed)
		if err := wait(ctx, delay); err != nil {
			return visionMatchResult{}, captures, err
		}
		elapsed += delay
		match, err = observe(ctx)
		captures++
		if err != nil || match.Matched == wantPresent {
			return match, captures, err
		}
	}
	return match, captures, nil
}

func captureAndMatch(ctx context.Context, invocation nodeadapter.Invocation, templateBytes []byte, region visionRegion, threshold float64, counters map[string]int64) (visionMatchResult, int64, error) {
	captureStarted := time.Now()
	frame, captureBytes, err := captureTemplateFrame(ctx, invocation)
	counters["capture_ms"] += time.Since(captureStarted).Milliseconds()
	if err != nil {
		return visionMatchResult{}, 0, err
	}
	counters["capture_bytes"] += captureBytes
	matchStarted := time.Now()
	match, err := matchTemplateFrame(frame, templateBytes, region, threshold)
	counters["match_ms"] += time.Since(matchStarted).Milliseconds()
	return match, captureBytes, err
}

func captureTemplateFrame(ctx context.Context, invocation nodeadapter.Invocation) (_ *image.RGBA, captureBytes int64, runErr error) {
	handle, err := openConfiguredTarget(ctx, invocation, installed.KindCapture, installed.CaptureOperations())
	if err != nil {
		return nil, 0, mapAutomationFailure(err)
	}
	defer func() { runErr = errors.Join(runErr, invocation.Targets.Drop(context.WithoutCancel(ctx), handle)) }()
	request, err := artifact.Marshal(installed.CaptureRequest{Format: installed.CaptureFormatRGBA})
	if err != nil {
		return nil, 0, templateFailure(installed.CodeContractViolation, err)
	}
	rawResponse, err := invocation.Targets.Invoke(ctx, handle, installed.OperationCapture, request)
	if err != nil {
		return nil, 0, mapAutomationFailure(err)
	}
	response, err := installed.OpenCaptureResponse(rawResponse)
	if err != nil {
		return nil, 0, mapAutomationFailure(err)
	}
	if response.Size <= 0 || response.Size > installed.MaxCaptureBytes {
		return nil, 0, templateFailure(installed.CodeContractViolation, errors.New("template capture exceeds its byte budget"))
	}
	var content bytes.Buffer
	content.Grow(int(response.Size))
	for offset := int64(0); offset < response.Size; {
		length := min(installed.MaxCaptureChunkBytes, response.Size-offset)
		payload, err := artifact.Marshal(installed.CaptureRangeRequest{Offset: offset, Length: length})
		if err != nil {
			return nil, 0, templateFailure(installed.CodeContractViolation, err)
		}
		chunk, err := invocation.Targets.Invoke(ctx, handle, installed.OperationReadCapture, payload)
		if err != nil {
			return nil, 0, mapAutomationFailure(err)
		}
		if int64(len(chunk)) != length {
			return nil, 0, templateFailure(installed.CodeContractViolation, errors.New("capture provider returned an invalid chunk length"))
		}
		_, _ = content.Write(chunk)
		offset += length
	}
	switch response.MediaType {
	case installed.CaptureMediaTypeRGBA:
		pixels := content.Bytes()
		return &image.RGBA{
			Pix: pixels, Stride: int(response.Width * 4),
			Rect: image.Rect(0, 0, int(response.Width), int(response.Height)),
		}, response.Size, nil
	case "image/png":
		frame, err := decodeVisionPNG(content.Bytes())
		if err != nil {
			return nil, 0, templateFailure(nodes.VisionImageInvalidCode, err)
		}
		return frame, response.Size, nil
	default:
		return nil, 0, templateFailure(installed.CodeContractViolation, errors.New("template capture returned an unsupported media type"))
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
