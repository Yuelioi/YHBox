package runprepare

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"image"
	"image/draw"
	"image/png"
	"io"
	"math"
	"sort"
	"sync"

	"github.com/yottaapp/yotta/internal/blob"
	"github.com/yottaapp/yotta/internal/workflow/compiler"
	"github.com/yottaapp/yotta/internal/workflow/schema"
	"github.com/yottaapp/yotta/pkg/vision"
)

const (
	imageScaleVersion     = "gray-bilinear-v1"
	maxCachedImageBytes   = 64 << 20
	maxCachedImageEntries = 256
)

type Planner struct {
	store imageBlobStore
	mu    sync.Mutex
	cache map[scaleKey][]byte
	order []scaleKey
	bytes int
}

type imageBlobStore interface {
	ReadRange(context.Context, blob.BlobRef, int64, int64) ([]byte, error)
	PutRetained(context.Context, string, io.Reader) (blob.BlobRef, blob.Retention, error)
}

type PreparedImages struct {
	Overrides  []compiler.ResourceOverride
	retentions []blob.Retention
}

func (prepared PreparedImages) Release() {
	for _, retention := range prepared.retentions {
		retention.Release()
	}
}

type scaleKey struct {
	digest       string
	sourceW      int
	sourceH      int
	targetW      int
	targetH      int
	scaleVersion string
}

func New(store imageBlobStore) (*Planner, error) {
	if store == nil {
		return nil, errors.New("run image planner requires a Blob store")
	}
	return &Planner{store: store, cache: make(map[scaleKey][]byte)}, nil
}

func ReferencedTargetSlots(sourceJSON []byte) ([]string, error) {
	source, diagnostics := schema.ParseSource(sourceJSON)
	if schema.HasErrors(diagnostics) {
		return nil, errors.New("run image planning requires a valid Workflow Source")
	}
	usage := referencedImageSlots(source)
	slots := make(map[string]bool)
	for _, resourceSlots := range usage {
		for slot := range resourceSlots {
			slots[slot] = true
		}
	}
	result := make([]string, 0, len(slots))
	for slot := range slots {
		result = append(result, slot)
	}
	sort.Strings(result)
	return result, nil
}

func referencedImageSlots(source schema.WorkflowSource) map[string]map[string]bool {
	images := make(map[string]bool)
	for _, resource := range source.Resources {
		images[resource.ID] = resource.Kind == schema.ResourceImage
	}
	defaultSlot, _ := schema.TargetDefaultSlot(source, "target")
	usage := make(map[string]map[string]bool)
	for _, graph := range source.Graphs {
		for _, node := range graph.Nodes {
			slot, _ := node.Config["slot"].(string)
			if slot == "" {
				slot = defaultSlot
			}
			if slot == "" {
				continue
			}
			for _, binding := range node.Bindings {
				if binding.Kind != schema.BindingResource || binding.Resource == nil || !images[binding.Resource.ResourceID] {
					continue
				}
				if usage[binding.Resource.ResourceID] == nil {
					usage[binding.Resource.ResourceID] = make(map[string]bool)
				}
				usage[binding.Resource.ResourceID][slot] = true
			}
		}
	}
	return usage
}

func (planner *Planner) Prepare(ctx context.Context, sourceJSON []byte, resolutions map[string][2]int) (_ PreparedImages, resultErr error) {
	if ctx == nil || planner == nil || planner.store == nil {
		return PreparedImages{}, errors.New("run image planning context is unavailable")
	}
	source, diagnostics := schema.ParseSource(sourceJSON)
	if schema.HasErrors(diagnostics) {
		return PreparedImages{}, errors.New("run image planning requires a valid Workflow Source")
	}
	slots := make([]string, 0, len(resolutions))
	for slot, resolution := range resolutions {
		if slot != "" && resolution[0] > 0 && resolution[1] > 0 {
			slots = append(slots, slot)
		}
	}
	sort.Strings(slots)
	usage := referencedImageSlots(source)
	prepared := PreparedImages{Overrides: make([]compiler.ResourceOverride, 0)}
	defer func() {
		if resultErr != nil {
			prepared.Release()
		}
	}()
	for _, resource := range source.Resources {
		if resource.Kind != schema.ResourceImage || resource.Image == nil {
			continue
		}
		for _, slot := range slots {
			if !usage[resource.ID][slot] {
				continue
			}
			target := resolutions[slot]
			variant, exact := pickVariant(resource.Image.Variants, target)
			if variant == nil {
				continue
			}
			selected := variant.Blob
			if !exact {
				if !sameAspect(variant.Resolution, target) {
					return PreparedImages{}, fmt.Errorf("image resource %q needs a %dx%d variant; closest is %dx%d with a different aspect ratio", resource.ID, target[0], target[1], variant.Resolution[0], variant.Resolution[1])
				}
				scaled, retention, scaleErr := planner.scaled(ctx, variant.Blob, variant.Resolution, target)
				if scaleErr != nil {
					return PreparedImages{}, fmt.Errorf("prepare image resource %q for %s: %w", resource.ID, slot, scaleErr)
				}
				selected = scaled
				prepared.retentions = append(prepared.retentions, retention)
			}
			prepared.Overrides = append(prepared.Overrides, compiler.ResourceOverride{ResourceID: resource.ID, TargetSlot: slot, Blob: selected})
		}
	}
	return prepared, nil
}

