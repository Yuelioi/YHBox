// cmd/yhbox 异环工具箱主入口。双击 exe 启动 wails3 应用。
package main

import (
	"context"
	"embed"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/rs/zerolog"
	"github.com/wailsapp/wails/v3/pkg/application"
	"github.com/wailsapp/wails/v3/pkg/events"

	"yhbox/internal/services"
	"yhbox/internal/services/actions"
	"yhbox/internal/services/actions/recording"
	actionsruntime "yhbox/internal/services/actions/runtime"
	"yhbox/internal/services/calibration"
	"yhbox/internal/services/tools"
	"yhbox/internal/services/container"
	containerruntime "yhbox/internal/services/container/runtime"
	"yhbox/internal/services/execution"
	"yhbox/internal/services/schedule"
	"yhbox/internal/bots"
	"yhbox/internal/hotkey"
	"yhbox/internal/services/template"
	"yhbox/pkg/actiontransform"
	"yhbox/pkg/botcore"
	"yhbox/pkg/capture"
	"yhbox/pkg/locale"
	"yhbox/pkg/platform"
	"yhbox/pkg/screenshot"
	"yhbox/pkg/winutil"
	"yhbox/tools/battle"
	"yhbox/tools/cook"
	"yhbox/tools/fish"
	"yhbox/tools/rhythm"
)

//go:embed all:frontend/dist
var assets embed.FS

// 用跟 exe icon 同一份 build/windows/icon.ico —— 多 size 容器，Windows 自动挑 16/32px 进托盘。
// 老 cmd/yhbox/winres/*.png 是旧资源生成器留下的，已废弃。
//
//go:embed build/windows/icon.ico
var trayIcon []byte

var version = "1.1.0"

