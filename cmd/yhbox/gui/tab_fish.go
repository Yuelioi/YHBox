package gui

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"sync/atomic"
	"time"

	"github.com/lxn/walk"
	. "github.com/lxn/walk/declarative"

	"yhbox/pkg/log"
	"yhbox/tools/fish"
)

// fishTab 装配并管理 Fish tab 的所有控件和运行状态。
type fishTab struct {
	app *App // 反向引用主 app，用于共享 logSink、settings、全局状态

	// 控件
	autoSell    *walk.CheckBox
	runGroup    *walk.GroupBox
	startBtn    *walk.PushButton
	pauseBtn    *walk.PushButton
	stopBtn     *walk.PushButton
	statsTime   *walk.Label
	statsTotal  *walk.Label
	statsCommon *walk.Label
	statsPurple *walk.Label
	statsGolden *walk.Label

	// 运行状态
	game    GameStatus
	ctrl    *guiControl
	cfg     *fish.Config // bot 当前持有的同一个指针；运行中改 AutoSell 用
	logFile *os.File
	running atomic.Bool

	// stats 共享：bot goroutine 写 latest，UI tick goroutine 读 latest 后 Synchronize 刷 Label
	statsMu     sync.RWMutex
	statsLatest fish.Stats
	statsCancel context.CancelFunc
}

// buildFishTab 返回 TabPage 装配描述。app 提供共享 logSink/settings/全局状态。
func buildFishTab(app *App) (*fishTab, TabPage) {
	t := &fishTab{app: app}

	page := TabPage{
		Title:  "钓鱼",
		Layout: VBox{},
		Children: []Widget{
			GroupBox{
				AssignTo: &t.runGroup,
				Title:    "运行: 空闲",
				Layout:   HBox{},
				Children: []Widget{
					PushButton{AssignTo: &t.startBtn, Text: "开始", OnClicked: t.onStart},
					PushButton{AssignTo: &t.pauseBtn, Text: "暂停", OnClicked: t.onPause, Enabled: false},
					PushButton{AssignTo: &t.stopBtn, Text: "停止", OnClicked: t.onStop, Enabled: false},
				},
			},
			GroupBox{
				Title:  "配置",
				Layout: HBox{},
				Children: []Widget{
					CheckBox{
						AssignTo:         &t.autoSell,
						Text:             "自动出售鱼（会消耗都市体力）",
						Checked:          app.settings.Fish.AutoSell,
						OnCheckedChanged: t.onAutoSellChanged,
					},
					HSpacer{},
				},
			},
			GroupBox{
				Title:  "统计",
				Layout: HBox{Spacing: 6},
				Children: []Widget{
					statCard("时长", &t.statsTime, "—", 0),
					statCard("总数", &t.statsTotal, "0", 0),
					statCard("普通", &t.statsCommon, "0", colorStatCommon),
					statCard("紫色", &t.statsPurple, "0", colorStatPurple),
					statCard("金色", &t.statsGolden, "0", colorStatGolden),
				},
			},
			// VSpacer 吃掉多余空间，让 GroupBox 紧贴顶部不被撑大
			VSpacer{},
		},
	}
	return t, page
}

// statCard 一张统计卡片：上排粗体大字数值、下排小字标签，两行均居中。
// AssignTo 指向 *walk.Label 用于后续 SetText。color 为 0 表示用默认颜色。
func statCard(name string, valueTarget **walk.Label, initial string, color walk.Color) Composite {
	return Composite{
		Layout: VBox{MarginsZero: true, Spacing: 2},
		Children: []Widget{
			Label{
				AssignTo:      valueTarget,
				Text:          initial,
				TextAlignment: AlignCenter,
				TextColor:     color,
				Font:          Font{Family: "Segoe UI", PointSize: 12, Bold: true},
			},
			Label{
				Text:          name,
				TextAlignment: AlignCenter,
			},
		},
	}
}

