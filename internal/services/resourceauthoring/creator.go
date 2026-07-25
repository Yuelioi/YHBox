// Package resourceauthoring creates portable Workflow Resources without
// publishing mutable Global Asset records.
package resourceauthoring

import (
	"bytes"
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"image/png"
	"sort"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/yottaapp/yotta/internal/blob"
	"github.com/yottaapp/yotta/internal/services/asset"
	"github.com/yottaapp/yotta/internal/services/inputclip"
	"github.com/yottaapp/yotta/internal/services/macro"
	"github.com/yottaapp/yotta/internal/workflow/schema"
)

const pngDataURLPrefix = "data:image/png;base64,"

type Metadata struct {
	Name        string   `json:"name"`
	Description string   `json:"description"`
	Category    string   `json:"category"`
	Tags        []string `json:"tags"`
}

type ImageDraft struct {
	Metadata
	DataURL    string     `json:"dataURL"`
	Resolution [2]int     `json:"resolution"`
	Region     [4]float32 `json:"region"`
}

type MacroDraft struct {
	Metadata
	Document macro.Document
}

type InputClipDraft struct {
	Metadata
	Clip inputclip.InputClip
}

// Creator is the deep Workflow Resource creation module. Callers supply
// authored content; encoding, validation, CAS publication, identity, and
// portable metadata projection remain behind this interface.
type Creator struct {
	blobs  *blob.Store
	assets *asset.Store
}

func NewCreator(blobs *blob.Store, assets *asset.Store) (*Creator, error) {
	if blobs == nil || assets == nil {
		return nil, errors.New("Workflow Resource creator requires shared Blob and Global Asset stores")
	}
	return &Creator{blobs: blobs, assets: assets}, nil
}

type Promotion struct {
	GUID     string `json:"guid"`
	Kind     string `json:"kind"`
	Revision uint64 `json:"revision"`
}

func (c *Creator) Promote(ctx context.Context, resource schema.WorkflowResource) (Promotion, error) {
	if err := schema.ValidateWorkflowResource(resource); err != nil {
		return Promotion{}, fmt.Errorf("validate Workflow Resource promotion: %w", err)
	}
	record := asset.AssetRecord{
		GUID: "asset-" + uuid.NewString(), Name: resource.Name,
		Description: resource.Description, Category: resource.Category,
		Tags:      append([]string(nil), resource.Tags...),
		Origin:    asset.Origin{Kind: "workflow-resource", SourceID: resource.ID},
		CreatedAt: time.Now().UTC(),
	}
	switch resource.Kind {
	case schema.ResourceImage:
		record.Kind = asset.KindTemplate
		record.Variants = make([]asset.Variant, len(resource.Image.Variants))
		for index, variant := range resource.Image.Variants {
			record.Variants[index] = asset.Variant{
				Resolution: variant.Resolution, BBox: variant.BBox,
				Regions: append([][4]int(nil), variant.Regions...), Blob: variant.Blob,
			}
		}
	case schema.ResourceMacro:
		record.Kind = asset.KindMacro
		ref := resource.Macro.Blob
		record.Blob = &ref
	case schema.ResourceInputClip:
		record.Kind = asset.KindClip
		ref := resource.InputClip.Blob
		record.Blob = &ref
	default:
		return Promotion{}, errors.New("Workflow Resource promotion kind is invalid")
	}
	if err := c.assets.PublishExistingRecord(ctx, record); err != nil {
		return Promotion{}, fmt.Errorf("publish promoted Global Asset: %w", err)
	}
	return Promotion{GUID: record.GUID, Kind: record.Kind, Revision: c.assets.Revision()}, nil
}

func (c *Creator) CreateImage(ctx context.Context, draft ImageDraft) (schema.WorkflowResource, error) {
	metadata, err := normalizeMetadata(draft.Metadata)
	if err != nil {
		return schema.WorkflowResource{}, err
	}
	if draft.Resolution[0] <= 0 || draft.Resolution[1] <= 0 {
		return schema.WorkflowResource{}, errors.New("Workflow Resource image resolution must be positive")
	}
	if !validRegion(draft.Region) {
		return schema.WorkflowResource{}, errors.New("Workflow Resource image region is invalid")
	}
	if !strings.HasPrefix(draft.DataURL, pngDataURLPrefix) {
		return schema.WorkflowResource{}, fmt.Errorf("data URL must start with %q", pngDataURLPrefix)
	}
	content, err := base64.StdEncoding.DecodeString(strings.TrimPrefix(draft.DataURL, pngDataURLPrefix))
	if err != nil {
		return schema.WorkflowResource{}, fmt.Errorf("decode Workflow Resource image: %w", err)
	}
	config, err := png.DecodeConfig(bytes.NewReader(content))
	if err != nil || config.Width <= 0 || config.Height <= 0 {
		return schema.WorkflowResource{}, errors.New("Workflow Resource image must contain a valid PNG")
	}
	bbox := regionBBox(draft.Resolution, draft.Region)
	if bbox[0] < 0 || bbox[1] < 0 || bbox[2] <= bbox[0] || bbox[3] <= bbox[1] ||
		bbox[2] > draft.Resolution[0] || bbox[3] > draft.Resolution[1] {
		return schema.WorkflowResource{}, errors.New("Workflow Resource image region has no valid pixels")
	}
	ref, err := c.blobs.Put(ctx, "image/png", bytes.NewReader(content))
	if err != nil {
		return schema.WorkflowResource{}, fmt.Errorf("publish Workflow Resource image: %w", err)
	}
	id := "image-" + uuid.NewString()
	return schema.WorkflowResource{
		ID: id, Kind: schema.ResourceImage,
		Name: metadata.Name, Description: metadata.Description,
		Category: metadata.Category, Tags: metadata.Tags,
		Image: &schema.ImageResource{Variants: []schema.ImageResourceVariant{{
			ID: "default", Resolution: draft.Resolution, BBox: bbox, Blob: ref,
		}}},
	}, nil
}

