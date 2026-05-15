package piano

import (
	"context"
	"sort"
	"sync/atomic"
	"time"

	"github.com/lxn/win"
	"github.com/rs/zerolog"

	"yhbox/pkg/input"
	"yhbox/pkg/runctl"
)

// activateDelay FakeActivate 后等 UE Slate 翻 IsActive 的时间。
// 参考 fish 模块经验，30ms 足够让 PostMessage 不被丢。Play 入口设一次即可。
const activateDelay = 30 * time.Millisecond

// Player 按时间表向游戏窗口发送键事件。
type Player struct {
	hwnd win.HWND
	cfg  *Config
	log  zerolog.Logger
	ctrl runctl.Control

	totalOffset int // 自动 + 手动八度偏移之和，Play 入口算好后整曲沿用

	seekTargetMs atomic.Int64
	seekPending  atomic.Bool
}

func NewPlayer(hwnd win.HWND, cfg *Config, logger zerolog.Logger, ctrl runctl.Control) *Player {
	return &Player{hwnd: hwnd, cfg: cfg, log: logger, ctrl: ctrl}
}

// SeekTo 请求跳转到 targetMs。Play 主循环在每次迭代顶端消费请求。
// 跨 goroutine 安全：Store(target) → Store(pending=true)，消费侧 CompareAndSwap 反向读，
// atomic happens-before 保证 pending=true 时 target 一定已经写完。
// 不叫 Seek 是为了避开 go vet stdmethods (io.Seeker.Seek 签名冲突)。
func (p *Player) SeekTo(targetMs int64) {
	p.seekTargetMs.Store(targetMs)
	p.seekPending.Store(true)
}

// Play 阻塞直到全部演奏完毕、ctx 取消或 ctrl.Stop。
// progress 可空；非空时按 100ms 频率推 (当前ms, 总ms) 供 UI 显示。
func (p *Player) Play(ctx context.Context, events []NoteEvent, progress func(curMs, totalMs int64)) {
	if len(events) == 0 {
		return
	}
	totalMs := events[len(events)-1].At.Milliseconds()
	start := time.Now()
	nextProgressAt := start

	// 整曲八度偏移 = 自动平移 + 用户手动微调。统一应用让旋律轮廓不变形。
	autoOff := 0
	if p.cfg.AutoTranspose {
		autoOff = AutoTranspose(events)
	}
	p.totalOffset = autoOff + p.cfg.OctaveOffset

	p.log.Info().Str("tag", "SYSTEM").Msgf("开始演奏 (mode=%s, 自动偏移=%+d, 手动偏移=%+d, %d 音符, 总时长 %.1fs)",
		p.cfg.Mode, autoOff, p.cfg.OctaveOffset, len(events), float64(totalMs)/1000)

	// 演奏开始前一次性 FakeActivate，让 UE 翻成 IsActive 接收后续 PostMessage。
	// 后续每个 chord 不再 FakeActivate，连续演奏中 IsActive 保持。
	input.FakeActivate(p.hwnd)
	time.Sleep(activateDelay)

	// 退出前清理：把所有可能按住的键 + Shift/Ctrl 都释放，
	// 防 Stop / ctx 取消时游戏里残留按住状态。
	defer p.releaseAllKeys()

	for i := 0; i < len(events); {
		if p.ctrl != nil {
			if p.ctrl.IsBack() {
				p.log.Info().Str("tag", "SYSTEM").Msg("停止演奏")
				return
			}
			if p.ctrl.IsPaused() {
				p.log.Info().Str("tag", "SYSTEM").Msg("已暂停")
				pausedAt := time.Now()
				if !p.ctrl.WaitUnpause() {
					return
				}
				start = start.Add(time.Since(pausedAt))
				p.log.Info().Str("tag", "SYSTEM").Msg("已恢复")
			}
		}

		// 用户拖进度条 → 跳转
		if p.seekPending.CompareAndSwap(true, false) {
			target := time.Duration(p.seekTargetMs.Load()) * time.Millisecond
			i = sort.Search(len(events), func(k int) bool { return events[k].At >= target })
			start = time.Now().Add(-target)
			ms := target.Milliseconds()
			p.log.Info().Str("tag", "SYSTEM").Msgf("跳转到 %d:%02d", ms/60000, (ms/1000)%60)
			if i >= len(events) {
				break
			}
			continue
		}

		// 收和弦：从 i 起，At 距首音 ≤ ChordWindow 的算同一拍。
		// 用 <= 保证同 At 的事件（多 track）一定在同一组。
		j := i + 1
		for j < len(events) && events[j].At-events[i].At <= p.cfg.ChordWindow {
			j++
		}
		chord := events[i:j]

		// 等到 chord[0].At。每 50ms 检查一次 ctrl 和 ctx，让 Stop / 暂停 即时响应。
		// 不能用单个 time.After(d)：用户 Stop 时 ctrl.IsBack() 变 true 但不会 cancel ctx，
		// 大间隔（如 1 音符 MIDI 等 50s）会让 Stop 没反应。
		target := start.Add(chord[0].At)
		const sleepStep = 50 * time.Millisecond
		stopped := false
	waitLoop:
		for {
			now := time.Now()
			if !now.Before(target) {
				break
			}
			if p.ctrl != nil && p.ctrl.IsBack() {
				stopped = true
				break
			}
			d := target.Sub(now)
			if d > sleepStep {
				d = sleepStep
			}
			select {
			case <-time.After(d):
			case <-ctx.Done():
				stopped = true
				break waitLoop
			}
		}
		if stopped {
			p.log.Info().Str("tag", "SYSTEM").Msg("停止演奏")
			return
		}

		p.playChord(chord)

		if progress != nil && time.Now().After(nextProgressAt) {
			progress(chord[0].At.Milliseconds(), totalMs)
			nextProgressAt = time.Now().Add(100 * time.Millisecond)
		}

		i = j
	}

	if progress != nil {
		progress(totalMs, totalMs)
	}
	p.log.Info().Str("tag", "SYSTEM").Msg("演奏完毕")
}

