// Package gui 用 lxn/walk 装配 YHBox 主窗口。
// 单进程：主线程跑 walk message loop，bot 在 goroutine 里。
package gui

import (
	"os"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/lxn/walk"
	. "github.com/lxn/walk/declarative"
	"github.com/lxn/win"

	"yhbox/pkg/log"
)

const (
	settingsFileName = "settings.json"
	logRingCapacity  = 500
)

// App 是顶层 GUI 应用，持有跨 tab 共享的状态。
type App struct {
	mw           *walk.MainWindow
	settings     *Settings
	settingsPath string
	logSink      *LogSink
	logger       *log.Logger // 长期 logger，仅挂 logSink，用于 settings/UI 层事件
	logEdit      *RichLogEdit
	logHost      *walk.GroupBox  // 装 logEdit 的「日志」GroupBox（imperative 创建 RichEdit 的父容器）
	logFrame     *walk.Composite // 包 logHost 的最外层 Composite；切日志可见性时一起隐藏，避免空白占位
	tabs         *walk.TabWidget

	game            GameStatus  // 最新游戏窗口检测结果，所有 tab 共用
	gameStatusLabel *walk.Label // 设置 tab 的游戏窗口状态显示

	fishTab   *fishTab
	cookTab   *cookTab
	pianoTab  *pianoTab
	battleTab *battleTab

	// tabContents[i] 是第 i 个 TabPage 的直接子控件快照。adaptToActiveTab 用它来切可见性：
	// 把非活动 page 的子控件全部 SetVisible(false)，walk 的 itemsToLayout 会跳过不可见非 spacer
	// 控件 (layout.go:766)，inactive page 的 MinSize 因此塌成 ~margins，TabWidget 整体 MinSize
	// 跟着 active page 走。
	// 不能用 SetLayout(nil)——walk 0.0.0-20210112 在 LayoutBase.SetContainer(nil) 和
	// ContainerBase.SetLayout(nil) 之间会死循环（验证过会 stack overflow）。
	tabContents [][]walk.Widget

	botMu    sync.Mutex // 保护 botOwner
	botOwner string     // "fish" / "cook" / "piano" / "" 表示空闲

	// shutdown 协调：关窗口时如果有 bot 跑，通知它 stop（不等 — 让 bot goroutine 自己 ReleaseBot）
	shuttingDown atomic.Bool
}

