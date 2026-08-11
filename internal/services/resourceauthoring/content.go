package resourceauthoring

import (
	"bytes"
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/yottaapp/yotta/internal/blob"
	"github.com/yottaapp/yotta/internal/services/inputclip"
	"github.com/yottaapp/yotta/internal/services/macro"
	"github.com/yottaapp/yotta/internal/workflow/schema"
)

const maxResourceEventPageSize = 200

const (
	EditMacroDocument = "macro-document"
	EditInputClipTrim = "input-clip-trim"
)

// Content is the decoded, verified authoring view of one portable resource.
// Presentation metadata stays in WorkflowResource; carrier-derived facts live
// here and cannot silently drift from the Source projection.
type Content struct {
	Kind      schema.ResourceKind `json:"kind"`
	Macro     *macro.Document     `json:"macro,omitempty"`
	InputClip *InputClipContent   `json:"inputClip,omitempty"`
}

type InputClipContent struct {
	DurationUs     uint64                  `json:"durationUs"`
	EventCount     int                     `json:"eventCount"`
	RecordingMode  inputclip.RecordingMode `json:"recordingMode"`
	MouseMode      string                  `json:"mouseMode"`
	BaseResolution [2]int                  `json:"baseResolution"`
	MouseCounts360 int                     `json:"mouseCounts360"`
	StopHotkeyVK   uint32                  `json:"stopHotkeyVk"`
	Tracks         []inputclip.EventTrack  `json:"tracks"`
}

type Edit struct {
	Kind      string         `json:"kind"`
	Macro     *MacroEdit     `json:"macro,omitempty"`
	InputClip *InputClipEdit `json:"inputClip,omitempty"`
}

type MacroEdit struct {
	Document macro.Document `json:"document"`
}

type InputClipEdit struct {
	TrimStartUs uint64 `json:"trimStartUs"`
	TrimEndUs   uint64 `json:"trimEndUs"`
}

type EventPage struct {
	Items  []inputclip.Event `json:"items"`
	Total  int               `json:"total"`
	Offset int               `json:"offset"`
	Limit  int               `json:"limit"`
}

func (c *Creator) Open(ctx context.Context, resource schema.WorkflowResource) (Content, error) {
	if err := schema.ValidateWorkflowResource(resource); err != nil {
		return Content{}, fmt.Errorf("validate Workflow Resource content: %w", err)
	}
	switch resource.Kind {
	case schema.ResourceImage:
		for _, variant := range resource.Image.Variants {
			if err := c.blobs.Verify(ctx, variant.Blob); err != nil {
				return Content{}, fmt.Errorf("verify Workflow Resource image variant %q: %w", variant.ID, err)
			}
		}
		return Content{Kind: resource.Kind}, nil
	case schema.ResourceMacro:
		document, err := c.openMacro(ctx, resource)
		if err != nil {
			return Content{}, err
		}
		return Content{Kind: resource.Kind, Macro: &document}, nil
	case schema.ResourceInputClip:
		clip, err := c.openInputClip(ctx, resource)
		if err != nil {
			return Content{}, err
		}
		return Content{Kind: resource.Kind, InputClip: inputClipContent(clip)}, nil
	default:
		return Content{}, errors.New("workflow Resource kind is invalid")
	}
}