// playChord 一组音同时按住：KeyDown 全部（KeyStagger 防丢键）→ 保持 KeyHold → KeyUp 全部。
// 跨修饰组只能串行（Shift 是全局态）；只用 Shift 策略 → 多数和弦在单组内一次性齐按。
//
// MelodyOnly=true 时只保留 chord 里最高音（top note），抛掉伴奏，旋律线最干净。
func (p *Player) playChord(chord []NoteEvent) {
	const maxOctaveShift = 1
	type act struct {
		key string
		mod Modifier
	}

	notes := chord
	if p.cfg.MelodyOnly && len(notes) > 1 {
		top := notes[0]
		for _, ev := range notes[1:] {
			if ev.Key > top.Key {
				top = ev
			}
		}
		notes = []NoteEvent{top}
	}

	var acts []act
	for _, ev := range notes {
		midi := int(ev.Key) + p.totalOffset*12
		a, ok := MapNoteWithOctaveShift(midi, p.cfg.Mode, maxOctaveShift)
		if !ok {
			continue
		}
		acts = append(acts, act{key: a.Key, mod: a.Mod})
	}
	if len(acts) == 0 {
		return
	}

	for _, mod := range []Modifier{ModNone, ModShift} {
		var group []string
		for _, a := range acts {
			if a.mod == mod {
				group = append(group, a.key)
			}
		}
		if len(group) == 0 {
			continue
		}

		if mod == ModShift {
			input.KeyDown(p.hwnd, "shift")
			time.Sleep(p.cfg.KeyStagger)
		}

		for k, key := range group {
			if k > 0 && p.cfg.KeyStagger > 0 {
				time.Sleep(p.cfg.KeyStagger)
			}
			input.KeyDown(p.hwnd, key)
		}

		time.Sleep(p.cfg.KeyHold)

		for k, key := range group {
			if k > 0 && p.cfg.KeyStagger > 0 {
				time.Sleep(p.cfg.KeyStagger)
			}
			input.KeyUp(p.hwnd, key)
		}

		if mod == ModShift {
			input.KeyUp(p.hwnd, "shift")
		}
	}
}

// releaseAllKeys 强制 KeyUp 所有可能按住的键。Play 退出前 defer 调用，
// 防 Stop / ctx 取消时 chord 中途留下卡住的键 + Shift。
func (p *Player) releaseAllKeys() {
	for _, row := range octaveKeys {
		for _, k := range row {
			input.KeyUp(p.hwnd, k)
		}
	}
	input.KeyUp(p.hwnd, "shift")
	input.KeyUp(p.hwnd, "ctrl")
}

func sleepCtx(ctx context.Context, d time.Duration) bool {
	select {
	case <-time.After(d):
		return true
	case <-ctx.Done():
		return false
	}
}
