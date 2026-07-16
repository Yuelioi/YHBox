package nodes31runtime_test

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
	"github.com/yottaapp/yotta/internal/nodes31"
	"github.com/yottaapp/yotta/internal/nodes31runtime"
	"github.com/yottaapp/yotta/internal/resource"
	run31 "github.com/yottaapp/yotta/internal/run"
	"github.com/yottaapp/yotta/internal/workflow/compiler"
)

type automationCaptureProvider struct {
	data    []byte
	closed  int
	capture int
	err     error
}

func (provider *automationCaptureProvider) Open(_ context.Context, request resource.ProviderOpenRequest) (any, error) {
	if request.Kind != automationinstalled.KindCapture || fmt.Sprint(request.Operations) != "[capture read-capture]" || string(request.Config) != `{}` || string(request.CapabilityScope) != `{"operation":"capture"}` {
		return nil, fmt.Errorf("unexpected automation capture open request: %#v", request)
	}
	return struct{}{}, nil
}

func (provider *automationCaptureProvider) Invoke(_ context.Context, _ any, operation string, payload []byte) ([]byte, error) {
	if provider.err != nil {
		return nil, provider.err
	}
	switch operation {
	case automationinstalled.OperationCapture:
		if string(payload) != `{}` {
			return nil, errors.New("unexpected capture payload")
		}
		provider.capture++
		return artifact.Marshal(automationinstalled.CaptureResponse{MediaType: "image/png", Size: int64(len(provider.data))})
	case automationinstalled.OperationReadCapture:
		var request automationinstalled.CaptureRangeRequest
		if err := json.Unmarshal(payload, &request); err != nil || request.Offset < 0 || request.Length <= 0 || request.Offset+request.Length > int64(len(provider.data)) {
			return nil, errors.New("unexpected capture range")
		}
		return append([]byte(nil), provider.data[request.Offset:request.Offset+request.Length]...), nil
	default:
		return nil, errors.New("unexpected capture operation")
	}
}

func (provider *automationCaptureProvider) Close(context.Context, any) error {
	provider.closed++
	return nil
}

