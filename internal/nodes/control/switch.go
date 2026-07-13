// internal/nodes/control/switch.go
// Switch — 多 case 路由 (named-by-value). Value 跟 config.cases[] 里每个 case 字符串逐一比较,
// 命中走同名 exec 出口 (case 值即出口 pin 名), 全不命中走 default。
//
// 动态输出由 config.cases 的声明式端口契约推导；case 值即出口名。
//
// 跟 Loop 不同 — Switch 不是 region (无 body), 走 Run() 而非 RunRegion().
package control

import (
	"github.com/yottaapp/yotta/internal/node"
)

func init() { node.Register(&Switch{}) }

type Switch struct{}

const (
	swInExec     = "In"
	swInValue    = "Value"
	swOutDefault = "default"
)

func (Switch) Spec() node.Spec {
	return node.Spec{
		Kind:     "Switch",
		Category: "Control",
		DynamicPorts: []node.DynamicPortSpec{{
			Role: node.DynamicPortOutput, ConfigKey: "cases", Shape: node.DynamicPortNames,
			FixedType: node.TypeExec, MinItems: 1, MaxItems: 256,
		}},
		Inputs: []node.InputSpec{
			{Name: swInExec, Type: "Exec"},
			{Name: swInValue, Type: "String", Required: true,
				Widget: node.WidgetSpec{Kind: "text"}},
		},
		Outputs: []node.OutputSpec{
			// 唯一静态出口 — 兜底；case 出口由 descriptor 生成。
			{Name: swOutDefault, Type: "Exec"},
		},
	}
}

func (Switch) Run(ctx node.Ctx, in node.Inputs) (node.Outputs, error) {
	v := in.String(swInValue)
	for _, c := range switchCases(in) {
		if v == c {
			return ctx.Out(c).Fire(), nil
		}
	}
	return ctx.Out(swOutDefault).Fire(), nil
}

// switchCases 从 config.cases (顶层 metadata, buildConfigFor 拷进 inputs) 取 case 值列表,
// 防御性规整: 过滤非字符串 / 空串 / 重复。镜像 container.ParseSwitchConfig (跨包不共享, 各一份)。
func switchCases(in node.Inputs) []string {
	arr, ok := in.Raw("cases").([]any)
	if !ok {
		return nil
	}
	out := make([]string, 0, len(arr))
	seen := map[string]bool{}
	for _, item := range arr {
		s, ok := item.(string)
		if !ok || s == "" || seen[s] {
			continue
		}
		seen[s] = true
		out = append(out, s)
	}
	return out
}
