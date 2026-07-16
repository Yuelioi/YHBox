package nodes31runtime

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/yottaapp/yotta/internal/ai"
	"github.com/yottaapp/yotta/internal/workflow/compiler"
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
	action := compiler.AdapterAction{Facts: map[string]string{}}
	addAIRequestSummary(&action, ai.GenerateRequest{Prompt: rendered, Output: &output, ToolSet: toolSet.Digest()})
	outputDigest, err := output.Digest()
	if err != nil {
		t.Fatal(err)
	}
	if action.Facts["prompt_manifest"] != manifest.Digest().String() ||
		action.Facts["output_schema"] != outputDigest.String() ||
		action.Facts["tool_set"] != toolSet.Digest().String() {
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

func TestAgentToolTurnRejectsPartialOrMixedProviderOutput(t *testing.T) {
	call := ai.OutputItem{Kind: ai.OutputToolCall, ToolCall: &ai.ToolCall{CallID: "call-1", Name: "tool", Arguments: json.RawMessage(`{}`)}}
	text := ai.OutputItem{Kind: ai.OutputText, Text: &ai.TextOutput{Text: "partial"}}
	for _, outcome := range []ai.Outcome{
		{Finish: ai.Finish{Kind: ai.FinishToolCalls}},
		{Finish: ai.Finish{Kind: ai.FinishToolCalls}, Items: []ai.OutputItem{call, text}},
		{Finish: ai.Finish{Kind: ai.FinishCompleted}, Items: []ai.OutputItem{call}},
	} {
		if _, err := agentToolCalls(outcome); err == nil {
			t.Fatalf("accepted partial Agent outcome %#v", outcome)
		}
	}
}
