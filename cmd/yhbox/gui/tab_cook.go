package gui

import (
	"context"
	"os"
	"sync/atomic"
	"time"

	"github.com/lxn/walk"
	. "github.com/lxn/walk/declarative"

	"yhbox/pkg/log"
	"yhbox/tools/cook"
)

// cookTab 装配并管理 Cook tab 的所有控件和运行状态。
type cookTab struct {
	app *App // 反向引用主 app，用于共享 logSink、settings、全局状态

	// 控件
	intervalEdit *walk.NumberEdit
	runGroup     *walk.GroupBox
	startBtn     *walk.PushButton
	pauseBtn     *walk.PushButton
	stopBtn      *walk.PushButton

	// 运行状态
	game    GameStatus
	ctrl    *guiControl
	logFile *os.File
	running atomic.Bool
}

// buildCookTab 返回 TabPage 装配描述。app 提供共享 logSink/settings/全局状态。
func buildCookTab(app *App) (*cookTab, TabPage) {
	t := &cookTab{app: app}

	page := TabPage{
		Title:  "店长",
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
					Label{Text: "点击间隔 (ms):"},
					NumberEdit{
						AssignTo:       &t.intervalEdit,
						Value:          float64(app.settings.Cook.IntervalMs),
						MinValue:       100,
						MaxValue:       10000,
						OnValueChanged: t.onIntervalChanged,
					},
				},
			},
			// VSpacer 吃掉多余空间，让 GroupBox 紧贴顶部不被撑大
			VSpacer{},
		},
	}
	return t, page
}

// IsRunning 给 app 查全局状态用。
func (t *cookTab) IsRunning() bool { return t.running.Load() }

// SetStartEnabled 给 app 在切 tab 或对侧 bot 状态变化时调。
func (t *cookTab) SetStartEnabled(enabled bool) {
	if t.startBtn != nil {
		t.startBtn.SetEnabled(enabled && !t.running.Load() && t.game.OK)
	}
}

// RefreshGameStatus 由 app 在初次启动 / 切 tab / 设置 tab 手动重检时统一调。
func (t *cookTab) RefreshGameStatus(g GameStatus) {
	t.game = g
	t.SetStartEnabled(true)
}

func (t *cookTab) onIntervalChanged() {
	t.app.settings.Cook.IntervalMs = int(t.intervalEdit.Value())
	t.app.SaveSettings()
}

func (t *cookTab) onStart() {
	t.app.detectAndPropagate() // safety check：游戏可能在切 tab 之后被关
	if !t.game.OK {
		walk.MsgBox(t.app.mw, "YHBox", "未检测到异环窗口", walk.MsgBoxIconWarning)
		return
	}
	if !t.app.AcquireBot("cook") {
		return // 对侧 bot 正在跑
	}

	cfg := cook.DefaultConfig()
	cfg.Interval = time.Duration(t.app.settings.Cook.IntervalMs) * time.Millisecond

	// 准备 logger。按 settings.UI.Logger.WriteFile 决定是否同时写日志文件。
	var logger *log.Logger
	t.logFile, logger = t.app.openBotLogger("yh_cook")

	t.ctrl = NewGUIControl()
	t.running.Store(true)
	t.setRunningUI()

	ctx, cancel := context.WithCancel(context.Background())
	go t.runBot(ctx, cancel, cfg, logger)
}

func (t *cookTab) runBot(ctx context.Context, cancel context.CancelFunc, cfg *cook.Config, logger *log.Logger) {
	defer cancel()
	defer t.afterBot()
	cook.Run(ctx, t.game.HWND, cfg, logger, t.ctrl)
}

func (t *cookTab) afterBot() {
	if t.logFile != nil {
		t.logFile.Close()
		t.logFile = nil
	}
	t.running.Store(false)
	t.app.ReleaseBot()
	if t.app.mw != nil {
		t.app.mw.Synchronize(func() {
			t.setIdleUI()
		})
	}
}

func (t *cookTab) onPause() {
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

func (t *cookTab) onStop() {
	if t.ctrl == nil {
		return
	}
	t.ctrl.Stop()
	t.stopBtn.SetEnabled(false) // 防双击
	// goroutine 跑完会调 afterBot → setIdleUI
}

func (t *cookTab) setIdleUI() {
	t.runGroup.SetTitle("运行: 空闲")
	t.startBtn.SetEnabled(t.game.OK)
	t.pauseBtn.SetEnabled(false)
	t.pauseBtn.SetText("暂停")
	t.stopBtn.SetEnabled(false)
	t.intervalEdit.SetEnabled(true)
}

func (t *cookTab) setRunningUI() {
	t.runGroup.SetTitle("运行: 运行中")
	t.startBtn.SetEnabled(false)
	t.pauseBtn.SetEnabled(true)
	t.pauseBtn.SetText("暂停")
	t.stopBtn.SetEnabled(true)
	t.intervalEdit.SetEnabled(false)
}

func (t *cookTab) setPausedUI() {
	t.runGroup.SetTitle("运行: 已暂停")
	t.pauseBtn.SetText("继续")
}
