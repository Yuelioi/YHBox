package runtime

import (
	"fmt"

	"yotta/internal/automation/controller"
	"yotta/internal/automation/target"
	automationtrace "yotta/internal/automation/trace"
)

type RuntimeControllerFactory interface {
	NewController(target.Target, automationtrace.Recorder) (controller.Controller, error)
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
		tg = windowHandleToTarget(wh)
	}
	rec := traceRecorderWithSource(rt.TraceRecorder(), source)
	if tg.Kind == target.KindWin32Window {
		return rt.win32Controller(tg, rec, need)
	}
	if rt.ControllerFactory == nil {
		return nil, fmt.Errorf("no controller factory for active target kind %s", tg.Kind)
	}
	return rt.ControllerFactory.NewController(tg, rec)
}

func (rt *RuntimeContext) win32Controller(tg target.Target, rec automationtrace.Recorder, need controllerNeed) (controller.Controller, error) {
	deps := controller.Win32Deps{Trace: rec}
	if need.Input {
		if rt.Input == nil {
			return nil, fmt.Errorf("input backend not initialised (setupRuntime not run)")
		}
		deps.Input = runtimeWin32Input{backend: rt.Input}
		deps.Backend = rt.Input.Name()
	}
	if need.Capture {
		if rt.Capture == nil {
			return nil, fmt.Errorf("capture backend not initialised")
		}
		deps.Capture = runtimeWin32Capture{backend: rt.Capture}
		deps.Backend = rt.Capture.Name()
	}
	return controller.NewWin32Controller(tg, deps)
}
