package bots

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/lxn/win"
	"github.com/rs/zerolog"
	"github.com/wailsapp/wails/v3/pkg/application"

	"yhbox/tools/piano"
	"yhbox/internal/services"
)

// pianoSelection 当前选中的 MIDI（运行前用户必须选一个）。
type pianoSelection struct {
	BuiltinAsset string // 非空时表示选了内置曲（互斥）
	ExternalPath string // 非空时表示选了外部文件
	Data         *piano.MIDIData
}

func (sel *pianoSelection) sourceDesc() string {
	if sel.BuiltinAsset != "" {
		return "(内置) " + sel.BuiltinAsset
	}
	return sel.ExternalPath
}

// PianoService Phase 3 完整实现。
type PianoService struct {
	app *services.App
	run *botRun

	selMu sync.RWMutex
	sel   *pianoSelection

	playerMu sync.Mutex
	player   *piano.Player

	totalMs int64 // 当前选中 MIDI 总时长（Start 后填）

	progressMu sync.Mutex
	lastEmit   time.Time
}

// LibraryEntry 跟 piano.LibraryEntry 同 shape — wails3 binding 暴露给 JS 用。
// 注意：tools/piano 的 LibraryEntry 字段大写无 JSON tag，
// 这里重新定义带 JSON tag 让前端拿到 camelCase。
type LibraryEntry struct {
	Asset   string `json:"asset"`
	Display string `json:"display"`
}

// TrackInfo 让前端展示 track 列表（智能/全部/具体 track i）。
type TrackInfo struct {
	Index     int    `json:"index"`
	Name      string `json:"name"`
	NoteCount int    `json:"noteCount"`
}

// MIDIInfo 给前端拿曲长 / 轨道数。LoadFile/LoadBuiltin 返回。
type MIDIInfo struct {
	Path    string      `json:"path"`
	TotalMs int64       `json:"totalMs"`
	Tracks  []TrackInfo `json:"tracks"`
}

func NewPianoService(app *services.App) *PianoService {
	return &PianoService{app: app, run: newBotRun("piano", app)}
}

func init() {
	services.RegisterBot(services.BotMeta{
		Kind: services.KindPiano,
		Construct: func(a *services.App) (services.BotService, application.Service) {
			s := NewPianoService(a)
			return s, application.NewService(s)
		},
	})
}

// GetLibrary 返回内置 MIDI 列表（按文件名排序）。
func (s *PianoService) GetLibrary() []LibraryEntry {
	src := piano.BuiltinLibrary()
	out := make([]LibraryEntry, len(src))
	for i, e := range src {
		out[i] = LibraryEntry{Asset: e.Asset, Display: e.Display}
	}
	return out
}

// LoadBuiltin 加载内置曲。前端选择内置下拉项后调。
func (s *PianoService) LoadBuiltin(asset string) (MIDIInfo, error) {
	if asset == "" {
		return MIDIInfo{}, errors.New("empty asset")
	}
	data, err := piano.LoadEmbeddedMIDI(asset)
	if err != nil {
		return MIDIInfo{}, fmt.Errorf("加载内置 MIDI: %w", err)
	}
	sel := &pianoSelection{BuiltinAsset: asset, Data: data}
	s.selMu.Lock()
	s.sel = sel
	s.selMu.Unlock()

	// 内置曲选择时清空 lastMidiPath（区分外部文件持久化）
	s.app.UpdateSettings(func(st *services.Settings) { st.Piano.LastMidiPath = "" })
	_ = s.app.SaveSettings()

	return midiInfoFromData(sel.BuiltinAsset, data), nil
}

