// wire_actions.go 适配器：action service 跟 services / runtime 类型对接。
//
// 为什么需要适配器：actions 包不能 import services（main.go 同时构造两者，
// 反向 import 会 cycle）。actions 包暴露纯接口（BotGate/GameProvider/Runner/
// RecorderHost/WindowOpener），适配器在 main 里把具体类型塞进接口。
package main

import (
	"context"
	"fmt"
	"net/url"
	"sync"
	"time"

	"github.com/lxn/win"
	"github.com/wailsapp/wails/v3/pkg/application"
	"github.com/wailsapp/wails/v3/pkg/events"

	"yhbox/internal/services"
	"yhbox/internal/services/actions"
	"yhbox/internal/services/actions/recording"
	actionsruntime "yhbox/internal/services/actions/runtime"
	"yhbox/internal/services/execution"
	"yhbox/internal/services/expr"
	"yhbox/pkg/input"
)

// ---- BotGate: services.App → actions.BotGate ----

type actionBotGateAdapter struct {
	app *services.App
}

func (a *actionBotGateAdapter) AcquireBot(name string) (actions.BotLease, error) {
	lease, err := a.app.AcquireBot(name)
	if err != nil {
		return nil, err
	}
	return lease, nil // *services.BotLease 自带 Release()，满足 actions.BotLease
}

// ---- GameProvider: services.App.Game() → actions.GameProvider ----

type actionGameProviderAdapter struct {
	app *services.App
}

func (a *actionGameProviderAdapter) HWND() (uintptr, bool) {
	g := a.app.Game()
	if g == nil || !g.OK {
		return 0, false
	}
	return uintptr(g.HWND), true
}

func (a *actionGameProviderAdapter) BringToForeground(hwnd uintptr) {
	input.BringToForeground(win.HWND(hwnd))
}

// ---- Runner: *runtime.Runner → actions.Runner ----
//
// runtime.Runner.SetHWND 接 win.HWND，Stop 接 StopMode；actions.Runner 接口
// 用 uintptr + 无参 Stop。这层把类型对齐。

type actionRunnerAdapter struct {
	r *actionsruntime.Runner
}

func (a *actionRunnerAdapter) Start(act *actions.Action) error {
	return a.r.Start(act)
}

func (a *actionRunnerAdapter) Stop() error {
	return a.r.Stop(actionsruntime.StopImmediate)
}

func (a *actionRunnerAdapter) SetHWND(h uintptr) {
	a.r.SetHWND(win.HWND(h))
}

// ---- RecorderHost: *recording.Recorder → actions.RecorderHost ----
//
// recording.Recorder.Start 接 win.HWND；actions.RecorderHost 接 uintptr。

type actionRecorderAdapter struct {
	r *recording.Recorder
}

func (a *actionRecorderAdapter) Start(gameHwnd uintptr, ignoreMods, ignoreVK uint32) (string, error) {
	return a.r.Start(win.HWND(gameHwnd), ignoreMods, ignoreVK)
}

func (a *actionRecorderAdapter) Stop() (actions.Action, error) { return a.r.Stop() }
func (a *actionRecorderAdapter) Cancel()                       { a.r.Cancel() }
func (a *actionRecorderAdapter) Active() bool                  { return a.r.Active() }

// ---- ActionInvoker: 把 actions.Action 当一个"原子"在 Container 里跑一遍 ----
//
// 跟 actions.Service.RunOnce 区别：
//   - 不发 action:state running/idle 事件（容器节点级日志已够）
//   - 不开 goroutine —— 同步执行，让 InvokeAction 节点等结束
//   - 每个 input step 外套 InputBus.Lock/Unlock 保证键鼠独占
//   - 共享 Container 的 ctx，cancel 立即停
//   - params 当前不消化（容器变量优先于 action 参数）

type actionInvokerAdapter struct {
	app    *services.App
	store  *actions.Store
	bus    *execution.InputBus
	driver *actionsruntime.Win32Driver
}

func (a *actionInvokerAdapter) Invoke(ctx context.Context, actionID string, params map[string]expr.Value) error {
	act, ok := a.store.Get(actionID)
	if !ok {
		return fmt.Errorf("action %q 不存在", actionID)
	}
	if len(params) > 0 {
		l := a.app.RootLogger()
		l.Warn().Str("tag", "INVOKE").Str("action", actionID).
			Int("params", len(params)).Msg("InvokeAction params 未实现，已忽略")
	}
	g := a.app.Game()
	if g == nil || !g.OK {
		return fmt.Errorf("游戏窗口未就绪，无法跑 Action")
	}
	hwnd := win.HWND(g.HWND)
	driver := a.driver
	if driver == nil {
		driver = &actionsruntime.Win32Driver{
			ActivateDelay:     30 * time.Millisecond,
			CursorSettleDelay: 20 * time.Millisecond,
		}
	}
	for i := range act.Steps {
		if err := ctx.Err(); err != nil {
			return err
		}
		s := &act.Steps[i]
		if actionsruntime.IsInputStep(s.Kind) {
			a.bus.Lock()
			err := actionsruntime.ExecuteStep(ctx, driver, hwnd, s)
			a.bus.Unlock()
			if err != nil {
				return err
			}
		} else {
			if err := actionsruntime.ExecuteStep(ctx, driver, hwnd, s); err != nil {
				return err
			}
		}
	}
	return nil
}

// ---- WindowOpener: *application.App → actions.WindowOpener ----
//
// 给每个 actionID 开一个独立编辑器窗口；同 id 重复打开 → focus 已有的，
// 不再开第二个。窗口 Close 事件清掉 map。

type actionWindowAdapter struct {
	wailsApp *application.App
	mu       sync.Mutex
	windows  map[string]*application.WebviewWindow
}

func newActionWindowAdapter(w *application.App) *actionWindowAdapter {
	return &actionWindowAdapter{
		wailsApp: w,
		windows:  map[string]*application.WebviewWindow{},
	}
}

func (a *actionWindowAdapter) OpenEditor(actionID string) error {
	if a.wailsApp == nil {
		return fmt.Errorf("wails app 未初始化")
	}
	a.mu.Lock()
	if existing, ok := a.windows[actionID]; ok {
		a.mu.Unlock()
		existing.Focus()
		return nil
	}
	a.mu.Unlock()

	// hash router 模式 + Frameless：让前端自己画 title bar 跟主窗口风格一致
	hashURL := "/#/action-editor?id=" + url.QueryEscape(actionID)
	w := a.wailsApp.Window.NewWithOptions(application.WebviewWindowOptions{
		Title:     "编辑动作",
		Width:     760,
		Height:    820,
		MinWidth:  560,
		MinHeight: 480,
		URL:       hashURL,
		Frameless: true,
	})

	a.mu.Lock()
	a.windows[actionID] = w
	a.mu.Unlock()

	// 窗口关闭时清 map，下次同 id 重新开窗
	w.OnWindowEvent(events.Common.WindowClosing, func(_ *application.WindowEvent) {
		a.mu.Lock()
		delete(a.windows, actionID)
		a.mu.Unlock()
	})
	return nil
}
