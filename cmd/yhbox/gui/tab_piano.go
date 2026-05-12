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
	"yhbox/tools/piano"
)

type pianoTab struct {
	app *App

	libraryCombo     *walk.ComboBox
	pathEdit         *walk.LineEdit
	browseBtn        *walk.PushButton
	modeCombo        *walk.ComboBox
	trackCombo       *walk.ComboBox
	autoTransposeBox *walk.CheckBox
	melodyOnlyBox    *walk.CheckBox
	offsetEdit       *walk.NumberEdit
	progSlider       *walk.Slider
	progressText     *walk.Label
	runGroup         *walk.GroupBox
	startBtn         *walk.PushButton
	pauseBtn         *walk.PushButton
	stopBtn          *walk.PushButton

	game    GameStatus
	ctrl    *guiControl
	logFile *os.File
	running atomic.Bool

	playerMu    sync.Mutex
	player      *piano.Player
	totalMs     atomic.Int64 // 当前曲长
	progSetting atomic.Bool  // 程序式更新 slider 时屏蔽 OnValueChanged

	library      []piano.LibraryEntry // 内置曲库快照
	builtinAsset string               // 选中的内置曲；非空时优先于 pathEdit
	midiData     *piano.MIDIData      // 当前 MIDI 解析结果（选曲后即时 parse，给 trackCombo 用）
}

func buildPianoTab(app *App) (*pianoTab, TabPage) {
	t := &pianoTab{app: app, library: piano.BuiltinLibrary()}

	page := TabPage{
		Title:  "弹琴",
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
				Title:  "MIDI 曲谱",
				Layout: Grid{Columns: 2},
				Children: []Widget{
					Label{Text: "内置:"},
					ComboBox{
						AssignTo:              &t.libraryCombo,
						Model:                 t.libraryModel(),
						OnCurrentIndexChanged: t.onLibrarySelect,
					},
					Label{Text: "文件:"},
					Composite{
						Layout: HBox{MarginsZero: true},
						Children: []Widget{
							LineEdit{
								AssignTo:    &t.pathEdit,
								ReadOnly:    true,
								Text:        app.settings.Piano.LastMidiPath,
								ToolTipText: app.settings.Piano.LastMidiPath,
							},
							PushButton{AssignTo: &t.browseBtn, Text: "浏览...", OnClicked: t.onBrowse},
						},
					},
				},
			},
			// 4 列 Grid: Col0 标签 / Col1 ComboBox (两行等宽自适应) / Col2 / Col3 固定项。
			// Grid 强制同列同宽 → 键位 ComboBox 跟选轨 ComboBox 永远对齐, 右侧
			// "微调/复选框" 也跟着对齐。
			GroupBox{
				Title:  "配置",
				Layout: Grid{Columns: 4},
				Children: []Widget{
					// Row 0
					Label{Text: "键位:"},
					ComboBox{
						AssignTo:              &t.modeCombo,
						Model:                 []string{"36键 (含半音)", "21键 (仅自然音)"},
						CurrentIndex:          app.settings.Piano.Mode,
						OnCurrentIndexChanged: t.onModeChanged,
					},
					Label{Text: "微调:"},
					NumberEdit{
						AssignTo:       &t.offsetEdit,
						Value:          float64(app.settings.Piano.OctaveOffset),
						MinValue:       -2,
						MaxValue:       2,
						Decimals:       0,
						MinSize:        Size{Width: 80},
						MaxSize:        Size{Width: 80},
						OnValueChanged: t.onOffsetChanged,
					},
					// Row 1
					Label{Text: "选轨:"},
					ComboBox{
						AssignTo:              &t.trackCombo,
						Model:                 []string{"智能", "全部"},
						CurrentIndex:          0,
						OnCurrentIndexChanged: t.onTrackPickChanged,
					},
					CheckBox{
						AssignTo:         &t.autoTransposeBox,
						Text:             "自动对齐音域",
						Checked:          app.settings.Piano.AutoTranspose,
						OnCheckedChanged: t.onAutoTransposeChanged,
					},
					CheckBox{
						AssignTo:         &t.melodyOnlyBox,
						Text:             "只弹主旋律",
						Checked:          app.settings.Piano.MelodyOnly,
						OnCheckedChanged: t.onMelodyOnlyChanged,
					},
				},
			},
			GroupBox{
				Title:  "进度",
				Layout: VBox{},
				Children: []Widget{
					Slider{
						AssignTo:       &t.progSlider,
						MinValue:       0,
						MaxValue:       1000,
						Tracking:       false, // 拖动结束才触发 OnValueChanged，避免拖动过程频繁 seek
						OnValueChanged: t.onSeek,
					},
					Label{AssignTo: &t.progressText, Text: "—"},
				},
			},
			VSpacer{},
		},
	}
	return t, page
}

