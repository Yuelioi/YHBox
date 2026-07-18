package noderuntime_test

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/yottaapp/yotta/internal/admission"
	"github.com/yottaapp/yotta/internal/artifact"
	automationinstalled "github.com/yottaapp/yotta/internal/automation/installed"
	"github.com/yottaapp/yotta/internal/blob"
	"github.com/yottaapp/yotta/internal/noderuntime"
	"github.com/yottaapp/yotta/internal/nodes"
	"github.com/yottaapp/yotta/internal/resource"
	run "github.com/yottaapp/yotta/internal/run"
	"github.com/yottaapp/yotta/internal/services/inputclip"
	"github.com/yottaapp/yotta/internal/workflow/compiler"
)

type automationPlaybackProvider struct {
	events   []automationinstalled.PlaybackEvent
	releases int
	closed   int
}

func (provider *automationPlaybackProvider) Open(_ context.Context, request resource.ProviderOpenRequest) (any, error) {
	if request.Kind != automationinstalled.KindPlayback || fmt.Sprint(request.Operations) != "[play-event release-held]" || string(request.Config) != `{}` || string(request.CapabilityScope) != `{"operation":"play"}` {
		return nil, fmt.Errorf("unexpected automation playback open request: %#v", request)
	}
	return struct{}{}, nil
}

func (provider *automationPlaybackProvider) Invoke(_ context.Context, _ any, operation string, payload []byte) ([]byte, error) {
	switch operation {
	case automationinstalled.OperationPlayEvent:
		var event automationinstalled.PlaybackEvent
		if err := json.Unmarshal(payload, &event); err != nil {
			return nil, err
		}
		provider.events = append(provider.events, event)
	case automationinstalled.OperationReleaseHeld:
		if string(payload) != `{}` {
			return nil, errors.New("unexpected release payload")
		}
		provider.releases++
	default:
		return nil, errors.New("unexpected playback operation")
	}
	return []byte(`{}`), nil
}

func (provider *automationPlaybackProvider) Close(context.Context, any) error {
	provider.closed++
	return nil
}