// Run 是 gui 包对外入口。version 由 main 传入用于窗口标题。
func Run(version string) error {
	app := &App{
		settingsPath: settingsPath(),
	}
	app.settings = LoadSettings(app.settingsPath)

	// logSink 在 mainWindow 装配前先建。onNewLines 通过 mw.Synchronize 调度到 UI 线程
	app.logSink = NewLogSink(logRingCapacity, app.appendLogLines)
	// 长期 logger 只挂 logSink（不写文件）—— 用于 settings/UI 事件，跟 bot 自己的 logger 解耦
	app.logger = log.New(app.logSink)

	// 装配 tabs
	fishT, fishPage := buildFishTab(app)
	cookT, cookPage := buildCookTab(app)
	pianoT, pianoPage := buildPianoTab(app)
	battleT, battlePage := buildBattleTab(app)
	app.fishTab = fishT
	app.cookTab = cookT
	app.pianoTab = pianoT
	app.battleTab = battleT

	// 初始 Bounds 用物理像素直接传 CreateWindowEx，跟 BoundsPixels 单位一致。
	initBounds := app.computeInitialBounds()

	form := MainWindow{
		AssignTo: &app.mw,
		Title:    "YHBox " + version,
		Bounds:   initBounds,
		// 卡死 walk 让用户没法把窗口拉到比 6 个 tab 标签栏还窄；不然 Settings/Help 这种
		// natural min width 很小的 tab 会让窗口缩到一半，tab 栏出现左右滚动箭头。
		MinSize: Size{Width: 400},
		Layout:  VBox{SpacingZero: true},
		Children: []Widget{
			TabWidget{
				AssignTo:              &app.tabs,
				Pages:                 []TabPage{fishPage, cookPage, pianoPage, battlePage, buildSettingsTab(app), buildHelpTab(), buildAboutTab(version)},
				OnCurrentIndexChanged: app.onTabChanged,
			},
			// 日志区：透明外层缩窄到对齐 TabWidget；白色中层提供 GroupBox 周围 9px 白边。
			Composite{
				AssignTo:      &app.logFrame,
				Layout:        HBox{Margins: Margins{Left: 1, Right: 3, Top: 0, Bottom: 0}, SpacingZero: true},
				StretchFactor: 100,
				Children: []Widget{
					Composite{
						Background: SolidColorBrush{Color: walk.RGB(255, 255, 255)},
						Layout:     VBox{Margins: Margins{Left: 9, Top: 9, Right: 9, Bottom: 9}},
						Children: []Widget{
							GroupBox{
								AssignTo:   &app.logHost,
								Title:      "日志",
								Background: SolidColorBrush{Color: walk.RGB(255, 255, 255)},
								Layout:     VBox{},
							},
						},
					},
				},
			},
		},
	}
	if err := form.Create(); err != nil {
		return err
	}

	re, err := NewRichLogEdit(app.logHost)
	if err != nil {
		return err
	}
	if f, ferr := walk.NewFont("Consolas", 9, 0); ferr == nil {
		re.SetFont(f)
	}
	re.SetWrapText(app.settings.UI.Logger.WrapText)
	app.logEdit = re

	// 标题栏 + 任务栏图标，从 winres 嵌入的 RT_GROUP_ICON "APP" 资源加载
	if ic, ierr := walk.NewIconFromResource("APP"); ierr == nil {
		_ = app.mw.SetIcon(ic)
	}

	app.mw.Closing().Attach(app.onClosing)

	// 快照每个 TabPage 的直接子控件，给 adaptToActiveTab 切可见性用
	if app.tabs != nil {
		n := app.tabs.Pages().Len()
		app.tabContents = make([][]walk.Widget, n)
		for i := range n {
			page := app.tabs.Pages().At(i)
			children := page.Children()
			ws := make([]walk.Widget, children.Len())
			for j := range children.Len() {
				ws[j] = children.At(j)
			}
			app.tabContents[i] = ws
		}
	}

	app.detectAndPropagate() // 初次检测 + 同步到所有 tab
	app.applyLogVisibility()
	app.pianoTab.preloadMIDI() // 启动时若有上次的 MIDI 路径，预解析填充 trackCombo
	app.onTabChanged()         // 初始 tab 状态（同时触发 adaptToActiveTab）

	// 如果上次关闭时战斗快捷键是 [启用] 状态，这里自动恢复（注册到 Win32）
	app.battleTab.AutoStartFromSettings()

	// 进消息循环（阻塞，直到关窗口）
	app.mw.Run()
	return nil
}

// settingsPath 返回 exe 同目录的 settings.json 路径。
func settingsPath() string {
	exe, err := os.Executable()
	if err != nil {
		return settingsFileName
	}
	return filepath.Join(filepath.Dir(exe), settingsFileName)
}

// computeInitialBounds 根据 settings 算窗口初始 bounds（物理像素）。
// 首次启动（X/Y < 0）居中到主显示器；否则用保存值。给 MainWindow.Bounds 用。
func (a *App) computeInitialBounds() Rectangle {
	w := a.settings.UI.Window
	r := Rectangle{X: w.X, Y: w.Y, Width: w.Width, Height: w.Height}
	if w.X < 0 || w.Y < 0 {
		scrW := int(win.GetSystemMetrics(win.SM_CXSCREEN))
		scrH := int(win.GetSystemMetrics(win.SM_CYSCREEN))
		r.X = (scrW - r.Width) / 2
		r.Y = (scrH - r.Height) / 2
	}
	return r
}

// SaveSettings 把当前 Settings 写到 settings.json。失败仅 log warning。
func (a *App) SaveSettings() {
	if err := SaveSettings(a.settingsPath, a.settings); err != nil {
		a.appendInternalLog("WARN: 保存 settings.json 失败: " + err.Error())
	}
}

// appendInternalLog 把一行内部消息推到 logSink。
// 走 a.logger 自动带时间戳 + [SYSTEM] 前缀，跟 bot 日志格式一致。
func (a *App) appendInternalLog(msg string) {
	if a.logger != nil {
		a.logger.Log(log.SYSTEM, "%s", msg)
		return
	}
	// 兜底：logger 还没初始化（理论不会发生，因为 logger 在 Run 入口就建好）
	a.logSink.Write([]byte(msg + "\n"))
}