// LoadFile 加载外部 MIDI 文件。path 必填，前端用 <input type="file"> 或手动粘贴路径。
func (s *PianoService) LoadFile(path string) (MIDIInfo, error) {
	if path == "" {
		return MIDIInfo{}, errors.New("path is required (file dialog 暂未实现，请直接传文件路径)")
	}
	data, err := piano.LoadMIDI(path)
	if err != nil {
		return MIDIInfo{}, fmt.Errorf("加载 MIDI 文件: %w", err)
	}
	sel := &pianoSelection{ExternalPath: path, Data: data}
	s.selMu.Lock()
	s.sel = sel
	s.selMu.Unlock()

	s.app.UpdateSettings(func(st *services.Settings) { st.Piano.LastMidiPath = path })
	_ = s.app.SaveSettings()

	return midiInfoFromData(path, data), nil
}

// Start 演奏开始。
func (s *PianoService) Start() error {
	s.run.opMu.Lock()
	defer s.run.opMu.Unlock()

	if s.run.state.Load() != bsIdle {
		return errors.New("piano 已在运行")
	}

	game := s.app.Game()
	if game == nil || !game.OK {
		return errors.New("未检测到异环窗口")
	}

	s.selMu.RLock()
	sel := s.sel
	s.selMu.RUnlock()
	if sel == nil || sel.Data == nil {
		return errors.New("请先加载 MIDI（选内置曲或文件）")
	}

	lease, err := s.app.AcquireBot("piano")
	if err != nil {
		return err
	}

	settings := s.app.Settings()
	cfg := piano.DefaultConfig()
	cfg.Mode = piano.Mode(settings.Piano.Mode)
	cfg.TrackPick = settings.Piano.TrackPick
	cfg.AutoTranspose = settings.Piano.AutoTranspose
	cfg.OctaveOffset = settings.Piano.OctaveOffset
	cfg.MelodyOnly = settings.Piano.MelodyOnly

	// 选事件
	events := pickEvents(sel.Data, cfg.TrackPick)
	if len(events) == 0 {
		lease.Release()
		return errors.New("MIDI 没有可演奏事件")
	}
	s.totalMs = events[len(events)-1].At.Milliseconds()

	ctx, cancel := context.WithCancel(context.Background())
	s.run.ctx = ctx
	s.run.cancel = cancel
	s.run.ctrl = NewBotControl()
	s.run.done = make(chan struct{})
	s.run.lease = lease
	s.run.exitOnce = sync.Once{}

	botLogger := zerolog.New(s.app.GetLogSink()).With().
		Timestamp().Str("bot", "piano").Logger()

	hwnd := win.HWND(uintptr(game.HWND))
	player := piano.NewPlayer(hwnd, cfg, botLogger, s.run.ctrl)
	s.playerMu.Lock()
	s.player = player
	s.playerMu.Unlock()

	// progress hook: 节流 200ms emit
	onProgress := func(curMs, totalMs int64) {
		s.progressMu.Lock()
		if time.Since(s.lastEmit) < 200*time.Millisecond {
			s.progressMu.Unlock()
			return
		}
		s.lastEmit = time.Now()
		s.progressMu.Unlock()
		s.app.Emit(services.EventPianoProgress, services.PianoProgressEvent{CurMs: curMs, TotalMs: totalMs})
	}

	// emit 初始 progress（reset）
	s.app.Emit(services.EventPianoProgress, services.PianoProgressEvent{CurMs: 0, TotalMs: s.totalMs})

	s.run.markRunning()

	go s.run.runWorker(botLogger, func() {
		player.Play(ctx, events, onProgress)
		// 演奏结束 emit 完成的 progress
		s.app.Emit(services.EventPianoProgress, services.PianoProgressEvent{CurMs: s.totalMs, TotalMs: s.totalMs})
		s.playerMu.Lock()
		s.player = nil
		s.playerMu.Unlock()
	})

	return nil
}

// Pause 暂停正在运行的 bot。
func (s *PianoService) Pause() error {
	s.run.opMu.Lock()
	defer s.run.opMu.Unlock()
	if s.run.state.Load() != bsRunning {
		return errors.New("piano 未在运行")
	}
	s.run.pause()
	return nil
}

