package runtime

import (
	"fmt"

	"github.com/yottaapp/yotta/internal/automation/controller"
	"github.com/yottaapp/yotta/internal/automation/target"
	automationtrace "github.com/yottaapp/yotta/internal/automation/trace"
)

type BrowserCDPClientProvider interface {
	ClientForTarget(target.Target) (controller.CDPClient, error)
}

type DefaultControllerFactory struct {
	BrowserCDP BrowserCDPClientProvider
}

func (f DefaultControllerFactory) NewController(tg target.Target, trace automationtrace.Recorder) (controller.Controller, error) {
	switch tg.Kind {
	case target.KindAndroidADB:
		return controller.NewAndroidADBController(tg, controller.AndroidADBDeps{Trace: trace})
	case target.KindBrowserCDP:
		if f.BrowserCDP == nil {
			return nil, fmt.Errorf("browser cdp controller client is not wired")
		}
		client, err := f.BrowserCDP.ClientForTarget(tg)
		if err != nil {
			return nil, err
		}
		return controller.NewBrowserCDPController(tg, controller.BrowserCDPDeps{Client: client, Trace: trace})
	default:
		return nil, fmt.Errorf("unsupported active target kind %s", tg.Kind)
	}
}