// openBotLogger 给 fish/cook/piano 启动时调，统一处理「是否写日志文件」选项。
// 返回 file 可能 nil（用户关了 / mkdir 失败 / open 失败）；logger 永远非 nil，至少挂 logSink。
// 调用方在 bot 退出时若 file != nil 需 Close()。prefix 例如 "yh_fish" "yh_cook" "yh_piano"。
//
// log.New 的 sink 是 io.Writer 切片，直接传 nil *os.File 会因为接口包了 typed-nil 而 panic
// （walk Logger.Log 里直接调 w.Write）；因此 WriteFile=false 路径只把 logSink 这一个真值传进去。
func (a *App) openBotLogger(prefix string) (*os.File, *log.Logger) {
	if !a.settings.UI.Logger.WriteFile {
		return nil, log.New(a.logSink)
	}
	if err := os.MkdirAll("logs", 0755); err != nil {
		a.appendInternalLog("WARN: 创建 logs 目录失败: " + err.Error())
		return nil, log.New(a.logSink)
	}
	logPath := filepath.Join("logs", prefix+"_"+time.Now().Format("20060102_150405")+".log")
	f, err := os.Create(logPath)
	if err != nil {
		a.appendInternalLog("WARN: 日志文件创建失败 (" + logPath + "): " + err.Error())
		return nil, log.New(a.logSink)
	}
	logger := log.New(f, a.logSink)
	logger.Log(log.SYSTEM, "日志文件: %s", logPath)
	return f, logger
}

// AcquireBot 给 tab 调，独占 bot 槽位。返回 false 表示对侧已占用。
func (a *App) AcquireBot(who string) bool {
	a.botMu.Lock()
	if a.botOwner != "" {
		owner := a.botOwner
		a.botMu.Unlock()
		walk.MsgBox(a.mw, "YHBox",
			"另一个功能正在运行（"+owner+"），请先停止", walk.MsgBoxIconWarning)
		return false
	}
	a.botOwner = who
	a.botMu.Unlock()
	a.updateTabStartButtons()
	return true
}

// ReleaseBot 让 tab 在 bot goroutine 结束时调。
func (a *App) ReleaseBot() {
	a.botMu.Lock()
	a.botOwner = ""
	a.botMu.Unlock()
	if a.mw != nil {
		a.mw.Synchronize(a.updateTabStartButtons)
	}
}

// updateTabStartButtons 必须在 UI 线程调（直接调或通过 mw.Synchronize）。
func (a *App) updateTabStartButtons() {
	a.botMu.Lock()
	idle := a.botOwner == ""
	a.botMu.Unlock()
	if a.fishTab != nil {
		a.fishTab.SetStartEnabled(idle)
	}
	if a.cookTab != nil {
		a.cookTab.SetStartEnabled(idle)
	}
	if a.pianoTab != nil {
		a.pianoTab.SetStartEnabled(idle)
	}
}

// detectAndPropagate 跑一次 DetectGameWindow，更新 app.game 缓存 + 设置 tab 的状态 label，
// 并把结果同步到三个 bot tab（控制 Start 按钮可用性）。
// 触发点：① 初始启动 ② 切到 bot tab ③ Start 按下前 ④ 设置 tab 手动重新检测。
func (a *App) detectAndPropagate() {
	g := DetectGameWindow()
	a.game = g
	if a.gameStatusLabel != nil {
		a.gameStatusLabel.SetText(g.String())
	}
	if a.fishTab != nil {
		a.fishTab.RefreshGameStatus(g)
	}
	if a.cookTab != nil {
		a.cookTab.RefreshGameStatus(g)
	}
	if a.pianoTab != nil {
		a.pianoTab.RefreshGameStatus(g)
	}
}

// 日志色板：时间戳灰、各 TAG 一色、正文默认黑。
// 未在表里的 TAG 走 colorLogTagOther。
var (
	colorLogTime     = walk.RGB(150, 150, 150)
	colorLogBody     = walk.RGB(0, 0, 0)
	colorLogTagOther = walk.RGB(80, 80, 150)

	logTagColors = map[string]walk.Color{
		"SYSTEM":     walk.RGB(180, 60, 60),
		"READY":      walk.RGB(100, 100, 100),
		"IDLE":       walk.RGB(100, 100, 100),
		"SETUP":      walk.RGB(0, 128, 128),
		"WAITING":    walk.RGB(120, 120, 120),
		"CAST":       walk.RGB(150, 100, 200),
		"FIGHT":      walk.RGB(50, 130, 180),
		"FISHING":    walk.RGB(50, 130, 180),
		"RESULT":     walk.RGB(50, 150, 50),
		"RECOVERING": walk.RGB(200, 130, 0),
		"BUY":        walk.RGB(50, 150, 50),
		"BUYBAIT":    walk.RGB(50, 150, 50),
		"SHOPSELL":   walk.RGB(50, 150, 50),
		"EQUIP":      walk.RGB(120, 60, 180),
		"CHANGEBAIT": walk.RGB(120, 60, 180),
		"MISS":       walk.RGB(180, 60, 60),
		"STOP":       walk.RGB(180, 60, 60),
		"PAUSE":      walk.RGB(200, 130, 0),
	}
)

