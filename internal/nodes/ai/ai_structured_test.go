package ai

import (
	"errors"
	"testing"

	"yotta/internal/node"
	"yotta/internal/services/llm"
)

func structuredCfg() map[string]any {
	return map[string]any{
		"User": "extract",
		"Outputs": []any{
			map[string]any{"Name": "Count", "Type": "Integer"},
			map[string]any{"Name": "Label", "Type": "String"},
		},
	}
}

func TestAI_StructuredOutput_SetsTypedFields(t *testing.T) {
	fp := &fakeProvider{
		resp:         llm.ChatResponse{Text: `{"Count":3,"Label":"x"}`},
		structFields: map[string]any{"Count": float64(3), "Label": "x"},
	}
	svc := node.StubServices()
	svc.AI = fakeAIService{provider: fp}

	res := setupAI(t, svc, structuredCfg(), nil)
	if res.Error != nil || res.ExitName != "Done" {
		t.Fatalf("exit=%q err=%v", res.ExitName, res.Error)
	}
	if res.OutputData["Count"] != 3 { // Integer coerce → int(3)
		t.Errorf("Count = %#v, want int 3", res.OutputData["Count"])
	}
	if res.OutputData["Label"] != "x" {
		t.Errorf("Label = %v, want x", res.OutputData["Label"])
	}
	if res.OutputData[outText] == nil {
		t.Error("Text(原文)应恒有")
	}
	// 默认 Mode=auto 透传给 ChatStructured。
	if fp.gotMode != "auto" {
		t.Errorf("mode passed = %q, want auto (default)", fp.gotMode)
	}
	if len(fp.gotSchema.Fields) != 2 {
		t.Errorf("schema fields = %d, want 2", len(fp.gotSchema.Fields))
	}
}

func TestAI_StructuredCoerceFailure_FiresFailWithText(t *testing.T) {
	fp := &fakeProvider{
		resp:         llm.ChatResponse{Text: "raw model output"},
		structFields: map[string]any{"Count": 1.5}, // 非整 → coerce 失败
	}
	svc := node.StubServices()
	svc.AI = fakeAIService{provider: fp}

	res := setupAI(t, svc, structuredCfg(), nil)
	if res.ExitName != "Fail" {
		t.Fatalf("exit=%q, want Fail", res.ExitName)
	}
	if res.OutputData[outText] != "raw model output" {
		t.Errorf("Fail.Text = %v, want raw model output", res.OutputData[outText])
	}
	if res.OutputData[outCode] != "badRequest" {
		t.Errorf("Code = %v, want badRequest", res.OutputData[outCode])
	}
}

func TestAI_StructuredProviderError_FiresFailWithRawText(t *testing.T) {
	fp := &fakeProvider{
		resp:      llm.ChatResponse{Text: "model said this then failed"},
		structErr: &llm.CodedError{Kind: llm.KindUnsupported, Err: errors.New("native unsupported")},
	}
	svc := node.StubServices()
	svc.AI = fakeAIService{provider: fp}

	res := setupAI(t, svc, structuredCfg(), nil)
	if res.ExitName != "Fail" {
		t.Fatalf("exit=%q, want Fail", res.ExitName)
	}
	if res.OutputData[outText] != "model said this then failed" {
		t.Errorf("Fail.Text = %v (raw carried on structured failure)", res.OutputData[outText])
	}
	if res.OutputData[outCode] != "unsupported" {
		t.Errorf("Code = %v, want unsupported", res.OutputData[outCode])
	}
}
