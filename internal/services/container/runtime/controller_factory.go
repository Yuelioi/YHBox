package runtime

import (
	"fmt"

	"github.com/yottaapp/yotta/internal/automation/controller"
	"github.com/yottaapp/yotta/internal/automation/target"
	automationtrace "github.com/yottaapp/yotta/internal/automation/trace"
)

type RuntimeControllerFactory interface {
	NewController(target.Target, automationtrace.Recorder) (controller.Controller, error)
}

type win32ControllerProvider interface {
	NewController(target.Target, automationtrace.Recorder, controllerNeed) (controller.Controller, error)
	Close() error
}

type controllerNeed struct {
	Input   bool
	Capture bool
}

func (rt *RuntimeContext) controllerForActiveTarget(source automationtrace.ActionSource, need controllerNeed) (controller.Controller, error) {
	tg, ok := rt.ActiveTarget()
	if !ok {
		wh := rt.WindowHandle()
		if wh.HWND == 0 {
			return nil, ErrNoActiveWindow
		}
		tg = target.NewWin32WindowTarget(wh)
	}
	rec := traceRecorderWithSource(rt.TraceRecorder(), source)
	if tg.Kind == target.KindWin32Window {
		if rt.win32Factory != nil {
			return rt.win32Factory.NewController(tg, rec)
		}
		if rt.win32Controllers == nil {
			return nil, fmt.Errorf("win32 controller provider not initialised (setupRuntime not run)")
		}
		return rt.win32Controllers.NewController(tg, rec, need)
	}
	if rt.ControllerFactory == nil {
		return nil, fmt.Errorf("no controller factory for active target kind %s", tg.Kind)
	}
	return rt.ControllerFactory.NewController(tg, rec)
}
