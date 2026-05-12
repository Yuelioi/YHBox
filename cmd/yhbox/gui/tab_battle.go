package gui

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"

	"github.com/lxn/walk"
	. "github.com/lxn/walk/declarative"

	"yhbox/tools/battle"
)

// battleTab 战斗 tab。当前只有「队伍快捷键」一个功能：注册全局热键切换上阵队伍。
// 数字键固定 1~6，修饰键组合用户可选（Ctrl / Ctrl+Shift / Ctrl+Alt / Shift+Alt /
// Ctrl+Shift+Alt）。启动状态 + 选中的组合都持久化到 settings.json，下次启动自动恢复。
type battleTab struct {
	app *App

	modsCombo   *walk.ComboBox
	hotkeyBtn   *walk.PushButton
	hotkeyLabel *walk.Label

	det      *battle.Detector
	mgr      *HotkeyManager
	enabled  atomic.Bool // GUI 状态 = mgr.IsRunning，但 atomic 给非 UI 线程读
	switchMu sync.Mutex  // 一次只允许一个 SwitchTeam 跑，连按多个键自动丢弃后续

	// 正在跑的 SwitchTeam 的 cancel 句柄。关窗口 / 关热键 / 切修饰键时取消正在跑的那次，
	// 不让残留 goroutine 在用户已经"看见"关闭后继续朝游戏窗口发输入。
	cancelMu     sync.Mutex
	cancelActive context.CancelFunc
}

// modCombo 一项修饰键组合。Label 既作 ComboBox 显示字串又作 settings.json 持久值。
type modCombo struct {
	Label string
	Mods  uint32
}

// modCombos 5 种合法的修饰键组合。settings.json 存 Label，启动时映射回 Mods。
// 顺序就是 ComboBox 显示顺序。新增组合在这里加一行即可。
var modCombos = []modCombo{
	{"Ctrl", winMOD_CONTROL},
	{"Ctrl+Shift", winMOD_CONTROL | winMOD_SHIFT},
	{"Ctrl+Alt", winMOD_CONTROL | winMOD_ALT},
	{"Shift+Alt", winMOD_SHIFT | winMOD_ALT},
	{"Ctrl+Shift+Alt", winMOD_CONTROL | winMOD_SHIFT | winMOD_ALT},
}

// modsByLabel 反查表，启动 / settings 切换组合时用。未知 label 走默认 Ctrl+Shift。
func modsByLabel(label string) uint32 {
	for _, m := range modCombos {
		if m.Label == label {
			return m.Mods
		}
	}
	return winMOD_CONTROL | winMOD_SHIFT // fallback
}

// modLabels 给 ComboBox.Model 用的字符串数组。
func modLabels() []string {
	out := make([]string, len(modCombos))
	for i, m := range modCombos {
		out[i] = m.Label
	}
	return out
}

// modLabelIndex 返回 label 在 modCombos 里的下标；未知返回 1 (Ctrl+Shift)。
func modLabelIndex(label string) int {
	for i, m := range modCombos {
		if m.Label == label {
			return i
		}
	}
	return 1
}

// buildBattleTab 构造战斗 tab。Detector 在这里一次性加载，所有热键触发复用。
// 加载失败 → tab 仍构造但启动按钮 disabled + 状态 label 提示。
func buildBattleTab(app *App) (*battleTab, TabPage) {
	t := &battleTab{app: app}
	t.mgr = NewHotkeyManager()

	det, err := battle.NewDetector()
	if err != nil {
		app.appendInternalLog("WARN: battle Detector 加载失败: " + err.Error())
	} else {
		t.det = det
	}

	// LoadSettings 已归一过空 HotkeyMods → "Ctrl+Shift"，这里直接读
	curMods := app.settings.UI.Battle.HotkeyMods

	page := TabPage{
		Title:  "战斗",
		Layout: VBox{},
		Children: []Widget{
			GroupBox{
				Title:  "队伍快捷键",
				Layout: VBox{Alignment: AlignHNearVNear},
				Children: []Widget{
					Label{Text: "全局热键，数字键 1~6 切换到对应队伍"},
					Label{Text: "需要游戏在前台主界面（无弹窗占用 L 键），切换流程约 3~4 秒"},
					Composite{
						Layout: HBox{MarginsZero: true},
						Children: []Widget{
							Label{Text: "修饰键："},
							ComboBox{
								AssignTo:              &t.modsCombo,
								Model:                 modLabels(),
								CurrentIndex:          modLabelIndex(curMods),
								OnCurrentIndexChanged: t.onModsChanged,
								MinSize:               Size{Width: 140},
							},
							Label{Text: " + 1~6"},
							HSpacer{},
						},
					},
					Composite{
						Layout: HBox{MarginsZero: true},
						Children: []Widget{
							PushButton{
								AssignTo:  &t.hotkeyBtn,
								Text:      "启动",
								OnClicked: t.onToggle,
								MinSize:   Size{Width: 80},
							},
							Label{AssignTo: &t.hotkeyLabel, Text: "状态: 已关闭"},
							HSpacer{},
						},
					},
					HSpacer{},
				},
			},
			VSpacer{},
		},
	}
	return t, page
}