func TestPlayInputClipReadsNominalBlobAndUsesExclusivePlaybackSession(t *testing.T) {
	builtins, err := nodes.Build()
	if err != nil {
		t.Fatal(err)
	}
	clip := &inputclip.InputClip{
		Meta: inputclip.ClipMeta{RecordingMode: inputclip.RecordingModePrecise, MouseMode: "mixed", BaseResolution: [2]int{1920, 1080}, MouseCounts360: 400},
		Events: []inputclip.Event{
			{TUs: 0, Seq: 0, Type: inputclip.EventTypeKeyDown, A: 0x41},
			{TUs: 1000, Seq: 0, Type: inputclip.EventTypeRawDelta, B: 3, C: -2},
			{TUs: 1000, Seq: 1, Type: inputclip.EventTypeKeyUp, A: 0x41},
		},
	}
	clip.UpdateDuration()
	var carrier bytes.Buffer
	if err := inputclip.Encode(&carrier, clip); err != nil {
		t.Fatal(err)
	}
	store, err := blob.Open(t.TempDir(), blob.Limits{MaxBlobBytes: 1 << 20, MaxTotalBytes: 2 << 20})
	if err != nil {
		t.Fatal(err)
	}
	ref, err := store.Put(context.Background(), inputclip.MediaType, bytes.NewReader(carrier.Bytes()))
	if err != nil {
		t.Fatal(err)
	}
	blobProvider, err := blob.NewProvider(store, blob.ProviderLimits{MaxChunkBytes: 64 << 10, QueueCapacity: 4})
	if err != nil {
		t.Fatal(err)
	}
	playbackProvider := &automationPlaybackProvider{}
	providerDigest, err := artifact.Sum("yotta/test/automation-playback-provider/v1", []byte("exact-installed-window-playback"))
	if err != nil {
		t.Fatal(err)
	}
	const providerID, targetID, slot = "automation-playback-test", "automation-target/playback-test", "playback-test"
	playbackCapability, ok := builtins.Catalog.LookupCapability(nodes.AutomationPlaybackCapabilityID)
	if !ok {
		t.Fatal("automation playback capability is missing")
	}
	profileDraft := executionProfile(t, builtins)
	profileDraft.Providers = append(profileDraft.Providers, admission.ProviderDescriptor{
		ID: providerID, ArtifactDigest: providerDigest, ABI: automationinstalled.ProviderABI, PluginInstanceID: "builtin",
		OperatingSystems: []string{"windows"}, Architectures: []string{"amd64"}, HostAPIs: []string{"3.1"},
		Capabilities: []admission.ProviderCapability{{Capability: playbackCapability.Ref(), ResourceKind: automationinstalled.KindPlayback}},
	})
	profileDraft.Targets = append(profileDraft.Targets, admission.AutomationTarget{ID: targetID, Kind: automationinstalled.TargetKind, ProviderID: providerID})
	profileDraft.TargetSlots = append(profileDraft.TargetSlots, admission.TargetSlotBinding{Slot: slot, TargetID: targetID})
	program := compilePrimitiveProgram(t, builtins, automationPlaybackSource(builtins, slot, ref))
	now := time.Date(2026, 7, 16, 22, 0, 0, 0, time.UTC)
	consent, err := artifact.Sum("yotta/test/automation-playback-consent/v1", []byte(slot))
	if err != nil {
		t.Fatal(err)
	}
	_, owner, journal := admittedExecutionWithConsent(t, builtins, program, map[string]run.InstalledProvider{
		blob.ProviderID: {ArtifactDigest: blobProviderDigest(t), ABI: blob.ProviderABI, Provider: blobProvider},
		providerID:      {ArtifactDigest: providerDigest, ABI: automationinstalled.ProviderABI, Provider: playbackProvider},
	}, now, profileDraft, []artifact.Digest{consent})
	t.Cleanup(func() { _ = owner.Close(context.Background()) })
	adapters, err := noderuntime.Installed(builtins, testDependencies())
	if err != nil {
		t.Fatal(err)
	}
	_, err = compiler.NewExecutor(builtins.Catalog, adapters, compiler.ExecutorOptions{Now: func() time.Time { return now }}).Run(context.Background(), program, owner, journal)
	if err != nil {
		t.Fatal(err)
	}
	if len(playbackProvider.events) != 3 || playbackProvider.events[1].Kind != automationinstalled.PlaybackMoveRelative ||
		playbackProvider.events[1].DeltaX != 3 || playbackProvider.events[1].DeltaY != -2 || playbackProvider.events[1].SourceCounts360 != 400 {
		t.Fatalf("playback events = %#v", playbackProvider.events)
	}
	if playbackProvider.releases != 1 || playbackProvider.closed != 1 {
		t.Fatalf("releases=%d closed=%d", playbackProvider.releases, playbackProvider.closed)
	}
	actions := 0
	for _, entry := range journal.Current().Journal() {
		if entry.Kind == run.JournalAdapterAction {
			actions++
			if entry.Action != "automation.play-input-clip" || entry.ActionOutcome != run.ActionSucceeded ||
				entry.Summary.Counters["events"] != 3 || entry.Summary.Counters["blob_bytes"] != ref.Size || entry.Summary.Counters["duration_ms"] != 1 {
				t.Fatalf("playback action = %#v", entry)
			}
		}
	}
	if actions != 1 {
		t.Fatalf("actions=%d", actions)
	}
}

func automationPlaybackSource(builtins nodes.Builtins, slot string, ref blob.BlobRef) []byte {
	started, _ := builtins.Definition(nodes.RunStartedNodeID)
	playback, _ := builtins.Definition(nodes.PlayInputClipNodeID)
	return []byte(fmt.Sprintf(`{
		"format":"yotta.workflow","version":"3.1","workflow":{"id":"wf-automation-playback","name":"Automation Playback"},"revision":0,"entryGraph":"main",
		"graphs":[{"id":"main","kind":"main","nodes":[
			{"id":"start","nodeRef":{"nodeTypeId":%q,"version":"1.0.0","semanticDigest":%q},"position":{"x":0,"y":0},"config":{},"bindings":{}},
			{"id":"playback","nodeRef":{"nodeTypeId":%q,"version":"1.0.0","semanticDigest":%q},"position":{"x":1,"y":0},"config":{"slot":%q},
			 "bindings":{"clip":{"kind":"blob","blob":{"mediaType":%q,"digest":%q,"size":%d}}}}
		],"edges":[{"channel":"exec","from":{"nodeId":"start","portId":"started"},"to":{"nodeId":"playback","portId":"in"}}],"inputs":[],"outputs":[]}],"variables":[],"secretRefs":[]
	}`, started.Contract.NodeRef().NodeTypeID, started.Contract.NodeRef().SemanticDigest,
		playback.Contract.NodeRef().NodeTypeID, playback.Contract.NodeRef().SemanticDigest, slot, ref.MediaType, ref.Digest, ref.Size))
}
