package noderuntime

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"image"
	"sort"

	"github.com/yottaapp/yotta/internal/blob"
	"github.com/yottaapp/yotta/internal/datatype"
	"github.com/yottaapp/yotta/internal/nodeadapter"
	"github.com/yottaapp/yotta/internal/nodes"
	"github.com/yottaapp/yotta/pkg/vision"
)

const maxVisionAnalysisResults = 4096

type visionColorRange struct {
	Space   string `json:"space"`
	Minimum [3]int `json:"minimum"`
	Maximum [3]int `json:"maximum"`
}

type visionColorBlob struct {
	Area   int          `json:"area"`
	Center visionPoint  `json:"center"`
	Bounds visionRegion `json:"bounds"`
}

func findTemplateMatches(builtins nodes.Builtins) nodeadapter.Adapter {
	return func(ctx context.Context, invocation nodeadapter.Invocation) (_ nodeadapter.AdapterResult, runErr error) {
		counters := map[string]int64{}
		defer func() {
			runErr = errors.Join(runErr, recordAdapterOutcome(ctx, invocation, nodeadapter.AdapterAction{
				EffectID: nodes.FindTemplateMatchesEffectID, Action: "vision.find-template-matches", SummaryCode: "vision.find-template-matches", Counters: counters,
			}, nodes.VisionAnalysisFailedCode, runErr))
		}()
		threshold, err := numberInput(invocation, "threshold")
		if err != nil || threshold < 0 || threshold > 1 {
			return nodeadapter.AdapterResult{}, visionFailure(nodes.VisionAnalysisFailedCode, errors.Join(err, errors.New("threshold must be between 0 and 1")))
		}
		minimumDistance, err := integerInput(invocation, "minimum-distance")
		if err != nil || minimumDistance < 0 || minimumDistance > maxVisionImagePixels {
			return nodeadapter.AdapterResult{}, visionFailure(nodes.VisionAnalysisFailedCode, errors.Join(err, errors.New("minimum distance is outside its supported range")))
		}
		region, err := visionRegionInput(invocation)
		if err != nil {
			return nodeadapter.AdapterResult{}, visionFailure(nodes.VisionRegionInvalidCode, err)
		}
		frame, frameRef, err := loadVisionImage(ctx, invocation, "image")
		if err != nil {
			return nodeadapter.AdapterResult{}, visionFailure(nodes.VisionImageInvalidCode, err)
		}
		templateImage, templateRef, err := loadVisionImage(ctx, invocation, "template")
		if err != nil {
			return nodeadapter.AdapterResult{}, visionFailure(nodes.VisionTemplateInvalidCode, err)
		}
		searchRect, err := resolveVisionRegion(frame.Bounds(), region)
		if err != nil {
			return nodeadapter.AdapterResult{}, visionFailure(nodes.VisionRegionInvalidCode, err)
		}
		if templateImage.Bounds().Dx() > searchRect.Dx() || templateImage.Bounds().Dy() > searchRect.Dy() {
			return nodeadapter.AdapterResult{}, visionFailure(nodes.VisionTemplateInvalidCode, errors.New("template exceeds the search region"))
		}
		searchGray, searchWidth, searchHeight := vision.RGBAToGray(frame.SubImage(searchRect).(*image.RGBA))
		templateGray, templateWidth, templateHeight := vision.RGBAToGray(templateImage)
		if uniformVisionTemplate(templateGray) {
			return nodeadapter.AdapterResult{}, visionFailure(nodes.VisionTemplateInvalidCode, errors.New("template has no grayscale variance"))
		}
		hits := vision.MatchAll(searchGray, searchWidth, searchHeight, &vision.Template{Gray: templateGray, W: templateWidth, H: templateHeight},
			vision.DefaultParallel(), float32(threshold), int(minimumDistance))
		if len(hits) > maxVisionAnalysisResults {
			return nodeadapter.AdapterResult{}, visionFailure(nodes.VisionAnalysisFailedCode, errors.New("template match result exceeds the output budget"))
		}
		matches := make([]any, 0, len(hits))
		for _, hit := range hits {
			x, y := searchRect.Min.X+hit.X, searchRect.Min.Y+hit.Y
			matches = append(matches, map[string]any{
				"score":  boundedVisionScore(hit.Conf),
				"center": visionPoint{X: float64(x) + float64(templateWidth)/2, Y: float64(y) + float64(templateHeight)/2, Unit: "px"},
				"bounds": visionRegion{X: float64(x), Y: float64(y), Width: float64(templateWidth), Height: float64(templateHeight), Unit: "px"},
			})
		}
		counters["image_bytes"], counters["template_bytes"] = frameRef.Size, templateRef.Size
		counters["matches"] = int64(len(matches))
		return sealVisionOutputs(builtins, invocation, map[string]any{"matches": matches})
	}
}

