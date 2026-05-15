package actiontransform

import "yhbox/internal/services/actions"

// dragThresholdPx 判别 click vs drag 的阈值：down→up 间最远的 mouse_move
// 距离平方超过 25（= 5px Euclidean）才认为是 drag。低于 → click_<btn>。
const dragThresholdPx = 25

// rawDeltaWindowMs raw input 聚合窗口。raw input 一秒上百次太碎，按 50ms 一桶
// 累加成一个 mouse_move_rel step，回放精度够（驱动里再拆 16ms 一帧投递）。
const rawDeltaWindowMs uint32 = 50

// PairEvents 按下列语义把 raw events 配对成 steps（带 sleep 占位）：
//
//	keyboard:
//	  down + up      → key (durationMs = up - down)
//	  only down      → key_down（用户没松开就停录）
//	  only up        → 丢弃（录制开始时键已按着，下半段没意义）
//	  down + down + up（同键重按） → 第一个 down 当 key_down；第二个 down + up 合成 key
//
//	mouse 按键:
//	  down + up（中间没移动）            → click_<btn>
//	  down + 有 move + up（超过阈值距离） → mouse_drag from down 位置 to up 位置
//	  only down/up                       → 丢弃
//
//	mouse_scroll:                        → 单 step_scroll，notches = ev.WheelNotches
//
//	mouse_move（无 down 关联）:           → 丢弃（裸鼠标移动不产 step；只在 drag 检测里当轨迹用）
//
// 短 click (durationMs < 20) → normalize 到 20ms 避抖动。
// 事件之间 >=50ms gap 插一个 step_sleep（具体 sleep 合并由 MergeSleeps 处理）。
func PairEvents(events []Event, clientW, clientH int) []actions.Step {
	var steps []actions.Step
	if len(events) == 0 {
		return steps
	}

	// 预扫键盘事件配对 + 标记 wrapper（外层修饰键，内有其它 key 嵌套）。
	// 这样 Shift_down → W_down → W_up → Shift_up 序列输出成：
	//   key_down(Shift) / key(W, 100ms) / key_up(Shift)
	// 而不是错的 key(W) + key(Shift)。
	keyPairs := scanKeyPairs(events)
	markKeyWrappers(keyPairs)
	keyDownInfo := make(map[int]*keyPair, len(keyPairs))
	keyUpInfo := make(map[int]*keyPair, len(keyPairs))
	for i := range keyPairs {
		keyDownInfo[keyPairs[i].DownIdx] = &keyPairs[i]
		if keyPairs[i].UpIdx >= 0 {
			keyUpInfo[keyPairs[i].UpIdx] = &keyPairs[i]
		}
	}

	pendingMouse := map[MouseBtn]int{} // btn → 未配对 down 的 index

	// raw delta 聚合器：50ms 窗口累加 dx/dy，碰到非 raw 事件或窗口超时 → 出一个 step
	var rawHasData bool
	var rawStartT, rawEndT uint32
	var rawDx, rawDy int

	var lastEmitTime uint32 = events[0].TimestampMs
	emitGap := func(t uint32) {
		if t <= lastEmitTime {
			return
		}
		gap := t - lastEmitTime
		if gap < 50 {
			return
		}
		steps = append(steps, actions.Step{
			Kind:       actions.StepSleep,
			DurationMs: int(gap),
		})
	}

	// flushRawDelta 把当前聚合的 raw delta 写成一个 mouse_move_rel step。
	// 在以下时机调：碰到非 raw 事件 / 窗口超出 50ms / 扫完事件流。
	flushRawDelta := func() {
		if !rawHasData {
			return
		}
		emitGap(rawStartT)
		duration := int(rawEndT - rawStartT)
		if duration < 16 {
			duration = 16 // 至少一帧
		}
		steps = append(steps, actions.Step{
			Kind:       actions.StepMouseMoveRel,
			Dx:         rawDx,
			Dy:         rawDy,
			DurationMs: duration,
		})
		lastEmitTime = rawEndT
		rawHasData = false
		rawDx, rawDy = 0, 0
	}

	toRatio := func(x, y int) (float32, float32) {
		var xR, yR float32
		if clientW > 0 {
			xR = float32(x) / float32(clientW)
		}
		if clientH > 0 {
			yR = float32(y) / float32(clientH)
		}
		return xR, yR
	}

	// 判 click vs drag：扫 downIdx 到 upIdx 之间的 mouse_move，找最远 manhattan
	// 距离平方；超阈值 → drag。
	classifyDrag := func(downIdx, upIdx int) bool {
		down := events[downIdx]
		for j := downIdx + 1; j < upIdx; j++ {
			if events[j].Type != EventMouseMove {
				continue
			}
			dx := events[j].X - down.X
			dy := events[j].Y - down.Y
			if dx*dx+dy*dy > dragThresholdPx {
				return true
			}
		}
		return false
	}

	for i, ev := range events {
		// 任何非 raw delta 事件先 flush 当前 raw 聚合；保证 mouse_move_rel step
		// 时间上落在跟其它 step 同一序列里。
		if ev.Type != EventRawMouseDelta {
			// raw 窗口超时也要在 raw 自己分支 flush；这里只处理"被非 raw 打断"
			flushRawDelta()
		}
		switch ev.Type {
		case EventKeyboard:
			if ev.IsDown {
				p := keyDownInfo[i]
				// Wrapper（外层修饰键）或孤立 down → 在 down 时机就 emit key_down
				if p != nil && (p.IsWrapper || p.UpIdx < 0) {
					emitGap(ev.TimestampMs)
					steps = append(steps, actions.Step{
						Kind: actions.StepKeyDown,
						Vk:   ev.Vk,
					})
					lastEmitTime = ev.TimestampMs
				}
				// 普通成对（非 wrapper、有 up）→ skip，等 up 那一刻 emit key(N)
			} else {
				p := keyUpInfo[i]
				if p == nil {
					break // 孤立 up 丢
				}
				if p.IsWrapper {
					// wrapper key_up 用 up 时刻算 gap（key_down 已在 down 时刻 emit 过）
					emitGap(ev.TimestampMs)
					steps = append(steps, actions.Step{
						Kind: actions.StepKeyUp,
						Vk:   ev.Vk,
					})
				} else {
					// 普通 key step：gap 算到 down 时刻，duration 覆盖 down→up
					emitGap(events[p.DownIdx].TimestampMs)
					duration := int(ev.TimestampMs - events[p.DownIdx].TimestampMs)
					if duration < 20 {
						duration = 20
					}
					steps = append(steps, actions.Step{
						Kind:       actions.StepKey,
						Vk:         ev.Vk,
						DurationMs: duration,
					})
				}
				lastEmitTime = ev.TimestampMs
			}

		case EventMouse:
			if ev.IsDown {
				pendingMouse[ev.MouseBtn] = i
			} else {
				if downIdx, has := pendingMouse[ev.MouseBtn]; has {
					down := events[downIdx]
					emitGap(down.TimestampMs)
					duration := int(ev.TimestampMs - down.TimestampMs)
					if duration < 20 {
						duration = 20
					}
					if classifyDrag(downIdx, i) {
						xR1, yR1 := toRatio(down.X, down.Y)
						xR2, yR2 := toRatio(ev.X, ev.Y)
						steps = append(steps, actions.Step{
							Kind:        actions.StepMouseDrag,
							XRatio:      xR1,
							YRatio:      yR1,
							X2Ratio:     xR2,
							Y2Ratio:     yR2,
							MouseButton: mouseButtonName(ev.MouseBtn),
							DurationMs:  duration,
						})
					} else {
						xR, yR := toRatio(down.X, down.Y)
						steps = append(steps, actions.Step{
							Kind:       mouseStepKind(ev.MouseBtn),
							XRatio:     xR,
							YRatio:     yR,
							DurationMs: duration,
						})
					}
					lastEmitTime = ev.TimestampMs
					delete(pendingMouse, ev.MouseBtn)
				}
			}

		case EventMouseScroll:
			emitGap(ev.TimestampMs)
			steps = append(steps, actions.Step{
				Kind:        actions.StepScroll,
				ScrollDelta: ev.WheelNotches,
			})
			lastEmitTime = ev.TimestampMs

		case EventMouseMove:
			// 裸移动不产 step；只用于 classifyDrag 的轨迹判断（在 EventMouse up 分支查）

		case EventRawMouseDelta:
			// 同一 50ms 窗口内的累加；窗口超时 → 先 flush 再开新桶
			if rawHasData && ev.TimestampMs-rawStartT > rawDeltaWindowMs {
				flushRawDelta()
			}
			if !rawHasData {
				rawStartT = ev.TimestampMs
				rawHasData = true
			}
			rawEndT = ev.TimestampMs
			rawDx += ev.Dx
			rawDy += ev.Dy
		}
	}
	// 收尾：可能还有一桶 raw 没 flush
	flushRawDelta()
	// 孤立 keyboard down（无配对 up）已在主循环 emit 为 key_down，不需要收尾。
	// pendingMouse 还在的孤立 down 丢弃。

	return steps
}

