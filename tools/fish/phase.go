package fish

import (
	"context"
	"fmt"
	"image"
	"time"

	"yhbox/pkg/capture"
	"yhbox/pkg/input"
)

// UIPhase 表示当前游戏画面实际所处的阶段，由全部检测器投票得出，
// 与状态机的内部 State 解耦——用于在状态机和真实画面失同步时纠偏。
type UIPhase int

const (
	PhaseUnknown       UIPhase = iota
	PhaseSetup                 // 准备界面（开始钓鱼按钮可见）
	PhaseReady                 // 钓鱼点 hook icon 可见
	PhaseFighting              // 耐力条可见，溜鱼中
	PhaseSettleWin             // 结算关闭按钮可见（钓上来了）
	PhaseSettleFail            // "鱼儿溜走了"文字可见
	PhaseNeedBait              // 鱼饵不足提示可见
	PhaseWarehouseFull         // 鱼仓已满提示可见
)

// phaseEntry 把 (UIPhase 枚举 → 检测器 → 显示标签) 三者绑在一起，
// inspectPhaseFrame 用表 for 循环替代 6 个独立 if 分支。
type phaseEntry struct {
	phase  UIPhase
	label  string // String() 短标签
	debug  string // 日志 slot 名
	detect DetectFn
}

var phaseTable = []phaseEntry{
	{PhaseSettleWin, "WIN", "result", (*Detector).Result},
	{PhaseSettleFail, "FAIL", "fish_escape", (*Detector).FishEscape},
	{PhaseWarehouseFull, "FULL", "warehouse_full", (*Detector).WarehouseFull},
	{PhaseNeedBait, "BAIT", "need_bait", (*Detector).NeedBait},
	{PhaseSetup, "SETUP", "start_fish", (*Detector).StartFish},
	{PhaseReady, "READY", "hook_icon", (*Detector).HookIconDim},
}

func (p UIPhase) String() string {
	if p == PhaseFighting {
		return "FIGHT"
	}
	if p == PhaseUnknown {
		return "UNKNOWN"
	}
	for _, e := range phaseTable {
		if e.phase == p {
			return e.label
		}
	}
	return "UNKNOWN"
}

// inspectPhase 抓一帧并按优先级跑全部检测器，返回当前画面实际处于哪个阶段。
// 用于没有现成 frame 的状态（如 WAITING）。
func (m *machine) inspectPhase() (UIPhase, string) {
	frame, err := capture.Frame(m.hwnd)
	if err != nil {
		return PhaseUnknown, fmt.Sprintf("抓帧失败: %v", err)
	}
	return m.inspectPhaseFrame(frame)
}

// inspectPhaseFrame 复用调用方已抓取的全帧，避免重复抓帧。
// 优先级见 phaseTable 顺序：结算成功 > 结算失败 > 鱼仓满 > 鱼饵不足 > 准备界面 > 抛竿就绪 > 溜鱼条。
// 注意：上钩文字只显示 ~1s，到探测时早没了，所以不在这里检查。
func (m *machine) inspectPhaseFrame(frame *image.RGBA) (UIPhase, string) {
	for _, e := range phaseTable {
		if hit, ok := e.detect(m.det, frame); ok {
			return e.phase, fmt.Sprintf("%s=%.2f", e.debug, hit.Conf)
		}
	}
	// bar 检测放最后（特殊语义：要抓 ROI 子帧）
	barX, barY, barW, barH := m.det.PickBarROI(m.clientW, m.clientH)
	if barW <= 0 || barH <= 0 {
		return PhaseUnknown, ""
	}
	if barFrame, err := capture.FrameROI(m.hwnd, barX, barY, barW, barH); err == nil {
		bar := m.det.FishingBarDirect(barFrame)
		if bar.CursorX >= 0 && bar.TargetX >= 0 && bar.Confidence >= confBar {
			return PhaseFighting, fmt.Sprintf("bar=%.2f", bar.Confidence)
		}
	}
	return PhaseUnknown, ""
}

// routePhase 根据探测到的阶段把状态机切换到对应分支，必要时执行结算/ESC 等副作用。
// 返回 true 表示已自动恢复并改写了 m.state，调用方应直接 return；
// 返回 false 表示 phase=PhaseUnknown，由调用方决定下一步（通常发 ESC 或回 IDLE）。
func (m *machine) routePhase(ctx context.Context, phase UIPhase) bool {
	switch phase {
	case PhaseSettleWin:
		m.recordOutcome(true, "（兜底探测：结算界面）")
		m.fishingBarMissingStart = time.Time{}
		m.fc.reset()
		m.pressEscUntilClear(ctx)
		m.setState(IDLE)
		return true
	case PhaseSettleFail:
		m.recordOutcome(false, "（兜底探测：鱼儿溜走）")
		m.fishingBarMissingStart = time.Time{}
		m.fc.reset()
		_ = m.sleep(ctx, delayLong)
		m.setState(IDLE)
		return true
	case PhaseSetup:
		m.logState("兜底探测：准备界面 → SETUP")
		m.setState(SETUP)
		return true
	case PhaseWarehouseFull:
		m.logState("兜底探测：鱼仓已满 → SHOPSELL")
		input.ReleaseAll(m.hwnd)
		m.setState(SHOPSELL)
		return true
	case PhaseNeedBait:
		m.logState("兜底探测：鱼饵不足 → BUYBAIT")
		input.ReleaseAll(m.hwnd)
		m.setState(BUYBAIT)
		return true
	case PhaseReady:
		m.logState("兜底探测：F 按钮可见 → IDLE 重抛")
		input.ReleaseAll(m.hwnd)
		m.setState(IDLE)
		return true
	case PhaseFighting:
		m.logState("兜底探测：耐力条仍在 → FISHING 继续溜鱼")
		m.fishingStart = time.Now()
		m.fishingLogNext = time.Time{}
		m.fishingFullFrameAt = time.Time{}
		m.fishingBarMissingStart = time.Time{}
		// 完整 reset：含 controlDir。否则 chooseDirection 命中相同 dir 时
		// applyDirection 会因 dir==controlDir 短路，导致 A/D 键不重发
		m.fc.reset()
		m.emptyCastStreak = 0 // 兜底进 FISHING = 有进展
		m.setState(FISHING)
		return true
	}
	return false
}