func (t *pianoTab) IsRunning() bool { return t.running.Load() }

func (t *pianoTab) SetStartEnabled(enabled bool) {
	if t.startBtn != nil {
		hasMidi := t.builtinAsset != "" || t.app.settings.Piano.LastMidiPath != ""
		ok := enabled && !t.running.Load() && t.game.OK && hasMidi
		t.startBtn.SetEnabled(ok)
	}
}

func (t *pianoTab) RefreshGameStatus(g GameStatus) {
	t.game = g
	t.SetStartEnabled(true)
}

// libraryModel 给 ComboBox.Model 用的字符串列表。第 0 项是占位符。
func (t *pianoTab) libraryModel() []string {
	out := make([]string, 0, len(t.library)+1)
	out = append(out, "(选一首内置曲)")
	for _, e := range t.library {
		out = append(out, e.Display)
	}
	return out
}

// onLibrarySelect ComboBox 切换：第 0 项 = 取消内置选择；其它项 = 选中对应内置曲。
// 跟外部文件路径互斥，所以选中内置时清空 pathEdit 显示。
func (t *pianoTab) onLibrarySelect() {
	idx := t.libraryCombo.CurrentIndex()
	if idx <= 0 {
		t.builtinAsset = ""
		t.midiData = nil
		t.refreshTrackCombo()
		t.SetStartEnabled(true)
		return
	}
	t.builtinAsset = t.library[idx-1].Asset
	t.pathEdit.SetText("(内置) " + t.library[idx-1].Display)
	t.pathEdit.SetToolTipText(t.builtinAsset)
	t.preloadMIDI()
	t.SetStartEnabled(true)
}

func (t *pianoTab) onBrowse() {
	dlg := walk.FileDialog{
		Filter: "MIDI 文件 (*.mid *.midi)|*.mid;*.midi|所有文件 (*.*)|*.*",
		Title:  "选择 MIDI 曲谱",
	}
	if t.app.settings.Piano.LastMidiPath != "" {
		dlg.InitialDirPath = filepath.Dir(t.app.settings.Piano.LastMidiPath)
	}
	ok, err := dlg.ShowOpen(t.app.mw)
	if err != nil || !ok {
		return
	}
	// 选了外部文件 → 清掉内置选择互斥
	t.builtinAsset = ""
	if t.libraryCombo != nil {
		t.libraryCombo.SetCurrentIndex(0)
	}
	t.app.settings.Piano.LastMidiPath = dlg.FilePath
	t.app.SaveSettings()
	t.pathEdit.SetText(dlg.FilePath)
	t.pathEdit.SetToolTipText(dlg.FilePath)
	t.preloadMIDI()
	t.SetStartEnabled(true)
}

func (t *pianoTab) onModeChanged() {
	t.app.settings.Piano.Mode = t.modeCombo.CurrentIndex()
	t.app.SaveSettings()
}

func (t *pianoTab) onAutoTransposeChanged() {
	t.app.settings.Piano.AutoTranspose = t.autoTransposeBox.Checked()
	t.app.SaveSettings()
}

// onTrackPickChanged ComboBox 项映射到 settings.Piano.TrackPick：
//
//	idx 0  → -1 (智能)
//	idx 1  → -2 (全部)
//	idx 2+ → 0+ (具体 track 索引 = idx - 2)
func (t *pianoTab) onTrackPickChanged() {
	idx := t.trackCombo.CurrentIndex()
	switch idx {
	case 0:
		t.app.settings.Piano.TrackPick = piano.TrackPickSmart
	case 1:
		t.app.settings.Piano.TrackPick = piano.TrackPickAll
	default:
		t.app.settings.Piano.TrackPick = idx - 2
	}
	t.app.SaveSettings()
}

