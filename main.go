// cmd/yhbox 异环工具箱主入口。双击 exe 启动 wails3 应用。
package main

import (
	"context"
	"embed"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/rs/zerolog"
	"github.com/wailsapp/wails/v3/pkg/application"
	"github.com/wailsapp/wails/v3/pkg/events"

	"yhbox/internal/hotkey"
	"yhbox/internal/node"
	_ "yhbox/internal/nodes/control"   // Start/Stop/Sleep/Break/Continue/Switch/If
	_ "yhbox/internal/nodes/detect"    // CheckTemplate/WaitTemplate/ClickTemplate/DetectColor/DetectColorHSV/ROIColorScan/Screenshot/ColorBarTrack
	_ "yhbox/internal/nodes/input"     // KeyPress/ClickAt/MouseMoveRel/Scroll/KeyHold*/MouseHold*/BringGameForeground/OnEvent
	_ "yhbox/internal/nodes/io"        // Log/Toast/PlayClip
	_ "yhbox/internal/nodes/purefunc"  // Add/Sub/.../Select + Expr (22+1, pure-data stubs)
	_ "yhbox/internal/nodes/stopwatch" // StopwatchStart/Stop/Read
	_ "yhbox/internal/nodes/system"    // Subgraph/SubgraphIn/Out/CollapsedNode/Throw/WindowTarget/MouseCalibration/CommentBox
	_ "yhbox/internal/nodes/variable"  // SetVar/IncVar/GetVar/GetSys/GetParam
	"yhbox/internal/services"
	"yhbox/internal/services/calibration"
	"yhbox/internal/services/container"
	"yhbox/internal/services/container/library"
	containerruntime "yhbox/internal/services/container/runtime"
	"yhbox/internal/services/execution"
	"yhbox/internal/services/schedule"
	"yhbox/internal/services/template"
	"yhbox/internal/services/tools"
	"yhbox/pkg/capture"
	"yhbox/pkg/locale"
	"yhbox/pkg/platform"
	"yhbox/pkg/screenshot"
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

	// 日志栈：zerolog → LogSink → wails3 Event.Emit + 顺便 append 到 logs/yhfish-YYYYMMDD.log
	logSink := services.NewLogSink(nil) // emit 在 wailsApp 构造后装配
	// 启动期先按 default 接通 (settings 未加载, 用 "logs" 兜底)
	logSink.SetFileWriter("logs")
	rootLog := zerolog.New(logSink).With().Timestamp().Logger()

	// v2 一次性数据迁移：旧 layout（actions/ + 单文件 containers/<id>.json + 全局 templates/）
	// → 新 layout（containers/<id>/{container.json,subgraphs/,templates/} + library/）。
	// 检测信号：bin/data/actions/ 存在 或 bin/data/templates/_index.json 存在（旧全局模板库）。
	// 命中即 rename 整个 bin/data 到 bin/data.legacy-2026-05-16/。Best-effort，失败仅日志。
	backupLegacyDataIfNeeded(rootLog)

	ensureV2DataLayout(rootLog)

	// App 协调器（不暴露 JS）
	app := services.NewApp("", logSink, rootLog) // settingsPath="" 走默认（exe 同目录）

	// app 加载完 settings 后按 settings.UI.Logger.WriteFile + FileDir 重新接通
	{
		ls := app.Settings().UI.Logger
		dir := ls.FileDir
		if !ls.WriteFile {
			dir = ""
		}
		logSink.SetFileWriter(dir)
	}

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
	_ = loc // locale 保留给后续 Locale 设置项使用

	wailsServices := make([]application.Service, 0, 13)

	// 共享 HotkeyManager。Win32 RegisterHotKey 是 process-wide unique（hWnd=NULL 时
	// 跟线程绑定），全 app 必须共享同一个实例 —— action / recorder 都注册到这里。
	// 两个 manager 就两个 hotkey 线程互相覆盖反注册，热键全丢。
	sharedHotkeys := hotkey.NewHotkeyManager()

	settingsSvc := services.NewSettingsService(app)
	gameSvc := services.NewGameService(app)

	// 数据根：<exeDir>/data/ —— Action / Container / Schedule / Template 全在这下面。
	dataDir := "data"
	if exe, err := os.Executable(); err == nil {
		dataDir = filepath.Join(filepath.Dir(exe), "data")
	}

	// ---- HotkeyRegistry：所有热键的中央 manifest ----
	// 系统热键 (execution-stop) + container 热键全部走这条路。
	// 用户可在 Settings → 快捷键 tab 改任意一条，hot reload 立即生效。
	hotkeyRegistry := hotkey.NewHotkeyRegistry(sharedHotkeys)
	hotkeyRegistry.SetCallbacks(
		// onActionHotkeyChange：v2 不再有 per-action hotkey，允许 nil。
		nil,
		// onSystemHotkeyChange：写回 settings.UI.ActionStopHotkey
		func(key, newStr string) error {
			cur := app.SnapshotSettings()
			switch key {
			case "system.execution-stop":
				cur.UI.ActionStopHotkey = newStr
			case "recording.stop":
				cur.UI.RecordingStopHotkey = newStr
			case "recording.pause":
				cur.UI.RecordingPauseHotkey = newStr
			case "system.calibrate-toggle":
				cur.UI.CalibrateHotkey = newStr
			case "tools.window-capture":
				cur.UI.WindowCaptureHotkey = newStr
			default:
				return nil
			}
			app.SwapSettings(cur)
			return app.SaveSettings()
		},
		// emit 回调：广播给所有 webview
		func() { app.Emit("hotkey:changed", map[string]any{}) },
	)

	// 暴露 HotkeyService RPC 给前端
	hotkeySvc := hotkey.NewHotkeyService(hotkeyRegistry)
	// 「重置默认」用的内置热键出厂默认 (跟 services.defaultSettings 一致, 也是下方各 Register 的 fallback)。
	// 容器热键是用户数据, 不在内 — 容器侧另给「一键清空」。
	hotkeySvc.SetSystemDefaults(map[string]string{
		"system.execution-stop":   "Ctrl+Shift+F9",
		"system.calibrate-toggle": "F8",
		"tools.window-capture":    "F9",
		"recording.stop":          "F12",
		"recording.pause":         "F11",
	})

	// Container / Schedule 数据层

	// 节点系统. 模板节点 (WaitTemplate/ClickTemplate/CheckTemplate/OnEvent) 的 Template 字段
	// 走 "template-picker" widget — inspector 直接用 TemplatePicker 读 templateSvc.List(containerID),
	// 不再走 mock asyncSource.
	nodeSvc := node.NewService()

	// v2: 库 store + service (Task 1.22)
	libStore, err := library.NewStore(filepath.Join(dataDir, "library"))
	if err != nil {
		rootLog.Fatal().Err(err).Str("tag", "STARTUP").Msg("library store init")
	}
	librarySvc := library.NewService(libStore)

	containerStore, err := container.NewStore(filepath.Join(dataDir, "containers"))
	if err != nil {
		rootLog.Fatal().Err(err).Str("tag", "STARTUP").Msg("container store init")
	}
	containerSvc := container.NewService(containerStore)

	// 模板库 (per-container, dataRoot 注入). 截模板按 containerID 经 containerSvc 解析目标窗口.
	templateSvc := template.NewService(dataDir, &templateCaptureAdapter{containers: containerSvc})

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

	// 真模板匹配 + 真颜色检测
	// input backend 由 ContainerRunner.setupRuntime 从 WindowTarget 节点解析, 不走 main.go 全局注入.
	// per-container template store 按需加载, containerID 作为 Detect 参数传入.
	//
	// container:warning emit 同时 zerolog.Warn — 让 warning 进
	// logs/yhfish-*.log (LogSink) 给 post-mortem 用.
	templateMatcher := newTemplateMatcherAdapter(dataDir, func(name string, payload map[string]any) {
		app.Emit(name, payload)
		if name == "container:warning" {
			rootLog.Warn().Interface("payload", payload).Msg("container warning")
		}
	})
	containerColor := &containerColorAdapter{
		fcEntries: make(map[uintptr]frameCacheEntry),
	}

	// InputClip: 容器级 + 库级 Service. 提前构造以便注入 PlayClip 节点需要的 ClipResolver.
	clipSvc, libClipSvc := newInputClipServices(dataDir)

	// Worker.RunFunc：load container → 构造 ContainerRunner → Run
	// container:warning 也走 zerolog 落盘 (跟 templateMatcher 同款).
	emitForRuntime := func(name string, data any) {
		app.Emit(name, data)
		if name == "container:warning" {
			rootLog.Warn().Interface("payload", data).Msg("container warning")
		}
	}
	runFunc := func(ctx context.Context, target execution.TargetRef) error {
		if target.Kind != "container" {
			return fmt.Errorf("unsupported target kind %q", target.Kind)
		}
		c, ok := containerStore.Get(target.ID)
		if !ok {
			return fmt.Errorf("container %q not found", target.ID)
		}
		rt := containerruntime.NewRuntimeContext(
			&c, inputBus, templateMatcher, containerColor,
			newGameProviderAdapter(app), emitForRuntime,
			clipSvc, app.Settings().ActiveMouseCounts360(),
		)
		r := containerruntime.NewContainerRunner(rt)
		// 把 zerolog 注入到 ServiceBundle.Log, dispatch 真节点时生效.
		r.SetLogger(rootLog)
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

	// 容器热键在「快捷键」中心页 rebind → 回写 container.json (直接 store.Save, 不走 service.Update
	// 以免 emitChange→Refresh→registry 再抢锁死锁; registry entry 已由 Update 更新, 两边一致)。
	hotkeyRegistry.SetContainerHotkeyChange(func(containerID, newStr string) error {
		c, ok := containerStore.Get(containerID)
		if !ok {
			return nil
		}
		c.Hotkey = newStr
		return containerStore.Save(&c)
	})

	// ScheduleDaemon：注册 enabled schedules 到 cron/hotkey/once，触发后入 queue
	scheduleHotkeyAdapter := &scheduleHotkeyRegistrar{reg: hotkeyRegistry}
	scheduleDaemon := schedule.NewDaemon(scheduleStore, execQueue, scheduleHotkeyAdapter)
	scheduleDaemon.Start()

	// Schedule CRUD 后重注册 cron / hotkey trigger
	scheduleSvc.SetOnChange(scheduleDaemon.Reload)

	// 全局强停热键：清 execution queue + cancel 当前 container worker run。
	// 设置面板里 UI.ActionStopHotkey 改这一条；空 → 默认 Ctrl+Shift+F9。
	stopAllHk := strings.TrimSpace(app.Settings().UI.ActionStopHotkey)
	if stopAllHk == "" {
		stopAllHk = "Ctrl+Shift+F9"
	}
	if err := hotkeyRegistry.Register("system.execution-stop", hotkey.HotkeySourceSystem,
		"hotkeys.label.system.execution_stop", nil, stopAllHk, "",
		func() {
			execQueue.CancelAll()
			worker.CancelCurrent()
		}); err != nil {
		rootLog.Warn().Err(err).Str("tag", "SYSTEM").Str("hotkey", stopAllHk).Msg("注册全局强停热键失败")
	}

	// 校准 F8 走自治 LL hook (calibrationSvc 持有), 不走 OS RegisterHotKey (游戏 reserve).
	// emit 把 'calibration:toggle' 广播给前端推进状态机; vkGetter 读热键中心当前 F8 绑定。
	calibrationSvc := calibration.NewService(
		func(name string, data any) { app.Emit(name, data) },
		func() uint32 {
			e, ok := hotkeyRegistry.Get("system.calibrate-toggle")
			if !ok || e.HotkeyStr == "" {
				return calibration.VKF8
			}
			_, vk, err := hotkey.ParseHotkey(e.HotkeyStr)
			if err != nil || vk == 0 {
				return calibration.VKF8
			}
			return vk
		},
	)

	// clipSvc / libClipSvc 提前构造 (runFunc 注入 PlayClip 用 ClipResolver).
	// 容器级走 wails RPC 给前端 (录制产物 CRUD); 库级暂保留构造, 后续 library 集成挂上.

	// recording Service 集成 clipSvc — Stop 落盘 InputClip + emit 'recording:completed'.
	recordingSvc := newRecordingService(app, clipSvc, hotkeyRegistry)

	// tools 杂项工具服务：MousePos / 鼠标 HUD / ScreenPicker 等。
	// wailsApp 还没建，先建 Service 注册到 service 列表，下面 wailsApp 后再 SetApp 注入。
	toolsSvc := tools.NewService(containerSvc)
	// 校准 HUD 窗关闭兜底: 卸 F8 钩 + 停 session (ESC/Alt+F4/崩溃都覆盖, 不依赖前端正常关)。
	toolsSvc.SetCalibratorCloseHandler(func() {
		calibrationSvc.StopHotkeyWatch()
		_, _ = calibrationSvc.Stop()
	})
	// 窗口捕获键走热键中心: 捕获时读 tools.window-capture 当前绑定 (mods+vk)，回退 F9。
	toolsSvc.SetCaptureHotkeyGetter(func() (uint32, uint32) {
		e, ok := hotkeyRegistry.Get("tools.window-capture")
		if !ok || e.HotkeyStr == "" {
			return 0, 0x78 // VK_F9
		}
		mods, vk, err := hotkey.ParseHotkey(e.HotkeyStr)
		if err != nil || vk == 0 {
			return 0, 0x78
		}
		return mods, vk
	})

	// DPI 校准 toggle 热键：默认 F8，从 settings.UI 读（rebind 经 onSystemHotkeyChange 写回）。
	// LL-hook 机制 (值持有条目, 不占 OS RegisterHotKey — 游戏会 reserve, 切游戏后失效)。
	// 真正装钩由 calibrationSvc 在校准窗开关时做 (StartHotkeyWatch 读上面的 vkGetter);
	// 命中 emit 'calibration:toggle' 推进前端状态机。热键中心仍可见 + rebind + 冲突检测。
	calibHk := strings.TrimSpace(app.Settings().UI.CalibrateHotkey)
	if calibHk == "" {
		calibHk = "F8"
	}
	if err := hotkeyRegistry.RegisterLLHook("system.calibrate-toggle", hotkey.HotkeySourceSystem,
		"hotkeys.label.system.calibrate_toggle", calibHk, ""); err != nil {
		rootLog.Warn().Err(err).Str("tag", "SYSTEM").Str("hotkey", calibHk).Msg("注册 DPI 校准热键失败")
	}

	// 录制热键 (LL-hook 全局拦截, 不占 OS RegisterHotKey — 游戏会 reserve)。
	// 默认从 settings.UI 读; registry 是编辑权威, rebind 经 onSystemHotkeyChange 写回 settings.UI。
	recStopHk := strings.TrimSpace(app.Settings().UI.RecordingStopHotkey)
	if recStopHk == "" {
		recStopHk = "F12"
	}
	if err := hotkeyRegistry.RegisterLLHook("recording.stop", hotkey.HotkeySourceRecording,
		"hotkeys.label.recording.stop", recStopHk, ""); err != nil {
		rootLog.Warn().Err(err).Str("tag", "SYSTEM").Str("hotkey", recStopHk).Msg("注册停录热键失败")
	}
	recPauseHk := strings.TrimSpace(app.Settings().UI.RecordingPauseHotkey)
	if recPauseHk == "" {
		recPauseHk = "F11"
	}
	if err := hotkeyRegistry.RegisterLLHook("recording.pause", hotkey.HotkeySourceRecording,
		"hotkeys.label.recording.pause", recPauseHk, ""); err != nil {
		rootLog.Warn().Err(err).Str("tag", "SYSTEM").Str("hotkey", recPauseHk).Msg("注册暂停录制热键失败")
	}

	// 窗口捕获键 (NodeInspector「捕获目标窗口」按下它抓前台游戏窗口)。
	// 值持有者条目 (mechanism=ll-hook, 不持久占 OS) — 进热键中心可见 + 可 rebind + 冲突检测；
	// 真正注册由 toolsSvc 捕获时临时做 (读下方 SetCaptureHotkeyGetter)。默认 F9。
	winCapHk := strings.TrimSpace(app.Settings().UI.WindowCaptureHotkey)
	if winCapHk == "" {
		winCapHk = "F9"
	}
	if err := hotkeyRegistry.RegisterLLHook("tools.window-capture", hotkey.HotkeySourceSystem,
		"hotkeys.label.system.window_capture", winCapHk, ""); err != nil {
		rootLog.Warn().Err(err).Str("tag", "SYSTEM").Str("hotkey", winCapHk).Msg("注册窗口捕获热键失败")
	}

	wailsServices = append(wailsServices,
		application.NewService(settingsSvc),
		application.NewService(services.NewAppInfoService()),
		application.NewService(gameSvc),
		application.NewService(hotkeySvc),
		application.NewService(templateSvc),
		application.NewService(containerSvc),
		application.NewService(librarySvc),
		application.NewService(scheduleSvc),
		application.NewService(calibrationSvc),
		application.NewService(recordingSvc),
		application.NewService(toolsSvc),
		application.NewService(clipSvc),
		application.NewService(nodeSvc),
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

	containerSvc.SetWindowOpener(newContainerWindowAdapter(wailsApp))
	librarySvc.SetEmit(func(name string, data any) { wailsApp.Event.Emit(name, data) })
	librarySvc.SetContainersRoot(filepath.Join(dataDir, "containers"))
	// B11: 注入容器 lookup, ImportToContainer dry-run 算 MissingGlobals diff.
	librarySvc.SetContainerLookup(containerStore.Get)
	// recording: emit 'recording:completed' 给前端 (Stop / F12 停录后落 Subgraph 走这条)
	recordingSvc.SetEmit(func(name string, data any) { wailsApp.Event.Emit(name, data) })
	// 录制完产物是 *container.Subgraph, 直接走 containerStore.SaveSubgraph 落到容器 subgraphs/
	recordingSvc.SetContainerSaver(containerStore)
	// Start 时按 containerID 拉 container, 取 WindowTarget 节点解析 hwnd
	recordingSvc.SetContainerGetter(containerStore)

	// inputclip: emit 'clip:changed' 给前端 (Save/Delete/Update 触发列表刷新)
	clipSvc.SetEmit(func(name string, data any) { wailsApp.Event.Emit(name, data) })
	libClipSvc.SetEmit(func(name string, data any) { wailsApp.Event.Emit(name, data) })

	// tools 服务也是 wailsApp 创建后才能用（要开新窗口）
	toolsSvc.SetApp(wailsApp)

	// 装配 LogSink emit（这样 logSink Write 触发 flush 时会推 log:lines 给前端）
	logSink.SetEmit(func(e services.LogLinesEvent) {
		wailsApp.Event.Emit(services.EventLogLines, e)
	})

	// 注册事件 payload 类型（让 bindings 生成器产 TS 类型）
	application.RegisterEvent[services.GameStatusEvent](services.EventGameStatus)
	application.RegisterEvent[services.LogLinesEvent](services.EventLogLines)

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
		URL:              "/#/containers",
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

	// 应用 logger 写一条启动日志，证明日志桥路打通
	rootLog.Info().Str("tag", "SYSTEM").Str("version", version).Msg("YHBox started")

	// 节点 registry 锁死: init() 注册完毕, RPC handler 之后只读.
	node.Freeze()

	// 阻塞直到关窗口
	if err := wailsApp.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "wails app run failed: %v\n", err)
		os.Exit(1)
	}

	// 退出钩子: flush log sink
	app.Shutdown()
}

