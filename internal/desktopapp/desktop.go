// Package desktopapp composes the Wails presentation adapter around the
// constructor-complete Workflow Application runtime.
package desktopapp

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

	"github.com/yottaapp/yotta/internal/ai"
	"github.com/yottaapp/yotta/internal/aiauthoring"
	"github.com/yottaapp/yotta/internal/appbootstrap"
	"github.com/yottaapp/yotta/internal/appcontrol"
	yottaapplication "github.com/yottaapp/yotta/internal/application"
	"github.com/yottaapp/yotta/internal/appruntime"
	automationinstalled "github.com/yottaapp/yotta/internal/automation/installed"
	"github.com/yottaapp/yotta/internal/blob"
	"github.com/yottaapp/yotta/internal/hotkey"
	"github.com/yottaapp/yotta/internal/httpegress"
	"github.com/yottaapp/yotta/internal/nodepackage"
	"github.com/yottaapp/yotta/internal/noderuntime"
	"github.com/yottaapp/yotta/internal/scriptengine"
	"github.com/yottaapp/yotta/internal/securestore"
	"github.com/yottaapp/yotta/internal/services"
	"github.com/yottaapp/yotta/internal/services/asset"
	"github.com/yottaapp/yotta/internal/services/calibration"
	"github.com/yottaapp/yotta/internal/services/recording"
	"github.com/yottaapp/yotta/internal/services/schedule"
	"github.com/yottaapp/yotta/internal/services/tools"
	"github.com/yottaapp/yotta/internal/services/workflow"
	"github.com/yottaapp/yotta/internal/wasmrunner"
	"github.com/yottaapp/yotta/pkg/locale"
	"github.com/yottaapp/yotta/pkg/platform"
	"github.com/yottaapp/yotta/pkg/screenshot"
	"github.com/yottaapp/yotta/pkg/version"
)

type Config struct {
	Assets   embed.FS
	TrayIcon []byte
}