func TestCaptureWindowCommitsNominalImageBlobAndBoundedJournal(t *testing.T) {
	builtins, err := nodes31.Build()
	if err != nil {
		t.Fatal(err)
	}
	want := bytes.Repeat([]byte("png-chunk"), 9000)
	captureProvider := &automationCaptureProvider{data: want}
	store, err := blob.Open(t.TempDir(), blob.Limits{MaxBlobBytes: 1 << 20, MaxTotalBytes: 2 << 20})
	if err != nil {
		t.Fatal(err)
	}
	blobProvider, err := blob.NewProvider(store, blob.ProviderLimits{MaxChunkBytes: 64 << 10, QueueCapacity: 4})
	if err != nil {
		t.Fatal(err)
	}
	providerDigest, err := artifact.Sum("yotta/test/automation-capture-provider/v1", []byte("exact-installed-window-capture"))
	if err != nil {
		t.Fatal(err)
	}
	const providerID, targetID, slot = "automation-capture-test", "automation-target/capture-test", "capture-test"
	captureCapability, ok := builtins.Catalog.LookupCapability(nodes31.AutomationCaptureCapabilityID)
	if !ok {
		t.Fatal("automation capture capability is missing")
	}
	profileDraft := executionProfile(t, builtins)
	profileDraft.Providers = append(profileDraft.Providers, admission.ProviderDescriptor{
		ID: providerID, ArtifactDigest: providerDigest, ABI: automationinstalled.ProviderABI, PluginInstanceID: "builtin",
		OperatingSystems: []string{"windows"}, Architectures: []string{"amd64"}, HostAPIs: []string{"3.1"},
		Capabilities: []admission.ProviderCapability{{Capability: captureCapability.Ref(), ResourceKind: automationinstalled.KindCapture}},
	})
	profileDraft.Targets = append(profileDraft.Targets, admission.AutomationTarget{ID: targetID, Kind: automationinstalled.TargetKind, ProviderID: providerID})
	profileDraft.TargetSlots = append(profileDraft.TargetSlots, admission.TargetSlotBinding{Slot: slot, TargetID: targetID})
	program := compilePrimitiveProgram(t, builtins, automationCaptureSource(builtins, slot))
	now := time.Date(2026, 7, 16, 21, 0, 0, 0, time.UTC)
	consent, err := artifact.Sum("yotta/test/automation-capture-consent/v1", []byte(slot))
	if err != nil {
		t.Fatal(err)
	}
	_, owner, journal := admittedExecutionWithConsent(t, builtins, program, map[string]run31.InstalledProvider{
		blob.ProviderID: {ArtifactDigest: blobProviderDigest(t), ABI: blob.ProviderABI, Provider: blobProvider},
		providerID:      {ArtifactDigest: providerDigest, ABI: automationinstalled.ProviderABI, Provider: captureProvider},
	}, now, profileDraft, []artifact.Digest{consent})
	t.Cleanup(func() { _ = owner.Close(context.Background()) })
	adapters, err := nodes31runtime.Installed(builtins, testDependencies())
	if err != nil {
		t.Fatal(err)
	}
	execution, err := compiler.NewExecutor(builtins.Catalog, adapters, compiler.ExecutorOptions{Now: func() time.Time { return now }}).Run(context.Background(), program, owner, journal)
	if err != nil {
		t.Fatal(err)
	}
	ref, ok := execution.NodeOutputs["capture"]["image"].BlobRef()
	if !ok || ref.MediaType != "image/png" || ref.Size != int64(len(want)) {
		t.Fatalf("image BlobRef = %#v, ok=%v", ref, ok)
	}
	got, err := store.ReadRange(context.Background(), ref, 0, ref.Size)
	if err != nil || !bytes.Equal(got, want) {
		t.Fatalf("stored capture bytes=%d error=%v", len(got), err)
	}
	if captureProvider.capture != 1 || captureProvider.closed != 1 {
		t.Fatalf("capture=%d closed=%d", captureProvider.capture, captureProvider.closed)
	}
	actions := 0
	for _, entry := range journal.Current().Journal() {
		if entry.Kind == run31.JournalAdapterAction {
			actions++
			if entry.Action != "automation.capture-window" || entry.ActionOutcome != run31.ActionSucceeded || entry.Summary.Counters["bytes"] != int64(len(want)) || entry.Summary.Counters["chunks"] != 2 {
				t.Fatalf("capture action = %#v", entry)
			}
		}
	}
	if actions != 1 {
		t.Fatalf("actions=%d", actions)
	}
}

func automationCaptureSource(builtins nodes31.Builtins, slot string) []byte {
	started, _ := builtins.Definition(nodes31.RunStartedNodeID)
	capture, _ := builtins.Definition(nodes31.CaptureWindowNodeID)
	return []byte(fmt.Sprintf(`{
		"format":"yotta.workflow","version":"3.1","workflow":{"id":"wf-automation-capture","name":"Automation Capture"},"revision":0,"entryGraph":"main",
		"graphs":[{"id":"main","kind":"main","nodes":[
			{"id":"start","nodeRef":{"nodeTypeId":%q,"semanticDigest":%q},"position":{"x":0,"y":0},"config":{},"bindings":{}},
			{"id":"capture","nodeRef":{"nodeTypeId":%q,"semanticDigest":%q},"position":{"x":1,"y":0},"config":{"slot":%q},"bindings":{}}
		],"edges":[{"channel":"exec","from":{"nodeId":"start","portId":"started"},"to":{"nodeId":"capture","portId":"in"}}],"inputs":[],"outputs":[]}],"variables":[],"secretRefs":[]
	}`, started.Contract.NodeRef().NodeTypeID, started.Contract.NodeRef().SemanticDigest,
		capture.Contract.NodeRef().NodeTypeID, capture.Contract.NodeRef().SemanticDigest, slot))
}