func pickVariant(variants []schema.ImageResourceVariant, target [2]int) (*schema.ImageResourceVariant, bool) {
	best := -1
	bestDistance := math.MaxFloat64
	for index := range variants {
		variant := &variants[index]
		if variant.Resolution == target {
			return variant, true
		}
		if variant.Resolution[0] <= 0 || variant.Resolution[1] <= 0 {
			continue
		}
		distance := math.Abs(math.Log(float64(max(target[0], target[1])) / float64(max(variant.Resolution[0], variant.Resolution[1]))))
		if distance < bestDistance {
			best, bestDistance = index, distance
		}
	}
	if best < 0 {
		return nil, false
	}
	return &variants[best], false
}

func sameAspect(source, target [2]int) bool {
	left := float64(source[0]) / float64(source[1])
	right := float64(target[0]) / float64(target[1])
	return math.Abs(left-right)/left <= 0.01
}

func (planner *Planner) scaled(ctx context.Context, source blob.BlobRef, sourceResolution, targetResolution [2]int) (blob.BlobRef, blob.Retention, error) {
	key := scaleKey{digest: source.Digest.String(), sourceW: sourceResolution[0], sourceH: sourceResolution[1], targetW: targetResolution[0], targetH: targetResolution[1], scaleVersion: imageScaleVersion}
	planner.mu.Lock()
	if cached, ok := planner.cache[key]; ok {
		planner.mu.Unlock()
		return planner.store.PutRetained(ctx, "image/png", bytes.NewReader(cached))
	}
	planner.mu.Unlock()
	raw, err := planner.store.ReadRange(ctx, source, 0, source.Size)
	if err != nil {
		return blob.BlobRef{}, nil, err
	}
	decoded, err := png.Decode(bytes.NewReader(raw))
	if err != nil {
		return blob.BlobRef{}, nil, fmt.Errorf("decode PNG: %w", err)
	}
	rgba := image.NewRGBA(decoded.Bounds())
	draw.Draw(rgba, rgba.Bounds(), decoded, decoded.Bounds().Min, draw.Src)
	gray, width, height := vision.RGBAToGray(rgba)
	scale := float64(targetResolution[0]) / float64(sourceResolution[0])
	targetWidth := int(math.Round(float64(width) * scale))
	targetHeight := int(math.Round(float64(height) * scale))
	resized := vision.ResizeGray(gray, width, height, targetWidth, targetHeight)
	if len(resized) != targetWidth*targetHeight {
		return blob.BlobRef{}, nil, errors.New("scaled template dimensions are invalid")
	}
	output := image.NewGray(image.Rect(0, 0, targetWidth, targetHeight))
	for index, value := range resized {
		output.Pix[index] = uint8(math.Round(float64(value) * 255))
	}
	var encoded bytes.Buffer
	if err := png.Encode(&encoded, output); err != nil {
		return blob.BlobRef{}, nil, err
	}
	content := append([]byte(nil), encoded.Bytes()...)
	ref, retention, err := planner.store.PutRetained(ctx, "image/png", bytes.NewReader(content))
	if err != nil {
		return blob.BlobRef{}, nil, err
	}
	planner.mu.Lock()
	planner.rememberLocked(key, content)
	planner.mu.Unlock()
	return ref, retention, nil
}

func (planner *Planner) rememberLocked(key scaleKey, content []byte) {
	if len(content) > maxCachedImageBytes {
		return
	}
	if _, exists := planner.cache[key]; exists {
		return
	}
	for len(planner.order) >= maxCachedImageEntries || planner.bytes+len(content) > maxCachedImageBytes {
		oldest := planner.order[0]
		planner.order = planner.order[1:]
		planner.bytes -= len(planner.cache[oldest])
		delete(planner.cache, oldest)
	}
	planner.cache[key] = content
	planner.order = append(planner.order, key)
	planner.bytes += len(content)
}
