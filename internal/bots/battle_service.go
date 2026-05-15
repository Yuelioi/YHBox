package bots

import (
	"context"
	"errors"
	"fmt"
	"sync"

	"github.com/lxn/win"
	"github.com/rs/zerolog"

	"yhbox/internal/hotkey"
	"yhbox/internal/services"

	"yhbox/tools/battle"
)

// modCombo 一种合法的修饰键组合。
type modCombo struct {
	Label string
	Mods  uint32
}

var modCombos = []modCombo{
	{"Ctrl", hotkey.MOD_CONTROL},
	{"Ctrl+Shift", hotkey.MOD_CONTROL | hotkey.MOD_SHIFT},
	{"Ctrl+Alt", hotkey.MOD_CONTROL | hotkey.MOD_ALT},
	{"Shift+Alt", hotkey.MOD_SHIFT | hotkey.MOD_ALT},
	{"Ctrl+Shift+Alt", hotkey.MOD_CONTROL | hotkey.MOD_SHIFT | hotkey.MOD_ALT},
}

func modsByLabel(label string) uint32 {
	for _, m := range modCombos {
		if m.Label == label {
			return m.Mods
		}
	}
	return hotkey.MOD_CONTROL | hotkey.MOD_SHIFT
}

// BattleService 队伍切换全局热键。
//
// 跟 fish/cook/piano 不同：不用 botRun（没有长跑 worker）。
// hotkey.HotkeyManager 跑在自己锁定的 OS 线程，由 Enable/Disable 控制。
// onHotkeyFired 在 hotkey 线程被调，派发 goroutine 跑实际 SwitchTeam，
// 用 switchMu.TryLock 单流防重入。
type BattleService struct {
	app *services.App
	mu  sync.Mutex // 串行 Enable/Disable/SetMods
	mgr *hotkey.HotkeyManager
	det *battle.Detector

	switchMu sync.Mutex // 一次只允许一个 SwitchTeam 跑，连按多键自动丢

	cancelMu     sync.Mutex
	cancelActive context.CancelFunc
}

// NewBattleService 接受外部 hotkey.HotkeyManager。Win32 RegisterHotKey 是 process-wide
// 的（hWnd=NULL 时跟线程绑定），全 app 必须共享同一个 manager，battle/action/recorder
// 等所有 hotkey-owner 都注册到这一个实例上。main.go 负责构造 + 传入。
func NewBattleService(app *services.App, mgr *hotkey.HotkeyManager) *BattleService {
	bs := &BattleService{
		app: app,
		mgr: mgr,
	}
	// 加载 Detector — 失败也允许构造，Enable 时再报错
	if det, err := battle.NewDetector(); err == nil {
		bs.det = det
	} else {
		l := app.RootLogger()
		l.Warn().Err(err).Str("tag", "SYSTEM").Msg("battle Detector 加载失败")
	}
	return bs
}

// Enable 注册 1~6 数字键的全局热键。失败时 mgr 不变（hotkey 线程不启动）。
func (s *BattleService) Enable() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.det == nil {
		return errors.New("Battle 模板未加载")
	}
	if s.mgr.IsRunning() {
		return errors.New("热键已启用")
	}

	settings := s.app.Settings()
	modsLabel := settings.UI.Battle.HotkeyMods
	mods := modsByLabel(modsLabel) | hotkey.MOD_NOREPEAT

	specs := make([]hotkey.HotkeySpec, battle.TeamCount)
	for i := 0; i < battle.TeamCount; i++ {
		vk := hotkey.VK_1 + uint32(i)
		specs[i] = hotkey.HotkeySpec{
			ID:   i + 1,
			Mods: mods,
			VK:   vk,
			Name: fmt.Sprintf("%s+%d", modsLabel, i+1),
		}
	}

	if err := s.mgr.Start(specs, s.onHotkeyFired); err != nil {
		return err
	}

	// 持久化 HotkeyEnabled=true
	s.app.UpdateSettings(func(st *services.Settings) { st.UI.Battle.HotkeyEnabled = true })
	_ = s.app.SaveSettings()

	s.app.Emit(services.EventBattleState, services.BattleStateEvent{Enabled: true, Mods: modsLabel})
	return nil
}

// Disable 反注册热键。打断正在跑的 SwitchTeam。
// 用户在 UI 主动关闭走这里：持久化 HotkeyEnabled=false。
func (s *BattleService) Disable() {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.cancelRunning()
	if s.mgr.IsRunning() {
		s.mgr.Stop()
	}

	var mods string
	s.app.UpdateSettings(func(st *services.Settings) {
		st.UI.Battle.HotkeyEnabled = false
		mods = st.UI.Battle.HotkeyMods
	})
	_ = s.app.SaveSettings()

	s.app.Emit(services.EventBattleState, services.BattleStateEvent{Enabled: false, Mods: mods})
}