func Run(config Config) error {
	// 日志栈：zerolog process/Workflow diagnostics → LogSink → 单一 log:batch 事件 + 可选 JSONL.
	logSink := services.NewLogSink(nil) // emit 在 wailsApp 构造后装配
	rootLog := zerolog.New(logSink).With().Timestamp().Logger()
	// App 构造即加载并应用日志策略，让 persisted off/level 在任何启动日志前生效。
	app := services.NewConfiguredApp("", logSink, rootLog) // settingsPath="" 走默认（exe 同目录）
	aiSecrets := services.NewAISecrets(securestore.New())

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

	wailsServices := make([]application.Service, 0, 14)

	// 共享 HotkeyManager。Win32 RegisterHotKey 是 process-wide unique（hWnd=NULL 时
	// 跟线程绑定），全 app 必须共享同一个实例 —— action / recorder 都注册到这里。
	// 两个 manager 就两个 hotkey 线程互相覆盖反注册，热键全丢。
	sharedHotkeys := hotkey.NewHotkeyManager()

	settingsSvc := services.NewSettingsService(app, aiSecrets)

	// 数据根：<exeDir>/data/。各 3.1 Store 只创建并管理自己的目录。
	dataDir := "data"
	if exe, err := os.Executable(); err == nil {
		dataDir = filepath.Join(filepath.Dir(exe), "data")
	}
	// Screenshot 节点写盘根目录 = dataDir (绝对). 不设的话节点回落到相对 "bin/data"，
	// 在 exeDir 已是 bin/ 时会拼成 bin/bin/data/... 还跟模板里的 screenshots/ 段重复。
	if err := os.Setenv("YOTTA_DATA_DIR", dataDir); err != nil {
		rootLog.Error().Err(err).Str("tag", "SYSTEM").Msg("set image output data directory")
	}
	sharedBlobStore, err := blob.Open(filepath.Join(dataDir, "blobs"), blob.Limits{MaxBlobBytes: 256 << 20, MaxTotalBytes: 4 << 30})
	if err != nil {
		return fmt.Errorf("initialize shared Blob Store: %w", err)
	}
	assetStore, err := asset.NewStore(dataDir, sharedBlobStore)
	if err != nil {
		return fmt.Errorf("initialize asset store: %w", err)
	}
	const runGrantTTL = 5 * time.Minute
	aiInstallations, err := ai.Install(app.Settings().AI.InstallationDrafts(), aiSecrets)
	if err != nil {
		return fmt.Errorf("initialize AI model installations: %w", err)
	}
	httpInstallations, err := httpegress.Install(app.Settings().Network.InstallationDrafts())
	if err != nil {
		return fmt.Errorf("initialize HTTP origin installations: %w", err)
	}
	applicationInstallations, err := appcontrol.Install(app.Settings().Applications.InstallationDrafts())
	if err != nil {
		return fmt.Errorf("initialize installed applications: %w", err)
	}
	automationDrafts, err := app.Settings().Automation.InstallationDrafts(app.Settings().Applications)
	if err != nil {
		return fmt.Errorf("read installed automation target settings: %w", err)
	}
	automationInstallations, err := automationinstalled.Install(automationDrafts)
	if err != nil {
		return fmt.Errorf("initialize installed automation targets: %w", err)
	}
	authoringTargets, err := automationinstalled.NewAuthoringTargets(automationInstallations)
	if err != nil {
		return fmt.Errorf("project installed authoring targets: %w", err)
	}
	executable, err := os.Executable()
	if err != nil {
		return fmt.Errorf("resolve script worker location: %w", err)
	}
	scriptRuntime, err := scriptengine.NewRuntime(scriptengine.RuntimeOptions{
		Executable:         filepath.Join(filepath.Dir(executable), scriptengine.WorkerExecutableName),
		ProcessMemoryBytes: scriptengine.DefaultMemoryBytes,
		JobMemoryBytes:     scriptengine.DefaultMemoryBytes,
	})
	if err != nil {
		return fmt.Errorf("initialize script runtime: %w", err)
	}
	nodePackageStore, _, err := nodepackage.OpenStoreIfPresent(context.Background(), filepath.Join(dataDir, "node-packages"))
	if err != nil {
		return fmt.Errorf("initialize node package store: %w", err)
	}
	workflowRuntime, err := appbootstrap.Build(appbootstrap.Config{
		DataRoot: dataDir, BlobStore: sharedBlobStore,
		Limits: appbootstrap.Limits{
			MaxSources: 4096, MaxPrograms: 16384, MaxRuns: 65536,
			MaxResourcePayloadBytes: 4 << 20,
			BlobChunkBytes:          64 << 10, BlobQueueCapacity: 8, StreamCapacity: 16, StreamChunkBytes: 64 << 10,
		},
		AIInstallations: aiInstallations, HTTPInstallations: httpInstallations, ApplicationInstallations: applicationInstallations, AutomationInstallations: automationInstallations, ScriptRuntime: scriptRuntime,
		NodePackageStore: nodePackageStore, WasmRunnerExecutable: filepath.Join(filepath.Dir(executable), wasmrunner.WorkerExecutableName),
		LogEmitter: newWorkflowLogEmitter(rootLog),
		GrantTTL:   runGrantTTL, OwnerCloseTimeout: 10 * time.Second, Now: time.Now,
		OnRunEvent: func(event yottaapplication.RunEvent) {
			payload := map[string]any{
				"runId": event.RunID, "status": event.Status, "generation": event.Generation, "recordDigest": event.Digest,
			}
			if event.Err != nil {
				payload["failed"] = true
				rootLog.Warn().Err(event.Err).Str("tag", "RUN").Str("runId", event.RunID).Msg("workflow Run completed with error")
			}
			app.Emit("run:changed", payload)
		},
		OnDebugEvent: func(event yottaapplication.DebugEvent) {
			app.Emit("debug:changed", map[string]any{
				"runId": event.RunID, "snapshot": event.Snapshot,
			})
		},
	})
	if err != nil {
		return fmt.Errorf("initialize workflow runtime: %w", err)
	}
	var scheduleSvc *schedule.Service
	workflowSvc, err := workflow.NewService(workflowRuntime.Application, workflow.WithBundleManager(workflowRuntime.Bundles), workflow.WithReferenceResolver(func(workflowID string) []workflow.SourceReference {
		references := make([]workflow.SourceReference, 0)
		if scheduleSvc != nil {
			for _, configured := range scheduleSvc.List() {
				for _, target := range configured.Targets {
					if target.Kind == schedule.TargetWorkflow && target.ID == workflowID {
						references = append(references, workflow.SourceReference{Kind: "schedule", ID: configured.ID, Label: configured.Name})
						break
					}
				}
			}
		}
		for _, block := range app.Settings().UI.LauncherItems {
			if block.Type == "workflow" && block.WorkflowID == workflowID {
				label := block.Label
				if label == "" {
					label = block.ID
				}
				references = append(references, workflow.SourceReference{Kind: "launcher", ID: block.ID, Label: label})
			}
		}
		return references
	}))
	if err != nil {
		return fmt.Errorf("initialize workflow service: %w", err)
	}
	aiAuthoring, err := aiauthoring.NewManager(workflowRuntime.Application, workflowRuntime.Builtins, time.Now)
	if err != nil {
		return fmt.Errorf("initialize AI authoring: %w", err)
	}

	// ---- HotkeyRegistry：所有热键的中央 manifest ----
	// 系统、录制、Schedule 与 editor 热键全部走这条路。
	// 用户可在 Settings → 快捷键 tab 改任意一条，hot reload 立即生效。
	hotkeyRegistry := hotkey.NewHotkeyRegistryWithCallbacks(sharedHotkeys, hotkey.Callbacks{
		// OnSystemChange 写回 settings.UI 的 exact binding。
		OnSystemChange: func(key, newStr string) error {
			switch key {
			case "system.execution-stop", "recording.stop", "recording.pause",
				"system.calibrate-toggle", "system.launcher-toggle", "tools.window-capture":
			default:
				return nil
			}
			_, _, err := app.MutateSettings(func(cur *services.Settings) error {
				switch key {
				case "system.execution-stop":
					cur.UI.ActionStopHotkey = newStr
				case "recording.stop":
					cur.UI.RecordingStopHotkey = newStr
				case "recording.pause":
					cur.UI.RecordingPauseHotkey = newStr
				case "system.calibrate-toggle":
					cur.UI.CalibrateHotkey = newStr
				case "system.launcher-toggle":
					cur.UI.LauncherToggleHotkey = newStr
				case "tools.window-capture":
					cur.UI.WindowCaptureHotkey = newStr
				}
				return nil
			})
			var committed interface{ Committed() bool }
			if err != nil && errors.As(err, &committed) && committed.Committed() {
				rootLog.Warn().Err(err).Str("tag", "SETTINGS").Str("hotkey", key).
					Msg("hotkey settings committed but durability sync failed")
				return nil
			}
			return err
		},
		EmitChanged: func() { app.Emit("hotkey:changed", map[string]any{}) },
	})

	// 暴露 HotkeyService RPC 给前端
	// 「重置默认」用的内置热键出厂默认 (跟 services.defaultSettings 一致, 也是下方各 Register 的 fallback)。
	hotkeySvc := hotkey.NewHotkeyService(hotkeyRegistry, map[string]string{
		"system.execution-stop":   "Ctrl+Shift+F9",
		"system.calibrate-toggle": "F8",
		"system.launcher-toggle":  "",
		"tools.window-capture":    "F9",
		"recording.stop":          "F12",
		"recording.pause":         "F11",
	})

	// Asset authoring captures exact installed targets; no Workflow
	// document can inject a native window selector.
	assetSvc := asset.NewService(assetStore, authoringTargets, workflowRuntime.Application)

	scheduleStore, err := schedule.NewStore(filepath.Join(dataDir, "schedules"))
	if err != nil {
		return fmt.Errorf("initialize schedule store: %w", err)
	}
	// Schedule triggers enter the same durable Workflow Run command as GUI.
	scheduleHotkeyAdapter := &scheduleHotkeyRegistrar{reg: hotkeyRegistry}
	scheduleDaemon := schedule.NewDaemon(scheduleStore, &workflowRunStarter{application: workflowRuntime.Application}, scheduleHotkeyAdapter)
	scheduleSvc = schedule.NewService(scheduleStore, scheduleDaemon.Reload)

	// InputClip remains an authoring asset service; 3.1 playback reads the
	// exposed nominal BlobRef through explicit blob-read and playback grants.
	clipSvc := newClipService(assetStore, app.Emit)

	// 全局强停热键取消唯一 Application worker 的 queued/running Runs。
	// 设置面板里 UI.ActionStopHotkey 改这一条；空 → 默认 Ctrl+Shift+F9。
	stopAllHk := hotkeyOrDefault(app.Settings().UI.ActionStopHotkey, "Ctrl+Shift+F9")
	if err := hotkeyRegistry.Register("system.execution-stop", hotkey.HotkeySourceSystem,
		"hotkeys.label.system.execution_stop", nil, stopAllHk, "",
		func() {
			stopAllForHotkey(func() error { return workflowRuntime.Application.CancelAll(context.Background()) }, rootLog)
		}); err != nil {
		rootLog.Warn().Err(err).Str("tag", "SYSTEM").Str("hotkey", stopAllHk).Msg("注册全局强停热键失败")
	}

	// 校准 F8 走自治 LL hook (calibrationSvc 持有), 不走 OS RegisterHotKey (游戏 reserve).
	// emit 把 'calibration:toggle' 广播给前端推进状态机; vkGetter 读热键中心当前 F8 绑定。
	calibrationSvc := calibration.NewService(
		func(name string, data any) { app.Emit(name, data) },
		func() uint32 {
			_, vk := registryHotkey(hotkeyRegistry, "system.calibrate-toggle", calibration.VKF8)
			return vk
		},
	)

	// recording Service 集成 clipSvc — Stop 落盘 InputClip + emit 'recording:completed'.
	recordingSvc := newRecordingService(app, clipSvc, hotkeyRegistry, authoringTargets, app.Emit)

	// tools 杂项工具服务：MousePos / 鼠标 HUD / ScreenPicker 等。
	// Wails app 尚未创建；先把可延迟 attach 的 presentation adapter 注入 tools core。
	toolsPresenter := &wailsToolsPresenter{}
	var quitForElevatedRestart func()
	toolsSvc := tools.NewServiceWithOptions(authoringTargets, toolsPresenter, tools.Options{
		OnCalibratorClose: func() {
			calibrationSvc.StopHotkeyWatch()
			_, _ = calibrationSvc.Stop()
		},
		CaptureHotkey: func() (uint32, uint32) {
			return registryHotkey(hotkeyRegistry, "tools.window-capture", 0x78)
		},
		RestartElevated: func() error {
			if err := launchElevated(); err != nil {
				return err
			}
			if quitForElevatedRestart != nil {
				quitForElevatedRestart()
			}
			return nil
		},
	})

	// 悬浮窗启动器 呼出/隐藏 热键：默认未绑（空），从 settings.UI 读，rebind 经 onSystemHotkeyChange 写回。
	// os-global 机制（跟 execution-stop 一致）。按键 → toggle 启动器悬浮窗显隐。
	launcherToggleHk := strings.TrimSpace(app.Settings().UI.LauncherToggleHotkey)
	if err := hotkeyRegistry.Register("system.launcher-toggle", hotkey.HotkeySourceSystem,
		"hotkeys.label.system.launcher_toggle", nil, launcherToggleHk, "",
		func() { _ = toolsSvc.ToggleLauncher() }); err != nil {
		rootLog.Warn().Err(err).Str("tag", "SYSTEM").Str("hotkey", launcherToggleHk).Msg("注册启动器 toggle 热键失败")
	}

	// DPI 校准 toggle 热键：默认 F8，从 settings.UI 读（rebind 经 onSystemHotkeyChange 写回）。
	// LL-hook 机制 (值持有条目, 不占 OS RegisterHotKey — 游戏会 reserve, 切游戏后失效)。
	// 真正装钩由 calibrationSvc 在校准窗开关时做 (StartHotkeyWatch 读上面的 vkGetter);
	// 命中 emit 'calibration:toggle' 推进前端状态机。热键中心仍可见 + rebind + 冲突检测。
	calibHk := hotkeyOrDefault(app.Settings().UI.CalibrateHotkey, "F8")
	if err := hotkeyRegistry.RegisterLLHook("system.calibrate-toggle", hotkey.HotkeySourceSystem,
		"hotkeys.label.system.calibrate_toggle", calibHk, ""); err != nil {
		rootLog.Warn().Err(err).Str("tag", "SYSTEM").Str("hotkey", calibHk).Msg("注册 DPI 校准热键失败")
	}

	// 录制热键 (LL-hook 全局拦截, 不占 OS RegisterHotKey — 游戏会 reserve)。
	// 默认从 settings.UI 读; registry 是编辑权威, rebind 经 onSystemHotkeyChange 写回 settings.UI。
	recStopHk := hotkeyOrDefault(app.Settings().UI.RecordingStopHotkey, "F12")
	if err := hotkeyRegistry.RegisterLLHook("recording.stop", hotkey.HotkeySourceRecording,
		"hotkeys.label.recording.stop", recStopHk, ""); err != nil {
		rootLog.Warn().Err(err).Str("tag", "SYSTEM").Str("hotkey", recStopHk).Msg("注册停录热键失败")
	}
	recPauseHk := hotkeyOrDefault(app.Settings().UI.RecordingPauseHotkey, "F11")
	if err := hotkeyRegistry.RegisterLLHook("recording.pause", hotkey.HotkeySourceRecording,
		"hotkeys.label.recording.pause", recPauseHk, ""); err != nil {
		rootLog.Warn().Err(err).Str("tag", "SYSTEM").Str("hotkey", recPauseHk).Msg("注册暂停录制热键失败")
	}

	// 窗口捕获键 (NodeInspector「捕获目标窗口」按下它抓前台游戏窗口)。
	// 值持有者条目 (mechanism=ll-hook, 不持久占 OS) — 进热键中心可见 + 可 rebind + 冲突检测；
	// 真正监听由 toolsSvc 捕获时临时读取 constructor-pinned getter。默认 F9。
	winCapHk := hotkeyOrDefault(app.Settings().UI.WindowCaptureHotkey, "F9")
	if err := hotkeyRegistry.RegisterLLHook("tools.window-capture", hotkey.HotkeySourceSystem,
		"hotkeys.label.system.window_capture", winCapHk, ""); err != nil {
		rootLog.Warn().Err(err).Str("tag", "SYSTEM").Str("hotkey", winCapHk).Msg("注册窗口捕获热键失败")
	}

	// Runtime resources are declared in dependency order and close in reverse.
	// Triggers stop before the single Workflow worker and its Run Owners.
	applicationRuntime := appruntime.New(
		appruntime.Resource{
			Name:  "workflow-runtime-3.1",
			Start: workflowRuntime.Start,
			Close: workflowRuntime.Close,
		},
		appruntime.Resource{
			Name:  "hotkey-registry",
			Start: func(context.Context) error { return nil },
			Close: hotkeyRegistry.Shutdown,
		},
		appruntime.Resource{
			Name:  "schedule-daemon",
			Start: func(context.Context) error { scheduleDaemon.Start(); return nil },
			Close: scheduleDaemon.StopContext,
		},
		appruntime.Resource{
			Name:  "recording",
			Start: func(context.Context) error { return nil },
			Close: func(ctx context.Context) error { return recording.Shutdown(ctx, recordingSvc) },
		},
		appruntime.Resource{
			Name:  "calibration",
			Start: func(context.Context) error { return nil },
			Close: func(ctx context.Context) error { return calibration.Shutdown(ctx, calibrationSvc) },
		},
		appruntime.Resource{
			Name:  "tools-presentation",
			Start: func(context.Context) error { return nil },
			Close: func(ctx context.Context) error {
				err := tools.Shutdown(ctx, toolsSvc)
				toolsPresenter.Detach()
				return err
			},
		},
	)

	wailsServices = append(wailsServices,
		application.NewService(settingsSvc),
		application.NewService(services.NewAppInfoService()),
		application.NewService(workflowSvc),
		application.NewService(hotkeySvc),
		application.NewService(assetSvc),
		application.NewService(scheduleSvc),
		application.NewService(calibrationSvc),
		application.NewService(recordingSvc),
		application.NewService(toolsSvc),
		application.NewService(clipSvc),
		application.NewService(services.NewAIService(app, aiSecrets, aiAuthoring)),
		application.NewService(services.NewNetworkService(app)),
		application.NewService(services.NewApplicationService(app)),
		application.NewService(services.NewAutomationService(app)),
	)
	// wails3 application
	wailsApp := application.New(application.Options{
		Name:        "Yotta",
		Description: "节点编排，自动执行",
		Services:    wailsServices,
		Windows:     wailsWindowsOptions(),
		Assets: application.AssetOptions{
			Handler: application.AssetFileServerFS(config.Assets),
		},
	})
	quitForElevatedRestart = wailsApp.Quit
	if err := app.AttachEmitter(func(name string, data any) { wailsApp.Event.Emit(name, data) }); err != nil {
		return fmt.Errorf("attach presentation emitter: %w", err)
	}

	// tools secondary windows/event delivery become ready with the GUI runtime.
	toolsPresenter.Attach(wailsApp)

	// 装配统一日志 transport（system/runtime/dump/trace 都经 log:batch）。
	logSink.SetEmit(func(e services.LogBatchEvent) {
		if app.LogStreamingEnabled() {
			wailsApp.Event.Emit(services.EventLogBatch, e)
		}
	})

	// 注册事件 payload 类型（让 bindings 生成器产 TS 类型）
	application.RegisterEvent[services.LogBatchEvent](services.EventLogBatch)

	// 主窗口尺寸读 settings（用户上次拖到的尺寸），frameless 让前端自己画 title bar
	winCfg := app.Settings().UI.Window
	mainWin := wailsApp.Window.NewWithOptions(mainWindowOptions(winCfg.Width, winCfg.Height))
	toolsPresenter.AttachMain(mainWin)

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
	tray.SetIcon(config.TrayIcon).SetTooltip("Yotta " + version.Version)
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
	trayMenu.Add("退出 Yotta").OnClick(func(*application.Context) { wailsApp.Quit() })
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

	if err := applicationRuntime.Start(context.Background()); err != nil {
		rootLog.Error().Err(err).Str("tag", "STARTUP").Msg("application runtime start")
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		_ = app.ShutdownContext(shutdownCtx)
		cancel()
		return fmt.Errorf("start application runtime: %w", err)
	}
	rootLog.Info().Str("tag", "SYSTEM").Str("version", version.Version).Msg("Yotta started")

	// 阻塞直到关窗口
	runErr := wailsApp.Run()

	shutdownCtx, cancelShutdown := context.WithTimeout(context.Background(), 5*time.Second)
	if err := applicationRuntime.Close(shutdownCtx); err != nil {
		rootLog.Warn().Err(err).Str("tag", "SHUTDOWN").Msg("application runtime close")
	}
	cancelShutdown()

	// 最后排空 presentation/log transport；使用独立 deadline，不能复用已被
	// runtime Close 消耗或取消的 context。
	presentationCtx, cancelPresentation := context.WithTimeout(context.Background(), 5*time.Second)
	if err := app.ShutdownContext(presentationCtx); err != nil {
		fmt.Fprintf(os.Stderr, "application presentation shutdown failed: %v\n", err)
	}
	cancelPresentation()
	if runErr != nil {
		return fmt.Errorf("run Wails application: %w", runErr)
	}
	return nil
}