// parseLogLine 切出 "HH:MM:SS [TAG] body" 三段。无法识别则 ts/tag 空、整行进 body。
func parseLogLine(line string) (ts, tag, body string) {
	if len(line) < 9 || line[2] != ':' || line[5] != ':' || line[8] != ' ' {
		return "", "", line
	}
	ts = line[:8]
	rest := line[9:]
	if len(rest) > 0 && rest[0] == '[' {
		if end := strings.IndexByte(rest, ']'); end > 0 {
			tag = rest[1:end]
			body = strings.TrimLeft(rest[end+1:], " ")
			return
		}
	}
	body = rest
	return
}

// appendLogLines 由 logSink 在 debounce 后调用，增量追加新行（不清空重渲）。
// 滚动：插入前记录 firstVisibleLine；on → 滚到底；off → 还原视野（防 RichEdit
// 因 caret 在末尾、获焦时自动拽底）。
func (a *App) appendLogLines(lines []string) {
	if a.logEdit == nil || a.mw == nil || len(lines) == 0 {
		return
	}
	a.mw.Synchronize(func() { a.renderLogLines(lines) })
}

// renderLogLines 把若干 raw 行渲染到 logEdit，按 settings.UI.Logger 决定是否显示时间/标签。
// 必须在 UI 线程调用。appendLogLines / rerenderLog 共用。
func (a *App) renderLogLines(lines []string) {
	autoScroll := a.settings.UI.Logger.AutoScroll
	showTime := a.settings.UI.Logger.ShowTime
	showTag := a.settings.UI.Logger.ShowTag
	firstVisible := a.logEdit.FirstVisibleLine()

	for _, line := range lines {
		ts, tag, body := parseLogLine(line)
		if ts != "" && showTime {
			a.logEdit.AppendColored(ts+" ", colorLogTime)
		}
		if tag != "" && showTag {
			c, ok := logTagColors[strings.TrimSpace(tag)]
			if !ok {
				c = colorLogTagOther
			}
			a.logEdit.AppendColored("["+tag+"] ", c)
		}
		a.logEdit.AppendColored(body+"\v", colorLogBody) // \v 软换行（同段落内）
	}
	// 超 cap 就从头丢。+1 是末尾 \v 让 LineCount 多算的那个空行。
	if excess := a.logEdit.LineCount() - (logRingCapacity + 1); excess > 0 {
		a.logEdit.DeleteFirstLines(excess)
		firstVisible -= excess
		if firstVisible < 0 {
			firstVisible = 0
		}
	}
	if autoScroll {
		a.logEdit.ScrollToEnd()
	} else {
		a.logEdit.LineScrollTo(firstVisible)
	}
}

// rerenderLog 清空 logEdit 并按当前 settings 把 logSink 的全部缓存行重新渲染一遍。
// 用户在「设置」勾改 显示时间/显示标签 时调，避免历史行带时间标签、新行不带的视觉割裂。
// 必须在 UI 线程调用。
func (a *App) rerenderLog() {
	if a.logEdit == nil {
		return
	}
	a.logEdit.Clear()
	snap := a.logSink.Snapshot()
	if snap == "" {
		return
	}
	a.renderLogLines(strings.Split(snap, "\r\n"))
}

// isBotTab 当前 tab 是 fish(0) / cook(1) / piano(2) / battle(3) 才显示日志区。
// 设置/帮助/关于 不显示。
func (a *App) isBotTab() bool {
	if a.tabs == nil {
		return true
	}
	idx := a.tabs.CurrentIndex()
	return idx == 0 || idx == 1 || idx == 2 || idx == 3
}

// applyLogVisibility 按 settings.UI.Logger.Show + 当前 tab 切日志区可见性。
// 切 logFrame（最外层 Composite）让整个日志区域连白底一起消失，
// 否则只关 GroupBox 会留白底空块占位。
// 用户允许窗口在切 tab 时随之收缩到当前 tab 的自然尺寸（adaptToActiveTab 处理），
// 所以这里不再额外调用 shrinkToFit——调用方（onTabChanged / applyLogVisibilityAndShrink）
// 自己决定是否触发 shrink。
func (a *App) applyLogVisibility() {
	if a.logFrame == nil {
		return
	}
	onBotTab := a.isBotTab()
	a.logFrame.SetVisible(onBotTab && a.settings.UI.Logger.Show)
}

// applyLogVisibilityAndShrink 用户在设置 tab 切换"显示日志"时调，刷新可见性 + 压缩多余空间。
func (a *App) applyLogVisibilityAndShrink() {
	a.applyLogVisibility()
	a.shrinkToFit()
}

