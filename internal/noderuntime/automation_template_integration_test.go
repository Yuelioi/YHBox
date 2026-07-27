package noderuntime_test

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"image"
	"image/color"
	"image/draw"
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
	"github.com/yottaapp/yotta/internal/workflow/compiler"
)

type templateAutomationProvider struct {
	frame    []byte
	clicks   []automationinstalled.ClickRequest
	closed   int
	captures int
}

func (provider *templateAutomationProvider) Open(_ context.Context, request resource.ProviderOpenRequest) (any, error) {
	switch request.Kind {
	case automationinstalled.KindCapture:
		if fmt.Sprint(request.Operations) != "[capture read-capture]" || string(request.CapabilityScope) != `{"operation":"capture"}` {
			return nil, fmt.Errorf("unexpected template capture open: %#v", request)
		}
		return automationinstalled.KindCapture, nil
	case automationinstalled.KindInput:
		if fmt.Sprint(request.Operations) != "[click]" || string(request.CapabilityScope) != `{"operation":"click"}` {
			return nil, fmt.Errorf("unexpected template input open: %#v", request)
		}
		return automationinstalled.KindInput, nil
	default:
		return nil, fmt.Errorf("unexpected template resource kind %q", request.Kind)
	}
}

func (provider *templateAutomationProvider) Invoke(_ context.Context, object any, operation string, payload []byte) ([]byte, error) {
	switch object {
	case automationinstalled.KindCapture:
		switch operation {
		case automationinstalled.OperationCapture:
			provider.captures++
			return artifact.Marshal(automationinstalled.CaptureResponse{MediaType: "image/png", Size: int64(len(provider.frame))})
		case automationinstalled.OperationReadCapture:
			var request automationinstalled.CaptureRangeRequest
			if err := json.Unmarshal(payload, &request); err != nil || request.Offset < 0 || request.Length <= 0 || request.Offset+request.Length > int64(len(provider.frame)) {
				return nil, errors.New("unexpected template capture range")
			}
			return append([]byte(nil), provider.frame[request.Offset:request.Offset+request.Length]...), nil
		}
	case automationinstalled.KindInput:
		if operation != automationinstalled.OperationClick {
			return nil, errors.New("unexpected template input operation")
		}
		var request automationinstalled.ClickRequest
		if err := json.Unmarshal(payload, &request); err != nil {
			return nil, err
		}
		provider.clicks = append(provider.clicks, request)
		return []byte(`{}`), nil
	}
	return nil, errors.New("unexpected template automation invocation")
}

func (provider *templateAutomationProvider) Close(context.Context, any) error {
	provider.closed++
	return nil
}