// 统计区颜色：普通=天蓝、紫色=紫罗兰、金色=goldenrod（纯 #FFD700 在白底偏淡）
var (
	colorStatCommon = walk.RGB(30, 144, 255)
	colorStatPurple = walk.RGB(155, 48, 255)
	colorStatGolden = walk.RGB(218, 165, 32)
)

// IsRunning 给 app 查全局状态用。
func (t *fishTab) IsRunning() bool { return t.running.Load() }

// SetStartEnabled 给 app 在切 tab 或对侧 bot 状态变化时调。
func (t *fishTab) SetStartEnabled(enabled bool) {
	if t.startBtn != nil {
		t.startBtn.SetEnabled(enabled && !t.running.Load() && t.game.OK)
	}
}

// RefreshGameStatus 由 app 在初次启动 / 切 tab / 设置 tab 手动重检时统一调。
func (t *fishTab) RefreshGameStatus(g GameStatus) {
	t.game = g
	t.SetStartEnabled(true)
}

func (t *fishTab) onAutoSellChanged() {
	v := t.autoSell.Checked()
	t.app.settings.Fish.AutoSell = v
	t.app.SaveSettings()
	// bot 在跑就 atomic 写到它持有的 cfg 上，立刻生效（cfg 是同一指针）
	if t.cfg != nil {
		t.cfg.AutoSell.Store(v)
		t.app.appendInternalLog(fmt.Sprintf("AutoSell = %v（运行中立刻生效）", v))
	} else {
		t.app.appendInternalLog(fmt.Sprintf("AutoSell = %v", v))
	}
}

func (t *fishTab) onStart() {
	t.app.detectAndPropagate() // safety check：游戏可能在切 tab 之后被关
	if !t.game.OK {
		walk.MsgBox(t.app.mw, "YHBox", "未检测到异环窗口", walk.MsgBoxIconWarning)
		return
	}
	if !t.app.AcquireBot("fish") {
		return // 对侧 bot 正在跑
	}

	cfg := fish.DefaultConfig()
	cfg.AutoSell.Store(t.app.settings.Fish.AutoSell)
	t.cfg = cfg // 存指针，运行中改 checkbox 时同步写回它，让 bot 立刻看到

	// 准备 log file + logger（双 sink）。MkdirAll/Create 失败仅 warn，bot 继续跑（GUI sink 可见）。
	if err := os.MkdirAll("logs", 0755); err != nil {
		t.app.appendInternalLog("WARN: 创建 logs 目录失败: " + err.Error())
	}
	logPath := filepath.Join("logs", "yh_fish_"+time.Now().Format("20060102_150405")+".log")
	f, ferr := os.Create(logPath)
	if ferr != nil {
		t.app.appendInternalLog("WARN: 日志文件创建失败 (" + logPath + "): " + ferr.Error())
	}
	t.logFile = f
	logger := log.New(f, t.app.logSink)
	if f != nil {
		logger.Log(log.SYSTEM, "日志文件: %s", logPath)
	}

	t.ctrl = NewGUIControl()
	t.running.Store(true)
	t.setRunningUI()

	// 重置 stats 显示
	t.statsMu.Lock()
	t.statsLatest = fish.Stats{}
	t.statsMu.Unlock()
	t.updateStatsLabels(fish.Stats{})

	ctx, cancel := context.WithCancel(context.Background())
	tickCtx, tickCancel := context.WithCancel(ctx)
	t.statsCancel = tickCancel
	go t.statsTicker(tickCtx)
	go t.runBot(ctx, cancel, cfg, logger)
}

func (t *fishTab) runBot(ctx context.Context, cancel context.CancelFunc, cfg *fish.Config, logger *log.Logger) {
	defer cancel()
	defer t.afterBot()
	_ = fish.Run(ctx, t.game.HWND, cfg, logger, t.ctrl, t.onStatsChange)
}

// onStatsChange bot goroutine 调，仅落到本地 latest 缓存，刷 UI 交给 statsTicker。
func (t *fishTab) onStatsChange(s fish.Stats) {
	t.statsMu.Lock()
	t.statsLatest = s
	t.statsMu.Unlock()
}