// Resume 恢复暂停中的 bot。
func (s *PianoService) Resume() error {
	s.run.opMu.Lock()
	defer s.run.opMu.Unlock()
	if s.run.state.Load() != bsPaused {
		return errors.New("piano 未暂停")
	}
	s.run.resume()
	return nil
}

// Stop 通知 bot 停止（不阻塞等待退出）。
func (s *PianoService) Stop() error {
	s.run.opMu.Lock()
	defer s.run.opMu.Unlock()
	if s.run.state.Load() == bsIdle {
		return errors.New("piano 未运行")
	}
	s.run.stop()
	return nil
}

// SeekMs 跳到指定毫秒。仅运行中有效。
// 不叫 Seek 是为了避开 go vet stdmethods (io.Seeker.Seek 签名冲突)。
func (s *PianoService) SeekMs(ms int64) error {
	s.playerMu.Lock()
	player := s.player
	s.playerMu.Unlock()
	if player == nil {
		return errors.New("piano 未在演奏")
	}
	player.SeekTo(ms)
	return nil
}

// SetMode 0=36键 1=21键
func (s *PianoService) SetMode(m int) error {
	if m != 0 && m != 1 {
		return fmt.Errorf("mode must be 0 or 1: got %d", m)
	}
	s.app.UpdateSettings(func(st *services.Settings) { st.Piano.Mode = m })
	return s.app.SaveSettings()
}

func (s *PianoService) SetTrackPick(p int) error {
	if p < -2 {
		return fmt.Errorf("trackPick must be >= -2: got %d", p)
	}
	s.app.UpdateSettings(func(st *services.Settings) { st.Piano.TrackPick = p })
	return s.app.SaveSettings()
}

func (s *PianoService) SetAutoTranspose(v bool) error {
	s.app.UpdateSettings(func(st *services.Settings) { st.Piano.AutoTranspose = v })
	return s.app.SaveSettings()
}

func (s *PianoService) SetMelodyOnly(v bool) error {
	s.app.UpdateSettings(func(st *services.Settings) { st.Piano.MelodyOnly = v })
	return s.app.SaveSettings()
}

func (s *PianoService) SetOctaveOffset(v int) error {
	if v < -2 || v > 2 {
		return fmt.Errorf("octaveOffset out of range [-2, 2]: %d", v)
	}
	s.app.UpdateSettings(func(st *services.Settings) { st.Piano.OctaveOffset = v })
	return s.app.SaveSettings()
}

// StopSync App.Shutdown 调；阻塞直到 worker 真正退出。
func (s *PianoService) StopSync() {
	s.run.opMu.Lock()
	defer s.run.opMu.Unlock()
	if s.run.state.Load() == bsIdle {
		return
	}
	s.run.stopSync()
}

// ------ helpers ------

func midiInfoFromData(path string, data *piano.MIDIData) MIDIInfo {
	tracks := make([]TrackInfo, len(data.Tracks))
	for i, t := range data.Tracks {
		name := t.Name
		if strings.TrimSpace(name) == "" {
			name = fmt.Sprintf("Track %d", i)
		}
		tracks[i] = TrackInfo{Index: i, Name: name, NoteCount: t.NoteCount}
	}
	var totalMs int64
	all := data.AllEvents()
	if len(all) > 0 {
		totalMs = all[len(all)-1].At.Milliseconds()
	}
	return MIDIInfo{Path: path, TotalMs: totalMs, Tracks: tracks}
}

// pickEvents 按 TrackPick 取事件。
func pickEvents(data *piano.MIDIData, pick int) []piano.NoteEvent {
	switch {
	case pick == piano.TrackPickSmart:
		return data.SmartEvents()
	case pick == piano.TrackPickAll:
		return data.AllEvents()
	default:
		if pick >= 0 && pick < len(data.Tracks) {
			return data.TrackEvents(pick)
		}
		return data.SmartEvents()
	}
}
