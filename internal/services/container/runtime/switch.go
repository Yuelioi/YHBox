package runtime

import (
	"context"

	"yhbox/internal/services/container"
	"yhbox/internal/services/expr"
)

// execSwitch N 路 exec 分派, 按 value 表达式字符串相等匹配 case.
// value resolve 失败 / nil / case miss → 走 default pin.
// pin 名 = case value 直接 (无前缀). default pin 始终存在不可关.
func (r *ContainerRunner) execSwitch(ctx context.Context, n *container.GraphNode, tok ExecToken) ([]ExecToken, error) {
	cfg, _ := container.ParseSwitchConfig(n) // typed config 共享 (跟 validator + pin schema 同源)
	v, err := r.configExpr(n, "value")
	if err != nil {
		// value 表达式 resolve 失败 → default (跟 case miss 同语义)
		return r.edges.next(n.ID+".default", tok.LoopStack), nil
	}
	// 显式处理 nil (不依赖 expr.AsString(nil) 隐式行为, 防未来 AsString impl 漂移)
	if v == nil {
		return r.edges.next(n.ID+".default", tok.LoopStack), nil
	}
	s := expr.AsString(v)
	for _, cs := range cfg.Cases {
		if cs == s {
			return r.edges.next(n.ID+"."+cs, tok.LoopStack), nil
		}
	}
	return r.edges.next(n.ID+".default", tok.LoopStack), nil
}
