package ai

import (
	"strings"
	"testing"

	"github.com/yottaapp/yotta/internal/node"
	"github.com/yottaapp/yotta/internal/services/llm"
)

func TestAI_CollectsImageInput_AsVisionAndPlaceholder(t *testing.T) {
	fp := &fakeProvider{resp: llm.ChatResponse{Text: "a cat"}}
	svc := node.StubServices()
	svc.AI = fakeAIService{provider: fp}

	cfg := map[string]any{
		"User":   "look at {{Pic}}",
		"Inputs": []any{map[string]any{"Name": "Pic", "Type": "any"}},
	}
	dataWire := map[string]any{"Pic": node.Image{Format: "png", Data: []byte("IMG")}}

	res := setupAI(t, svc, cfg, dataWire)
	if res.Error != nil || res.ExitName != "Done" {
		t.Fatalf("exit=%q err=%v", res.ExitName, res.Error)
	}
	last := fp.gotReq.Messages[len(fp.gotReq.Messages)-1]
	if len(last.Images) != 1 || last.Images[0].Format != "png" || string(last.Images[0].Data) != "IMG" {
		t.Fatalf("user message images = %+v, want 1 png 'IMG'", last.Images)
	}
	if !strings.Contains(last.Content, "[image]") {
		t.Errorf("image input in {{}} should render [image], got %q", last.Content)
	}
}

func TestAI_MultipleImages_DeclarationOrder(t *testing.T) {
	fp := &fakeProvider{resp: llm.ChatResponse{Text: "ok"}}
	svc := node.StubServices()
	svc.AI = fakeAIService{provider: fp}

	cfg := map[string]any{
		"User": "compare",
		"Inputs": []any{
			map[string]any{"Name": "First", "Type": "any"},
			map[string]any{"Name": "Second", "Type": "any"},
		},
	}
	dataWire := map[string]any{
		"First":  node.Image{Format: "png", Data: []byte("A")},
		"Second": node.Image{Format: "jpeg", Data: []byte("B")},
	}
	res := setupAI(t, svc, cfg, dataWire)
	if res.Error != nil {
		t.Fatalf("err: %v", res.Error)
	}
	imgs := fp.gotReq.Messages[len(fp.gotReq.Messages)-1].Images
	if len(imgs) != 2 || string(imgs[0].Data) != "A" || string(imgs[1].Data) != "B" {
		t.Errorf("images not in declaration order: %+v", imgs)
	}
}
