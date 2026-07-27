package noderuntime

import (
	"bytes"
	"context"
	"errors"
	"fmt"
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
		if err := emitTemplateStatus(ctx, invocation, nodes.AutomationTemplateWaitingStatus, map[string]int64{
			"timeout_ms": timeout.Milliseconds(), "poll_ms": poll.Milliseconds(),
		}); err != nil {
			return nodeadapter.AdapterResult{}, err
		}

		wantPresent := nodeTypeID != nodes.WaitTemplateGoneNodeID
		match, captures, err := waitForTemplateState(ctx, invocation, templateBytes, region, threshold, timeout, poll, wantPresent)
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
				if relocated, _, relocateErr := captureAndMatch(ctx, invocation, templateBytes, region, threshold); relocateErr == nil && relocated.Matched {
					match = relocated
					counters["captures"]++
				}
			}
			if err := emitTemplateMatchStatus(ctx, invocation, match.Matched, captures); err != nil {
				return nodeadapter.AdapterResult{}, err
			}
			return templateResult(builtins, invocation, match, choose(match.Matched, "found", "timeout"))
		case nodes.WaitTemplateGoneNodeID:
			if err := emitTemplateMatchStatus(ctx, invocation, !match.Matched, captures); err != nil {
				return nodeadapter.AdapterResult{}, err
			}
			return templateResult(builtins, invocation, match, choose(!match.Matched, "gone", "timeout"))
		case nodes.ClickTemplateNodeID:
			if !match.Matched {
				if err := emitTemplateMatchStatus(ctx, invocation, false, captures); err != nil {
					return nodeadapter.AdapterResult{}, err
				}
				return templateResult(builtins, invocation, match, "timeout")
			}
			if settle > 0 {
				if err := invocation.Wait(ctx, settle); err != nil {
					return nodeadapter.AdapterResult{}, err
				}
				relocated, _, relocateErr := captureAndMatch(ctx, invocation, templateBytes, region, threshold)
				if relocateErr != nil {
					return nodeadapter.AdapterResult{}, templateNodeFailure(relocateErr)
				}
				counters["captures"]++
				if !relocated.Matched {
					if err := emitTemplateMatchStatus(ctx, invocation, false, captures); err != nil {
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
			if err := emitTemplateMatchStatus(ctx, invocation, true, captures); err != nil {
				return nodeadapter.AdapterResult{}, err
			}
			return templateResult(builtins, invocation, match, "completed")
		default:
			return nodeadapter.AdapterResult{}, templateFailure(installed.CodeContractViolation, errors.New("template automation node is not installed"))
		}
	}
}

func emitTemplateMatchStatus(ctx context.Context, invocation nodeadapter.Invocation, matched bool, captures int) error {
	code := nodes.AutomationTemplateTimeoutStatus
	if matched {
		code = nodes.AutomationTemplateMatchedStatus
	}
	return emitTemplateStatus(ctx, invocation, code, map[string]int64{"captures": int64(captures)})
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

func waitForTemplateState(ctx context.Context, invocation nodeadapter.Invocation, templateBytes []byte, region visionRegion, threshold float64, timeout, poll time.Duration, wantPresent bool) (visionMatchResult, int, error) {
	return pollTemplateState(ctx, invocation.Wait, timeout, poll, wantPresent, func(observeCtx context.Context) (visionMatchResult, error) {
		match, _, err := captureAndMatch(observeCtx, invocation, templateBytes, region, threshold)
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

func captureAndMatch(ctx context.Context, invocation nodeadapter.Invocation, templateBytes []byte, region visionRegion, threshold float64) (visionMatchResult, int64, error) {
	frame, err := captureTemplateFrame(ctx, invocation)
	if err != nil {
		return visionMatchResult{}, 0, err
	}
	match, err := matchTemplateBytes(frame, templateBytes, region, threshold)
	return match, int64(len(frame)), err
}

func captureTemplateFrame(ctx context.Context, invocation nodeadapter.Invocation) (_ []byte, runErr error) {
	session := invocation.Sessions["capture-target"]
	if session == nil {
		return nil, templateFailure(installed.CodeContractViolation, errors.New("template capture capability session is missing"))
	}
	handle, err := session.Open(ctx, installed.CaptureOperations(), []byte(`{}`))
	if err != nil {
		return nil, mapAutomationFailure(err)
	}
	defer func() { runErr = errors.Join(runErr, session.Drop(context.WithoutCancel(ctx), handle)) }()
	rawResponse, err := session.Invoke(ctx, handle, installed.OperationCapture, []byte(`{}`))
	if err != nil {
		return nil, mapAutomationFailure(err)
	}
	response, err := installed.OpenCaptureResponse(rawResponse)
	if err != nil {
		return nil, mapAutomationFailure(err)
	}
	if response.MediaType != "image/png" || response.Size <= 0 || response.Size > maxVisionBlobBytes {
		return nil, templateFailure(installed.CodeContractViolation, errors.New("template capture must be a bounded image/png"))
	}
	var content bytes.Buffer
	content.Grow(int(response.Size))
	for offset := int64(0); offset < response.Size; {
		length := min(captureChunkBytes, response.Size-offset)
		payload, err := artifact.Marshal(installed.CaptureRangeRequest{Offset: offset, Length: length})
		if err != nil {
			return nil, templateFailure(installed.CodeContractViolation, err)
		}
		chunk, err := session.Invoke(ctx, handle, installed.OperationReadCapture, payload)
		if err != nil {
			return nil, mapAutomationFailure(err)
		}
		if int64(len(chunk)) != length {
			return nil, templateFailure(installed.CodeContractViolation, errors.New("capture provider returned an invalid chunk length"))
		}
		_, _ = content.Write(chunk)
		offset += length
	}
	return content.Bytes(), nil
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
	session := invocation.Sessions["input-target"]
	if session == nil {
		return templateFailure(installed.CodeContractViolation, errors.New("template input capability session is missing"))
	}
	handle, err := session.Open(ctx, []string{installed.OperationClick}, []byte(`{}`))
	if err != nil {
		return mapAutomationFailure(err)
	}
	defer func() { runErr = errors.Join(runErr, session.Drop(context.WithoutCancel(ctx), handle)) }()
	raw, err := session.Invoke(ctx, handle, installed.OperationClick, payload)
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