// ShutdownUnregister 应用退出时调，仅反注册热键，**不**改 settings。
// 这样下次启动 AutoStartFromSettings 能凭 HotkeyEnabled=true 自动恢复。
func (s *BattleService) ShutdownUnregister() {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.cancelRunning()
	if s.mgr.IsRunning() {
		s.mgr.Stop()
	}
}

// SetMods 切修饰键组合。
//   - 未启用：仅写 settings
//   - 已启用：先打断 SwitchTeam → Stop 旧热键 → 试 Start 新组合
//     注册失败 → 回滚 settings.HotkeyMods 到旧值
func (s *BattleService) SetMods(label string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if !services.ValidHotkeyMods(label) {
		return fmt.Errorf("invalid mods label: %q", label)
	}

	oldLabel := s.app.Settings().UI.Battle.HotkeyMods

	if label == oldLabel {
		return nil
	}

	wasRunning := s.mgr.IsRunning()
	if !wasRunning {
		s.app.UpdateSettings(func(st *services.Settings) { st.UI.Battle.HotkeyMods = label })
		_ = s.app.SaveSettings()
		s.app.Emit(services.EventBattleState, services.BattleStateEvent{Enabled: false, Mods: label})
		return nil
	}

	// 已启用：cancel → stop → 尝试新 mods
	s.cancelRunning()
	s.mgr.Stop()

	s.app.UpdateSettings(func(st *services.Settings) { st.UI.Battle.HotkeyMods = label })

	if err := s.startHotkeysLocked(); err != nil {
		// 回滚
		s.app.UpdateSettings(func(st *services.Settings) { st.UI.Battle.HotkeyMods = oldLabel })
		_ = s.app.SaveSettings()
		s.app.Emit(services.EventBattleState, services.BattleStateEvent{Enabled: false, Mods: oldLabel})
		return fmt.Errorf("注册新修饰键 %s 失败（已回滚 %s）: %w", label, oldLabel, err)
	}
	_ = s.app.SaveSettings()
	s.app.Emit(services.EventBattleState, services.BattleStateEvent{Enabled: true, Mods: label})
	return nil
}

// startHotkeysLocked s.mu 已持时调，注册热键。
func (s *BattleService) startHotkeysLocked() error {
	modsLabel := s.app.Settings().UI.Battle.HotkeyMods
	mods := modsByLabel(modsLabel) | hotkey.MOD_NOREPEAT

	specs := make([]hotkey.HotkeySpec, battle.TeamCount)
	for i := 0; i < battle.TeamCount; i++ {
		specs[i] = hotkey.HotkeySpec{
			ID:   i + 1,
			Mods: mods,
			VK:   hotkey.VK_1 + uint32(i),
			Name: fmt.Sprintf("%s+%d", modsLabel, i+1),
		}
	}
	return s.mgr.Start(specs, s.onHotkeyFired)
}

// onHotkeyFired 在 hotkey 线程被调，必须尽快返回 — 派发 goroutine 跑实际切换。
func (s *BattleService) onHotkeyFired(id int) {
	if id < 1 || id > battle.TeamCount {
		return
	}
	target := id

	botLogger := zerolog.New(s.app.GetLogSink()).With().
		Timestamp().Str("bot", "battle").Logger()

	go func() {
		if !s.switchMu.TryLock() {
			botLogger.Warn().Str("tag", "SYSTEM").Msgf("队伍切换正在进行，忽略快捷键 -> %d", target)
			return
		}
		defer s.switchMu.Unlock()

		game := s.app.Game()
		if game == nil || !game.OK {
			botLogger.Warn().Str("tag", "SYSTEM").Msgf("快捷键 -> %d 触发：未检测到异环窗口，跳过", target)
			return
		}

		ctx, cancel := context.WithCancel(context.Background())
		s.cancelMu.Lock()
		s.cancelActive = cancel
		s.cancelMu.Unlock()
		defer func() {
			s.cancelMu.Lock()
			s.cancelActive = nil
			s.cancelMu.Unlock()
			cancel()
		}()

		hwnd := win.HWND(uintptr(game.HWND))
		if err := battle.SwitchTeam(ctx, hwnd, s.det, botLogger, target); err != nil {
			botLogger.Error().Err(err).Str("tag", "SYSTEM").Msgf("切换队伍 %d 失败", target)
		}
	}()
}

// cancelRunning 打断正在跑的 SwitchTeam（关热键/切修饰键/Shutdown 时调）。
func (s *BattleService) cancelRunning() {
	s.cancelMu.Lock()
	cancel := s.cancelActive
	s.cancelMu.Unlock()
	if cancel != nil {
		cancel()
	}
}

// AutoStartFromSettings 应用启动后，若 settings 记录热键已启用则自动恢复。
func (s *BattleService) AutoStartFromSettings() {
	want := s.app.Settings().UI.Battle.HotkeyEnabled
	if !want {
		return
	}
	if err := s.Enable(); err != nil {
		l := s.app.RootLogger()
		l.Warn().Err(err).Str("tag", "SYSTEM").Msg("Battle 热键自动启动失败")
	}
}