func compareImages(builtins nodes.Builtins) nodeadapter.Adapter {
	return func(ctx context.Context, invocation nodeadapter.Invocation) (_ nodeadapter.AdapterResult, runErr error) {
		counters := map[string]int64{}
		defer func() {
			runErr = errors.Join(runErr, recordAdapterOutcome(ctx, invocation, nodeadapter.AdapterAction{
				EffectID: nodes.CompareImagesEffectID, Action: "vision.compare-images", SummaryCode: "vision.compare-images", Counters: counters,
			}, nodes.VisionAnalysisFailedCode, runErr))
		}()
		gridSize, err := integerInput(invocation, "grid-size")
		if err != nil || gridSize < 1 || gridSize > 256 {
			return nodeadapter.AdapterResult{}, visionFailure(nodes.VisionAnalysisFailedCode, errors.Join(err, errors.New("grid size must be between 1 and 256")))
		}
		cellDelta, err := integerInput(invocation, "cell-delta")
		if err != nil || cellDelta < 0 || cellDelta > 255 {
			return nodeadapter.AdapterResult{}, visionFailure(nodes.VisionAnalysisFailedCode, errors.Join(err, errors.New("cell delta must be between 0 and 255")))
		}
		region, err := visionRegionInput(invocation)
		if err != nil {
			return nodeadapter.AdapterResult{}, visionFailure(nodes.VisionRegionInvalidCode, err)
		}
		before, beforeRef, err := loadVisionImage(ctx, invocation, "before")
		if err != nil {
			return nodeadapter.AdapterResult{}, visionFailure(nodes.VisionImageInvalidCode, fmt.Errorf("before: %w", err))
		}
		after, afterRef, err := loadVisionImage(ctx, invocation, "after")
		if err != nil {
			return nodeadapter.AdapterResult{}, visionFailure(nodes.VisionImageInvalidCode, fmt.Errorf("after: %w", err))
		}
		beforeRect, err := resolveVisionRegion(before.Bounds(), region)
		if err != nil {
			return nodeadapter.AdapterResult{}, visionFailure(nodes.VisionRegionInvalidCode, fmt.Errorf("before: %w", err))
		}
		afterRect, err := resolveVisionRegion(after.Bounds(), region)
		if err != nil {
			return nodeadapter.AdapterResult{}, visionFailure(nodes.VisionRegionInvalidCode, fmt.Errorf("after: %w", err))
		}
		beforeSignature := vision.Downsample(before.SubImage(beforeRect).(*image.RGBA), int(gridSize))
		afterSignature := vision.Downsample(after.SubImage(afterRect).(*image.RGBA), int(gridSize))
		changedRatio := vision.GridChangedRatio(beforeSignature, afterSignature, int(cellDelta))
		meanDifference := vision.GridMeanDiff(beforeSignature, afterSignature)
		counters["before_bytes"], counters["after_bytes"] = beforeRef.Size, afterRef.Size
		counters["grid_cells"] = gridSize * gridSize
		return sealVisionOutputs(builtins, invocation, map[string]any{"changed-ratio": changedRatio, "mean-difference": meanDifference})
	}
}

