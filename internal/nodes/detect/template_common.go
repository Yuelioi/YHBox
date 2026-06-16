// template_common.go — WaitTemplate / ClickTemplate / CheckTemplate 共用的多模板辅助.
// 三个节点的 Template 字段是 Templates 列表 (GUID, + MatchMode any/all); 依赖抽取逻辑一致,
// 抽这里避免三处重复. 资产改 GUID 后无节点级格式校验 (合法性=存在性, 走 container validator_deps).
package detect

import (
	"time"

	"yotta/internal/node"
)

// waitOrCancel 等 d; 期间 graph stop (ctx 取消) 则返 ctx.Err()。d<=0 立即返回 nil。
// SettleMs 延迟 / 重试间隔共用。
func waitOrCancel(ctx node.Ctx, d time.Duration) error {
	if d <= 0 {
		return nil
	}
	select {
	case <-ctx.Context().Done():
		return ctx.Context().Err()
	case <-time.After(d):
		return nil
	}
}

// matchOnce 用单帧 (timeout=0) 查模板此刻在不在 —— 命中后重定位 / 点完验证消失没 共用。
// pt==nil 即本帧没越过阈值 (模板已不在画面)。
func matchOnce(ctx node.Ctx, keys []string, threshold float64, mode string) (*node.Point, float64, error) {
	return ctx.Vision().WaitMatch(ctx.Context(), keys, threshold, mode, 0)
}

// settleAfterMatch: 命中后可选稳定延迟 (SettleMs)。模板"刚冒出来"那一刻常还在转场/加载/动画,
// 这会儿点 (ClickTemplate) 或据此动作 (WaitTemplate 输出 Point 给下游) 都可能太早。等 settle 让它
// 真正就位, 再用新鲜帧重定位一次返回最新坐标 (元素动了也跟得上; 重定位丢了退回原坐标)。
// settle<=0 → 原样返回 (零开销, 行为同旧)。可取消 (settle 期间 graph stop → 返 ctx.Err())。
// WaitTemplate / ClickTemplate 共用。
func settleAfterMatch(ctx node.Ctx, keys []string, threshold float64, mode string, settle time.Duration, pt *node.Point, conf float64) (*node.Point, float64, error) {
	if settle <= 0 {
		return pt, conf, nil
	}
	if err := waitOrCancel(ctx, settle); err != nil {
		return nil, 0, err
	}
	if pt2, conf2, err := matchOnce(ctx, keys, threshold, mode); err == nil && pt2 != nil {
		return pt2, conf2, nil
	}
	return pt, conf, nil
}

// templateDeps 把模板 GUID 列表转成 library scanner 用的 template 依赖 (每 GUID 一条).
func templateDeps(guids []string) []node.Dependency {
	deps := make([]node.Dependency, 0, len(guids))
	for _, guid := range guids {
		if guid != "" {
			deps = append(deps, node.Dependency{Kind: "template", Key: guid})
		}
	}
	return deps
}
