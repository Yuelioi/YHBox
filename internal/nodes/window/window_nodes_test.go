package window

import (
	"context"
	"testing"
	"time"

	"yotta/internal/node"
	"yotta/pkg/winutil"
)

func TestGetWindow_ResolvesToDoneWindow(t *testing.T) {
	orig := resolveWindowFn
	defer func() { resolveWindowFn = orig }()
	resolveWindowFn = func(_ context.Context, _ winutil.MatchSpec, _, _ time.Duration) (winutil.WindowHandle, error) {
		return winutil.WindowHandle{HWND: 42, Title: "记事本", ClientW: 800, ClientH: 600}, nil
	}

	node.ResetRegistryForTest()
	node.Register(&GetWindow{})
	rn, _ := node.Get("GetWindow")
	svc := node.StubServices()

	r := node.RunNode(context.Background(), rn, nil,
		map[string]any{selInTitle: "记事本"}, nil, svc, false)
	if r.Error != nil {
		t.Fatal(r.Error)
	}
	if r.ExitName != gwDone {
		t.Fatalf("应走 Done, got %q", r.ExitName)
	}
	w, ok := r.OutputData["Window"].(node.Window)
	if !ok || w.HWND != 42 || w.Title != "记事本" {
		t.Fatalf("Done.Window 错: %+v (ok=%v)", w, ok)
	}
}
