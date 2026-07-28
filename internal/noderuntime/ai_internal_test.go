package noderuntime

import (
	"bytes"
	"encoding/json"
	"image"
	"image/color"
	"image/jpeg"
	"image/png"
	"strings"
	"testing"

	"github.com/yottaapp/yotta/internal/ai"
	"github.com/yottaapp/yotta/internal/nodeadapter"
)

func TestAIRequestSummaryRecordsCompleteArtifactLineage(t *testing.T) {
	manifest, err := ai.SealPromptManifest(ai.PromptManifestDraft{
		ID: "yotta.test.runtime", Version: "1.0.0", Owner: "tests", Instructions: "Trusted instructions.",
	})
	if err != nil {
		t.Fatal(err)
	}
	rendered, err := ai.RenderPrompt(manifest, []ai.PromptBlock{{Kind: ai.PromptBlockUser, Content: "untrusted input"}})
	if err != nil {
		t.Fatal(err)
	}
	schema := json.RawMessage(`{"type":"object","properties":{"value":{"type":"string"}},"required":["value"],"additionalProperties":false}`)
	output, err := ai.CompileStructuredOutput("result", schema)
	if err != nil {
		t.Fatal(err)
	}
	toolSet, err := ai.SealToolSet(ai.ToolSetDraft{
		ID: "yotta.test.tools", Version: "1.0.0", Owner: "tests",
		Tools: []ai.ToolManifestDraft{{Name: "lookup", Description: "Look up a value.", Authority: ai.ToolAuthorityPure, InputSchema: schema, OutputSchema: schema}},
	})
	if err != nil {
		t.Fatal(err)
	}
	action := nodeadapter.AdapterAction{Facts: map[string]string{}}
	addAIRequestSummary(&action, ai.GenerateRequest{
		Prompt: rendered, Image: &ai.ImageInput{MediaType: "image/jpeg", Data: []byte("image")},
		Output: &output, ToolSet: toolSet.Digest(),
	})
	outputDigest, err := output.Digest()
	if err != nil {
		t.Fatal(err)
	}
	if action.Facts["prompt_manifest"] != manifest.Digest().String() ||
		action.Facts["output_schema"] != outputDigest.String() ||
		action.Facts["tool_set"] != toolSet.Digest().String() ||
		action.Facts["image_media_type"] != "image/jpeg" ||
		action.Counters["image_bytes"] != 5 {
		t.Fatalf("AI request lineage = %#v", action.Facts)
	}
	for _, leaked := range []string{"Trusted instructions.", "untrusted input", string(schema)} {
		for name, value := range action.Facts {
			if strings.Contains(value, leaked) {
				t.Fatalf("AI request fact %q leaked raw content", name)
			}
		}
	}
}

func TestAIRequestCompilesFriendlyFieldsIntoStrictSchema(t *testing.T) {
	manifest, err := ai.SealPromptManifest(ai.PromptManifestDraft{
		ID: "yotta.test.extract", Version: "1.0.0", Owner: "tests", Instructions: "Extract.",
	})
	if err != nil {
		t.Fatal(err)
	}
	request, err := aiRequest(map[string]any{
		"fields": []any{
			map[string]any{"name": "title", "type": "string", "description": "Short title"},
			map[string]any{"name": "score", "type": "number", "nullable": true},
		},
	}, "extract values", nil, true, manifest)
	if err != nil {
		t.Fatal(err)
	}
	if request.Output == nil {
		t.Fatal("structured output was not compiled")
	}
	var schema map[string]any
	if err := json.Unmarshal(request.Output.Schema, &schema); err != nil {
		t.Fatal(err)
	}
	properties := schema["properties"].(map[string]any)
	if properties["title"].(map[string]any)["type"] != "string" {
		t.Fatalf("title schema = %#v", properties["title"])
	}
	if len(properties["score"].(map[string]any)["anyOf"].([]any)) != 2 {
		t.Fatalf("nullable score schema = %#v", properties["score"])
	}
}

func TestPrepareAIImageProducesBoundedProviderInput(t *testing.T) {
	source := image.NewRGBA(image.Rect(0, 0, 1920, 1080))
	for y := 0; y < source.Bounds().Dy(); y++ {
		for x := 0; x < source.Bounds().Dx(); x++ {
			source.SetRGBA(x, y, color.RGBA{
				R: uint8((x + y) % 255), G: uint8((x*3 + y) % 255), B: uint8((x + y*5) % 255), A: 255,
			})
		}
	}
	var encoded bytes.Buffer
	if err := png.Encode(&encoded, source); err != nil {
		t.Fatal(err)
	}
	prepared, err := prepareAIImage(encoded.Bytes())
	if err != nil {
		t.Fatal(err)
	}
	if prepared.MediaType != "image/jpeg" || len(prepared.Data) == 0 || len(prepared.Data) > ai.MaxImageInputBytes {
		t.Fatalf("prepared AI image = type %q, bytes %d", prepared.MediaType, len(prepared.Data))
	}
	config, err := jpeg.DecodeConfig(bytes.NewReader(prepared.Data))
	if err != nil {
		t.Fatal(err)
	}
	if config.Width > 1568 || config.Height > 1568 {
		t.Fatalf("prepared AI image dimensions = %dx%d", config.Width, config.Height)
	}
}
