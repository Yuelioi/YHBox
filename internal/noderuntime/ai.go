package noderuntime

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"image"
	imagedraw "image/draw"
	"image/jpeg"
	"image/png"
	"strings"
	"time"

	"github.com/yottaapp/yotta/internal/ai"
	"github.com/yottaapp/yotta/internal/artifact"
	"github.com/yottaapp/yotta/internal/datatype"
	"github.com/yottaapp/yotta/internal/nodeadapter"
	"github.com/yottaapp/yotta/internal/nodes"
	"github.com/yottaapp/yotta/internal/runid"
	"github.com/yottaapp/yotta/pkg/imageutil"
)

const (
	maxAIImageSourceBytes = 32 << 20
	maxAIImagePixels      = 16 << 20
)

func aiGenerate(builtins nodes.Builtins, structured bool) nodeadapter.Adapter {
	effectID := nodes.AIGenerateEffectID
	operation := ai.OperationGenerate
	promptManifest := builtins.AIGeneratePrompt
	if structured {
		effectID = nodes.AIExtractEffectID
		operation = ai.OperationGenerateStructured
		promptManifest = builtins.AIExtractPrompt
	}
	return func(ctx context.Context, invocation nodeadapter.Invocation) (_ nodeadapter.AdapterResult, runErr error) {
		runCtx := ctx
		action := nodeadapter.AdapterAction{
			EffectID: effectID, Action: "ai.provider-response", SummaryCode: "ai.generation",
			Counters: map[string]int64{}, Facts: map[string]string{},
		}
		defer func() {
			runErr = errors.Join(runErr, recordAdapterOutcome(runCtx, invocation, action, "ai.generation_failed", runErr))
		}()
		timeout, err := configInt64(invocation.Config["timeoutMilliseconds"])
		if err != nil || timeout < nodes.MinAITimeoutMilliseconds || timeout > nodes.MaxAITimeoutMilliseconds {
			return nodeadapter.AdapterResult{}, errors.New("AI timeout config is invalid")
		}
		action.Counters["timeout_ms"] = timeout
		attemptCtx, cancelAttempt := context.WithTimeout(runCtx, time.Duration(timeout)*time.Millisecond)
		defer cancelAttempt()
		defer func() {
			if errors.Is(runErr, context.DeadlineExceeded) && runCtx.Err() == nil {
				runErr = &nodeadapter.NodeFailure{
					Code: "ai.generation_failed", Output: "failed", Cause: errors.New("AI generation timed out"),
				}
			}
		}()
		ctx = attemptCtx
		promptEnvelope, ok := invocation.Inputs["prompt"]
		if !ok || len(promptEnvelope.InlineJSON()) == 0 {
			return nodeadapter.AdapterResult{}, errors.New("AI prompt input is missing")
		}
		var prompt string
		if err := json.Unmarshal(promptEnvelope.InlineJSON(), &prompt); err != nil || prompt == "" {
			return nodeadapter.AdapterResult{}, errors.New("AI prompt input must be a non-empty string")
		}
		var imageInput *ai.ImageInput
		if _, connected := invocation.Inputs["image"]; connected {
			carrier, _, err := readBlobInput(ctx, invocation, "image", "image/png", maxAIImageSourceBytes)
			if err != nil {
				return nodeadapter.AdapterResult{}, fmt.Errorf("read AI image input: %w", err)
			}
			prepared, err := prepareAIImage(carrier)
			if err != nil {
				return nodeadapter.AdapterResult{}, err
			}
			imageInput = &prepared
		}
		request, err := aiRequest(invocation.Config, prompt, imageInput, structured, promptManifest)
		if err != nil {
			return nodeadapter.AdapterResult{}, err
		}
		addAIRequestSummary(&action, request)
		session := invocation.Sessions["model"]
		if session == nil {
			return nodeadapter.AdapterResult{}, errors.New("AI model capability session is missing")
		}
		handle, err := session.Open(ctx, []string{operation}, []byte(`{}`))
		if err != nil {
			return nodeadapter.AdapterResult{}, err
		}
		defer func() { runErr = errors.Join(runErr, session.Drop(context.WithoutCancel(ctx), handle)) }()
		payload, err := artifact.Marshal(request)
		if err != nil {
			return nodeadapter.AdapterResult{}, err
		}
		rawOutcome, err := session.Invoke(ctx, handle, operation, payload)
		if err != nil {
			return nodeadapter.AdapterResult{}, err
		}
		outcome, err := ai.OpenOutcome(rawOutcome)
		if err != nil {
			return nodeadapter.AdapterResult{}, err
		}
		addAIOutcomeSummary(&action, outcome)
		if outcome.Finish.Kind != ai.FinishCompleted {
			return nodeadapter.AdapterResult{}, &nodeadapter.NodeFailure{
				Code: "ai.generation_failed", Output: "failed", Cause: fmt.Errorf("AI generation finished as %s", outcome.Finish.Kind),
			}
		}
		var rawValue json.RawMessage
		if structured {
			if len(outcome.Items) != 1 || outcome.Items[0].Kind != ai.OutputStructured || outcome.Items[0].Structured == nil {
				return nodeadapter.AdapterResult{}, errors.New("AI structured generation returned no exact structured item")
			}
			rawValue = outcome.Items[0].Structured.Value
		} else {
			var text strings.Builder
			for _, item := range outcome.Items {
				if item.Kind != ai.OutputText || item.Text == nil {
					return nodeadapter.AdapterResult{}, errors.New("AI text generation returned a non-text item")
				}
				text.WriteString(item.Text.Text)
			}
			if text.Len() == 0 {
				return nodeadapter.AdapterResult{}, errors.New("AI text generation returned an empty result")
			}
			rawValue, err = json.Marshal(text.String())
			if err != nil {
				return nodeadapter.AdapterResult{}, err
			}
		}
		resolved, ok := invocation.OutputTypes["result"]
		if !ok {
			return nodeadapter.AdapterResult{}, errors.New("AI result output type is unresolved")
		}
		envelope, err := datatype.SealInlineJSON(builtins.Catalog, resolved, rawValue)
		if err != nil {
			return nodeadapter.AdapterResult{}, err
		}
		return nodeadapter.AdapterResult{Outputs: map[string]datatype.ValueEnvelope{"result": envelope}, ExecOutputs: []string{"completed"}}, nil
	}
}

