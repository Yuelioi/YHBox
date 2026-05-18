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
	// v4: try data-in pin "value" first; v3 fall back to configExpr.
	var v expr.Value
	if dv, err := r.pullDataPin(n.ID, "value"); err == nil && dv != nil {
		v = dv
	} else {
		ev, eerr := r.configExpr(n, "value")
		if eerr != nil {
			return r.edges.next(n.ID+".default", tok.LoopStack), nil
		}
		v = ev
	}
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