// refreshTrackCombo 根据当前 midiData 刷新轨道下拉。前两项固定为智能/全部，
// 后面跟着每条 track 的描述。保留用户先前选的 TrackPick 设置（若仍有效）。
func (t *pianoTab) refreshTrackCombo() {
	items := []string{"智能", "全部"}
	if t.midiData != nil {
		for i, tr := range t.midiData.Tracks {
			name := tr.Name
			if name == "" {
				name = "未命名"
			}
			items = append(items, fmt.Sprintf("Track %d: %s (%d 音符)", i, name, tr.NoteCount))
		}
	}
	t.trackCombo.SetModel(items)
	// 还原选择
	pick := t.app.settings.Piano.TrackPick
	var idx int
	switch pick {
	case piano.TrackPickSmart:
		idx = 0
	case piano.TrackPickAll:
		idx = 1
	default:
		if pick >= 0 && pick+2 < len(items) {
			idx = pick + 2
		} else {
			idx = 0 // 越界则回到智能
		}
	}
	t.trackCombo.SetCurrentIndex(idx)
}

// preloadMIDI 选 MIDI 时立即 parse，缓存 midiData 并刷新 trackCombo。
func (t *pianoTab) preloadMIDI() {
	var data *piano.MIDIData
	var err error
	if t.builtinAsset != "" {
		data, err = piano.LoadEmbeddedMIDI(t.builtinAsset)
	} else if t.app.settings.Piano.LastMidiPath != "" {
		data, err = piano.LoadMIDI(t.app.settings.Piano.LastMidiPath)
	}
	if err != nil {
		t.app.appendInternalLog("WARN: MIDI 预解析失败: " + err.Error())
		t.midiData = nil
	} else {
		t.midiData = data
	}
	t.refreshTrackCombo()
}

func (t *pianoTab) onMelodyOnlyChanged() {
	t.app.settings.Piano.MelodyOnly = t.melodyOnlyBox.Checked()
	t.app.SaveSettings()
}

func (t *pianoTab) onOffsetChanged() {
	t.app.settings.Piano.OctaveOffset = int(t.offsetEdit.Value())
	t.app.SaveSettings()
}

func (t *pianoTab) onStart() {
	t.app.detectAndPropagate() // safety check：游戏可能在切 tab 之后被关
	if !t.game.OK {
		walk.MsgBox(t.app.mw, "YHBox", "未检测到异环窗口", walk.MsgBoxIconWarning)
		return
	}
	builtin := t.builtinAsset
	midiPath := t.app.settings.Piano.LastMidiPath
	if builtin == "" && midiPath == "" {
		walk.MsgBox(t.app.mw, "YHBox", "请先选内置曲或浏览 MIDI 文件", walk.MsgBoxIconWarning)
		return
	}
	if !t.app.AcquireBot("piano") {
		return
	}

	cfg := piano.DefaultConfig()
	cfg.Mode = piano.Mode(t.app.settings.Piano.Mode)
	cfg.TrackPick = t.app.settings.Piano.TrackPick
	cfg.AutoTranspose = t.app.settings.Piano.AutoTranspose
	cfg.OctaveOffset = t.app.settings.Piano.OctaveOffset
	cfg.MelodyOnly = t.app.settings.Piano.MelodyOnly

	if err := os.MkdirAll("logs", 0755); err != nil {
		t.app.appendInternalLog("WARN: 创建 logs 目录失败: " + err.Error())
	}
	logPath := filepath.Join("logs", "yh_piano_"+time.Now().Format("20060102_150405")+".log")
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

	ctx, cancel := context.WithCancel(context.Background())
	go t.runBot(ctx, cancel, cfg, builtin, midiPath, logger)
}

func (t *pianoTab) runBot(ctx context.Context, cancel context.CancelFunc, cfg *piano.Config, builtinAsset, midiPath string, logger *log.Logger) {
	defer cancel()
	defer t.afterBot()

	// 重置进度状态，防上次成功播放残留 totalMs 影响这次 seek 计算
	t.totalMs.Store(0)

	var data *piano.MIDIData
	var err error
	var source string
	if builtinAsset != "" {
		data, err = piano.LoadEmbeddedMIDI(builtinAsset)
		source = "(内置) " + builtinAsset
	} else {
		data, err = piano.LoadMIDI(midiPath)
		source = midiPath
	}
	if err != nil {
		logger.Log(log.SYSTEM, "MIDI 加载失败: %v", err)
		return
	}

	var events []piano.NoteEvent
	var pickDesc string
	switch p := cfg.TrackPick; {
	case p == piano.TrackPickSmart:
		events = data.SmartEvents()
		pickDesc = "智能选轨"
	case p == piano.TrackPickAll:
		events = data.AllEvents()
		pickDesc = "全部 track"
	default:
		events = data.TrackEvents(p)
		if events == nil { // 越界 → 退化到智能
			events = data.SmartEvents()
			pickDesc = fmt.Sprintf("Track %d 越界，回退智能", p)
		} else {
			pickDesc = fmt.Sprintf("Track %d", p)
		}
	}
	logger.Log(log.SYSTEM, "MIDI 已加载: %s (%s: %d 音符 / 总 %d track)",
		source, pickDesc, len(events), len(data.Tracks))

	if len(events) > 0 {
		t.totalMs.Store(events[len(events)-1].At.Milliseconds())
	}

	p := piano.NewPlayer(t.game.HWND, cfg, logger, t.ctrl)
	t.playerMu.Lock()
	t.player = p
	t.playerMu.Unlock()
	p.Play(ctx, events, t.onProgress)

	t.playerMu.Lock()
	t.player = nil
	t.playerMu.Unlock()
}

