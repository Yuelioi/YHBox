package recording

import (
	"testing"

	"github.com/yottaapp/yotta/internal/blob"
	"github.com/yottaapp/yotta/internal/nodecontract"
	"github.com/yottaapp/yotta/internal/nodes"
	"github.com/yottaapp/yotta/internal/services/inputclip"
)

func TestRecordingPreviewAndDraftExpandSimpleKeysAndClick(t *testing.T) {
	result := &StopResult{
		Meta: inputclip.ClipMeta{RecordingMode: inputclip.RecordingModeSimple, BaseResolution: [2]int{1280, 720}},
		Events: []inputclip.Event{
			{TUs: 0, Type: inputclip.EventTypeKeyDown, A: int32(VK_CONTROL)},
			{TUs: 10_000, Type: inputclip.EventTypeKeyDown, A: 'C'},
			{TUs: 40_000, Type: inputclip.EventTypeKeyUp, A: 'C'},
			{TUs: 50_000, Type: inputclip.EventTypeKeyUp, A: int32(VK_CONTROL)},
			{TUs: 200_000, Type: inputclip.EventTypeMouseBtnDown, A: int32(HookBtnLeft), B: 640, C: 360},
			{TUs: 250_000, Type: inputclip.EventTypeMouseBtnUp, A: int32(HookBtnLeft), B: 640, C: 360},
		},
	}
	preview := recordingPreview(result)
	if preview.Mode != "simple" || preview.KeyActions != 1 || preview.ClickActions != 1 || len(preview.Steps) != 2 {
		t.Fatalf("preview = %+v", preview)
	}
	draft := buildWorkflowDraft(result, "editor", blob.BlobRef{})
	if draft.Mode != inputclip.RecordingModeSimple || len(draft.Nodes) != 3 {
		t.Fatalf("draft = %+v", draft)
	}
	keys, ok := draft.Nodes[0].Values["keys"].([]string)
	if !ok || len(keys) != 2 || keys[0] != "CTRL" || keys[1] != "C" {
		t.Fatalf("keys = %#v", draft.Nodes[0].Values["keys"])
	}
	if draft.Nodes[1].NodeTypeID != delayNodeID || draft.Nodes[1].Values["duration-milliseconds"] != int64(150) {
		t.Fatalf("delay = %+v", draft.Nodes[1])
	}
	point, ok := draft.Nodes[2].Values["point"].(Point)
	if !ok || point.X != 0.5 || point.Y != 0.5 || point.Unit != "ratio" {
		t.Fatalf("point = %#v", draft.Nodes[2].Values["point"])
	}
	assertDraftContracts(t, draft)
}

func TestRecordingDraftRetainsPreciseRecordingAsInputClip(t *testing.T) {
	result := &StopResult{Meta: inputclip.ClipMeta{RecordingMode: inputclip.RecordingModePrecise}, Events: []inputclip.Event{
		{TUs: 0, Type: inputclip.EventTypeRawDelta, B: 12, C: -4},
		{TUs: 25_000, Type: inputclip.EventTypeRawDelta, B: 8, C: 2},
	}}
	preview := recordingPreview(result)
	if preview.Mode != "precise" || preview.RawDeltas != 2 {
		t.Fatalf("preview = %+v", preview)
	}
	ref := blob.BlobRef{MediaType: inputclip.MediaType, Digest: "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa", Size: 64}
	draft := buildWorkflowDraft(result, "game", ref)
	if draft.Mode != inputclip.RecordingModePrecise || len(draft.Nodes) != 1 || draft.Nodes[0].Blobs["clip"] != ref {
		t.Fatalf("draft = %+v", draft)
	}
	assertDraftContracts(t, draft)
}

func assertDraftContracts(t *testing.T, draft WorkflowDraft) {
	t.Helper()
	builtins, err := nodes.Build()
	if err != nil {
		t.Fatal(err)
	}
	for _, draftNode := range draft.Nodes {
		entry, ok := builtins.Catalog.Lookup(draftNode.NodeTypeID)
		if !ok {
			t.Fatalf("draft node %q is not in the Catalog", draftNode.NodeTypeID)
		}
		machine := entry.Contract.Machine()
		if !hasSignalInput(machine.Ports.ExecInputs, draftNode.ExecInput) || !hasSignalOutput(machine.Ports.ExecOutputs, draftNode.ExecOutput) {
			t.Fatalf("draft signal ports do not match %q", draftNode.NodeTypeID)
		}
		for portID := range draftNode.Values {
			if !hasDataInput(machine.Ports.DataInputs, portID) {
				t.Fatalf("draft value port %q is not on %q", portID, draftNode.NodeTypeID)
			}
		}
		for portID := range draftNode.Blobs {
			if !hasDataInput(machine.Ports.DataInputs, portID) {
				t.Fatalf("draft blob port %q is not on %q", portID, draftNode.NodeTypeID)
			}
		}
	}
}

func hasSignalInput(ports []nodecontract.SignalPort, id string) bool {
	for _, port := range ports {
		if port.ID == id {
			return true
		}
	}
	return false
}

func hasSignalOutput(ports []nodecontract.SignalPort, id string) bool {
	return hasSignalInput(ports, id)
}

func hasDataInput(ports []nodecontract.DataInputPort, id string) bool {
	for _, port := range ports {
		if port.ID == id {
			return true
		}
	}
	return false
}