func aiRequest(config map[string]any, prompt string, imageInput *ai.ImageInput, structured bool, manifest ai.PromptManifest) (ai.GenerateRequest, error) {
	attemptID, err := runid.New()
	if err != nil {
		return ai.GenerateRequest{}, err
	}
	rendered, err := ai.RenderPrompt(manifest, []ai.PromptBlock{{Kind: ai.PromptBlockUser, Content: prompt}})
	if err != nil {
		return ai.GenerateRequest{}, err
	}
	request := ai.GenerateRequest{AttemptID: attemptID, Prompt: rendered, Image: imageInput, Retention: ai.RetentionNoApplicationState}
	if value, exists := config["temperature"]; exists {
		temperature, err := configFloat(value)
		if err != nil {
			return ai.GenerateRequest{}, errors.New("AI temperature config is invalid")
		}
		request.Limits.Temperature = &temperature
	}
	if value, exists := config["maxOutputTokens"]; exists {
		maximum, err := configInt64(value)
		if err != nil {
			return ai.GenerateRequest{}, errors.New("AI max output tokens config is invalid")
		}
		request.Limits.MaxOutputTokens = &maximum
	}
	if structured {
		fields, ok := config["fields"]
		if !ok {
			return ai.GenerateRequest{}, errors.New("AI Extract output fields are missing")
		}
		spec, err := ai.CompileStructuredFields("result", fields)
		if err != nil {
			return ai.GenerateRequest{}, err
		}
		request.Output = &spec
	}
	if err := request.Validate(); err != nil {
		return ai.GenerateRequest{}, err
	}
	return request, nil
}

