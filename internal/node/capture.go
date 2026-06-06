// internal/node/capture.go
// Capture — 产出型节点把一个输出值"捕获"进用户命名变量的统一入口。
// captureField 是节点上的可选 String 配置框名（Advanced+Semantic=capture）；
// 框里填了变量名(trim 后非空)才写，scope 固定 auto（同 SetVar）。空白 = 没配 = no-op。
package node

import "strings"

func Capture(ctx Ctx, in Inputs, captureField string, value any) {
	name := strings.TrimSpace(in.String(captureField))
	if name == "" {
		return
	}
	ctx.Vars().SetScoped(name, "auto", value)
}