// keyPair 一对键盘 down/up 事件的索引。UpIdx<0 = 孤立 down。
type keyPair struct {
	Vk        string
	DownIdx   int
	UpIdx     int
	IsWrapper bool // 其它 pair 完全嵌套在内（外层修饰键如 Shift+W 的 Shift）
}

// scanKeyPairs 一次扫 events 把所有键盘 down/up 配对。
//
// **Auto-repeat 处理**：Windows 在用户按住键不松开时会持续发 KEYDOWN（按
// keyboard repeat rate ~30ms/次），LL hook 全部捕获。recorder 不区分初次按下和
// auto-repeat。pending 用单值 map（不是栈）：同 vk 已经在 pending 时再来 down
// 视为 auto-repeat 忽略，直到对应 up 才闭合。
//
// 副作用：放弃了原来"down/down/up = 重按"的栈语义 —— 物理上一个键没松手不可能
// 再按一次，旧实现 auto-repeat 下会产 N 个孤立 key_down step，回放严重错误。
func scanKeyPairs(events []Event) []keyPair {
	pending := map[string]int{} // vk → 当前未配对 down 的索引（最多一个）
	pairs := []keyPair{}
	for i, ev := range events {
		if ev.Type != EventKeyboard {
			continue
		}
		if ev.IsDown {
			// auto-repeat：已经在 pending 就跳过
			if _, has := pending[ev.Vk]; has {
				continue
			}
			pending[ev.Vk] = i
		} else {
			if downIdx, has := pending[ev.Vk]; has {
				pairs = append(pairs, keyPair{
					Vk:      ev.Vk,
					DownIdx: downIdx,
					UpIdx:   i,
				})
				delete(pending, ev.Vk)
			}
			// 孤立 up 丢
		}
	}
	// 剩余 pending = 孤立 down（用户没松手就停录）
	for vk, downIdx := range pending {
		pairs = append(pairs, keyPair{Vk: vk, DownIdx: downIdx, UpIdx: -1})
	}
	return pairs
}

// markKeyWrappers pair[i] 是 wrapper 当且仅当存在 pair[j] (i≠j) 完全嵌套：
// j.DownIdx > i.DownIdx && j.UpIdx < i.UpIdx。孤立 down 不是 wrapper。
func markKeyWrappers(pairs []keyPair) {
	for i := range pairs {
		if pairs[i].UpIdx < 0 {
			continue
		}
		for j := range pairs {
			if i == j || pairs[j].UpIdx < 0 {
				continue
			}
			if pairs[j].DownIdx > pairs[i].DownIdx && pairs[j].UpIdx < pairs[i].UpIdx {
				pairs[i].IsWrapper = true
				break
			}
		}
	}
}

func mouseButtonName(b MouseBtn) string {
	switch b {
	case BtnMiddle:
		return "middle"
	case BtnRight:
		return "right"
	default:
		return "left"
	}
}

func mouseStepKind(b MouseBtn) actions.StepKind {
	switch b {
	case BtnMiddle:
		return actions.StepClickMiddle
	case BtnRight:
		return actions.StepClickRight
	default:
		return actions.StepClickLeft
	}
}
