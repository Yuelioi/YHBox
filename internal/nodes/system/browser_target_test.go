package system

import (
	"context"
	"testing"

	"yotta/internal/automation/target"
	"yotta/internal/node"
	"yotta/internal/services/browsercdp"
)

func TestBrowserTarget_SpecHasAsyncBrowserID(t *testing.T) {
	s := BrowserTarget{}.Spec()
	if s.Kind != "BrowserTarget" {
		t.Fatalf("kind = %q, want BrowserTarget", s.Kind)
	}
	var browserID node.InputSpec
	for _, p := range s.Inputs {
		if p.Name == btInBrowserID {
			browserID = p
		}
	}
	if browserID.Widget.Kind != "async-dropdown" {
		t.Fatalf("BrowserID widget = %q, want async-dropdown", browserID.Widget.Kind)
	}
	props := browserID.Widget.Props
	if props["asyncSource"] != browsercdp.AsyncSourceTargets {
		t.Fatalf("BrowserID asyncSource = %#v", props["asyncSource"])
	}
}

func TestBrowserTarget_RunSetsActiveTarget(t *testing.T) {
	node.ResetRegistryForTest()
	node.Register(&BrowserTarget{})
	rn, _ := node.Get("BrowserTarget")

	svc := node.StubServices()
	r := node.RunNode(context.Background(), rn, nil,
		map[string]any{
			btInEndpoint:     "http://127.0.0.1:9222",
			btInBrowserID:    "page-1",
			btInName:         "Home",
			btInWebSocketURL: "ws://127.0.0.1/devtools/page/page-1",
			btInWidth:        1280,
			btInHeight:       720,
		},
		nil, svc, false)
	if r.Error != nil {
		t.Fatal(r.Error)
	}
	if r.ExitName != btOutDone {
		t.Fatalf("exit = %q, want %q", r.ExitName, btOutDone)
	}
	tg, ok := svc.Target.Active()
	if !ok {
		t.Fatal("active target missing")
	}
	if tg.Kind != target.KindBrowserCDP || tg.Ref.BrowserID != "page-1" {
		t.Fatalf("active target = %#v", tg)
	}
	if tg.DisplayName != "Home" || tg.Resolution.W != 1280 || tg.Resolution.H != 720 {
		t.Fatalf("active target metadata = %#v", tg)
	}
	if tg.Metadata["webSocketDebuggerUrl"] != "ws://127.0.0.1/devtools/page/page-1" {
		t.Fatalf("metadata = %#v", tg.Metadata)
	}
}

func TestBrowserTarget_ValidateRejectsInvalidResolution(t *testing.T) {
	in := node.NewInputsFromConfig(map[string]any{
		btInBrowserID: "page-1",
		btInWidth:     0,
		btInHeight:    -1,
	})
	errs := BrowserTarget{}.Validate(in)
	if len(errs) != 2 {
		t.Fatalf("Validate errors = %#v, want 2", errs)
	}
}