// shrinkToFit 把窗口高度设到 1 触发 walk startLayout 弹回到 layout 自然最小，
// 切控件可见性后自动适配 — 隐藏 GroupBox 不留 VSpacer 空白，显示也会撑大。
func (a *App) shrinkToFit() {
	if a.mw == nil {
		return
	}
	a.mw.RequestLayout()
	b := a.mw.BoundsPixels()
	b.Height = 1
	_ = a.mw.SetBoundsPixels(b)
}

// onTabChanged TabWidget.CurrentIndex 变化时触发。仅 fish/cook/piano tab 显示日志区，
// 同时切非活动 page 的子控件可见性 + shrinkToFit 让窗口贴当前 tab 自然尺寸。
func (a *App) onTabChanged() {
	a.applyLogVisibility()
	// 切到任一 bot tab 自动重新检测，刷新 Start 按钮可用性
	if a.isBotTab() {
		a.detectAndPropagate()
	}
	a.adaptToActiveTab()
}

// adaptToActiveTab 把非活动 TabPage 的直接子控件 SetVisible(false)、活动 page 的 SetVisible(true)，
// 然后 shrinkToFit 收到当前 tab 的自然尺寸。
//
// 背景：walk.tabWidgetLayoutItem.MinSize() 取所有 page 的 max，所以切到矮 tab 后窗口仍按
// 最高 tab 撑高，矮 tab 底部出现大片留白。绕道：walk 的 itemsToLayout 会跳过不可见非 spacer
// 控件 (layout.go:766)，把 inactive page 的 GroupBox 们全部隐藏 → 该 page 的 MinSize 塌成
// ~margins → TabWidget MinSize 跟随 active page。
//
// 关键：必须**无条件** SetVisible，不能先看 widget.Visible() 跳过。walk.Visible() 走的是
// Win32 IsWindowVisible（祖先链全可见才 true），所以 inactive tab 的 child 报告 false。
// 但 walk 的 layout 用的是 widgetBase.visible 内部字段 (layout.go:37 `lib.visible = ... .visible`)
// —— 这个字段只在 SetVisible 里改。跳过 SetVisible 会让 inactive child 的 visible 字段
// 永远是 true，MinSize 仍把它算进去，等于没改。
//
// 用 SetSuspended 包住切换：WM_SETREDRAW 暂停 paint/layout，等所有改动完成统一刷新，
// 避免切 tab 瞬间出现「先空白后渲染」的闪烁。
func (a *App) adaptToActiveTab() {
	if a.mw == nil || a.tabs == nil || len(a.tabContents) == 0 {
		return
	}
	idx := a.tabs.CurrentIndex()
	if idx < 0 {
		return
	}
	a.mw.SetSuspended(true)
	for i, widgets := range a.tabContents {
		visible := i == idx
		for _, w := range widgets {
			w.SetVisible(visible)
		}
	}
	a.mw.SetSuspended(false)
	a.shrinkToFit()
}

// onClosing 关窗口前的收尾：保存窗口尺寸；如果有 bot 在跑，发 Stop 让它干净退。
// 不阻塞窗口关闭 — bot goroutine 会自己 ReleaseBot + close log file。
func (a *App) onClosing(canceled *bool, reason walk.CloseReason) {
	if !a.shuttingDown.CompareAndSwap(false, true) {
		return
	}

	// 保存窗口位置 + 尺寸（物理像素，跟 MainWindow.Bounds 单位一致，HiDPI 屏 round-trip 安全）
	if a.mw != nil {
		b := a.mw.BoundsPixels()
		a.settings.UI.Window.X = b.X
		a.settings.UI.Window.Y = b.Y
		a.settings.UI.Window.Width = b.Width
		a.settings.UI.Window.Height = b.Height
		a.SaveSettings()
	}

	// 反注册全局热键，避免泄漏 ID（被别的应用占）
	if a.battleTab != nil {
		a.battleTab.Shutdown()
	}

	a.botMu.Lock()
	owner := a.botOwner
	a.botMu.Unlock()

	if owner == "" {
		return // 没 bot，直接关
	}

	// 通知对应 tab stop
	switch owner {
	case "fish":
		if a.fishTab != nil && a.fishTab.ctrl != nil {
			a.fishTab.ctrl.Stop()
		}
	case "cook":
		if a.cookTab != nil && a.cookTab.ctrl != nil {
			a.cookTab.ctrl.Stop()
		}
	case "piano":
		if a.pianoTab != nil && a.pianoTab.ctrl != nil {
			a.pianoTab.ctrl.Stop()
		}
	}
}