// AutoStartFromSettings 启动时由 app 调，如果 settings 里 HotkeyEnabled=true 就自动注册热键。
// 失败时静默落到关闭态 + 日志提示，不弹窗（启动期弹窗用户体验差）。
//
// 错误分类：
//   - Detector 未加载（模板缺失）= 永久错误 → 写回 HotkeyEnabled=false，不然每次启动都报警
//   - 热键被占用 = 临时错误 → 不动 settings，下次启动重试
func (t *battleTab) AutoStartFromSettings() {
	if !t.app.settings.UI.Battle.HotkeyEnabled {
		return
	}
	if t.det == nil {
		// 永久错误：模板缺失不会自己修好
		t.app.settings.UI.Battle.HotkeyEnabled = false
		t.app.SaveSettings()
		t.app.appendInternalLog("WARN: Battle 模板未加载，自动启动取消（HotkeyEnabled 已重置）")
		return
	}
	if err := t.startHotkeys(); err != nil {
		t.app.appendInternalLog("WARN: 自动启动队伍快捷键失败: " + err.Error())
		// 临时错误（被占用）不改 settings，下次启动还会再试
	}
}

// startHotkeys 按当前 ComboBox 选中的 mods 注册 6 个数字键。
// 调用方负责检查 IsRunning。失败原样返回 err（含具体哪个组合被占）。
func (t *battleTab) startHotkeys() error {
	if t.det == nil {
		return fmt.Errorf("Battle 模板未加载")
	}
	mods := modsByLabel(t.app.settings.UI.Battle.HotkeyMods) | winMOD_NOREPEAT
	specs := make([]HotkeySpec, battle.TeamCount)
	for i := range battle.TeamCount {
		vk := uint32(winVK_1 + i) // 0x31..0x36
		specs[i] = HotkeySpec{
			ID:   i + 1,
			Mods: mods,
			VK:   vk,
			Name: fmt.Sprintf("%s+%d", t.app.settings.UI.Battle.HotkeyMods, i+1),
		}
	}
	if err := t.mgr.Start(specs, t.onHotkeyFired); err != nil {
		return err
	}
	t.enabled.Store(true)
	t.refreshUI()
	return nil
}

// onToggle 启动/关闭按钮回调。
func (t *battleTab) onToggle() {
	if t.mgr.IsRunning() {
		t.cancelRunning() // 先打断正在跑的 SwitchTeam，不让残留输入继续敲游戏
		t.mgr.Stop()
		t.enabled.Store(false)
		t.refreshUI()
		t.app.settings.UI.Battle.HotkeyEnabled = false
		t.app.SaveSettings()
		t.app.appendInternalLog("已关闭队伍快捷键")
		return
	}
	if err := t.startHotkeys(); err != nil {
		walk.MsgBox(t.app.mw, "YHBox", "启动热键失败: "+err.Error(), walk.MsgBoxIconError)
		t.app.appendInternalLog("WARN: 启动队伍快捷键失败: " + err.Error())
		return
	}
	t.app.settings.UI.Battle.HotkeyEnabled = true
	t.app.SaveSettings()
	t.app.appendInternalLog(fmt.Sprintf("已启用队伍快捷键 %s+1~6",
		t.app.settings.UI.Battle.HotkeyMods))
}