func TestClickTemplateCapturesMatchesAndClicksTheSameExactTarget(t *testing.T) {
	builtins, err := nodes.Build()
	if err != nil {
		t.Fatal(err)
	}
	frame := image.NewRGBA(image.Rect(0, 0, 80, 60))
	draw.Draw(frame, frame.Bounds(), &image.Uniform{C: color.RGBA{R: 18, G: 22, B: 28, A: 255}}, image.Point{}, draw.Src)
	templateImage := image.NewRGBA(image.Rect(0, 0, 12, 10))
	for y := range 10 {
		for x := range 12 {
			templateImage.SetRGBA(x, y, color.RGBA{R: uint8(20 + x*13), G: uint8(30 + y*17), B: uint8(40 + (x+y)*7), A: 255})
		}
	}
	draw.Draw(frame, image.Rect(24, 18, 36, 28), templateImage, image.Point{}, draw.Src)
	framePNG, templatePNG := encodeVisionPNG(t, frame), encodeVisionPNG(t, templateImage)

	store, err := blob.Open(t.TempDir(), blob.Limits{MaxBlobBytes: 1 << 20, MaxTotalBytes: 2 << 20})
	if err != nil {
		t.Fatal(err)
	}
	templateRef, err := store.Put(context.Background(), "image/png", bytes.NewReader(templatePNG))
	if err != nil {
		t.Fatal(err)
	}
	blobProvider, err := blob.NewProvider(store, blob.ProviderLimits{MaxChunkBytes: 64 << 10, QueueCapacity: 4})
	if err != nil {
		t.Fatal(err)
	}
	provider := &templateAutomationProvider{frame: framePNG}
	providerDigest, err := artifact.Sum("yotta/test/template-automation-provider/v1", []byte("same-exact-target"))
	if err != nil {
		t.Fatal(err)
	}
	const providerID, targetID, slot = "template-automation-test", "automation-target/template", "template-target"
	profileDraft := executionProfile(t, builtins)
	captureCapability, _ := builtins.Catalog.LookupCapability(nodes.AutomationCaptureCapabilityID)
	inputCapability, _ := builtins.Catalog.LookupCapability(nodes.AutomationInputCapabilityID)
	profileDraft.Providers = append(profileDraft.Providers, admission.ProviderDescriptor{
		ID: providerID, ArtifactDigest: providerDigest, ABI: automationinstalled.ProviderABI, PluginInstanceID: "builtin",
		OperatingSystems: []string{"windows"}, Architectures: []string{"amd64"}, HostAPIs: []string{"1.0"},
		Capabilities: []admission.ProviderCapability{
			{Capability: captureCapability.Ref(), ResourceKind: automationinstalled.KindCapture},
			{Capability: inputCapability.Ref(), ResourceKind: automationinstalled.KindInput},
		},
	})
	profileDraft.Targets = append(profileDraft.Targets, admission.AutomationTarget{ID: targetID, Kind: automationinstalled.TargetKindDesktopWindow, ProviderID: providerID})
	profileDraft.TargetSlots = append(profileDraft.TargetSlots, admission.TargetSlotBinding{Slot: slot, TargetID: targetID})
	program := compilePrimitiveProgram(t, builtins, clickTemplateSource(builtins, slot, templateRef))
	now := time.Date(2026, 7, 17, 15, 0, 0, 0, time.UTC)
	consent, _ := artifact.Sum("yotta/test/template-automation-consent/v1", []byte(slot))
	_, owner, journal := admittedExecutionWithConsent(t, builtins, program, map[string]run.InstalledProvider{
		blob.ProviderID: {ArtifactDigest: blobProviderDigest(t), ABI: blob.ProviderABI, Provider: blobProvider},
		providerID:      {ArtifactDigest: providerDigest, ABI: automationinstalled.ProviderABI, Provider: provider},
	}, now, profileDraft, []artifact.Digest{consent})
	t.Cleanup(func() { _ = owner.Close(context.Background()) })
	adapters, err := noderuntime.Installed(builtins, testDependencies())
	if err != nil {
		t.Fatal(err)
	}
	result, err := compiler.NewExecutor(builtins.Catalog, adapters, compiler.ExecutorOptions{Now: func() time.Time { return now }}).Run(context.Background(), program, owner, journal)
	if err != nil {
		t.Fatal(err)
	}
	if len(provider.clicks) != 1 || provider.clicks[0].Point.Unit != "ratio" || provider.clicks[0].Point.X != 0.375 || provider.clicks[0].Point.Y != 23.0/60.0 {
		t.Fatalf("template clicks = %#v", provider.clicks)
	}
	matched := result.NodeOutputs["click"]["matched"].InlineJSON()
	if string(matched) != "true" || provider.closed != 2 {
		t.Fatalf("matched=%s closed=%d", matched, provider.closed)
	}
	var statuses []string
	for _, entry := range journal.Current().Journal() {
		if entry.Kind == run.JournalNodeStatus {
			statuses = append(statuses, entry.StatusCode)
		}
	}
	if fmt.Sprint(statuses) != "[automation.template.waiting automation.template.matched]" {
		t.Fatalf("template statuses = %v", statuses)
	}
}

func clickTemplateSource(builtins nodes.Builtins, slot string, template blob.BlobRef) []byte {
	started, _ := builtins.Definition(nodes.RunStartedNodeID)
	click, _ := builtins.Definition(nodes.ClickTemplateNodeID)
	return []byte(fmt.Sprintf(`{
		"format":"yotta.workflow","version":"1","workflow":{"id":"wf-click-template","name":"Click Template"},"revision":0,"entryGraph":"main",
		"graphs":[{"id":"main","kind":"main","nodes":[
			{"id":"start","nodeRef":{"nodeTypeId":%q,"version":"1.0.0","semanticDigest":%q},"position":{"x":0,"y":0},"config":{},"bindings":{}},
			{"id":"click","nodeRef":{"nodeTypeId":%q,"version":"1.0.0","semanticDigest":%q},"position":{"x":1,"y":0},"config":{"slot":%q},
			 "bindings":{"template":{"kind":"blob","blob":{"mediaType":%q,"digest":%q,"size":%d}},"region":{"kind":"default"},"threshold":{"kind":"default"},"timeout":{"kind":"default"},"poll-interval":{"kind":"default"},"settle-duration":{"kind":"value","value":0},"button":{"kind":"default"},"hold-duration":{"kind":"default"}}}
		],"edges":[{"channel":"exec","from":{"nodeId":"start","portId":"started"},"to":{"nodeId":"click","portId":"in"}}],"inputs":[],"outputs":[]}],"variables":[],"resources":[],"targetProfileDefinitions":[],"credentialRequirements":[],"dependencies":[]
	}`, started.Contract.NodeRef().NodeTypeID, started.Contract.NodeRef().SemanticDigest,
		click.Contract.NodeRef().NodeTypeID, click.Contract.NodeRef().SemanticDigest, slot,
		template.MediaType, template.Digest, template.Size))
}
