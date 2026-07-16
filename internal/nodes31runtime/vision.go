package nodes31runtime

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"image"
	"image/draw"
	"image/png"
	"math"

	"github.com/yottaapp/yotta/internal/artifact"
	"github.com/yottaapp/yotta/internal/blob"
	"github.com/yottaapp/yotta/internal/datatype"
	"github.com/yottaapp/yotta/internal/nodes31"
	"github.com/yottaapp/yotta/internal/workflow/compiler"
	"github.com/yottaapp/yotta/pkg/vision"
)

const (
	visionBlobReadChunkBytes = int64(64 << 10)
	maxVisionBlobBytes       = int64(32 << 20)
	maxVisionImagePixels     = 16_777_216
)

type visionRegion struct {
	X      float64 `json:"x"`
	Y      float64 `json:"y"`
	Width  float64 `json:"width"`
	Height float64 `json:"height"`
	Unit   string  `json:"unit"`
}

type visionPoint struct {
	X    float64 `json:"x"`
	Y    float64 `json:"y"`
	Unit string  `json:"unit"`
}

func matchTemplate(builtins nodes31.Builtins) compiler.Adapter {
	return func(ctx context.Context, invocation compiler.Invocation) (_ compiler.AdapterResult, runErr error) {
		counters := map[string]int64{}
		defer func() {
			runErr = errors.Join(runErr, recordAdapterOutcome(ctx, invocation, compiler.AdapterAction{
				EffectID: nodes31.MatchTemplateEffectID, Action: "vision.match-template", SummaryCode: "vision.match-template", Counters: counters,
			}, nodes31.VisionMatchFailedCode, runErr))
		}()

		threshold, err := numberInput(invocation, "threshold")
		if err != nil || threshold < 0 || threshold > 1 {
			return compiler.AdapterResult{}, visionFailure(nodes31.VisionMatchFailedCode, errors.Join(err, errors.New("threshold must be between 0 and 1")))
		}
		region, err := visionRegionInput(invocation)
		if err != nil {
			return compiler.AdapterResult{}, visionFailure(nodes31.VisionRegionInvalidCode, err)
		}

		imageRef, err := visionBlobInput(invocation, "image")
		if err != nil {
			return compiler.AdapterResult{}, visionFailure(nodes31.VisionImageInvalidCode, err)
		}
		templateRef, err := visionBlobInput(invocation, "template")
		if err != nil {
			return compiler.AdapterResult{}, visionFailure(nodes31.VisionTemplateInvalidCode, err)
		}
		imageBytes, err := readVisionBlob(ctx, invocation, imageRef)
		if err != nil {
			return compiler.AdapterResult{}, visionFailure(nodes31.VisionMatchFailedCode, fmt.Errorf("read image: %w", err))
		}
		templateBytes, err := readVisionBlob(ctx, invocation, templateRef)
		if err != nil {
			return compiler.AdapterResult{}, visionFailure(nodes31.VisionMatchFailedCode, fmt.Errorf("read template: %w", err))
		}
		frame, err := decodeVisionPNG(imageBytes)
		if err != nil {
			return compiler.AdapterResult{}, visionFailure(nodes31.VisionImageInvalidCode, err)
		}
		templateImage, err := decodeVisionPNG(templateBytes)
		if err != nil {
			return compiler.AdapterResult{}, visionFailure(nodes31.VisionTemplateInvalidCode, err)
		}
		searchRect, err := resolveVisionRegion(frame.Bounds(), region)
		if err != nil {
			return compiler.AdapterResult{}, visionFailure(nodes31.VisionRegionInvalidCode, err)
		}
		if templateImage.Bounds().Dx() > searchRect.Dx() || templateImage.Bounds().Dy() > searchRect.Dy() {
			return compiler.AdapterResult{}, visionFailure(nodes31.VisionTemplateInvalidCode, fmt.Errorf(
				"template dimensions %dx%d exceed search region %dx%d",
				templateImage.Bounds().Dx(), templateImage.Bounds().Dy(), searchRect.Dx(), searchRect.Dy(),
			))
		}

		searchGray, searchWidth, searchHeight := vision.RGBAToGray(frame.SubImage(searchRect).(*image.RGBA))
		templateGray, templateWidth, templateHeight := vision.RGBAToGray(templateImage)
		if uniformVisionTemplate(templateGray) {
			return compiler.AdapterResult{}, visionFailure(nodes31.VisionTemplateInvalidCode, errors.New("template has no grayscale variance"))
		}
		x, y, score := vision.MatchFast(searchGray, searchWidth, searchHeight, &vision.Template{
			Gray: templateGray, W: templateWidth, H: templateHeight,
		}, vision.DefaultParallel())

		matched := x >= 0 && y >= 0 && score >= float32(threshold)
		center := visionPoint{Unit: "px"}
		bounds := visionRegion{Unit: "px"}
		if x >= 0 && y >= 0 {
			absoluteX, absoluteY := searchRect.Min.X+x, searchRect.Min.Y+y
			center.X = float64(absoluteX) + float64(templateWidth)/2
			center.Y = float64(absoluteY) + float64(templateHeight)/2
			bounds.X, bounds.Y = float64(absoluteX), float64(absoluteY)
			bounds.Width, bounds.Height = float64(templateWidth), float64(templateHeight)
		} else {
			score = -1
		}
		counters["image_bytes"] = imageRef.Size
		counters["template_bytes"] = templateRef.Size
		counters["search_pixels"] = int64(searchWidth * searchHeight)
		counters["template_pixels"] = int64(templateWidth * templateHeight)
		return sealVisionMatchOutputs(builtins, invocation, matched, float64(score), center, bounds)
	}
}