// backupLegacyDataIfNeeded 检测旧 v1 数据布局，命中则整体 rename 备份。
// 调用点：main() 启动早期，settings/services 初始化之前。
func backupLegacyDataIfNeeded(log zerolog.Logger) {
	exeDir, err := os.Executable()
	if err != nil {
		log.Warn().Err(err).Str("tag", "MIGRATE").Msg("无法定位 exe，跳过 legacy 数据备份")
		return
	}
	dataRoot := filepath.Join(filepath.Dir(exeDir), "data")
	actionsDir := filepath.Join(dataRoot, "actions")
	tplIndex := filepath.Join(dataRoot, "templates", "_index.json")

	needBackup := false
	if st, err := os.Stat(actionsDir); err == nil && st.IsDir() {
		needBackup = true
	}
	if _, err := os.Stat(tplIndex); err == nil {
		needBackup = true
	}
	if !needBackup {
		return
	}

	backupDir := filepath.Join(filepath.Dir(exeDir), "data.legacy-2026-05-16")
	if _, err := os.Stat(backupDir); err == nil {
		// 已经备份过一次就不要二次覆盖
		log.Info().Str("tag", "MIGRATE").Str("path", backupDir).Msg("legacy 备份目录已存在，跳过")
		return
	}
	if err := os.Rename(dataRoot, backupDir); err != nil {
		log.Error().Err(err).Str("tag", "MIGRATE").Msg("rename data → data.legacy-2026-05-16 失败")
		return
	}
	log.Info().Str("tag", "MIGRATE").Str("backup", backupDir).Msg("v1 数据已备份；从空 layout 重新起步")
}

// ensureV2DataLayout 保证 v2 新数据 layout 的 4 个目录存在。
// 调用点：main()，紧跟 backupLegacyDataIfNeeded 之后。
func ensureV2DataLayout(log zerolog.Logger) {
	exeDir, err := os.Executable()
	if err != nil {
		log.Warn().Err(err).Str("tag", "MIGRATE").Msg("无法定位 exe，跳过 v2 layout 创建")
		return
	}
	base := filepath.Join(filepath.Dir(exeDir), "data")
	dirs := []string{
		filepath.Join(base, "containers"),
		filepath.Join(base, "library"),
		filepath.Join(base, "library", "subgraphs"),
		filepath.Join(base, "library", "templates"),
	}
	for _, d := range dirs {
		if err := os.MkdirAll(d, 0o755); err != nil {
			log.Error().Err(err).Str("tag", "MIGRATE").Str("dir", d).Msg("mkdir 失败")
		}
	}
}
