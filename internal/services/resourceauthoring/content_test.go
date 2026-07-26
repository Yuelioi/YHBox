package resourceauthoring

import (
	"context"
	"testing"

	"github.com/yottaapp/yotta/internal/services/inputclip"
	"github.com/yottaapp/yotta/internal/services/macro"
	"github.com/yottaapp/yotta/internal/workflow/schema"
)

func TestCreatorOpensAndRewritesMacroWithoutGlobalAsset(t *testing.T) {
	creator, _, assets := newTestCreator(t)
	resource, err := creator.CreateMacro(context.Background(), MacroDraft{
		Metadata: Metadata{Name: "Shared macro"},
		Document: macro.Document{
			SchemaVersion: macro.SchemaVersion, BaseResolution: [2]int{1280, 720},
			Actions: []macro.Action{{ID: "wait", Kind: macro.ActionSleep, DurationUs: 10_000}},
		},
	})
	if err != nil {
		t.Fatal(err)
	}

	opened, err := creator.Open(context.Background(), resource)
	if err != nil {
		t.Fatal(err)
	}
	if opened.Kind != schema.ResourceMacro || opened.Macro == nil ||
		len(opened.Macro.Actions) != 1 || opened.InputClip != nil {
		t.Fatalf("opened Macro = %#v", opened)
	}

	document := *opened.Macro
	document.Actions = []macro.Action{
		{ID: "wait-1", Kind: macro.ActionSleep, DurationUs: 20_000},
		{ID: "wait-2", Kind: macro.ActionSleep, DurationUs: 30_000},
	}
	updated, err := creator.Rewrite(context.Background(), resource, Edit{
		Kind: EditMacroDocument, Macro: &MacroEdit{Document: document},
	})
	if err != nil {
		t.Fatal(err)
	}
	if updated.ID != resource.ID || updated.Name != resource.Name || updated.Kind != resource.Kind ||
		updated.Macro.ActionCount != 2 || updated.Macro.DurationUs != 50_000 ||
		updated.Macro.Blob == resource.Macro.Blob {
		t.Fatalf("rewritten Macro = %#v", updated)
	}
	if records := assets.List(); len(records) != 0 {
		t.Fatalf("Workflow Resource rewrite created Global Assets: %#v", records)
	}
}

func TestCreatorOpensPagesAndTrimsInputClipFromPortableCarrier(t *testing.T) {
	creator, _, assets := newTestCreator(t)
	resource, err := creator.CreateInputClip(context.Background(), InputClipDraft{
		Metadata: Metadata{Name: "Shared clip"},
		Clip: inputclip.InputClip{
			Meta: inputclip.ClipMeta{
				RecordingMode: inputclip.RecordingModePrecise, MouseMode: "relative",
				BaseResolution: [2]int{1920, 1080}, MouseCounts360: 1440, StopHotkeyVK: 0x7B,
			},
			Events: []inputclip.Event{
				{TUs: 0, Type: inputclip.EventTypeRawDelta, B: 1},
				{TUs: 100, Seq: 1, Type: inputclip.EventTypeRawDelta, B: 2},
				{TUs: 200, Seq: 2, Type: inputclip.EventTypeScroll, A: 1},
			},
		},
	})
	if err != nil {
		t.Fatal(err)
	}

	opened, err := creator.Open(context.Background(), resource)
	if err != nil {
		t.Fatal(err)
	}
	if opened.InputClip == nil || opened.InputClip.EventCount != 3 ||
		opened.InputClip.DurationUs != 200 || opened.InputClip.MouseCounts360 != 1440 ||
		len(opened.InputClip.Tracks) != 2 {
		t.Fatalf("opened InputClip = %#v", opened)
	}
	page, err := creator.Events(context.Background(), resource, 1, 2)
	if err != nil {
		t.Fatal(err)
	}
	if page.Total != 3 || page.Offset != 1 || len(page.Items) != 2 || page.Items[0].TUs != 100 {
		t.Fatalf("event page = %#v", page)
	}

	updated, err := creator.Rewrite(context.Background(), resource, Edit{
		Kind: EditInputClipTrim, InputClip: &InputClipEdit{TrimStartUs: 100, TrimEndUs: 200},
	})
	if err != nil {
		t.Fatal(err)
	}
	if updated.ID != resource.ID || updated.InputClip.DurationUs != 100 ||
		updated.InputClip.EventCount != 2 || updated.InputClip.MouseCounts360 != 1440 ||
		updated.InputClip.Blob == resource.InputClip.Blob {
		t.Fatalf("trimmed InputClip = %#v", updated)
	}
	if records := assets.List(); len(records) != 0 {
		t.Fatalf("Workflow Resource trim created Global Assets: %#v", records)
	}
}

