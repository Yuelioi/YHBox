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
	frame         []byte
	clicks        []automationinstalled.ClickRequest
	closed        int
	captures      int
	captureFormat string
}

func (provider *templateAutomationProvider) Open(_ context.Context, request resource.ProviderOpenRequest) (any, error) {
	switch request.Kind {
	case automationinstalled.KindCapture:
		if fmt.Sprint(request.Operations) != "[capture read-capture]" || len(request.CapabilityScope) != 0 {
			return nil, fmt.Errorf("unexpected template capture open: %#v", request)
		}
		return automationinstalled.KindCapture, nil
	case automationinstalled.KindInput:
		if fmt.Sprint(request.Operations) != "[click]" || len(request.CapabilityScope) != 0 {
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
			var request struct {
				Format string `json:"format,omitempty"`
			}
			if err := json.Unmarshal(payload, &request); err != nil {
				return nil, err
			}
			provider.captureFormat = request.Format
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
	const targetID, slot = "automation-target/template", "template-target"
	program := compilePrimitiveProgram(t, builtins, clickTemplateSource(builtins, slot, templateRef))
	now := time.Date(2026, 7, 17, 15, 0, 0, 0, time.UTC)
	_, owner, journal := admittedExecution(t, builtins, program, map[string]run.InstalledProvider{
		blob.ProviderID: {ArtifactDigest: blobProviderDigest(t), ABI: blob.ProviderABI, Provider: blobProvider},
	}, now)
	t.Cleanup(func() { _ = owner.Close(context.Background()) })
	targets := configuredTargetRun(t, slot, targetID, provider)
	adapters, err := noderuntime.Installed(builtins, testDependencies())
	if err != nil {
		t.Fatal(err)
	}
	startedAt := time.Now()
	result, err := compiler.NewExecutor(builtins.Catalog, adapters, compiler.ExecutorOptions{Now: func() time.Time { return now }}).
		RunWithTargets(context.Background(), program, owner, targets, journal)
	if err != nil {
		t.Fatal(err)
	}
	if elapsed := time.Since(startedAt); elapsed >= 2*time.Second {
		t.Fatalf("first-frame template match waited %s before clicking", elapsed)
	}
	if len(provider.clicks) != 1 || provider.clicks[0].Point.Unit != "ratio" || provider.clicks[0].Point.X != 0.375 || provider.clicks[0].Point.Y != 23.0/60.0 {
		t.Fatalf("template clicks = %#v", provider.clicks)
	}
	matched := result.NodeOutputs["click"]["matched"].InlineJSON()
	if string(matched) != "true" || provider.closed != 2 {
		t.Fatalf("matched=%s closed=%d", matched, provider.closed)
	}
	if provider.captureFormat != "rgba" {
		t.Fatalf("template capture format = %q, want rgba fast path", provider.captureFormat)
	}
	var statuses []string
	var matchedCounters map[string]int64
	for _, entry := range journal.Current().Journal() {
		if entry.Kind == run.JournalNodeStatus {
			statuses = append(statuses, entry.StatusCode)
			if entry.StatusCode == nodes.AutomationTemplateMatchedStatus {
				matchedCounters = entry.Summary.Counters
			}
		}
	}
	if fmt.Sprint(statuses) != "[automation.template.waiting automation.template.matched]" {
		t.Fatalf("template statuses = %v", statuses)
	}
	for _, counter := range []string{"captures", "capture_bytes", "capture_ms", "match_ms"} {
		if _, ok := matchedCounters[counter]; !ok {
			t.Fatalf("matched status omitted %q timing counter: %v", counter, matchedCounters)
		}
	}
}

func TestClickTemplateReusesOneCapturedImageAcrossNodes(t *testing.T) {
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
	const targetID, slot = "automation-target/template-reuse", "template-reuse-target"
	program := compilePrimitiveProgram(t, builtins, clickTemplateReuseSource(builtins, slot, templateRef))
	now := time.Date(2026, 7, 30, 1, 0, 0, 0, time.UTC)
	_, owner, journal := admittedExecution(t, builtins, program, map[string]run.InstalledProvider{
		blob.ProviderID: {ArtifactDigest: blobProviderDigest(t), ABI: blob.ProviderABI, Provider: blobProvider},
	}, now)
	t.Cleanup(func() { _ = owner.Close(context.Background()) })
	targets := configuredTargetRun(t, slot, targetID, provider)
	adapters, err := noderuntime.Installed(builtins, testDependencies())
	if err != nil {
		t.Fatal(err)
	}
	if _, err := compiler.NewExecutor(builtins.Catalog, adapters, compiler.ExecutorOptions{Now: func() time.Time { return now }}).
		RunWithTargets(context.Background(), program, owner, targets, journal); err != nil {
		t.Fatal(err)
	}
	if provider.captures != 1 {
		t.Fatalf("capture calls = %d, want exactly the explicit Capture Window call", provider.captures)
	}
	if len(provider.clicks) != 2 {
		t.Fatalf("template clicks = %#v", provider.clicks)
	}
	for _, click := range provider.clicks {
		if click.Point.Unit != "ratio" || click.Point.X != 0.375 || click.Point.Y != 23.0/60.0 {
			t.Fatalf("template click = %#v", click)
		}
	}
	for _, entry := range journal.Current().Journal() {
		if entry.Action == "automation.click-template" && entry.Summary.Counters["captures"] != 0 {
			t.Fatalf("click template captured despite a bound source image: %#v", entry.Summary.Counters)
		}
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
			 "bindings":{"template":{"kind":"blob","blob":{"mediaType":%q,"digest":%q,"size":%d}},"region":{"kind":"default"},"threshold":{"kind":"default"},"timeout":{"kind":"value","value":5000},"poll-interval":{"kind":"value","value":100},"settle-duration":{"kind":"value","value":0},"button":{"kind":"default"},"hold-duration":{"kind":"default"}}}
		],"edges":[{"channel":"exec","from":{"nodeId":"start","portId":"started"},"to":{"nodeId":"click","portId":"in"}}],"inputs":[],"outputs":[]}],"variables":[],"resources":[],"targetProfileDefinitions":[],"credentialRequirements":[],"dependencies":[]
	}`, started.Contract.NodeRef().NodeTypeID, started.Contract.NodeRef().SemanticDigest,
		click.Contract.NodeRef().NodeTypeID, click.Contract.NodeRef().SemanticDigest, slot,
		template.MediaType, template.Digest, template.Size))
}

func clickTemplateReuseSource(builtins nodes.Builtins, slot string, template blob.BlobRef) []byte {
	started, _ := builtins.Definition(nodes.RunStartedNodeID)
	capture, _ := builtins.Definition(nodes.CaptureWindowNodeID)
	click, _ := builtins.Definition(nodes.ClickTemplateNodeID)
	clickBindings := fmt.Sprintf(`"template":{"kind":"blob","blob":{"mediaType":%q,"digest":%q,"size":%d}},"region":{"kind":"default"},"threshold":{"kind":"default"},"timeout":{"kind":"value","value":5000},"poll-interval":{"kind":"value","value":100},"settle-duration":{"kind":"value","value":0},"button":{"kind":"default"},"hold-duration":{"kind":"default"}`,
		template.MediaType, template.Digest, template.Size)
	return []byte(fmt.Sprintf(`{
		"format":"yotta.workflow","version":"1","workflow":{"id":"wf-click-template-reuse","name":"Click Template Reuse"},"revision":0,"entryGraph":"main",
		"graphs":[{"id":"main","kind":"main","nodes":[
			{"id":"start","nodeRef":{"nodeTypeId":%q,"version":"1.0.0","semanticDigest":%q},"position":{"x":0,"y":0},"config":{},"bindings":{}},
			{"id":"capture","nodeRef":{"nodeTypeId":%q,"version":"1.0.0","semanticDigest":%q},"position":{"x":1,"y":0},"config":{"slot":%q},"bindings":{}},
			{"id":"click-1","nodeRef":{"nodeTypeId":%q,"version":"1.0.0","semanticDigest":%q},"position":{"x":2,"y":0},"config":{"slot":%q},"bindings":{%s}},
			{"id":"click-2","nodeRef":{"nodeTypeId":%q,"version":"1.0.0","semanticDigest":%q},"position":{"x":3,"y":0},"config":{"slot":%q},"bindings":{%s}}
		],"edges":[
			{"channel":"exec","from":{"nodeId":"start","portId":"started"},"to":{"nodeId":"capture","portId":"in"}},
			{"channel":"exec","from":{"nodeId":"capture","portId":"completed"},"to":{"nodeId":"click-1","portId":"in"}},
			{"channel":"exec","from":{"nodeId":"click-1","portId":"completed"},"to":{"nodeId":"click-2","portId":"in"}},
			{"channel":"data","from":{"nodeId":"capture","portId":"image"},"to":{"nodeId":"click-1","portId":"image"}},
			{"channel":"data","from":{"nodeId":"capture","portId":"image"},"to":{"nodeId":"click-2","portId":"image"}}
		],"inputs":[],"outputs":[]}],"variables":[],"resources":[],"targetProfileDefinitions":[],"credentialRequirements":[],"dependencies":[]
	}`, started.Contract.NodeRef().NodeTypeID, started.Contract.NodeRef().SemanticDigest,
		capture.Contract.NodeRef().NodeTypeID, capture.Contract.NodeRef().SemanticDigest, slot,
		click.Contract.NodeRef().NodeTypeID, click.Contract.NodeRef().SemanticDigest, slot, clickBindings,
		click.Contract.NodeRef().NodeTypeID, click.Contract.NodeRef().SemanticDigest, slot, clickBindings))
}
