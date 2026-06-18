// internal/node/capture.go
// Capture — fire-time 捕获助手. 供产出型 exec 节点把出口 Data 值写进用户命名变量.
// 使用: node.Capture(ctx, in, "CapturePoint", pt) 即读 in.String("CapturePoint") 里的
// 变量名, trim 后非空则 ctx.Vars().SetScoped(name,"auto",value). 空/纯空白 = 未配 = 不写.
// spec §贯穿约束: 捕获只写该 exit 实际带的字段 — 调用方确保只在对应出口路径调 Capture.
package node

import "strings"

// Capture 把 value 写进 in.String(capturePinName) 指定的变量 (auto scope).
// 空变量名 / nil Vars → 静默 no-op.
func Capture(ctx Ctx, in Inputs, capturePinName string, value any) {
	name := strings.TrimSpace(in.String(capturePinName))
	if name == "" {
		return
	}
	if ctx.Vars() == nil {
		return
	}
	ctx.Vars().SetScoped(name, "auto", value)
}