func visionRegionInput(invocation compiler.Invocation) (visionRegion, error) {
	input, ok := invocation.Inputs["region"]
	if !ok || len(input.InlineJSON()) == 0 {
		return visionRegion{}, errors.New("search region is missing")
	}
	var region visionRegion
	if err := json.Unmarshal(input.InlineJSON(), &region); err != nil {
		return visionRegion{}, fmt.Errorf("decode search region: %w", err)
	}
	return region, nil
}

func visionBlobInput(invocation compiler.Invocation, id string) (blob.BlobRef, error) {
	input, ok := invocation.Inputs[id]
	if !ok {
		return blob.BlobRef{}, fmt.Errorf("%s input is missing", id)
	}
	ref, ok := input.BlobRef()
	if !ok || ref.Validate() != nil || ref.MediaType != "image/png" || ref.Size <= 0 || ref.Size > maxVisionBlobBytes {
		return blob.BlobRef{}, fmt.Errorf("%s must be a bounded image/png BlobRef", id)
	}
	return ref, nil
}

func readVisionBlob(ctx context.Context, invocation compiler.Invocation, ref blob.BlobRef) ([]byte, error) {
	session := invocation.Sessions["blob-read"]
	if session == nil {
		return nil, errors.New("blob-read capability session is missing")
	}
	config, err := artifact.Marshal(blob.ReadConfig{Blob: ref})
	if err != nil {
		return nil, err
	}
	handle, err := session.Open(ctx, []string{blob.OperationReadRange}, config)
	if err != nil {
		return nil, err
	}
	defer func() { _ = session.Drop(context.WithoutCancel(ctx), handle) }()

	var content bytes.Buffer
	content.Grow(int(ref.Size))
	for offset := int64(0); offset < ref.Size; {
		length := min(visionBlobReadChunkBytes, ref.Size-offset)
		payload, err := artifact.Marshal(blob.RangeRequest{Offset: offset, Length: length})
		if err != nil {
			return nil, err
		}
		chunk, err := session.Invoke(ctx, handle, blob.OperationReadRange, payload)
		if err != nil {
			return nil, err
		}
		if int64(len(chunk)) != length {
			return nil, errors.New("blob provider returned an invalid image chunk length")
		}
		_, _ = content.Write(chunk)
		offset += length
	}
	return content.Bytes(), nil
}