// onModsChanged 用户切修饰键组合。
//   - 未启用：仅写 settings，下次启动按新组合注册
//   - 已启用：cancel 正在跑的切换 → Stop 旧热键 → 试 Start 新组合
//     重注册失败 → 回滚 settings.HotkeyMods 到旧值 + 回滚 ComboBox 选项 + 不写 enabled
//     避免出现「settings 写了新 mods + enabled=false」的半截状态
func (t *battleTab) onModsChanged() {
	idx := t.modsCombo.CurrentIndex()
	if idx < 0 || idx >= len(modCombos) {
		return
	}
	newLabel := modCombos[idx].Label
	oldLabel := t.app.settings.UI.Battle.HotkeyMods
	if newLabel == oldLabel {
		return
	}

	if !t.mgr.IsRunning() {
		// 未启用：直接存
		t.app.settings.UI.Battle.HotkeyMods = newLabel
		t.app.SaveSettings()
		t.app.appendInternalLog("修饰键改为 " + newLabel + "（下次启动生效）")
		return
	}

	// 已启用：先尝试用新 mods 注册，成功了才写 settings
	t.cancelRunning()
	t.mgr.Stop()
	t.enabled.Store(false)
	t.app.settings.UI.Battle.HotkeyMods = newLabel // startHotkeys 会读这个
	if err := t.startHotkeys(); err != nil {
		// 回滚：mods 复原 + ComboBox 复原 + 不动 HotkeyEnabled（仍是 true，但当前没注册）
		t.app.settings.UI.Battle.HotkeyMods = oldLabel
		t.app.SaveSettings()
		t.modsCombo.SetCurrentIndex(modLabelIndex(oldLabel))
		walk.MsgBox(t.app.mw, "YHBox",
			"切换到 "+newLabel+" 失败: "+err.Error()+"\n已回滚到 "+oldLabel+"，但热键当前未运行，请重新启动。",
			walk.MsgBoxIconWarning)
		t.refreshUI()
		return
	}
	t.app.SaveSettings()
	t.app.appendInternalLog(fmt.Sprintf("修饰键改为 %s，已重注册", newLabel))
}

// onHotkeyFired 在 hotkey 线程被调，必须尽快返回 —— 派发到 goroutine 跑实际切换。
// switchMu.TryLock 保证一次只有一个切换在跑；重复按键被丢（用户能从日志看到）。
// cancelActive 暴露 cancel 句柄给 cancelRunning() 用，关窗口/关热键时能打断正在跑的那次。
func (t *battleTab) onHotkeyFired(id int) {
	if id < 1 || id > battle.TeamCount {
		return
	}
	target := id
	go func() {
		if !t.switchMu.TryLock() {
			t.app.appendInternalLog(fmt.Sprintf("队伍切换正在进行，忽略快捷键 -> %d", target))
			return
		}
		defer t.switchMu.Unlock()

		if !t.app.game.OK {
			t.app.appendInternalLog(fmt.Sprintf("快捷键 -> %d 触发：未检测到异环窗口，跳过", target))
			return
		}
		ctx, cancel := context.WithCancel(context.Background())
		t.cancelMu.Lock()
		t.cancelActive = cancel
		t.cancelMu.Unlock()
		defer func() {
			t.cancelMu.Lock()
			t.cancelActive = nil
			t.cancelMu.Unlock()
			cancel()
		}()
		if err := battle.SwitchTeam(ctx, t.app.game.HWND, t.det, t.app.logger, target); err != nil {
			t.app.appendInternalLog(fmt.Sprintf("切换队伍 %d 失败: %v", target, err))
		}
	}()
}

// cancelRunning 打断正在跑的 SwitchTeam（如果有）。Shutdown/onToggle(关闭)/
// onModsChanged 用：用户已经"决定停了"就不该让残留输入继续敲游戏。
func (t *battleTab) cancelRunning() {
	t.cancelMu.Lock()
	cancel := t.cancelActive
	t.cancelMu.Unlock()
	if cancel != nil {
		cancel()
	}
}

// refreshUI 必须在 UI 线程调。Synchronize 包一下给非 UI 线程用。
func (t *battleTab) refreshUI() {
	if t.app.mw == nil {
		return
	}
	t.app.mw.Synchronize(func() {
		if t.hotkeyBtn == nil || t.hotkeyLabel == nil {
			return
		}
		mods := t.app.settings.UI.Battle.HotkeyMods
		if t.enabled.Load() {
			t.hotkeyBtn.SetText("关闭")
			t.hotkeyLabel.SetText(fmt.Sprintf("状态: 已启用（%s+1~6）", mods))
		} else {
			t.hotkeyBtn.SetText("启动")
			t.hotkeyLabel.SetText("状态: 已关闭")
		}
	})
}

// Shutdown 关窗口前调。先 cancel 正在跑的 SwitchTeam（不让残留输入打游戏），
// 再 mgr.Stop() 反注册全局热键（Stop 阻塞到 hotkey 线程退出，hWnd=NULL 时
// UnregisterHotKey 必须在注册线程上跑才有效）。
func (t *battleTab) Shutdown() {
	t.cancelRunning()
	t.mgr.Stop()
}