func decodeQR(builtins nodes.Builtins) nodeadapter.Adapter {
	return func(ctx context.Context, invocation nodeadapter.Invocation) (_ nodeadapter.AdapterResult, runErr error) {
		counters := map[string]int64{}
		defer func() {
			runErr = errors.Join(runErr, recordAdapterOutcome(ctx, invocation, nodeadapter.AdapterAction{
				EffectID: nodes.DecodeQREffectID, Action: "vision.decode-qr", SummaryCode: "vision.decode-qr", Counters: counters,
			}, nodes.VisionAnalysisFailedCode, runErr))
		}()
		region, err := visionRegionInput(invocation)
		if err != nil {
			return nodeadapter.AdapterResult{}, visionFailure(nodes.VisionRegionInvalidCode, err)
		}
		frame, ref, err := loadVisionImage(ctx, invocation, "image")
		if err != nil {
			return nodeadapter.AdapterResult{}, visionFailure(nodes.VisionImageInvalidCode, err)
		}
		searchRect, err := resolveVisionRegion(frame.Bounds(), region)
		if err != nil {
			return nodeadapter.AdapterResult{}, visionFailure(nodes.VisionRegionInvalidCode, err)
		}
		hits, err := vision.DecodeQRFromImage(frame.SubImage(searchRect))
		if err != nil {
			return nodeadapter.AdapterResult{}, visionFailure(nodes.VisionAnalysisFailedCode, err)
		}
		if len(hits) > maxVisionAnalysisResults {
			return nodeadapter.AdapterResult{}, visionFailure(nodes.VisionAnalysisFailedCode, errors.New("QR result exceeds the output budget"))
		}
		codes := make([]any, 0, len(hits))
		for _, hit := range hits {
			points := make([]any, 0, len(hit.Points))
			for _, point := range hit.Points {
				points = append(points, visionPoint{X: float64(searchRect.Min.X + point[0]), Y: float64(searchRect.Min.Y + point[1]), Unit: "px"})
			}
			codes = append(codes, map[string]any{"text": hit.Text, "points": points})
		}
		counters["image_bytes"], counters["codes"] = ref.Size, int64(len(codes))
		return sealVisionOutputs(builtins, invocation, map[string]any{"codes": codes})
	}
}

func analyzeColor(builtins nodes.Builtins) nodeadapter.Adapter {
	return func(ctx context.Context, invocation nodeadapter.Invocation) (_ nodeadapter.AdapterResult, runErr error) {
		counters := map[string]int64{}
		defer func() {
			runErr = errors.Join(runErr, recordAdapterOutcome(ctx, invocation, nodeadapter.AdapterAction{
				EffectID: nodes.AnalyzeColorEffectID, Action: "vision.analyze-color", SummaryCode: "vision.analyze-color", Counters: counters,
			}, nodes.VisionAnalysisFailedCode, runErr))
		}()
		colorRange, err := visionColorRangeInput(invocation)
		if err != nil {
			return nodeadapter.AdapterResult{}, visionFailure(nodes.VisionColorRangeInvalidCode, err)
		}
		region, err := visionRegionInput(invocation)
		if err != nil {
			return nodeadapter.AdapterResult{}, visionFailure(nodes.VisionRegionInvalidCode, err)
		}
		frame, ref, err := loadVisionImage(ctx, invocation, "image")
		if err != nil {
			return nodeadapter.AdapterResult{}, visionFailure(nodes.VisionImageInvalidCode, err)
		}
		searchRect, err := resolveVisionRegion(frame.Bounds(), region)
		if err != nil {
			return nodeadapter.AdapterResult{}, visionFailure(nodes.VisionRegionInvalidCode, err)
		}
		count, sumX, sumY := scanVisionColor(frame, searchRect, colorRange, nil)
		centroid := visionPoint{Unit: "px"}
		if count > 0 {
			centroid.X, centroid.Y = sumX/float64(count), sumY/float64(count)
		}
		counters["image_bytes"], counters["matched_pixels"] = ref.Size, int64(count)
		return sealVisionOutputs(builtins, invocation, map[string]any{
			"pixel-count": count, "fraction": float64(count) / float64(searchRect.Dx()*searchRect.Dy()), "centroid": centroid,
		})
	}
}

