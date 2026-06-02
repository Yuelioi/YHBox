// Package runtime — ClipPlayer: InputClip 回放调度核心.
//
// 关键算法 (否则 cut 段后会 freeze 等待):
//   不能用 event.TUs 直接做 schedule target. 否则 cut 掉一段后下个 event 的
//   时间不变 → wallclock 多等 cut 长度的秒数.
// 正解: 对每个 event 计算 effectiveTUs = event.TUs - accumulatedCut, accumulatedCut
//   是"按 keepRanges 跳过的总时长". 进入 range r 之前, 累计跨段的 gap.
//
// keepRanges: 不指定 / 空 = 整段播 (等价单 range [0, DurationUs]).
// 取消语义: ctx 取消 或 Cancel() → ReleaseHeld 释放所有按下的 key/btn 防卡键.
package runtime

import (
	"context"
	"errors"
	"fmt"
	"math"
	"sort"
	"sync"

	"yotta/internal/services/inputclip"
	"yotta/internal/services/inputclip/backends"
)

// ErrCancelled ClipPlayer 因 Cancel() 或 ctx 取消而终止 Wait() 返这个.
var ErrCancelled = errors.New("clip playback cancelled")

// IClipPlayable 抽象: 调度方 (PlayClip 节点) 只看接口.
type IClipPlayable interface {
	Start(ctx context.Context)
	Wait() error
	Cancel()
	Done() <-chan struct{}
}

// ClipPlayer InputClip 调度器. 单次播放, 不复用.
type ClipPlayer struct {
	clip            *inputclip.InputClip
	keepRanges      []inputclip.Range
	backend         backends.IInputBackend
	policy          PlaybackPolicy
	targetCounts360 int
	// windowProvider 返回回放目标窗口的 clientW/clientH.
	// nil 或返回 0 → 拿不到目标窗口尺寸, 按原始坐标回放不缩放.
	windowProvider func() (clientW, clientH int)

	done     chan struct{}
	cancelCh chan struct{}
	err      error

	mu         sync.Mutex
	cancelOnce sync.Once

	// run 写, Wait 后读. 不需要锁 — Wait 阻塞到 done close, happens-after.
	sentCount int

	// RawDelta 缩放的小数余量 (误差扩散 carry). 每事件独立截断会把小 delta (1~3 counts)
	// 成片抹平 (360° 塌成 120° 的根因); 累积余量保住缩放总量. run-only, 单次播放零初始化即可.
	residX, residY float64
}

// SentCount Wait 返回后调. 真正调 backend.Send 的次数.
func (p *ClipPlayer) SentCount() int { return p.sentCount }

// NewClipPlayer 构造. keepRanges 可为 nil / 空 = 整段播.
// windowProvider 可选 (可传 nil): 返回回放目标窗口 clientW/clientH, 用于
// 跨分辨率绝对坐标缩放. nil 或返回 0 → 不缩放.
func NewClipPlayer(
	clip *inputclip.InputClip,
	keepRanges []inputclip.Range,
	backend backends.IInputBackend,
	policy PlaybackPolicy,
	targetCounts360 int,
	windowProvider func() (clientW, clientH int),
) *ClipPlayer {
	return &ClipPlayer{
		clip:            clip,
		keepRanges:      keepRanges,
		backend:         backend,
		policy:          policy,
		targetCounts360: targetCounts360,
		windowProvider:  windowProvider,
		done:            make(chan struct{}),
		cancelCh:        make(chan struct{}),
	}
}

// Start 异步启动调度 goroutine. 返回前 done chan 已就绪.
func (p *ClipPlayer) Start(ctx context.Context) {
	go p.run(ctx)
}

// Wait 阻塞到回放结束 / 取消 / err.
func (p *ClipPlayer) Wait() error {
	<-p.done
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.err
}

// Cancel 取消播放. 幂等. ReleaseHeld 在 run() defer 里调.
func (p *ClipPlayer) Cancel() {
	p.cancelOnce.Do(func() {
		close(p.cancelCh)
	})
}

// Done 给 select 用.
func (p *ClipPlayer) Done() <-chan struct{} {
	return p.done
}

func (p *ClipPlayer) setErr(err error) {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.err == nil {
		p.err = err
	}
}