func decodeVisionPNG(content []byte) (*image.RGBA, error) {
	config, err := png.DecodeConfig(bytes.NewReader(content))
	if err != nil || config.Width <= 0 || config.Height <= 0 || config.Height > maxVisionImagePixels || config.Width > maxVisionImagePixels/config.Height {
		return nil, errors.Join(errors.New("PNG dimensions are invalid or exceed the vision pixel budget"), err)
	}
	decoded, err := png.Decode(bytes.NewReader(content))
	if err != nil {
		return nil, fmt.Errorf("decode PNG: %w", err)
	}
	rgba := image.NewRGBA(image.Rect(0, 0, config.Width, config.Height))
	draw.Draw(rgba, rgba.Bounds(), decoded, decoded.Bounds().Min, draw.Src)
	return rgba, nil
}

func resolveVisionRegion(bounds image.Rectangle, region visionRegion) (image.Rectangle, error) {
	values := []float64{region.X, region.Y, region.Width, region.Height}
	for _, value := range values {
		if math.IsNaN(value) || math.IsInf(value, 0) {
			return image.Rectangle{}, errors.New("search region contains a non-finite value")
		}
	}
	if region.X < 0 || region.Y < 0 || region.Width <= 0 || region.Height <= 0 {
		return image.Rectangle{}, errors.New("search region must have a non-negative origin and positive size")
	}
	var x0, y0, x1, y1 float64
	switch region.Unit {
	case "ratio":
		if region.X+region.Width > 1 || region.Y+region.Height > 1 {
			return image.Rectangle{}, errors.New("ratio search region must remain inside the image")
		}
		x0, y0 = region.X*float64(bounds.Dx()), region.Y*float64(bounds.Dy())
		x1, y1 = (region.X+region.Width)*float64(bounds.Dx()), (region.Y+region.Height)*float64(bounds.Dy())
	case "px":
		x0, y0, x1, y1 = region.X, region.Y, region.X+region.Width, region.Y+region.Height
		if x1 > float64(bounds.Dx()) || y1 > float64(bounds.Dy()) {
			return image.Rectangle{}, errors.New("pixel search region must remain inside the image")
		}
	default:
		return image.Rectangle{}, fmt.Errorf("unsupported search region unit %q", region.Unit)
	}
	resolved := image.Rect(int(math.Floor(x0)), int(math.Floor(y0)), int(math.Ceil(x1)), int(math.Ceil(y1))).Add(bounds.Min)
	if resolved.Empty() || !resolved.In(bounds) {
		return image.Rectangle{}, errors.New("search region resolves outside the image")
	}
	return resolved, nil
}

func uniformVisionTemplate(gray []float32) bool {
	if len(gray) == 0 {
		return true
	}
	minimum, maximum := gray[0], gray[0]
	for _, value := range gray[1:] {
		minimum = min(minimum, value)
		maximum = max(maximum, value)
	}
	return maximum-minimum < 1e-6
}

func sealVisionMatchOutputs(builtins nodes31.Builtins, invocation compiler.Invocation, matched bool, score float64, center visionPoint, bounds visionRegion) (compiler.AdapterResult, error) {
	values := map[string]any{"matched": matched, "score": score, "center": center, "bounds": bounds}
	outputs := make(map[string]datatype.ValueEnvelope, len(values))
	for id, value := range values {
		resolved, ok := invocation.OutputTypes[id]
		if !ok {
			return compiler.AdapterResult{}, visionFailure(nodes31.VisionMatchFailedCode, fmt.Errorf("output type %q is unresolved", id))
		}
		raw, err := json.Marshal(value)
		if err != nil {
			return compiler.AdapterResult{}, visionFailure(nodes31.VisionMatchFailedCode, err)
		}
		envelope, err := datatype.SealInlineJSON(builtins.Catalog, resolved, raw)
		if err != nil {
			return compiler.AdapterResult{}, visionFailure(nodes31.VisionMatchFailedCode, fmt.Errorf("seal output %q: %w", id, err))
		}
		outputs[id] = envelope
	}
	return compiler.AdapterResult{Outputs: outputs}, nil
}

func visionFailure(code string, cause error) error {
	return &compiler.NodeFailure{Code: code, Cause: cause}
}
