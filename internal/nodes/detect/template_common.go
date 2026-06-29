// template_common.go — WaitTemplate / ClickTemplate / CheckTemplate 共用的多模板辅助.
// 三个节点的 Template 字段是 Templates 列表 (GUID, + MatchMode any/all); 依赖抽取逻辑一致,
// 抽这里避免三处重复. 资产改 GUID 后无节点级格式校验 (合法性=存在性, 走 container validator_deps).
package detect

import (
	"errors"
	"time"

	"yotta/internal/node"
)

const (
	tmplOutFail   = "Fail"
	tmplDataError = "Error"
	tmplDataCode  = "Code"
)

func templateFailOutputSpec() node.OutputSpec {
	return node.OutputSpec{Name: tmplOutFail, Type: node.TypeExec, Semantic: "error", Data: []node.DataField{
		{Name: tmplDataError, Type: "String"},
		{Name: tmplDataCode, Type: "String"},
	}}
}

func fireTemplateFail(ctx node.Ctx, err error) node.Outputs {
	code := node.CodeError
	var coded node.Coded
	if errors.As(err, &coded) {
		code = coded.ErrCode()
	}
	return ctx.Out(tmplOutFail).
		Set(tmplDataError, err.Error()).
		Set(tmplDataCode, string(code)).
		Fire()
}

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

func normalizePollInterval(d time.Duration) time.Duration {
	if d <= 0 {
		return visionPollInterval
	}
	return d
}

// matchOnce 单帧查模板此刻在不在。
func matchOnce(ctx node.Ctx, keys []string, threshold float64, roi node.Geometry) (node.MatchHit, error) {
	return ctx.Vision().WaitMatch(ctx.Context(), keys, threshold, roi, 0)
}

// settleAfterMatch: 命中后可选稳定延迟 (SettleMs) + 新鲜帧重定位一次。settle<=0 原样返回。
func settleAfterMatch(ctx node.Ctx, keys []string, threshold float64, roi node.Geometry, settle time.Duration, hit node.MatchHit) (node.MatchHit, error) {
	if settle <= 0 {
		return hit, nil
	}
	if err := waitOrCancel(ctx, settle); err != nil {
		return node.MatchHit{}, err
	}
	if hit2, err := matchOnce(ctx, keys, threshold, roi); err == nil && hit2.Found {
		return hit2, nil
	}
	return hit, nil
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