// run 调度 goroutine. defer ReleaseHeld 兜底防卡键.
func (p *ClipPlayer) run(ctx context.Context) {
	defer close(p.done)
	defer func() { _ = p.backend.ReleaseHeld() }()

	TimerSetPrecision1ms()
	defer TimerResetPrecision()

	if p.clip == nil || len(p.clip.Events) == 0 {
		return
	}

	// 归一化 ranges: 空 → 整段; 否则 sort + merge overlap.
	// UI ClipTimeline 拖拽可能产生乱序 / 重叠 ranges, 不归一化 accumulatedKept 算错事件发两次.
	ranges := normalizeRanges(p.keepRanges, p.clip.DurationUs)

	startQPC := QPCMicros()
	evIdx := 0
	accumulatedKept := uint64(0) // 前面所有 keep 段总长 (用于算 accumulatedCut)

	for _, r := range ranges {
		// 进入这段时 accumulatedCut = 段头位置 - 前面已 keep 总长.
		// 第一段: r[0].FromUs > 0 → accumulatedCut = r[0].FromUs (开头 cut).
		// 后续段: accumulatedCut = r[i].FromUs - sum(keep 前 i 段).
		var accumulatedCut uint64
		if r.FromUs > accumulatedKept {
			accumulatedCut = r.FromUs - accumulatedKept
		}

		// 跳过 < r.FromUs 的 events.
		for evIdx < len(p.clip.Events) && p.clip.Events[evIdx].TUs < r.FromUs {
			evIdx++
		}
		// 发送 events in [r.FromUs, r.ToUs].
		for evIdx < len(p.clip.Events) && p.clip.Events[evIdx].TUs <= r.ToUs {
			if err := ctx.Err(); err != nil {
				p.setErr(err)
				return
			}
			select {
			case <-p.cancelCh:
				p.setErr(ErrCancelled)
				return
			default:
			}

			ev := p.clip.Events[evIdx]
			effectiveTUs := ev.TUs - accumulatedCut
			WaitUntil(startQPC+effectiveTUs, p.policy)

			// 二次取消检查: WaitUntil 可能阻塞了一段时间.
			if err := ctx.Err(); err != nil {
				p.setErr(err)
				return
			}
			select {
			case <-p.cancelCh:
				p.setErr(ErrCancelled)
				return
			default:
			}

			outEv := ev
			if ev.Type == inputclip.EventTypeRawDelta {
				source := p.clip.Meta.MouseCounts360
				target := p.targetCounts360
				if source > 0 && target > 0 && source != target {
					factor := float64(target) / float64(source)
					// 误差扩散: 累积小数余量再 round, 防小 delta 截断成片丢失 (保住缩放总量).
					sx := float64(ev.B)*factor + p.residX
					sy := float64(ev.C)*factor + p.residY
					rx := math.Round(sx)
					ry := math.Round(sy)
					p.residX = sx - rx
					p.residY = sy - ry
					outEv.B = int32(rx)
					outEv.C = int32(ry)
				}
			} else if absScaleNeeded(ev.Type) {
				baseW := p.clip.Meta.BaseResolution[0]
				baseH := p.clip.Meta.BaseResolution[1]
				if baseW > 0 && baseH > 0 && p.windowProvider != nil {
					targetW, targetH := p.windowProvider()
					if targetW > 0 && targetH > 0 {
						scaleX := float64(targetW) / float64(baseW)
						scaleY := float64(targetH) / float64(baseH)
						if scaleX != 1.0 || scaleY != 1.0 {
							outEv.B = int32(float64(ev.B) * scaleX)
							outEv.C = int32(float64(ev.C) * scaleY)
						}
					}
				}
			}

			if err := p.backend.Send(outEv); err != nil {
				p.setErr(fmt.Errorf("backend.Send ev[%d]: %w", evIdx, err))
				return
			}
			p.sentCount++
			evIdx++
		}

		accumulatedKept += r.ToUs - r.FromUs
	}
}

// 编译期校验: ClipPlayer 实现 IClipPlayable.
var _ IClipPlayable = (*ClipPlayer)(nil)

// absScaleNeeded 判断 EventType 是否携带绝对屏幕坐标 (B=x, C=y), 需跨分辨率缩放.
func absScaleNeeded(t inputclip.EventType) bool {
	return t == inputclip.EventTypeMouseBtnDown ||
		t == inputclip.EventTypeMouseBtnUp ||
		t == inputclip.EventTypeMouseMove ||
		t == inputclip.EventTypeScroll
}

// normalizeRanges sort + merge overlap; nil/空 → 整段 [0, durationUs].
// 也截断超出 durationUs 的段, 防止越界.
func normalizeRanges(input []inputclip.Range, durationUs uint64) []inputclip.Range {
	if len(input) == 0 {
		return []inputclip.Range{{FromUs: 0, ToUs: durationUs}}
	}
	// 拷贝避免改 caller. 同时丢空 / 反向 / 越界段.
	cp := make([]inputclip.Range, 0, len(input))
	for _, r := range input {
		if r.ToUs > durationUs {
			r.ToUs = durationUs
		}
		if r.FromUs >= r.ToUs {
			continue
		}
		cp = append(cp, r)
	}
	if len(cp) == 0 {
		return []inputclip.Range{{FromUs: 0, ToUs: durationUs}}
	}
	sort.Slice(cp, func(i, j int) bool { return cp[i].FromUs < cp[j].FromUs })
	out := cp[:0]
	for _, r := range cp {
		if len(out) > 0 && r.FromUs <= out[len(out)-1].ToUs {
			if r.ToUs > out[len(out)-1].ToUs {
				out[len(out)-1].ToUs = r.ToUs
			}
			continue
		}
		out = append(out, r)
	}
	return out
}