func stopAllForHotkey(stopAll func() error, log zerolog.Logger) {
	if err := stopAll(); err != nil {
		log.Warn().Err(err).Str("tag", "SYSTEM").Msg("全局强停失败")
	}
}

func hotkeyOrDefault(value, fallback string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return fallback
	}
	return value
}

func registryHotkey(registry *hotkey.HotkeyRegistry, key string, fallback uint32) (uint32, uint32) {
	entry, ok := registry.Get(key)
	if !ok || entry.HotkeyStr == "" {
		return 0, fallback
	}
	mods, vk, err := hotkey.ParseHotkey(entry.HotkeyStr)
	if err != nil || vk == 0 {
		return 0, fallback
	}
	return mods, vk
}

func newWorkflowLogEmitter(log zerolog.Logger) noderuntime.LogEmitter {
	return noderuntime.LogEmitterFunc(func(ctx context.Context, entry noderuntime.LogEntry) error {
		if err := ctx.Err(); err != nil {
			return err
		}
		var event *zerolog.Event
		switch entry.Level {
		case "debug":
			event = log.Debug()
		case "warn":
			event = log.Warn()
		case "error":
			event = log.Error()
		default:
			event = log.Info()
		}
		event.Str("tag", "WORKFLOW").Str("graphId", entry.GraphID).Str("nodeId", entry.NodeID).
			Str("invocationId", entry.InvocationID).Int("attempt", entry.Attempt).Msg(entry.Message)
		return nil
	})
}
