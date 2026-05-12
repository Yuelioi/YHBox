package fish

import (
	"context"
	"image"
	"time"

	"yhbox/pkg/capture"
	"yhbox/pkg/input"
	"yhbox/pkg/log"
)

// shopSellFlow: 鱼仓已满 → 按 Q 开商店 → 鱼仓 tab → 一键出售 → 确认 → ESC 关
var shopSellFlow = flow{
	Name: "商店出售",
	Steps: []Step{
		logf("鱼仓已满 → 按 Q 开商店"),
		tap("q"),
		waitLong(),
		clickIfSeen("鱼仓 tab", (*Detector).ShopBagTab, missFailPause),
		waitLong(),
		clickIfSeen("一键出售", (*Detector).ShopSellAll, missFailPause),
		waitLong(),
		clickIfSeen("确认出售", (*Detector).ShopConfirmSell, missFailPause),
		waitLong(),
		logf("出售完成，ESC 关商店"),
		tap("esc"),
		waitDur(800 * time.Millisecond),
		retryIfStillSeen("商店未关", func(d *Detector, f *image.RGBA) (Hit, bool) {
			if h, ok := d.ShopSellAll(f); ok {
				return h, true
			}
			return d.GoFishing(f)
		}, tap("esc"), waitDur(800*time.Millisecond)),
	},
	OnDone: IDLE,
}

// buyBaitFlow: 鱼饵不足 → 按 R 开鱼饵商店 → 选万能鱼饵 → 拉满 → 购买 → 确认 → toast → ESC
var buyBaitFlow = flow{
	Name: "购买鱼饵",
	Steps: []Step{
		logf("鱼饵不足 → 等提示消散后按 R 开鱼饵商店"),
		waitLong(), // 等"鱼饵不足"提示消散 (~2s)
		tap("r"),
		waitLong(), // 进入鱼饵商店, UI 加载 2s
		multiSlotClick("万能鱼饵", (*Detector).BaitInShop),
		waitDur(time.Second),
		clickIfSeen("拉满", (*Detector).BuyMax, missFailPause),
		waitDur(time.Second),
		clickIfSeen("购买", (*Detector).BuyButton, missFailPause),
		waitDur(time.Second),
		abortIfNoCurrency(), // 钱不够 → ESC 关商店 + 标记，handleBuyBait 改派到 SHOPSELL
		clickIfSeen("二次确认", (*Detector).BuyConfirm, missSkip), // 钱不够 <99 时不弹
		waitDur(time.Second),
		// 成功 toast 是"单击空白区域关闭"，点 toast 中心可能落在文本上需要重试。
		// 用 clickUntilGone 确保 toast 真的消失，再 ESC，避免动画期 ESC 被吞。
		clickUntilGone("成功 toast", (*Detector).BuySuccess, missFailPause, 3, 800*time.Millisecond),
		tap("esc"),
		waitDur(time.Second),
		// 验证用 BuyMax(拉满) 而非 BuyButton：toast 关闭后 ESC 若被吞，BuyButton 可能仍被
		// 残留模态/动画遮挡导致检测假阴；BuyMax 在右侧，更稳。
		retryIfStillSeen("商店未关", (*Detector).BuyMax, tap("esc"), waitDur(time.Second)),
	},
	OnDone: CHANGEBAIT, // 买完接装备流程
}

// changeBaitFlow: 按 E 开更换 UI → 点更换 (验证按钮消失, 最多 3 次) → 1s 自动返回
var changeBaitFlow = flow{
	Name: "更换鱼饵",
	Steps: []Step{
		logf("按 E 进入更换鱼饵 UI"),
		tap("e"),
		waitLong(),
		// PostMessage 点击有概率被游戏丢，重试最多 2 次。每次点击后等 800ms 再 detect。
		clickUntilGone("更换", (*Detector).ChangeBaitConfirm, missFailPause, 3, 800*time.Millisecond),
		waitDur(time.Second), // 更换成功提示 ~0.75s 后自动返回，1s 兜底
	},
	OnDone: IDLE,
}

