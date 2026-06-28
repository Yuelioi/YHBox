package runtime

import (
	"fmt"

	"yotta/internal/automation/controller"
	"yotta/internal/automation/target"
	automationtrace "yotta/internal/automation/trace"
)

type DefaultControllerFactory struct{}

func (DefaultControllerFactory) NewController(tg target.Target, trace automationtrace.Recorder) (controller.Controller, error) {
	switch tg.Kind {
	case target.KindAndroidADB:
		return controller.NewAndroidADBController(tg, controller.AndroidADBDeps{Trace: trace})
	case target.KindBrowserCDP:
		return nil, fmt.Errorf("browser cdp controller client is not wired")
	default:
		return nil, fmt.Errorf("unsupported active target kind %s", tg.Kind)
	}
}