func findColorBlobs(builtins nodes.Builtins) nodeadapter.Adapter {
	return func(ctx context.Context, invocation nodeadapter.Invocation) (_ nodeadapter.AdapterResult, runErr error) {
		counters := map[string]int64{}
		defer func() {
			runErr = errors.Join(runErr, recordAdapterOutcome(ctx, invocation, nodeadapter.AdapterAction{
				EffectID: nodes.FindColorBlobsEffectID, Action: "vision.find-color-blobs", SummaryCode: "vision.find-color-blobs", Counters: counters,
			}, nodes.VisionAnalysisFailedCode, runErr))
		}()
		minimumArea, err := integerInput(invocation, "minimum-area")
		if err != nil || minimumArea < 1 || minimumArea > maxVisionImagePixels {
			return nodeadapter.AdapterResult{}, visionFailure(nodes.VisionAnalysisFailedCode, errors.Join(err, errors.New("minimum area is outside its supported range")))
		}
		colorRange, err := visionColorRangeInput(invocation)
		if err != nil {
			return nodeadapter.AdapterResult{}, visionFailure(nodes.VisionColorRangeInvalidCode, err)
		}
		region, err := visionRegionInput(invocation)
		if err != nil {
			return nodeadapter.AdapterResult{}, visionFailure(nodes.VisionRegionInvalidCode, err)
		}
		frame, ref, err := loadVisionImage(ctx, invocation, "image")
		if err != nil {
			return nodeadapter.AdapterResult{}, visionFailure(nodes.VisionImageInvalidCode, err)
		}
		searchRect, err := resolveVisionRegion(frame.Bounds(), region)
		if err != nil {
			return nodeadapter.AdapterResult{}, visionFailure(nodes.VisionRegionInvalidCode, err)
		}
		mask := make([]bool, searchRect.Dx()*searchRect.Dy())
		scanVisionColor(frame, searchRect, colorRange, mask)
		blobs, err := connectedVisionColorBlobs(mask, searchRect, int(minimumArea))
		if err != nil {
			return nodeadapter.AdapterResult{}, visionFailure(nodes.VisionAnalysisFailedCode, err)
		}
		values := make([]any, len(blobs))
		for index := range blobs {
			values[index] = blobs[index]
		}
		counters["image_bytes"], counters["blobs"] = ref.Size, int64(len(blobs))
		return sealVisionOutputs(builtins, invocation, map[string]any{"blobs": values})
	}
}

func trackDualColorBar(builtins nodes.Builtins) nodeadapter.Adapter {
	return func(ctx context.Context, invocation nodeadapter.Invocation) (_ nodeadapter.AdapterResult, runErr error) {
		counters := map[string]int64{}
		defer func() {
			runErr = errors.Join(runErr, recordAdapterOutcome(ctx, invocation, nodeadapter.AdapterAction{
				EffectID: nodes.TrackDualColorBarEffectID, Action: "vision.track-dual-color-bar", SummaryCode: "vision.track-dual-color-bar", Counters: counters,
			}, nodes.VisionAnalysisFailedCode, runErr))
		}()
		innerRange, err := visionColorRangeNamedInput(invocation, "inner-range")
		if err != nil {
			return nodeadapter.AdapterResult{}, visionFailure(nodes.VisionColorRangeInvalidCode, fmt.Errorf("inner range: %w", err))
		}
		outerRange, err := visionColorRangeNamedInput(invocation, "outer-range")
		if err != nil {
			return nodeadapter.AdapterResult{}, visionFailure(nodes.VisionColorRangeInvalidCode, fmt.Errorf("outer range: %w", err))
		}
		region, err := visionRegionInput(invocation)
		if err != nil {
			return nodeadapter.AdapterResult{}, visionFailure(nodes.VisionRegionInvalidCode, err)
		}
		frame, ref, err := loadVisionImage(ctx, invocation, "image")
		if err != nil {
			return nodeadapter.AdapterResult{}, visionFailure(nodes.VisionImageInvalidCode, err)
		}
		searchRect, err := resolveVisionRegion(frame.Bounds(), region)
		if err != nil {
			return nodeadapter.AdapterResult{}, visionFailure(nodes.VisionRegionInvalidCode, err)
		}
		innerMinimumWidth, err := integerInput(invocation, "inner-minimum-width")
		if err != nil {
			return nodeadapter.AdapterResult{}, visionFailure(nodes.VisionAnalysisFailedCode, err)
		}
		innerMaximumWidth, err := integerInput(invocation, "inner-maximum-width")
		if err != nil {
			return nodeadapter.AdapterResult{}, visionFailure(nodes.VisionAnalysisFailedCode, err)
		}
		outerMinimumWidth, err := integerInput(invocation, "outer-minimum-width")
		if err != nil {
			return nodeadapter.AdapterResult{}, visionFailure(nodes.VisionAnalysisFailedCode, err)
		}
		bandHeightRatio, err := numberInput(invocation, "band-height-ratio")
		if err != nil {
			return nodeadapter.AdapterResult{}, visionFailure(nodes.VisionAnalysisFailedCode, err)
		}
		bandInnerHeightRatio, err := numberInput(invocation, "band-inner-height-ratio")
		if err != nil {
			return nodeadapter.AdapterResult{}, visionFailure(nodes.VisionAnalysisFailedCode, err)
		}
		innerWeight, err := numberInput(invocation, "inner-confidence-weight")
		if err != nil {
			return nodeadapter.AdapterResult{}, visionFailure(nodes.VisionAnalysisFailedCode, err)
		}
		outerWeight, err := numberInput(invocation, "outer-confidence-weight")
		if err != nil {
			return nodeadapter.AdapterResult{}, visionFailure(nodes.VisionAnalysisFailedCode, err)
		}
		if innerMinimumWidth < 1 || innerMaximumWidth < 0 || outerMinimumWidth < 0 ||
			bandHeightRatio <= 0 || bandInnerHeightRatio <= 0 || innerWeight < 0 || outerWeight < 0 || innerWeight+outerWeight <= 0 {
			return nodeadapter.AdapterResult{}, visionFailure(nodes.VisionAnalysisFailedCode, errors.New("dual color bar options are outside their supported ranges"))
		}
		result := vision.AnalyzeDualColorBar(frame, searchRect,
			vision.ColorRange{Space: innerRange.Space, Minimum: innerRange.Minimum, Maximum: innerRange.Maximum},
			vision.ColorRange{Space: outerRange.Space, Minimum: outerRange.Minimum, Maximum: outerRange.Maximum},
			vision.DualColorBarOptions{
				InnerMinimumWidth: int(innerMinimumWidth), InnerMaximumWidth: int(innerMaximumWidth), OuterMinimumWidth: int(outerMinimumWidth),
				BandHeightRatio: bandHeightRatio, BandInnerHeightRatio: bandInnerHeightRatio,
				InnerConfidenceWeight: innerWeight, OuterConfidenceWeight: outerWeight,
			})
		counters["image_bytes"], counters["inner_pixels"], counters["outer_pixels"] = ref.Size, int64(result.InnerPixels), int64(result.OuterPixels)
		return sealVisionOutputs(builtins, invocation, map[string]any{
			"found": result.Found, "inner-x": result.InnerX, "outer-x": result.OuterX, "outer-width": result.OuterWidth,
			"confidence": result.Confidence, "inner-pixels": result.InnerPixels, "outer-pixels": result.OuterPixels,
		})
	}
}

