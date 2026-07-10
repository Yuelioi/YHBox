package runtime

import (
	"context"
	"strings"
	"testing"

	"github.com/yottaapp/yotta/internal/automation/controller"
	"github.com/yottaapp/yotta/internal/automation/target"
	automationtrace "github.com/yottaapp/yotta/internal/automation/trace"
)

func TestDefaultControllerFactoryAndroidADB(t *testing.T) {
	tg := target.Target{
		ID:         "android:emulator-5554",
		Kind:       target.KindAndroidADB,
		Ref:        target.TargetRef{ADBSerial: "emulator-5554"},
		Resolution: target.Size{W: 1080, H: 1920},
	}
	ctrl, err := DefaultControllerFactory{}.NewController(tg, automationtrace.NewMemoryRecorder())
	if err != nil {
		t.Fatalf("NewController() error = %v", err)
	}
	if _, ok := ctrl.(*controller.AndroidADBController); !ok {
		t.Fatalf("controller type = %T, want *controller.AndroidADBController", ctrl)
	}
}

func TestDefaultControllerFactoryBrowserCDPNotWired(t *testing.T) {
	tg := target.Target{
		ID:         "browser:tab-1",
		Kind:       target.KindBrowserCDP,
		Ref:        target.TargetRef{BrowserID: "tab-1"},
		Resolution: target.Size{W: 1280, H: 720},
	}
	_, err := DefaultControllerFactory{}.NewController(tg, nil)
	if err == nil || !strings.Contains(err.Error(), "browser cdp controller client is not wired") {
		t.Fatalf("NewController() error = %v", err)
	}
}

type fakeCDPProvider struct {
	client controller.CDPClient
	target target.Target
}

func (f *fakeCDPProvider) ClientForTarget(tg target.Target) (controller.CDPClient, error) {
	f.target = tg
	return f.client, nil
}

func TestDefaultControllerFactoryBrowserCDPWired(t *testing.T) {
	tg := target.Target{
		ID:         "browser:tab-1",
		Kind:       target.KindBrowserCDP,
		Ref:        target.TargetRef{BrowserID: "tab-1"},
		Resolution: target.Size{W: 1280, H: 720},
	}
	provider := &fakeCDPProvider{client: controller.CDPClientFunc(func(_ context.Context, _ string, _ map[string]any) (map[string]any, error) {
		return map[string]any{}, nil
	})}
	ctrl, err := DefaultControllerFactory{BrowserCDP: provider}.NewController(tg, nil)
	if err != nil {
		t.Fatalf("NewController() error = %v", err)
	}
	if _, ok := ctrl.(*controller.BrowserCDPController); !ok {
		t.Fatalf("controller type = %T, want *controller.BrowserCDPController", ctrl)
	}
	if provider.target.Ref.BrowserID != "tab-1" {
		t.Fatalf("provider target = %#v", provider.target)
	}
}