func TestCreatorRejectsCarrierMetadataDriftAndMissingBlob(t *testing.T) {
	creator, _, _ := newTestCreator(t)
	resource, err := creator.CreateMacro(context.Background(), MacroDraft{
		Metadata: Metadata{Name: "Macro"},
		Document: macro.Document{
			SchemaVersion: macro.SchemaVersion, BaseResolution: [2]int{1280, 720},
			Actions: []macro.Action{{ID: "wait", Kind: macro.ActionSleep, DurationUs: 10}},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	drifted := resource
	macroMetadata := *resource.Macro
	macroMetadata.ActionCount++
	drifted.Macro = &macroMetadata
	if _, err := creator.Open(context.Background(), drifted); err == nil {
		t.Fatal("Open accepted carrier metadata drift")
	}

	missing := resource
	missingMetadata := *resource.Macro
	missingMetadata.Blob.Digest = "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
	missing.Macro = &missingMetadata
	if _, err := creator.Open(context.Background(), missing); err == nil {
		t.Fatal("Open accepted a missing BlobRef")
	}
}

func TestCreatorDuplicatesAllResourceKindsWithoutCopyingContent(t *testing.T) {
	creator, _, _ := newTestCreator(t)
	imageResource, err := creator.CreateImage(context.Background(), ImageDraft{
		Metadata: Metadata{Name: "Image"}, DataURL: pngDataURL(t, 8, 8),
		Resolution: [2]int{8, 8}, Region: [4]float32{0, 0, 1, 1},
	})
	if err != nil {
		t.Fatal(err)
	}
	macroResource, err := creator.CreateMacro(context.Background(), MacroDraft{
		Metadata: Metadata{Name: "Macro"},
		Document: macro.Document{
			SchemaVersion: macro.SchemaVersion, BaseResolution: [2]int{1280, 720},
			Actions: []macro.Action{{ID: "wait", Kind: macro.ActionSleep, DurationUs: 10}},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	clipResource, err := creator.CreateInputClip(context.Background(), InputClipDraft{
		Metadata: Metadata{Name: "Clip"},
		Clip: inputclip.InputClip{
			Meta: inputclip.ClipMeta{
				RecordingMode: inputclip.RecordingModePrecise, MouseMode: "absolute",
				BaseResolution: [2]int{1280, 720},
			},
			Events: []inputclip.Event{{TUs: 0, Type: inputclip.EventTypeMouseMove}},
		},
	})
	if err != nil {
		t.Fatal(err)
	}

	for _, resource := range []schema.WorkflowResource{imageResource, macroResource, clipResource} {
		duplicate, err := creator.Duplicate(context.Background(), resource)
		if err != nil {
			t.Fatal(err)
		}
		if duplicate.ID == resource.ID || duplicate.Kind != resource.Kind || duplicate.Name != resource.Name {
			t.Fatalf("duplicate = %#v, source = %#v", duplicate, resource)
		}
		switch resource.Kind {
		case schema.ResourceImage:
			if duplicate.Image.Variants[0].Blob != resource.Image.Variants[0].Blob {
				t.Fatal("image duplicate copied content identity")
			}
		case schema.ResourceMacro:
			if duplicate.Macro.Blob != resource.Macro.Blob {
				t.Fatal("Macro duplicate copied content identity")
			}
		case schema.ResourceInputClip:
			if duplicate.InputClip.Blob != resource.InputClip.Blob {
				t.Fatal("InputClip duplicate copied content identity")
			}
		}
	}
}