func prepareAIImage(content []byte) (ai.ImageInput, error) {
	config, err := png.DecodeConfig(bytes.NewReader(content))
	if err != nil || config.Width <= 0 || config.Height <= 0 ||
		config.Height > maxAIImagePixels || config.Width > maxAIImagePixels/config.Height {
		return ai.ImageInput{}, errors.Join(errors.New("AI image dimensions are invalid or too large"), err)
	}
	decoded, err := png.Decode(bytes.NewReader(content))
	if err != nil {
		return ai.ImageInput{}, fmt.Errorf("decode AI image: %w", err)
	}
	source := image.NewRGBA(image.Rect(0, 0, config.Width, config.Height))
	imagedraw.Draw(source, source.Bounds(), decoded, decoded.Bounds().Min, imagedraw.Src)
	for _, maximumDimension := range []int{1568, 1280, 1024} {
		scaled := scaleAIImage(source, maximumDimension)
		for _, quality := range []int{88, 78, 68, 58} {
			var encoded bytes.Buffer
			if err := jpeg.Encode(&encoded, scaled, &jpeg.Options{Quality: quality}); err != nil {
				return ai.ImageInput{}, fmt.Errorf("encode AI image: %w", err)
			}
			if encoded.Len() <= ai.MaxImageInputBytes {
				return ai.ImageInput{MediaType: "image/jpeg", Data: encoded.Bytes()}, nil
			}
		}
	}
	return ai.ImageInput{}, errors.New("AI image remains too large after safe resizing")
}

func scaleAIImage(source *image.RGBA, maximumDimension int) *image.RGBA {
	width, height := source.Bounds().Dx(), source.Bounds().Dy()
	if width <= maximumDimension && height <= maximumDimension {
		return source
	}
	if width >= height {
		return imageutil.ScaleRGBA(source, maximumDimension, max(1, height*maximumDimension/width))
	}
	return imageutil.ScaleRGBA(source, max(1, width*maximumDimension/height), maximumDimension)
}

func addAIRequestSummary(action *nodeadapter.AdapterAction, request ai.GenerateRequest) {
	addFact(action.Facts, "prompt_manifest", request.Prompt.ManifestDigest.String())
	addFact(action.Facts, "tool_set", request.ToolSet.String())
	if request.Output != nil {
		digest, err := request.Output.Digest()
		if err == nil {
			addFact(action.Facts, "output_schema", digest.String())
		}
	}
	if request.Image != nil {
		addFact(action.Facts, "image_media_type", request.Image.MediaType)
		if action.Counters == nil {
			action.Counters = map[string]int64{}
		}
		action.Counters["image_bytes"] = int64(len(request.Image.Data))
	}
}

func configFloat(value any) (float64, error) {
	switch typed := value.(type) {
	case json.Number:
		return typed.Float64()
	case float64:
		return typed, nil
	default:
		return 0, errors.New("not a number")
	}
}

func configInt64(value any) (int64, error) {
	switch typed := value.(type) {
	case json.Number:
		return typed.Int64()
	case float64:
		if typed != float64(int64(typed)) {
			return 0, errors.New("not an integer")
		}
		return int64(typed), nil
	default:
		return 0, errors.New("not an integer")
	}
}

func addAIOutcomeSummary(action *nodeadapter.AdapterAction, outcome ai.Outcome) {
	addCounter := func(name string, value *int64) {
		if value != nil {
			action.Counters[name] = *value
		}
	}
	addCounter("input_tokens", outcome.Usage.InputTotal)
	addCounter("cache_read_tokens", outcome.Usage.CacheRead)
	addCounter("cache_write_tokens", outcome.Usage.CacheWrite)
	addCounter("output_tokens", outcome.Usage.OutputTotal)
	addCounter("reasoning_tokens", outcome.Usage.ReasoningOutput)
	addCounter("cost_microunits", outcome.Usage.CostMicrounits)
	addFact(action.Facts, "provider", string(outcome.Provider))
	addFact(action.Facts, "requested_model", outcome.RequestedModel)
	addFact(action.Facts, "resolved_model", outcome.ResolvedModel)
	addFact(action.Facts, "finish", string(outcome.Finish.Kind))
	addFact(action.Facts, "provider_request_id", outcome.ProviderRequestID)
	addFact(action.Facts, "provider_response_id", outcome.ProviderResponseID)
}

func addFact(target map[string]string, name, value string) {
	if value == "" || len(value) > 256 {
		return
	}
	for _, character := range value {
		if character < 0x20 || character > 0x7e {
			return
		}
	}
	target[name] = value
}