func loadVisionImage(ctx context.Context, invocation nodeadapter.Invocation, id string) (*image.RGBA, blob.BlobRef, error) {
	ref, err := visionBlobInput(invocation, id)
	if err != nil {
		return nil, blob.BlobRef{}, err
	}
	content, err := readVisionBlob(ctx, invocation, ref)
	if err != nil {
		return nil, blob.BlobRef{}, err
	}
	decoded, err := decodeVisionPNG(content)
	if err != nil {
		return nil, blob.BlobRef{}, err
	}
	return decoded, ref, nil
}

func visionColorRangeInput(invocation nodeadapter.Invocation) (visionColorRange, error) {
	return visionColorRangeNamedInput(invocation, "range")
}

func visionColorRangeNamedInput(invocation nodeadapter.Invocation, id string) (visionColorRange, error) {
	input, ok := invocation.Inputs[id]
	if !ok || len(input.InlineJSON()) == 0 {
		return visionColorRange{}, errors.New("color range is missing")
	}
	var colorRange visionColorRange
	if err := json.Unmarshal(input.InlineJSON(), &colorRange); err != nil {
		return visionColorRange{}, fmt.Errorf("decode color range: %w", err)
	}
	limits := [3]int{255, 255, 255}
	if colorRange.Space == "hsv" {
		limits = [3]int{360, 100, 100}
	} else if colorRange.Space != "rgb" {
		return visionColorRange{}, fmt.Errorf("unsupported color space %q", colorRange.Space)
	}
	for index := range 3 {
		if colorRange.Minimum[index] < 0 || colorRange.Maximum[index] > limits[index] || colorRange.Minimum[index] > colorRange.Maximum[index] {
			return visionColorRange{}, fmt.Errorf("color channel %d has an invalid inclusive range", index)
		}
	}
	return colorRange, nil
}

func scanVisionColor(frame *image.RGBA, searchRect image.Rectangle, colorRange visionColorRange, mask []bool) (count int, sumX, sumY float64) {
	width := searchRect.Dx()
	for y := searchRect.Min.Y; y < searchRect.Max.Y; y++ {
		for x := searchRect.Min.X; x < searchRect.Max.X; x++ {
			offset := frame.PixOffset(x, y)
			if !matchesVisionColor(frame.Pix[offset], frame.Pix[offset+1], frame.Pix[offset+2], colorRange) {
				continue
			}
			count++
			sumX, sumY = sumX+float64(x)+0.5, sumY+float64(y)+0.5
			if mask != nil {
				mask[(y-searchRect.Min.Y)*width+(x-searchRect.Min.X)] = true
			}
		}
	}
	return count, sumX, sumY
}