func (c *Creator) Rewrite(ctx context.Context, resource schema.WorkflowResource, edit Edit) (schema.WorkflowResource, error) {
	if _, err := c.Open(ctx, resource); err != nil {
		return schema.WorkflowResource{}, err
	}
	switch edit.Kind {
	case EditMacroDocument:
		if resource.Kind != schema.ResourceMacro || edit.Macro == nil || edit.InputClip != nil {
			return schema.WorkflowResource{}, errors.New("macro edit does not match the Workflow Resource")
		}
		if err := macro.Validate(edit.Macro.Document); err != nil {
			return schema.WorkflowResource{}, fmt.Errorf("validate Workflow Resource macro edit: %w", err)
		}
		var carrier bytes.Buffer
		if err := macro.Encode(&carrier, edit.Macro.Document); err != nil {
			return schema.WorkflowResource{}, fmt.Errorf("encode Workflow Resource macro edit: %w", err)
		}
		ref, err := c.blobs.Put(ctx, macro.MediaType, bytes.NewReader(carrier.Bytes()))
		if err != nil {
			return schema.WorkflowResource{}, fmt.Errorf("publish Workflow Resource macro edit: %w", err)
		}
		analysis := macro.Analyze(edit.Macro.Document)
		updated := cloneResource(resource)
		updated.Macro = &schema.MacroResource{
			Blob: ref, BaseResolution: edit.Macro.Document.BaseResolution,
			ActionCount: len(edit.Macro.Document.Actions), DurationUs: analysis.DurationUs,
		}
		return updated, nil
	case EditInputClipTrim:
		if resource.Kind != schema.ResourceInputClip || edit.InputClip == nil || edit.Macro != nil {
			return schema.WorkflowResource{}, errors.New("InputClip edit does not match the Workflow Resource")
		}
		clip, err := c.openInputClip(ctx, resource)
		if err != nil {
			return schema.WorkflowResource{}, err
		}
		events, err := inputclip.TrimEvents(clip.Events, edit.InputClip.TrimStartUs, edit.InputClip.TrimEndUs)
		if err != nil {
			return schema.WorkflowResource{}, fmt.Errorf("trim Workflow Resource InputClip: %w", err)
		}
		clip.Events = events
		clip.UpdateDuration()
		var carrier bytes.Buffer
		if err := inputclip.Encode(&carrier, clip); err != nil {
			return schema.WorkflowResource{}, fmt.Errorf("encode Workflow Resource InputClip edit: %w", err)
		}
		ref, err := c.blobs.Put(ctx, inputclip.MediaType, bytes.NewReader(carrier.Bytes()))
		if err != nil {
			return schema.WorkflowResource{}, fmt.Errorf("publish Workflow Resource InputClip edit: %w", err)
		}
		updated := cloneResource(resource)
		updated.InputClip = &schema.InputClipResource{
			Blob: ref, DurationUs: clip.DurationUs, EventCount: len(clip.Events),
			RecordingMode: string(clip.Meta.RecordingMode), MouseMode: clip.Meta.MouseMode,
			BaseResolution: clip.Meta.BaseResolution, MouseCounts360: clip.Meta.MouseCounts360,
			StopHotkeyVK: clip.Meta.StopHotkeyVK,
		}
		return updated, nil
	default:
		return schema.WorkflowResource{}, errors.New("workflow Resource edit kind is invalid")
	}
}

func (c *Creator) Events(ctx context.Context, resource schema.WorkflowResource, offset, limit int) (EventPage, error) {
	if offset < 0 || limit <= 0 || limit > maxResourceEventPageSize {
		return EventPage{}, errors.New("workflow Resource event page is outside the bounded range")
	}
	if err := schema.ValidateWorkflowResource(resource); err != nil {
		return EventPage{}, fmt.Errorf("validate Workflow Resource events: %w", err)
	}
	if resource.Kind != schema.ResourceInputClip {
		return EventPage{}, errors.New("raw events are available only for InputClip resources")
	}
	clip, err := c.openInputClip(ctx, resource)
	if err != nil {
		return EventPage{}, err
	}
	total := len(clip.Events)
	start := min(offset, total)
	end := min(start+limit, total)
	return EventPage{
		Items: append([]inputclip.Event(nil), clip.Events[start:end]...),
		Total: total, Offset: start, Limit: limit,
	}, nil
}