func (c *Creator) CreateMacro(ctx context.Context, draft MacroDraft) (schema.WorkflowResource, error) {
	metadata, err := normalizeMetadata(draft.Metadata)
	if err != nil {
		return schema.WorkflowResource{}, err
	}
	if err := macro.Validate(draft.Document); err != nil {
		return schema.WorkflowResource{}, fmt.Errorf("validate Workflow Resource macro: %w", err)
	}
	var carrier bytes.Buffer
	if err := macro.Encode(&carrier, draft.Document); err != nil {
		return schema.WorkflowResource{}, fmt.Errorf("encode Workflow Resource macro: %w", err)
	}
	ref, err := c.blobs.Put(ctx, macro.MediaType, bytes.NewReader(carrier.Bytes()))
	if err != nil {
		return schema.WorkflowResource{}, fmt.Errorf("publish Workflow Resource macro: %w", err)
	}
	analysis := macro.Analyze(draft.Document)
	return schema.WorkflowResource{
		ID: "macro-" + uuid.NewString(), Kind: schema.ResourceMacro,
		Name: metadata.Name, Description: metadata.Description,
		Category: metadata.Category, Tags: metadata.Tags,
		Macro: &schema.MacroResource{
			Blob: ref, BaseResolution: draft.Document.BaseResolution,
			ActionCount: len(draft.Document.Actions), DurationUs: analysis.DurationUs,
		},
	}, nil
}

func (c *Creator) CreateInputClip(ctx context.Context, draft InputClipDraft) (schema.WorkflowResource, error) {
	metadata, err := normalizeMetadata(draft.Metadata)
	if err != nil {
		return schema.WorkflowResource{}, err
	}
	clip := draft.Clip
	clip.UpdateDuration()
	var carrier bytes.Buffer
	if err := inputclip.Encode(&carrier, &clip); err != nil {
		return schema.WorkflowResource{}, fmt.Errorf("encode Workflow Resource InputClip: %w", err)
	}
	if _, err := inputclip.Decode(bytes.NewReader(carrier.Bytes())); err != nil {
		return schema.WorkflowResource{}, fmt.Errorf("verify Workflow Resource InputClip: %w", err)
	}
	ref, err := c.blobs.Put(ctx, inputclip.MediaType, bytes.NewReader(carrier.Bytes()))
	if err != nil {
		return schema.WorkflowResource{}, fmt.Errorf("publish Workflow Resource InputClip: %w", err)
	}
	mouseMode := clip.Meta.MouseMode
	if mouseMode != "relative" && mouseMode != "absolute" {
		mouseMode = "mixed"
	}
	return schema.WorkflowResource{
		ID: "clip-" + uuid.NewString(), Kind: schema.ResourceInputClip,
		Name: metadata.Name, Description: metadata.Description,
		Category: metadata.Category, Tags: metadata.Tags,
		InputClip: &schema.InputClipResource{
			Blob: ref, DurationUs: clip.DurationUs, EventCount: len(clip.Events),
			RecordingMode: string(clip.Meta.RecordingMode), MouseMode: mouseMode,
			BaseResolution: clip.Meta.BaseResolution, MouseCounts360: clip.Meta.MouseCounts360,
			StopHotkeyVK: clip.Meta.StopHotkeyVK,
		},
	}, nil
}

func normalizeMetadata(metadata Metadata) (Metadata, error) {
	metadata.Name = strings.TrimSpace(metadata.Name)
	if metadata.Name == "" || len([]rune(metadata.Name)) > 80 {
		return Metadata{}, errors.New("Workflow Resource name must contain 1 to 80 characters")
	}
	metadata.Description = strings.TrimSpace(metadata.Description)
	metadata.Category = strings.TrimSpace(metadata.Category)
	if len([]rune(metadata.Description)) > 4096 || len([]rune(metadata.Category)) > 128 {
		return Metadata{}, errors.New("Workflow Resource presentation metadata exceeds its size budget")
	}
	metadata.Tags = normalizeTags(metadata.Tags)
	if len(metadata.Tags) > 64 {
		return Metadata{}, errors.New("Workflow Resource tags exceed the count budget")
	}
	return metadata, nil
}

func normalizeTags(tags []string) []string {
	seen := make(map[string]struct{}, len(tags))
	result := make([]string, 0, len(tags))
	for _, raw := range tags {
		tag := strings.TrimSpace(raw)
		key := strings.ToLower(tag)
		if tag == "" {
			continue
		}
		if _, duplicate := seen[key]; duplicate {
			continue
		}
		seen[key] = struct{}{}
		result = append(result, tag)
	}
	sort.Strings(result)
	return result
}

func validRegion(region [4]float32) bool {
	return region[0] >= 0 && region[1] >= 0 && region[2] > 0 && region[3] > 0 &&
		region[0]+region[2] <= 1 && region[1]+region[3] <= 1
}

func regionBBox(resolution [2]int, region [4]float32) [4]int {
	return [4]int{
		int(region[0] * float32(resolution[0])),
		int(region[1] * float32(resolution[1])),
		int((region[0] + region[2]) * float32(resolution[0])),
		int((region[1] + region[3]) * float32(resolution[1])),
	}
}