func matchesVisionColor(r, g, b uint8, colorRange visionColorRange) bool {
	values := [3]int{int(r), int(g), int(b)}
	if colorRange.Space == "hsv" {
		values[0], values[1], values[2] = vision.RGBToHSV(r, g, b)
	}
	for index := range 3 {
		if values[index] < colorRange.Minimum[index] || values[index] > colorRange.Maximum[index] {
			return false
		}
	}
	return true
}

func connectedVisionColorBlobs(mask []bool, region image.Rectangle, minimumArea int) ([]visionColorBlob, error) {
	width, height := region.Dx(), region.Dy()
	visited := make([]bool, len(mask))
	queue := make([]int, 0, 256)
	blobs := make([]visionColorBlob, 0)
	for start, set := range mask {
		if !set || visited[start] {
			continue
		}
		visited[start] = true
		queue = append(queue[:0], start)
		area, sumX, sumY := 0, 0, 0
		minX, minY, maxX, maxY := width, height, -1, -1
		for len(queue) > 0 {
			current := queue[len(queue)-1]
			queue = queue[:len(queue)-1]
			x, y := current%width, current/width
			area, sumX, sumY = area+1, sumX+x, sumY+y
			minX, minY, maxX, maxY = min(minX, x), min(minY, y), max(maxX, x), max(maxY, y)
			neighbors := [4][2]int{{x - 1, y}, {x + 1, y}, {x, y - 1}, {x, y + 1}}
			for _, neighbor := range neighbors {
				nx, ny := neighbor[0], neighbor[1]
				if nx < 0 || nx >= width || ny < 0 || ny >= height {
					continue
				}
				index := ny*width + nx
				if mask[index] && !visited[index] {
					visited[index] = true
					queue = append(queue, index)
				}
			}
		}
		if area < minimumArea {
			continue
		}
		if len(blobs) == maxVisionAnalysisResults {
			return nil, errors.New("color blob result exceeds the output budget")
		}
		blobs = append(blobs, visionColorBlob{
			Area:   area,
			Center: visionPoint{X: float64(region.Min.X) + float64(sumX)/float64(area) + 0.5, Y: float64(region.Min.Y) + float64(sumY)/float64(area) + 0.5, Unit: "px"},
			Bounds: visionRegion{X: float64(region.Min.X + minX), Y: float64(region.Min.Y + minY), Width: float64(maxX - minX + 1), Height: float64(maxY - minY + 1), Unit: "px"},
		})
	}
	sort.Slice(blobs, func(i, j int) bool {
		if blobs[i].Area != blobs[j].Area {
			return blobs[i].Area > blobs[j].Area
		}
		if blobs[i].Bounds.Y != blobs[j].Bounds.Y {
			return blobs[i].Bounds.Y < blobs[j].Bounds.Y
		}
		return blobs[i].Bounds.X < blobs[j].Bounds.X
	})
	return blobs, nil
}

func sealVisionOutputs(builtins nodes.Builtins, invocation nodeadapter.Invocation, values map[string]any) (nodeadapter.AdapterResult, error) {
	outputs := make(map[string]datatype.ValueEnvelope, len(values))
	for id, value := range values {
		resolved, ok := invocation.OutputTypes[id]
		if !ok {
			return nodeadapter.AdapterResult{}, visionFailure(nodes.VisionAnalysisFailedCode, fmt.Errorf("output type %q is unresolved", id))
		}
		raw, err := json.Marshal(value)
		if err != nil {
			return nodeadapter.AdapterResult{}, visionFailure(nodes.VisionAnalysisFailedCode, err)
		}
		envelope, err := datatype.SealInlineJSON(builtins.Catalog, resolved, raw)
		if err != nil {
			return nodeadapter.AdapterResult{}, visionFailure(nodes.VisionAnalysisFailedCode, fmt.Errorf("seal output %q: %w", id, err))
		}
		outputs[id] = envelope
	}
	return nodeadapter.AdapterResult{Outputs: outputs}, nil
}
