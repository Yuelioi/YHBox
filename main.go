// Yotta 主入口。双击 exe 启动 wails3 应用。
package main

import (
	"context"
	"embed"
	"errors"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/mark3labs/mcp-go/server"
	"github.com/rs/zerolog"
	"github.com/wailsapp/wails/v3/pkg/application"
	"github.com/wailsapp/wails/v3/pkg/events"

	"github.com/yottaapp/yotta/internal/appruntime"
	"github.com/yottaapp/yotta/internal/hotkey"
	"github.com/yottaapp/yotta/internal/node"
	_ "github.com/yottaapp/yotta/internal/nodes/all"
	"github.com/yottaapp/yotta/internal/runclassify"
	"github.com/yottaapp/yotta/internal/services"
	"github.com/yottaapp/yotta/internal/services/androidadb"
	"github.com/yottaapp/yotta/internal/services/asset"
	"github.com/yottaapp/yotta/internal/services/browsercdp"
	"github.com/yottaapp/yotta/internal/services/calibration"
	"github.com/yottaapp/yotta/internal/services/codesnippet"
	"github.com/yottaapp/yotta/internal/services/container"
	containerruntime "github.com/yottaapp/yotta/internal/services/container/runtime"
	"github.com/yottaapp/yotta/internal/services/execution"
	"github.com/yottaapp/yotta/internal/services/inputclip"
	mcpserver "github.com/yottaapp/yotta/internal/services/mcpserver"
	"github.com/yottaapp/yotta/internal/services/nodeoptions"
	"github.com/yottaapp/yotta/internal/services/recording"
	"github.com/yottaapp/yotta/internal/services/schedule"
	"github.com/yottaapp/yotta/internal/services/tools"
	"github.com/yottaapp/yotta/pkg/locale"
	"github.com/yottaapp/yotta/pkg/platform"
	"github.com/yottaapp/yotta/pkg/screenshot"
	"github.com/yottaapp/yotta/pkg/version"
)

//go:embed all:frontend/dist
var assets embed.FS

// 用跟 exe icon 同一份 build/windows/icon.ico —— 多 size 容器，Windows 自动挑 16/32px 进托盘。
//
//go:embed build/windows/icon.ico
var trayIcon []byte

