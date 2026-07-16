package nodes31runtime_test

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"image"
	"image/color"
	"image/png"
	"testing"
	"time"

	"github.com/yottaapp/yotta/internal/blob"
	"github.com/yottaapp/yotta/internal/nodes31"
	"github.com/yottaapp/yotta/internal/nodes31runtime"
	run31 "github.com/yottaapp/yotta/internal/run"
	"github.com/yottaapp/yotta/internal/workflow/compiler"
)

func TestMatchTemplateReadsExplicitImagesAndReturnsTypedBestMatch(t *testing.T) {
	builtins, err := nodes31.Build()
	if err != nil {
		t.Fatal(err)
	}
	frame := image.NewRGBA(image.Rect(0, 0, 64, 48))
	for y := 0; y < 48; y++ {
		for x := 0; x < 64; x++ {
			frame.SetRGBA(x, y, color.RGBA{R: uint8((x*3 + y) % 97), G: uint8((y*5 + 11) % 101), B: uint8((x + y*2) % 89), A: 255})
		}
	}
	for y := 0; y < 8; y++ {
		for x := 0; x < 10; x++ {
			frame.SetRGBA(17+x, 11+y, color.RGBA{R: uint8(20 + x*19), G: uint8(230 - y*21), B: uint8((x*31 + y*17) % 255), A: 255})
		}
	}
	templateImage := image.NewRGBA(image.Rect(0, 0, 10, 8))
	for y := 0; y < 8; y++ {
		for x := 0; x < 10; x++ {
			templateImage.SetRGBA(x, y, frame.RGBAAt(17+x, 11+y))
		}
	}
	framePNG, templatePNG := encodeVisionPNG(t, frame), encodeVisionPNG(t, templateImage)
	store, err := blob.Open(t.TempDir(), blob.Limits{MaxBlobBytes: 1 << 20, MaxTotalBytes: 2 << 20})
	if err != nil {
		t.Fatal(err)
	}
	frameRef, err := store.Put(context.Background(), "image/png", bytes.NewReader(framePNG))
	if err != nil {
		t.Fatal(err)
	}
	templateRef, err := store.Put(context.Background(), "image/png", bytes.NewReader(templatePNG))
	if err != nil {
		t.Fatal(err)
	}
	provider, err := blob.NewProvider(store, blob.ProviderLimits{MaxChunkBytes: 64 << 10, QueueCapacity: 4})
	if err != nil {
		t.Fatal(err)
	}
	program := compilePrimitiveProgram(t, builtins, visionMatchSource(builtins, frameRef, templateRef))
	now := time.Date(2026, 7, 16, 23, 0, 0, 0, time.UTC)
	_, owner, journal := admittedExecution(t, builtins, program, map[string]run31.InstalledProvider{
		blob.ProviderID: {ArtifactDigest: blobProviderDigest(t), ABI: blob.ProviderABI, Provider: provider},
	}, now)
	t.Cleanup(func() { _ = owner.Close(context.Background()) })
	adapters, err := nodes31runtime.Installed(builtins, testDependencies())
	if err != nil {
		t.Fatal(err)
	}
	execution, err := compiler.NewExecutor(builtins.Catalog, adapters, compiler.ExecutorOptions{Now: func() time.Time { return now }}).Run(context.Background(), program, owner, journal)
	if err != nil {
		t.Fatal(err)
	}
	var matched bool
	if err := json.Unmarshal(execution.NodeOutputs["match"]["matched"].InlineJSON(), &matched); err != nil || !matched {
		t.Fatalf("matched=%v error=%v", matched, err)
	}
	var score float64
	if err := json.Unmarshal(execution.NodeOutputs["match"]["score"].InlineJSON(), &score); err != nil || score < 0.999 {
		t.Fatalf("score=%v error=%v", score, err)
	}
	var center struct {
		X, Y float64
		Unit string
	}
	if err := json.Unmarshal(execution.NodeOutputs["match"]["center"].InlineJSON(), &center); err != nil || center.X != 22 || center.Y != 15 || center.Unit != "px" {
		t.Fatalf("center=%#v error=%v", center, err)
	}
	var bounds struct {
		X, Y, Width, Height float64
		Unit                string
	}
	if err := json.Unmarshal(execution.NodeOutputs["match"]["bounds"].InlineJSON(), &bounds); err != nil ||
		bounds.X != 17 || bounds.Y != 11 || bounds.Width != 10 || bounds.Height != 8 || bounds.Unit != "px" {
		t.Fatalf("bounds=%#v error=%v", bounds, err)
	}
	actions := 0
	for _, entry := range journal.Current().Journal() {
		if entry.Kind == run31.JournalAdapterAction {
			actions++
			if entry.Action != "vision.match-template" || entry.ActionOutcome != run31.ActionSucceeded ||
				entry.Summary.Counters["search_pixels"] != 64*48 || entry.Summary.Counters["template_pixels"] != 10*8 {
				t.Fatalf("vision action = %#v", entry)
			}
		}
	}
	if actions != 1 {
		t.Fatalf("actions=%d", actions)
	}
}

func encodeVisionPNG(t *testing.T, source image.Image) []byte {
	t.Helper()
	var encoded bytes.Buffer
	if err := png.Encode(&encoded, source); err != nil {
		t.Fatal(err)
	}
	return encoded.Bytes()
}

func visionMatchSource(builtins nodes31.Builtins, frame, template blob.BlobRef) []byte {
	started, _ := builtins.Definition(nodes31.RunStartedNodeID)
	match, _ := builtins.Definition(nodes31.MatchTemplateNodeID)
	branch, _ := builtins.Definition(nodes31.BranchNodeID)
	return []byte(fmt.Sprintf(`{
		"format":"yotta.workflow","version":"3.1","workflow":{"id":"wf-vision-match","name":"Vision Match"},"revision":0,"entryGraph":"main",
		"graphs":[{"id":"main","kind":"main","nodes":[
			{"id":"start","nodeRef":{"nodeTypeId":%q,"semanticDigest":%q},"position":{"x":0,"y":0},"config":{},"bindings":{}},
			{"id":"match","nodeRef":{"nodeTypeId":%q,"semanticDigest":%q},"position":{"x":0,"y":1},"config":{},"bindings":{
				"image":{"kind":"blob","blob":{"mediaType":%q,"digest":%q,"size":%d}},
				"template":{"kind":"blob","blob":{"mediaType":%q,"digest":%q,"size":%d}},
				"region":{"kind":"default"},"threshold":{"kind":"default"}}},
			{"id":"branch","nodeRef":{"nodeTypeId":%q,"semanticDigest":%q},"position":{"x":1,"y":0},"config":{},"bindings":{}}
		],"edges":[
			{"channel":"exec","from":{"nodeId":"start","portId":"started"},"to":{"nodeId":"branch","portId":"in"}},
			{"channel":"data","from":{"nodeId":"match","portId":"matched"},"to":{"nodeId":"branch","portId":"condition"}}
		],"inputs":[],"outputs":[]}],"variables":[],"secretRefs":[]
	}`, started.Contract.NodeRef().NodeTypeID, started.Contract.NodeRef().SemanticDigest,
		match.Contract.NodeRef().NodeTypeID, match.Contract.NodeRef().SemanticDigest,
		frame.MediaType, frame.Digest, frame.Size, template.MediaType, template.Digest, template.Size,
		branch.Contract.NodeRef().NodeTypeID, branch.Contract.NodeRef().SemanticDigest))
}