func main() {
	platform.EnsureAdmin()

	// 日志栈：zerolog → LogSink → wails3 Event.Emit
	logSink := services.NewLogSink(nil) // emit 在 wailsApp 构造后装配
	rootLog := zerolog.New(logSink).With().Timestamp().Logger()

	// App 协调器（不暴露 JS）
	app := services.NewApp("", logSink, rootLog) // settingsPath="" 走默认（exe 同目录）

	// 截屏后端选择：auto 根据 OS build 自动选；其它按 settings 指定。
	// WGC / Mock 初始化失败时回退 GDI + warn，确保用户至少能跑。
	method := app.Settings().Capture.Method
	if method == "auto" {
		auto := capture.AutoBackend()
		method = auto.String()
		rootLog.Info().Str("tag", "SYSTEM").Uint32("build", capture.WindowsBuild()).Msgf("截屏 auto: 选中 %s", method)
	}
	switch method {
	case "wgc":
		if err := capture.SetBackend(capture.BackendWGC); err != nil {
			rootLog.Warn().Err(err).Str("tag", "SYSTEM").Msg("WGC 截屏初始化失败，回退到 GDI")
			_ = capture.SetBackend(capture.BackendGDI)
		} else {
			rootLog.Info().Str("tag", "SYSTEM").Msg("截屏后端: WGC (Windows Graphics Capture)")
		}
	case "mock":
		if err := capture.SetBackend(capture.BackendMock); err != nil {
			rootLog.Warn().Err(err).Str("tag", "SYSTEM").Msg("Mock 截屏初始化失败，回退到 GDI（mock-frames/ 目录里放 PNG 再切回 mock）")
			_ = capture.SetBackend(capture.BackendGDI)
		} else {
			rootLog.Info().Str("tag", "SYSTEM").Msg("截屏后端: Mock (离线回放，不读游戏窗口)")
		}
	default:
		_ = capture.SetBackend(capture.BackendGDI)
		rootLog.Info().Str("tag", "SYSTEM").Msg("截屏后端: GDI (PrintWindow)")
	}
	// WGC 的 SetIsBorderRequired 等非致命 API 失败会写到 LastInitWarning，
	// 这里读出来给用户看（黄框关不掉、cursor 抓帧关不掉等场景）。
	// 注意：这个变量在第一次 openWGCSession 时才填，所以延迟到首次 detect 之后才有值。
	// 想立即检查得在 SetBackend 时跑一次试探 session —— 暂时不做，等用户反馈再扩。

	// Screenshot writer: 给 bot 异步落盘带标注 PNG 用，调试调参时打开 Capture.DumpDebug
	// settings 在 debug/captures/<bot>/<date>/ 累积。默认关，写盘走独立 goroutine。
	// debug/ 是 gitignored 的开发者本地目录。
	screenshot.InitDefault(platform.DevOutputPath("captures"), 16, app.Settings().Capture.DumpDebug)
	defer screenshot.CloseDefault()

	// 按 settings.locale 加载每个 bot 的配置 + 视觉模板。
	// 失败分两种：
	//   - ErrLocaleNotImplemented：合法的"未实装"状态（manifest implemented:false），
	//     info 级别日志，前端把这个 bot 标灰；不当成错误。
	//   - 其它 error：真配置错（yaml 解析挂了、ROI 越界等），error 级别日志。
	loc := locale.Locale(app.Settings().Locale)
	if !locale.Valid(string(loc)) {
		rootLog.Warn().Str("tag", "SYSTEM").Msgf("settings.locale=%q 无效，回退 zh", loc)
		loc = locale.Zh
	}
	if err := rhythm.LoadConfig(loc); err != nil {
		if errors.Is(err, botcore.ErrLocaleNotImplemented) {
			rootLog.Info().Err(err).Str("tag", "SYSTEM").Msgf("rhythm: locale %s 未实装，bot 不可用", loc)
		} else {
			rootLog.Error().Err(err).Str("tag", "SYSTEM").Msgf("rhythm: locale %s 配置加载失败", loc)
		}
	} else {
		rootLog.Info().Str("tag", "SYSTEM").Msgf("rhythm: locale %s 配置加载成功", loc)
	}
	if err := fish.LoadConfig(loc); err != nil {
		if errors.Is(err, botcore.ErrLocaleNotImplemented) {
			rootLog.Info().Err(err).Str("tag", "SYSTEM").Msgf("fish: locale %s 未实装，bot 不可用", loc)
		} else {
			rootLog.Error().Err(err).Str("tag", "SYSTEM").Msgf("fish: locale %s 配置加载失败", loc)
		}
	} else {
		rootLog.Info().Str("tag", "SYSTEM").Msgf("fish: locale %s 配置加载成功", loc)
	}
	if err := cook.LoadConfig(loc); err != nil {
		if errors.Is(err, botcore.ErrLocaleNotImplemented) {
			rootLog.Info().Err(err).Str("tag", "SYSTEM").Msgf("cook: locale %s 未实装，bot 不可用", loc)
		} else {
			rootLog.Error().Err(err).Str("tag", "SYSTEM").Msgf("cook: locale %s 配置加载失败", loc)
		}
	} else {
		rootLog.Info().Str("tag", "SYSTEM").Msgf("cook: locale %s 配置加载成功", loc)
	}
	if err := battle.LoadConfig(loc); err != nil {
		if errors.Is(err, botcore.ErrLocaleNotImplemented) {
			rootLog.Info().Err(err).Str("tag", "SYSTEM").Msgf("battle: locale %s 未实装，bot 不可用", loc)
		} else {
			rootLog.Error().Err(err).Str("tag", "SYSTEM").Msgf("battle: locale %s 配置加载失败", loc)
		}
	} else {
		rootLog.Info().Str("tag", "SYSTEM").Msgf("battle: locale %s 配置加载成功", loc)
	}

	// 长跑型 bot service：从 registry 拿元数据，统一构造
	botMetas := services.RegisteredBots()
	botSvcs := make([]services.BotService, 0, len(botMetas))
	// cap = bot 数 + 11 个固定 services（battle/settings/game/action/hotkey/template/
	// container/schedule/calibration/tools/+1 spare）
	wailsServices := make([]application.Service, 0, len(botMetas)+11)
	for _, b := range botMetas {
		svc, ws := b.Construct(app)
		botSvcs = append(botSvcs, svc)
		wailsServices = append(wailsServices, ws)
	}
	app.RegisterBotServices(botSvcs)

	// 共享 HotkeyManager。Win32 RegisterHotKey 是 process-wide unique（hWnd=NULL 时
	// 跟线程绑定），全 app 必须共享同一个实例 —— battle / action / recorder 都注册到这里。
	// 两个 manager 就两个 hotkey 线程互相覆盖反注册，热键全丢。
	sharedHotkeys := hotkey.NewHotkeyManager()

	// 非 bot service（hotkey-driven 的 battle + 共享的 settings/game）
	battleSvc := bots.NewBattleService(app, sharedHotkeys)
	settingsSvc := services.NewSettingsService(app)
	gameSvc := services.NewGameService(app)
	app.RegisterBattleService(battleSvc)

	// 数据根：<exeDir>/data/ —— Action / Container / Schedule / Template 全在这下面。
	dataDir := "data"
	if exe, err := os.Executable(); err == nil {
		dataDir = filepath.Join(filepath.Dir(exe), "data")
	}

	// 启动期 v1 数据检测：含已废字段（hotkey/loopMode/...）的 action.json → wipe 整 dir。
	// 早期阶段无兼容性，用户已被告知重置。
	actionsRoot := filepath.Join(dataDir, "actions")
	if wiped, err := actions.WipeV1Data(actionsRoot); err != nil {
		rootLog.Warn().Err(err).Str("tag", "STARTUP").Msg("WipeV1Data 失败")
	} else if wiped {
		rootLog.Warn().Str("tag", "STARTUP").Msg("检测到 v1 actions 数据，已 wipe；请在 Container 编辑器内重新录制")
	}
	actionStore := actions.NewStore(actionsRoot)

	// actionSvc 用占位 nil 提前构造引用 —— 真正构造放到 runner emit 回调能拿到 actionSvc 后。
	// Runner emit 回调里要调 actionSvc.OnRunnerEvent 释放 lease；用闭包 + 后赋值的模式。
	var actionSvc *actions.Service
	runner := actionsruntime.NewRunner(
		&actionsruntime.Win32Driver{
			ActivateDelay:     30 * time.Millisecond,
			CursorSettleDelay: 20 * time.Millisecond,
		},
		func(ev actionsruntime.EventActionState) {
			// 1) 推到前端
			app.Emit("action:state", ev)
			// 2) 通知 service 释放 lease
			if actionSvc != nil {
				actionSvc.OnRunnerEvent(ev.ActionID, ev.Status)
			}
		},
	)
	actionSvc = actions.NewService(
		actionStore,
		&actionRunnerAdapter{r: runner},
		&actionBotGateAdapter{app: app},
		&actionGameProviderAdapter{app: app},
	)
	runner.SetLocalMouseCounts360Getter(func() int {
		return app.SnapshotSettings().UI.MouseCounts360
	})

	// Recorder：构造 + emit 回调推前端 (action:recorder-event)。
	// 录制 toggle 热键固定 Ctrl+Shift+R（v1 不可改；以后从 settings 读再说）。
	recorder := recording.NewRecorder(func(ev actiontransform.Event) {
		app.Emit("action:recorder-event", ev)
	})
	recorder.SetMouseCounts360Getter(func() int {
		return app.SnapshotSettings().UI.MouseCounts360
	})
	const (
		toggleMods = recording.MOD_CONTROL | recording.MOD_SHIFT
		toggleVK   = uint32('R')
	)
	actionSvc.SetRecorder(&actionRecorderAdapter{r: recorder}, toggleMods, toggleVK)
	actionSvc.SetEmit(func(name string, data any) { app.Emit(name, data) })

	// ---- HotkeyRegistry：所有热键的中央 manifest ----
	// 系统热键 (recorder-toggle / stop-action) + per-action 热键全部走这条路。
	// 用户可在 Settings → 快捷键 tab 改任意一条，hot reload 立即生效。
	hotkeyRegistry := hotkey.NewHotkeyRegistry(sharedHotkeys)
	hotkeyRegistry.SetCallbacks(
		// onActionHotkeyChange：Action 不再绑热键（容器架构下 hotkey 在 container 层）；
		// Container 接管后改为写回 container.Hotkey。registry 允许 nil。
		nil,
		// onSystemHotkeyChange：写回 settings.UI.ActionStopHotkey
		// system.recorder-toggle 当前不持久化（hardcode "Ctrl+Shift+R"，重启复位）
		func(key, newStr string) error {
			if key == "system.stop-action" {
				cur := app.SnapshotSettings()
				cur.UI.ActionStopHotkey = newStr
				app.SwapSettings(cur)
				return app.SaveSettings()
			}
			return nil
		},
		// emit 回调：广播给所有 webview
		func() { app.Emit("hotkey:changed", map[string]any{}) },
	)

	// 注册系统热键：录制 toggle
	if err := hotkeyRegistry.Register("system.recorder-toggle", hotkey.HotkeySourceSystem,
		"录制 toggle", "Ctrl+Shift+R", "",
		func() { app.Emit("action:recorder-toggle", map[string]any{}) }); err != nil {
		rootLog.Warn().Err(err).Str("tag", "SYSTEM").Msg("注册录制 toggle 热键失败")
	}

	// 全局强停热键（默认 Ctrl+Shift+F9）下面跟 system.execution-stop 一起注册。
	// 单一 hotkey 同时停 action 直接 run + execQueue + worker——避免双注册（Win32
	// RegisterHotKey 重复 mod+vk 会拒绝第二次，老版本两条都用 F9 导致第二条静默失败）。

	// 暴露 HotkeyService RPC 给前端
	hotkeySvc := hotkey.NewHotkeyService(hotkeyRegistry)

	// 模板库 / Container / Schedule 数据层
	templateStore, err := template.NewStore(filepath.Join(dataDir, "assets", "templates"))
	if err != nil {
		rootLog.Fatal().Err(err).Str("tag", "STARTUP").Msg("template store init")
	}
	templateSvc := template.NewService(templateStore, &templateCaptureAdapter{app: app})

	containerStore, err := container.NewStore(filepath.Join(dataDir, "containers"))
	if err != nil {
		rootLog.Fatal().Err(err).Str("tag", "STARTUP").Msg("container store init")
	}
	containerSvc := container.NewService(containerStore)

	scheduleStore, err := schedule.NewStore(filepath.Join(dataDir, "schedules"))
	if err != nil {
		rootLog.Fatal().Err(err).Str("tag", "STARTUP").Msg("schedule store init")
	}
	scheduleSvc := schedule.NewService(scheduleStore)

	// ---- 运行时层：ExecutionQueue + Worker + ScheduleDaemon ----
	//
	// 键鼠独占：单 worker，串行消费 queue；hotkey/schedule/UI 触发都进同一 queue。
	execQueue := execution.NewExecutionQueue()
	inputBus := execution.NewInputBus()

	// 真模板匹配 + 真 Action 调用 + 真输入驱动 + 真颜色检测
	templateMatcher := &templateMatcherAdapter{app: app, tplStore: templateStore}
	actionInvoker := &actionInvokerAdapter{app: app, store: actionStore, bus: inputBus}
	containerInputDrv := newContainerInputDriver(app)
	containerColor := &containerColorAdapter{app: app}

	// Worker.RunFunc：load container → 构造 ContainerRunner → Run
	emitForRuntime := func(name string, data any) { app.Emit(name, data) }
	runFunc := func(ctx context.Context, target execution.TargetRef) error {
		if target.Kind != "container" {
			return fmt.Errorf("unsupported target kind %q", target.Kind)
		}
		c, ok := containerStore.Get(target.ID)
		if !ok {
			return fmt.Errorf("container %q not found", target.ID)
		}
		// 前台模式：跑之前把游戏窗口拉到前台（必须前台运行的脚本/视觉确认场景）
		if c.RunMode == "foreground" {
			if targets := winutil.FindGame(); len(targets) > 0 {
				winutil.BringToFront(targets[0].HWND)
				time.Sleep(150 * time.Millisecond) // 让窗口完成 restore + 焦点切换
			}
		}
		rt := containerruntime.NewRuntimeContext(&c, inputBus, templateMatcher, actionInvoker, containerInputDrv, containerColor, emitForRuntime)
		r := containerruntime.NewContainerRunner(rt)
		return r.Run(ctx)
	}
	worker := execution.NewWorker(execQueue, runFunc, func(s execution.WorkerState) {
		app.Emit("execution:state", s)
	})
	worker.Start()

	// 给 containerSvc 注入运行入口 — 前端 ▶ 按钮走 Run，■ 走 StopAll
	containerSvc.SetRunner(&containerRunnerAdapter{queue: execQueue, worker: worker})

	// Container hotkey 绑定：扫所有容器，把 container.hotkey 注册到 registry。
	// CRUD 后重扫一次（containerSvc.SetOnChange 钩子）。触发后入队单 target manual run。
	containerHotkeys := newContainerHotkeyBinder(containerStore, hotkeyRegistry, execQueue)
	containerHotkeys.Refresh()
	containerSvc.SetOnChange(containerHotkeys.Refresh)

	// ScheduleDaemon：注册 enabled schedules 到 cron/hotkey/once，触发后入 queue
	scheduleHotkeyAdapter := &scheduleHotkeyRegistrar{reg: hotkeyRegistry}
	scheduleDaemon := schedule.NewDaemon(scheduleStore, execQueue, scheduleHotkeyAdapter)
	scheduleDaemon.Start()

	// Schedule CRUD 后重注册 cron / hotkey trigger
	scheduleSvc.SetOnChange(scheduleDaemon.Reload)

	// 全局强停热键：同时停 action runner（直接调 RunOnce 的回放）、清 execution
	// queue、cancel 当前 container worker run。设置面板里 UI.ActionStopHotkey 改
	// 这一条；空 → 默认 F9。
	stopAllHk := strings.TrimSpace(app.Settings().UI.ActionStopHotkey)
	if stopAllHk == "" {
		stopAllHk = "Ctrl+Shift+F9"
	}
	if err := hotkeyRegistry.Register("system.execution-stop", hotkey.HotkeySourceSystem,
		"强停所有运行", stopAllHk, "",
		func() {
			// action 直跑（不走 queue 的）—— actionSvc.StopRunning 是 noop-safe
			go func() { _ = actionSvc.StopRunning() }()
			execQueue.CancelAll()
			worker.CancelCurrent()
		}); err != nil {
		rootLog.Warn().Err(err).Str("tag", "SYSTEM").Str("hotkey", stopAllHk).Msg("注册全局强停热键失败")
	}

	calibrationSvc := calibration.NewService()

	// tools 杂项工具服务：MousePos / 鼠标 HUD / ScreenPicker 等。
	// wailsApp 还没建，先建 Service 注册到 service 列表，下面 wailsApp 后再 SetApp 注入。
	toolsSvc := tools.NewService(&toolsGameAdapter{app: app})

	// DPI 校准 toggle 热键：默认 F8。前端订阅 'calibration:toggle' event 推进状态机。
	// 注册无副作用——只有 SettingsInput 打开校准对话框时才有效。
	if err := hotkeyRegistry.Register("system.calibrate-toggle", hotkey.HotkeySourceSystem,
		"DPI 校准 启动/停止", "F8", "",
		func() {
			app.Emit("calibration:toggle", nil)
		}); err != nil {
		rootLog.Warn().Err(err).Str("tag", "SYSTEM").Msg("注册 DPI 校准热键失败")
	}

	wailsServices = append(wailsServices,
		application.NewService(battleSvc),
		application.NewService(settingsSvc),
		application.NewService(gameSvc),
		application.NewService(actionSvc),
		application.NewService(hotkeySvc),
		application.NewService(templateSvc),
		application.NewService(containerSvc),
		application.NewService(scheduleSvc),
		application.NewService(calibrationSvc),
		application.NewService(toolsSvc),
	)
	_ = scheduleDaemon // 防 import 未用 + 留扩展位

	// wails3 application
	wailsApp := application.New(application.Options{
		Name:        "YHBox",
		Description: "异环工具箱",
		Services:    wailsServices,
		Assets: application.AssetOptions{
			Handler: application.AssetFileServerFS(assets),
		},
	})
	app.AttachWailsApp(wailsApp)

	// 注入编辑器独立窗口的开窗器（wails3 多窗口 API）
	actionSvc.SetWindowOpener(newActionWindowAdapter(wailsApp))
	// tools 服务也是 wailsApp 创建后才能用（要开新窗口）
	toolsSvc.SetApp(wailsApp)

	// 装配 LogSink emit（这样 logSink Write 触发 flush 时会推 log:lines 给前端）
	logSink.SetEmit(func(e services.LogLinesEvent) {
		wailsApp.Event.Emit(services.EventLogLines, e)
	})

	// 注册事件 payload 类型（让 bindings 生成器产 TS 类型）
	// 长跑 bot 的 state 事件由 registry 派生
	for _, b := range botMetas {
		application.RegisterEvent[services.BotStateEvent](services.EventStateName(b.Kind))
	}
	// Bot-specific + 共享事件
	application.RegisterEvent[services.FishStatsEvent](services.EventFishStats)
	application.RegisterEvent[services.PianoProgressEvent](services.EventPianoProgress)
	application.RegisterEvent[services.BattleStateEvent](services.EventBattleState)
	application.RegisterEvent[services.GameStatusEvent](services.EventGameStatus)
	application.RegisterEvent[services.LogLinesEvent](services.EventLogLines)
	application.RegisterEvent[actionsruntime.EventActionState]("action:state")

	// 主窗口尺寸读 settings（用户上次拖到的尺寸），frameless 让前端自己画 title bar
	winCfg := app.Settings().UI.Window
	mainWin := wailsApp.Window.NewWithOptions(application.WebviewWindowOptions{
		Title:            "YHBox " + version,
		Width:            winCfg.Width,
		Height:           winCfg.Height,
		MinWidth:         900,
		MinHeight:        600,
		BackgroundColour: application.NewRGB(9, 9, 11), // zinc-950
		Frameless:        true,
		URL:              "/",
	})

	// 用户拖完才落盘（WindowDidResize 拖动期间会狂刷，没必要每帧写 IO）。
	// settings.UI.Window 不走 SettingsService.Update 的 patch 流程 —— 这只是 UI 状态，
	// 不需要前端订阅或校验，直接绕过 swap 直接写。
	mainWin.OnWindowEvent(events.Windows.WindowEndResize, func(_ *application.WindowEvent) {
		w, h := mainWin.Size()
		if w < 100 || h < 100 {
			return
		}
		app.UpdateWindowSize(w, h)
	})

	// 系统托盘：始终常驻。左键单击 toggle 窗口显隐；右键弹菜单可强制退出。
	// 即使 MinimizeToTray=false 也保留托盘 —— 用户万一拿不到主窗口时还有个出口。
	//
	// 故意不用 tray.AttachWindow()：它内部走 PositionWindow，会把窗口拽到托盘附近
	// （桌面右下角），还会让部分内容溢出屏幕。我们要的是"回到上次位置"，
	// 所以手动 OnClick → Show().Focus()（操作系统会把它还原到 Hide() 前的位置）。
	tray := wailsApp.SystemTray.New()
	tray.SetIcon(trayIcon).SetTooltip("YHBox " + version)
	tray.OnClick(func() {
		if mainWin.IsVisible() && !mainWin.IsMinimised() {
			mainWin.Hide()
		} else {
			mainWin.Show().Focus()
		}
	})
	trayMenu := application.NewMenu()
	trayMenu.Add("显示窗口").OnClick(func(*application.Context) { mainWin.Show().Focus() })
	trayMenu.AddSeparator()
	trayMenu.Add("退出 YHBox").OnClick(func(*application.Context) { wailsApp.Quit() })
	tray.SetMenu(trayMenu)

	// 关闭按钮（X）行为：MinimizeToTray=true 时 cancel 事件 + 隐藏到托盘；
	// false 时走默认（关窗 → app 退出，wailsApp.Run 返回）。
	mainWin.OnWindowEvent(events.Common.WindowClosing, func(e *application.WindowEvent) {
		if !app.Settings().UI.MinimizeToTray {
			return
		}
		e.Cancel()
		mainWin.Hide()
	})

	// 启动期同步注册表：用户上次设了 autostart=true，但 exe 可能被移动过，
	// 注册表里的路径已失效。每次启动都按当前 exe 路径重写一遍。
	if err := services.ApplyAutostart(app.Settings().UI.Autostart); err != nil {
		rootLog.Warn().Err(err).Str("tag", "SYSTEM").Msg("启动期自启注册表同步失败")
	}

	// 启动后异步触发一次游戏检测（emit game:status 给前端 hydrate）
	go gameSvc.Detect()

	// BattleService 启动后自动恢复上次的热键启用状态
	go battleSvc.AutoStartFromSettings()

	// 应用 logger 写一条启动日志，证明日志桥路打通
	rootLog.Info().Str("tag", "SYSTEM").Str("version", version).Msg("YHBox started")

	// 阻塞直到关窗口
	if err := wailsApp.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "wails app run failed: %v\n", err)
		os.Exit(1)
	}

	// 退出钩子（app.Shutdown 内部统一停所有 bot service）
	app.Shutdown()
}