func main() {
	platform.EnsureAdmin()

	// 日志栈：zerolog → LogSink → wails3 Event.Emit + 顺便 append 到 logs/yotta-YYYYMMDD.log
	logSink := services.NewLogSink(nil) // emit 在 wailsApp 构造后装配
	// 启动期先按 default 接通 (settings 未加载, 用 "logs" 兜底)
	logSink.SetFileWriter("logs")
	rootLog := zerolog.New(logSink).With().Timestamp().Logger()

	// v2 一次性数据迁移：旧 layout（actions/ + 单文件 containers/<id>.json + 全局 templates/）
	// → 新 layout（containers/<id>/{package.json,graph.json,installation.json,yotta-lock.json} + library/）。
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

	settingsSvc := services.NewSettingsService(app)

	// AI 节点用的按连接缓存 Provider 服务(进程级单例)。getter 每次读 live settings 快照,
	// 改连接 endpoint/key 后指纹自愈重建。注入进每个 ContainerRunner。
	aiProviderCache := services.NewAIProviderCache(func() *services.Settings {
		s := settingsSvc.Get()
		return &s
	})

	// 数据根：<exeDir>/data/ —— Action / Container / Schedule / Template 全在这下面。
	dataDir := "data"
	if exe, err := os.Executable(); err == nil {
		dataDir = filepath.Join(filepath.Dir(exe), "data")
	}
	// Screenshot 节点写盘根目录 = dataDir (绝对). 不设的话节点回落到相对 "bin/data"，
	// 在 exeDir 已是 bin/ 时会拼成 bin/bin/data/... 还跟模板里的 screenshots/ 段重复。
	if err := os.Setenv("YOTTA_DATA_DIR", dataDir); err != nil {
		rootLog.Error().Err(err).Str("tag", "SYSTEM").Msg("set image output data directory")
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
		"system.launcher-toggle":  "",
		"tools.window-capture":    "F9",
		"recording.stop":          "F12",
		"recording.pause":         "F11",
	})

	// Container / Schedule 数据层

	// 节点系统. 模板节点 (WaitTemplate/ClickTemplate/CheckTemplate) 的 Templates 字段 (GUID)
	// 走 "template-picker" widget — inspector 直接用 TemplatePicker 读 assetSvc.List() (全局).
	nodeSvc := node.NewService()
	androidadb.RegisterNodeAsyncSource(nodeSvc, androidadb.NewService(nil))
	browserCDPSvc := browsercdp.NewService("")
	browserCDPProvider := browsercdp.NewClientProvider(browserCDPSvc)

	// 全局资产库 (template + clip 统一): <dataDir>/{templates,clips,blobs} 平铺布局.
	// 单实例全局共享 — matcher / validator / library / asset RPC / clip resolver 都接这一个.
	assetStore, err := asset.NewStore(dataDir)
	if err != nil {
		rootLog.Fatal().Err(err).Str("tag", "STARTUP").Msg("asset store init")
	}

	// 全局子图池 (2026-06-12 全局化: 容器只引用不复制): <dataDir>/subgraphs/.
	sgStore, err := container.NewSubgraphStore(filepath.Join(dataDir, "subgraphs"))
	if err != nil {
		rootLog.Fatal().Err(err).Str("tag", "STARTUP").Msg("subgraph store init")
	}
	sgSvc := container.NewSubgraphService(sgStore)

	containerStore, err := container.NewStore(filepath.Join(dataDir, "containers"))
	if err != nil {
		rootLog.Fatal().Err(err).Str("tag", "STARTUP").Msg("container store init")
	}
	// 校验用引用闭包解析 (单一咽喉, 见 wire_subgraph.go).
	containerStore.SetSubgraphResolver(func(c *container.Container) []container.Subgraph {
		return subgraphClosureFor(c, sgStore)
	})
	containerStore.SetAssetStore(assetStore)
	containerSvc := container.NewService(containerStore)
	container.ConfigureSubgraphReferrerScanner(sgSvc, scanSubgraphReferrers(containerStore, sgStore))

	// 匿名子图 GC (mark-sweep, 幂等): 启动时 + 容器删除完成后. 锁序 Container → Subgraph.
	gcAnonymousSubgraphs := func() {
		removed, gcErr := sgStore.GCAnonymous(collectReferencedSubgraphIDs(containerStore, sgStore))
		if gcErr != nil {
			rootLog.Warn().Err(gcErr).Str("tag", "SUBGRAPH-GC").Msg("匿名子图 GC 失败")
		} else if len(removed) > 0 {
			rootLog.Info().Strs("removed", removed).Str("tag", "SUBGRAPH-GC").Msg("匿名子图回收")
		}
	}
	gcAnonymousSubgraphs()
	container.ConfigurePostDelete(containerSvc, gcAnonymousSubgraphs)
	// validator 存在性检查: 节点引用的模板/clip GUID 必须存在于全局 asset 库.
	container.ConfigureAssetExistence(containerSvc,
		assetExistence(assetStore, asset.KindTemplate),
		assetExistence(assetStore, asset.KindClip),
	)

	// 资产 RPC 服务 (全局, 无 containerID). 截模板按 containerID 经 containerSvc 解析目标窗口.
	assetSvc := asset.NewService(assetStore, &templateCaptureAdapter{containers: containerSvc})
	// 删资产前扫全部容器+子图引用, 返 Referrer 列表 (不阻断, FE 弹"被 N 处引用"警告).
	asset.ConfigureReferrerScanner(assetSvc, scanAssetReferrers(containerStore, sgStore))
	// 注: change listener 在 templateMatcher 构造后接 (见下), 让存资产立刻让 matcher 解码缓存失效.
	nodeoptions.RegisterAssetAsyncSources(nodeSvc, assetSvc, sgSvc)

	scheduleStore, err := schedule.NewStore(filepath.Join(dataDir, "schedules"))
	if err != nil {
		rootLog.Fatal().Err(err).Str("tag", "STARTUP").Msg("schedule store init")
	}
	scheduleSvc := schedule.NewService(scheduleStore)

	// 编辑器用户代码片段 (Script/Expr 放大编辑「片段」菜单): <dataDir>/snippets.json 整存整取.
	codeSnippetSvc := codesnippet.NewService(filepath.Join(dataDir, "snippets.json"))

	// ---- 运行时层：ExecutionQueue + Worker + ScheduleDaemon ----
	//
	// 键鼠独占：单 worker，串行消费 queue；hotkey/schedule/UI 触发都进同一 queue。
	execQueue := execution.NewExecutionQueue()
	inputBus := execution.NewInputBus()

	// 真模板匹配 + 真颜色检测
	// input backend 由 ContainerRunner.setupRuntime 从 Win32WindowTarget 节点解析, 不走 main.go 全局注入.
	// 资产全局 (asset.Store): matcher 按 guid 匹配, scaleTolerance 由 VisionAdapter (有容器上下文) 传入.
	//
	// container:warning emit 同时 zerolog.Warn — 让 warning 进
	// logs/yotta-*.log (LogSink) 给 post-mortem 用.
	templateMatcher := newTemplateMatcherAdapter(assetStore, func(name string, payload map[string]any) {
		app.Emit(name, payload)
		if name == "container:warning" {
			rootLog.Warn().Interface("payload", payload).Msg("container warning")
		}
	})
	// 用户在编辑器里存/删/重拍资产后, 让 matcher 丢弃解码缓存 (否则重拍换 blob 后旧图还在匹配).
	asset.ConfigureChangeListener(assetSvc, templateMatcher.Invalidate)
	// InputClip: 全局 clip Service (asset clip kind). 提前构造以便注入 PlayClip 节点需要的 ClipResolver.
	clipSvc := newClipService(assetStore)

	// Worker.RunFunc：load container → 构造 ContainerRunner → Run
	// container:warning 也走 zerolog 落盘 (跟 templateMatcher 同款).
	emitForRuntime := func(name string, data any) {
		app.Emit(name, data)
		if name == "container:warning" {
			rootLog.Warn().Interface("payload", data).Msg("container warning")
		}
	}
	newRunnerForContainer := func(containerID string) (*containerruntime.ContainerRunner, error) {
		c, ok := containerStore.Get(containerID)
		if !ok {
			return nil, fmt.Errorf("container %q not found", containerID)
		}
		rt := containerruntime.NewRuntimeContext(
			&c, inputBus, templateMatcher,
			newGameProviderAdapter(), emitForRuntime,
			clipSvc, app.Settings().ActiveMouseCounts360(),
		)
		rt.ControllerFactory = containerruntime.DefaultControllerFactory{BrowserCDP: browserCDPProvider}
		// 起跑时从全局池解析引用闭包 → 快照进 rt (跑中不回查 store, 编辑不影响在跑实例).
		rt.Subgraphs = subgraphClosureFor(&c, sgStore)
		r := containerruntime.NewContainerRunner(rt)
		// 把 zerolog 注入到 ServiceBundle.Log, dispatch 真节点时生效.
		r.SetLogger(rootLog)
		r.SetAIProvider(aiProviderCache)
		return r, nil
	}
	runFunc := func(ctx context.Context, target execution.TargetRef) error {
		if target.Kind != "container" {
			return fmt.Errorf("unsupported target kind %q", target.Kind)
		}
		r, err := newRunnerForContainer(target.ID)
		if err != nil {
			return err
		}
		return r.Run(ctx)
	}
	worker := execution.NewWorker(execQueue, runFunc, func(s execution.WorkerState) {
		app.Emit("execution:state", s)
	}, runclassify.RunError)
	// 给 containerSvc 注入运行入口 — 前端 ▶ 按钮走 Run，■ 走 StopAll
	debugManager := newContainerDebugManager(newRunnerForContainer, func(name string, data any) {
		app.Emit(name, data)
	}, worker.IsRunning)
	container.ConfigureRunner(containerSvc, &containerRunnerAdapter{queue: execQueue, worker: worker, debug: debugManager})

	// MCP 对外暴露 server (③): 复用执行标准件, 后台起 Streamable HTTP.
	mcpSrv := mcpserver.NewServer(mcpserver.Deps{
		Store:       containerStore,
		InputBus:    inputBus,
		Matcher:     templateMatcher,
		Game:        newGameProviderAdapter(),
		Clip:        clipSvc,
		MouseCounts: func() int { return app.Settings().ActiveMouseCounts360() },
		Armed:       func() bool { return app.Settings().MCP.Armed },
		Busy:        worker.IsRunning,
	})
	mcpCore := server.NewMCPServer("yotta-mcp", "0.1.0")
	mcpSrv.Register(mcpCore)
	mcpMux := http.NewServeMux()
	mcpMux.Handle("/mcp", server.NewStreamableHTTPServer(mcpCore))
	mcpHTTP := appruntime.NewHTTPServer("127.0.0.1:8765", mcpMux)

	// Container hotkey 绑定：扫所有容器，把 container.hotkey 注册到 registry。
	// CRUD 后经 ConfigureChangeListener 重扫一次。触发后入队单 target manual run。
	containerHotkeys := newContainerHotkeyBinder(containerStore, hotkeyRegistry, execQueue)
	containerHotkeys.Refresh()
	container.ConfigureChangeListener(containerSvc, containerHotkeys.Refresh)

	// 容器热键在「快捷键」中心页 rebind → 回写容器四件套 (直接 store.Save, 不走 service.Update
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

	// Schedule CRUD 后重注册 cron / hotkey trigger
	schedule.ConfigureChangeListener(scheduleSvc, scheduleDaemon.Reload)

	// 全局强停热键：清 execution queue + cancel 当前 container worker run。
	// 设置面板里 UI.ActionStopHotkey 改这一条；空 → 默认 Ctrl+Shift+F9。
	stopAllHk := strings.TrimSpace(app.Settings().UI.ActionStopHotkey)
	if stopAllHk == "" {
		stopAllHk = "Ctrl+Shift+F9"
	}
	if err := hotkeyRegistry.Register("system.execution-stop", hotkey.HotkeySourceSystem,
		"hotkeys.label.system.execution_stop", nil, stopAllHk, "",
		func() {
			stopAllForHotkey(containerSvc.StopAll, rootLog)
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

	// clipSvc 提前构造 (runFunc 注入 PlayClip 用 ClipResolver). 全局 clip 库走 wails RPC 给前端.

	// recording Service 集成 clipSvc — Stop 落盘 InputClip + emit 'recording:completed'.
	recordingSvc := newRecordingService(app, clipSvc, hotkeyRegistry)

	// tools 杂项工具服务：MousePos / 鼠标 HUD / ScreenPicker 等。
	// Wails app 尚未创建；先把可延迟 attach 的 presentation adapter 注入 tools core。
	toolsPresenter := &wailsToolsPresenter{}
	toolsSvc := tools.NewService(containerSvc, toolsPresenter)
	// 校准 HUD 窗关闭兜底: 卸 F8 钩 + 停 session (ESC/Alt+F4/崩溃都覆盖, 不依赖前端正常关)。
	tools.ConfigureCalibratorCloseHandler(toolsSvc, func() {
		calibrationSvc.StopHotkeyWatch()
		_, _ = calibrationSvc.Stop()
	})
	// 窗口捕获键走热键中心: 捕获时读 tools.window-capture 当前绑定 (mods+vk)，回退 F9。
	tools.ConfigureCaptureHotkeyGetter(toolsSvc, func() (uint32, uint32) {
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
	// 真正注册由 toolsSvc 捕获时临时做 (读下方 ConfigureCaptureHotkeyGetter)。默认 F9。
	winCapHk := strings.TrimSpace(app.Settings().UI.WindowCaptureHotkey)
	if winCapHk == "" {
		winCapHk = "F9"
	}
	if err := hotkeyRegistry.RegisterLLHook("tools.window-capture", hotkey.HotkeySourceSystem,
		"hotkeys.label.system.window_capture", winCapHk, ""); err != nil {
		rootLog.Warn().Err(err).Str("tag", "SYSTEM").Str("hotkey", winCapHk).Msg("注册窗口捕获热键失败")
	}

	// Runtime resources are declared in dependency order and close in reverse:
	// UI/tool activity first, then interactive hooks, producers, and the worker.
	applicationRuntime := appruntime.New(
		appruntime.Resource{
			Name:  "execution-worker",
			Start: func(context.Context) error { worker.Start(); return nil },
			Close: worker.StopContext,
		},
		appruntime.Resource{
			Name:  "debug-manager",
			Start: func(context.Context) error { return nil },
			Close: debugManager.CloseContext,
		},
		appruntime.Resource{
			Name:  "hotkey-registry",
			Start: func(context.Context) error { return nil },
			Close: hotkeyRegistry.Shutdown,
		},
		mcpHTTP.Resource("mcp-http"),
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
		application.NewService(hotkeySvc),
		application.NewService(assetSvc),
		application.NewService(containerSvc),
		application.NewService(sgSvc),
		application.NewService(scheduleSvc),
		application.NewService(calibrationSvc),
		application.NewService(recordingSvc),
		application.NewService(toolsSvc),
		application.NewService(clipSvc),
		application.NewService(nodeSvc),
		application.NewService(codeSnippetSvc),
		application.NewService(services.NewAIService()),
	)
	// wails3 application
	wailsApp := application.New(application.Options{
		Name:        "Yotta",
		Description: "节点编排，自动执行",
		Services:    wailsServices,
		Assets: application.AssetOptions{
			Handler: application.AssetFileServerFS(assets),
		},
	})
	if err := app.AttachEmitter(func(name string, data any) { wailsApp.Event.Emit(name, data) }); err != nil {
		rootLog.Fatal().Err(err).Str("tag", "STARTUP").Msg("attach presentation emitter")
	}

	container.ConfigureSubgraphEmitter(sgSvc, func(name string, data any) { wailsApp.Event.Emit(name, data) })
	// recording: emit 'recording:completed' 给前端 (Stop / F12 停录后落 Subgraph 走这条)
	recording.ConfigureEmitter(recordingSvc, func(name string, data any) { wailsApp.Event.Emit(name, data) })
	// 录制完产物是 *container.Subgraph, 直接入全局子图池 (sgStore.Create)
	recording.ConfigureSubgraphSaver(recordingSvc, sgStore)
	// Start 时按 containerID 拉 container, 取 Win32WindowTarget 节点解析 hwnd
	recording.ConfigureContainerGetter(recordingSvc, containerStore)

	// inputclip: emit 'clip:changed' 给前端 (Save/Delete/Update 触发列表刷新)
	inputclip.ConfigureEmitter(clipSvc, func(name string, data any) { wailsApp.Event.Emit(name, data) })

	// tools secondary windows/event delivery become ready with the GUI runtime.
	toolsPresenter.Attach(wailsApp)

	// 装配 LogSink emit（这样 logSink Write 触发 flush 时会推 log:lines 给前端）
	logSink.SetEmit(func(e services.LogLinesEvent) {
		wailsApp.Event.Emit(services.EventLogLines, e)
	})

	// 注册事件 payload 类型（让 bindings 生成器产 TS 类型）
	application.RegisterEvent[services.LogLinesEvent](services.EventLogLines)

	// 主窗口尺寸读 settings（用户上次拖到的尺寸），frameless 让前端自己画 title bar
	winCfg := app.Settings().UI.Window
	mainWin := wailsApp.Window.NewWithOptions(application.WebviewWindowOptions{
		Title:            "Yotta " + version.Version,
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
	tray.SetIcon(trayIcon).SetTooltip("Yotta " + version.Version)
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

	// 节点 registry 锁死: init() 注册完毕, RPC handler 之后只读.
	node.Freeze()
	if err := applicationRuntime.Start(context.Background()); err != nil {
		rootLog.Error().Err(err).Str("tag", "STARTUP").Msg("application runtime start")
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		_ = app.ShutdownContext(shutdownCtx)
		cancel()
		fmt.Fprintf(os.Stderr, "application runtime start failed: %v\n", err)
		os.Exit(1)
	}
	go func() {
		<-mcpHTTP.Done()
		if err := mcpHTTP.Err(); err != nil {
			rootLog.Warn().Err(err).Str("tag", "MCP").Msg("MCP HTTP server 退出")
		}
	}()
	rootLog.Info().Str("tag", "MCP").Msg("MCP server: http://127.0.0.1:8765/mcp")
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
		fmt.Fprintf(os.Stderr, "wails app run failed: %v\n", runErr)
		os.Exit(1)
	}
}

func stopAllForHotkey(stopAll func() error, log zerolog.Logger) {
	if err := stopAll(); err != nil {
		log.Warn().Err(err).Str("tag", "SYSTEM").Msg("全局强停失败")
	}
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

// ensureV2DataLayout 保证平铺数据 layout 的顶层目录存在 (类型即目录, 2026-06-12 大整理)。
func ensureV2DataLayout(log zerolog.Logger) {
	exeDir, err := os.Executable()
	if err != nil {
		log.Warn().Err(err).Str("tag", "MIGRATE").Msg("无法定位 exe，跳过 layout 创建")
		return
	}
	base := filepath.Join(filepath.Dir(exeDir), "data")
	dirs := []string{
		filepath.Join(base, "containers"),
		filepath.Join(base, "subgraphs"),
		filepath.Join(base, "templates"),
		filepath.Join(base, "clips"),
		filepath.Join(base, "blobs"),
		filepath.Join(base, "schedules"),
	}
	for _, d := range dirs {
		if err := os.MkdirAll(d, 0o755); err != nil {
			log.Error().Err(err).Str("tag", "MIGRATE").Str("dir", d).Msg("mkdir 失败")
		}
	}
}