// abortIfNoCurrency: 抓帧检测「货币不足」横幅。检出后行为分两支（看 cfg.AutoSell）：
//   - AutoSell=true：设 buyBaitAfterSell=true，handleBuyBait 改派 SHOPSELL，跑完
//     handleShopSell 再改派回 BUYBAIT，自动串联补货。
//   - AutoSell=false：与鱼仓满路径对齐——ctrl.Pause() 阻塞等用户手动卖完点[继续]，
//     设 buyBaitAbortIdle=true 让 handleBuyBait 改派 IDLE 重扫描（不去 CHANGEBAIT，
//     因为鱼饵根本没买到，按 E 换装会白按）。
//
// 不论走哪支，先 ESC 关掉鱼饵商店（BuyMax 仍可见 → 再 ESC 兜底），收尾后再返回 errFlowAbort。
func abortIfNoCurrency() Step {
	return func(ctx context.Context, m *machine) error {
		frame, err := capture.Frame(m.hwnd)
		if err != nil {
			return nil // 抓帧失败让后续步骤照常跑（其失败会触发常规 failPause）
		}
		hit, ok := m.det.NoCurrency(frame)
		if !ok {
			return nil
		}
		logSafe(m, "「货币不足无法购买」(conf=%.2f)", hit.Conf)
		// 关闭鱼饵商店；BuyMax 仍可见兜底再 ESC 一次（同 buyBaitFlow 末尾收尾策略）
		input.Tap(m.hwnd, "esc", delayMid, delayShort)
		if !m.sleep(ctx, delayLong) {
			return errFlowAbort
		}
		if f2, err := capture.Frame(m.hwnd); err == nil {
			if _, stillOpen := m.det.BuyMax(f2); stillOpen {
				input.Tap(m.hwnd, "esc", delayMid, delayShort)
				_ = m.sleep(ctx, delayLong)
			}
		}
		if m.cfg != nil && m.cfg.AutoSell.Load() {
			logSafe(m, "AutoSell=true → 先去卖鱼再回来买鱼饵")
			m.buyBaitAfterSell = true
			return errFlowAbort
		}
		// AutoSell=false：暂停等用户手动处理（鱼仓满路径同款 UX）
		logSafe(m, "AutoSell=false → 自动暂停等用户手动卖鱼/补货币")
		m.buyBaitAbortIdle = true
		input.ReleaseAll(m.hwnd)
		if m.ctrl != nil {
			if m.log != nil {
				m.log.Log(log.SYSTEM, "货币不足无法购买鱼饵，自动出售已禁用，已自动暂停。手动处理（卖鱼/补货币）后在 GUI 点[继续]，或点[停止]退出。")
			}
			m.ctrl.Pause()
			if !m.ctrl.WaitUnpause() {
				return errFlowAbort
			}
			if m.log != nil {
				m.log.Log(log.SYSTEM, "已恢复（state 回 IDLE 重新扫描）")
			}
		}
		return errFlowAbort
	}
}

// machine 上的 1 行 dispatcher
//
// handleBuyBait / handleShopSell 在 runFlow 之后检查货币不足标记，按 AutoSell 取值改派：
//   - buyBaitAfterSell=true（AutoSell=true）：BUYBAIT→SHOPSELL→BUYBAIT 自动串联
//   - buyBaitAbortIdle=true（AutoSell=false）：abortIfNoCurrency 已 pause 过，恢复后改派 IDLE
//   - 默认走 flow.OnDone（buyBait→CHANGEBAIT；shopSell→IDLE）
func (m *machine) handleShopSell(ctx context.Context) {
	runFlow(ctx, m, shopSellFlow)
	if m.buyBaitAfterSell {
		m.buyBaitAfterSell = false
		m.setState(BUYBAIT)
	}
}
func (m *machine) handleBuyBait(ctx context.Context) {
	runFlow(ctx, m, buyBaitFlow)
	if m.buyBaitAfterSell {
		m.setState(SHOPSELL)
		return
	}
	if m.buyBaitAbortIdle {
		m.buyBaitAbortIdle = false
		m.setState(IDLE)
	}
}
func (m *machine) handleChangeBait(ctx context.Context) { runFlow(ctx, m, changeBaitFlow) }