// statsTicker 每秒拉 latest + 算时长，Synchronize 刷 Label。ctx 取消即退出。
// 用 prev 缓存上次渲染过的快照和时长文本，没变化就不进 Synchronize（默认场景：
// 一分钟内只钓几条鱼，60 次 tick 里 ~55 次完全相同）。
func (t *fishTab) statsTicker(ctx context.Context) {
	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()
	var prev fish.Stats
	var prevTime string
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if t.app.shuttingDown.Load() {
				return
			}
			t.statsMu.RLock()
			snap := t.statsLatest
			t.statsMu.RUnlock()
			timeStr := formatDurationHM(snap.StartedAt)
			if snap == prev && timeStr == prevTime {
				continue
			}
			prev, prevTime = snap, timeStr
			if t.app.mw == nil {
				continue
			}
			t.app.mw.Synchronize(func() { t.updateStatsLabels(snap) })
		}
	}
}

func (t *fishTab) updateStatsLabels(s fish.Stats) {
	if t.statsTime == nil {
		return
	}
	t.statsTime.SetText(formatDurationHM(s.StartedAt))
	total := s.CommonCount + s.PurpleCount + s.GoldenCount
	t.statsTotal.SetText(fmt.Sprintf("%d", total))
	t.statsCommon.SetText(fmt.Sprintf("%d", s.CommonCount))
	t.statsPurple.SetText(fmt.Sprintf("%d", s.PurpleCount))
	t.statsGolden.SetText(fmt.Sprintf("%d", s.GoldenCount))
}

// formatDurationHM 把启动时间起算的时长格式化成 "Nh Mm"（秒不显示）。
// StartedAt 零值（未启动）显示 "—"。
func formatDurationHM(startedAt time.Time) string {
	if startedAt.IsZero() {
		return "—"
	}
	mins := int(time.Since(startedAt).Minutes())
	return fmt.Sprintf("%dh %dm", mins/60, mins%60)
}

func (t *fishTab) afterBot() {
	if t.statsCancel != nil {
		t.statsCancel()
		t.statsCancel = nil
	}
	if t.logFile != nil {
		t.logFile.Close()
		t.logFile = nil
	}
	t.cfg = nil // bot 已退，cfg 指针失效，避免下次 onAutoSellChanged 误写
	t.running.Store(false)
	t.app.ReleaseBot()
	if t.app.mw != nil {
		t.app.mw.Synchronize(func() {
			t.setIdleUI()
		})
	}
}

func (t *fishTab) onPause() {
	if t.ctrl == nil {
		return
	}
	if t.ctrl.IsPaused() {
		t.ctrl.Resume()
		t.setRunningUI()
	} else {
		t.ctrl.Pause()
		t.setPausedUI()
	}
}

func (t *fishTab) onStop() {
	if t.ctrl == nil {
		return
	}
	t.ctrl.Stop()
	t.stopBtn.SetEnabled(false) // 防双击
	// goroutine 跑完会调 afterBot → setIdleUI
}

func (t *fishTab) setIdleUI() {
	t.runGroup.SetTitle("运行: 空闲")
	t.startBtn.SetEnabled(t.game.OK)
	t.pauseBtn.SetEnabled(false)
	t.pauseBtn.SetText("暂停")
	t.stopBtn.SetEnabled(false)
	t.autoSell.SetEnabled(true)
}

func (t *fishTab) setRunningUI() {
	t.runGroup.SetTitle("运行: 运行中")
	t.startBtn.SetEnabled(false)
	t.pauseBtn.SetEnabled(true)
	t.pauseBtn.SetText("暂停")
	t.stopBtn.SetEnabled(true)
	t.autoSell.SetEnabled(true) // 运行中也允许改（下次满仓生效）
}

func (t *fishTab) setPausedUI() {
	t.runGroup.SetTitle("运行: 已暂停")
	t.pauseBtn.SetText("继续")
}