func (t *pianoTab) onProgress(curMs, totalMs int64) {
	if t.app.mw == nil || t.app.shuttingDown.Load() {
		return
	}
	var pct int
	if totalMs > 0 {
		pct = int(curMs * 1000 / totalMs)
	}
	text := fmt.Sprintf("%s / %s", formatMillis(curMs), formatMillis(totalMs))
	t.app.mw.Synchronize(func() {
		if t.progSlider != nil {
			t.progSetting.Store(true)
			t.progSlider.SetValue(pct)
			t.progSetting.Store(false)
		}
		if t.progressText != nil {
			t.progressText.SetText(text)
		}
	})
}

// onSeek 用户拖动 Slider 释放后触发。程序式 SetValue 不应触发 seek，用 progSetting 标志屏蔽。
func (t *pianoTab) onSeek() {
	if t.progSetting.Load() {
		return
	}
	t.playerMu.Lock()
	p := t.player
	t.playerMu.Unlock()
	if p == nil {
		return
	}
	totalMs := t.totalMs.Load()
	if totalMs <= 0 {
		return
	}
	pct := t.progSlider.Value() // 0..1000
	target := int64(pct) * totalMs / 1000
	p.Seek(target)
}

func formatMillis(ms int64) string {
	if ms < 0 {
		ms = 0
	}
	totalSec := ms / 1000
	return fmt.Sprintf("%d:%02d", totalSec/60, totalSec%60)
}

func (t *pianoTab) afterBot() {
	if t.logFile != nil {
		t.logFile.Close()
		t.logFile = nil
	}
	t.running.Store(false)
	t.app.ReleaseBot()
	if t.app.mw != nil {
		t.app.mw.Synchronize(func() { t.setIdleUI() })
	}
}

func (t *pianoTab) onPause() {
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

func (t *pianoTab) onStop() {
	if t.ctrl == nil {
		return
	}
	t.ctrl.Stop()
	t.stopBtn.SetEnabled(false)
}

func (t *pianoTab) setIdleUI() {
	t.runGroup.SetTitle("运行: 空闲")
	t.SetStartEnabled(true)
	t.pauseBtn.SetEnabled(false)
	t.pauseBtn.SetText("暂停")
	t.stopBtn.SetEnabled(false)
	t.libraryCombo.SetEnabled(true)
	t.browseBtn.SetEnabled(true)
	t.modeCombo.SetEnabled(true)
	t.trackCombo.SetEnabled(true)
	t.autoTransposeBox.SetEnabled(true)
	t.melodyOnlyBox.SetEnabled(true)
	t.offsetEdit.SetEnabled(true)
}

func (t *pianoTab) setRunningUI() {
	t.runGroup.SetTitle("运行: 演奏中")
	t.startBtn.SetEnabled(false)
	t.pauseBtn.SetEnabled(true)
	t.pauseBtn.SetText("暂停")
	t.stopBtn.SetEnabled(true)
	t.libraryCombo.SetEnabled(false)
	t.browseBtn.SetEnabled(false)
	t.modeCombo.SetEnabled(false)
	t.trackCombo.SetEnabled(false)
	t.autoTransposeBox.SetEnabled(false)
	t.melodyOnlyBox.SetEnabled(false)
	t.offsetEdit.SetEnabled(false)
}

func (t *pianoTab) setPausedUI() {
	t.runGroup.SetTitle("运行: 已暂停")
	t.pauseBtn.SetText("继续")
}