func (c *Creator) Duplicate(ctx context.Context, resource schema.WorkflowResource) (schema.WorkflowResource, error) {
	if _, err := c.Open(ctx, resource); err != nil {
		return schema.WorkflowResource{}, err
	}
	duplicate := cloneResource(resource)
	switch resource.Kind {
	case schema.ResourceImage:
		duplicate.ID = "image-" + uuid.NewString()
	case schema.ResourceMacro:
		duplicate.ID = "macro-" + uuid.NewString()
	case schema.ResourceInputClip:
		duplicate.ID = "clip-" + uuid.NewString()
	default:
		return schema.WorkflowResource{}, errors.New("workflow Resource kind is invalid")
	}
	return duplicate, nil
}

func (c *Creator) openMacro(ctx context.Context, resource schema.WorkflowResource) (macro.Document, error) {
	content, err := c.readBlob(ctx, resource.Macro.Blob)
	if err != nil {
		return macro.Document{}, fmt.Errorf("read Workflow Resource macro: %w", err)
	}
	document, err := macro.Decode(bytes.NewReader(content))
	if err != nil {
		return macro.Document{}, fmt.Errorf("decode Workflow Resource macro: %w", err)
	}
	analysis := macro.Analyze(document)
	if len(analysis.Issues) != 0 || resource.Macro.BaseResolution != document.BaseResolution ||
		resource.Macro.ActionCount != len(document.Actions) || resource.Macro.DurationUs != analysis.DurationUs {
		return macro.Document{}, errors.New("workflow Resource macro carrier does not match Source metadata")
	}
	return document, nil
}

func (c *Creator) openInputClip(ctx context.Context, resource schema.WorkflowResource) (*inputclip.InputClip, error) {
	content, err := c.readBlob(ctx, resource.InputClip.Blob)
	if err != nil {
		return nil, fmt.Errorf("read Workflow Resource InputClip: %w", err)
	}
	clip, err := inputclip.Decode(bytes.NewReader(content))
	if err != nil {
		return nil, fmt.Errorf("decode Workflow Resource InputClip: %w", err)
	}
	metadata := resource.InputClip
	if metadata.DurationUs != clip.DurationUs || metadata.EventCount != len(clip.Events) ||
		metadata.RecordingMode != string(clip.Meta.RecordingMode) || metadata.MouseMode != clip.Meta.MouseMode ||
		metadata.BaseResolution != clip.Meta.BaseResolution || metadata.MouseCounts360 != clip.Meta.MouseCounts360 ||
		metadata.StopHotkeyVK != clip.Meta.StopHotkeyVK {
		return nil, errors.New("workflow Resource InputClip carrier does not match Source metadata")
	}
	return clip, nil
}

func (c *Creator) readBlob(ctx context.Context, ref blob.BlobRef) ([]byte, error) {
	return c.blobs.ReadRange(ctx, ref, 0, ref.Size)
}

func inputClipContent(clip *inputclip.InputClip) *InputClipContent {
	return &InputClipContent{
		DurationUs: clip.DurationUs, EventCount: len(clip.Events),
		RecordingMode: clip.Meta.RecordingMode, MouseMode: clip.Meta.MouseMode,
		BaseResolution: clip.Meta.BaseResolution, MouseCounts360: clip.Meta.MouseCounts360,
		StopHotkeyVK: clip.Meta.StopHotkeyVK, Tracks: inputclip.EventTracks(clip.Events),
	}
}

func cloneResource(resource schema.WorkflowResource) schema.WorkflowResource {
	clone := resource
	clone.Tags = append([]string(nil), resource.Tags...)
	if resource.Image != nil {
		image := *resource.Image
		image.Variants = append([]schema.ImageResourceVariant(nil), resource.Image.Variants...)
		for index := range image.Variants {
			image.Variants[index].Regions = append([][4]int(nil), image.Variants[index].Regions...)
		}
		clone.Image = &image
	}
	if resource.Macro != nil {
		value := *resource.Macro
		clone.Macro = &value
	}
	if resource.InputClip != nil {
		value := *resource.InputClip
		clone.InputClip = &value
	}
	return clone
}
