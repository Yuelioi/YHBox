// internal/services/container/nodekind/specs/expr.go
package specs

import "yhbox/internal/services/container/nodekind"

func init() {
	nodekind.Register(&nodekind.Spec{
		Kind:    "Expr",
		Group:   "variables",
		ExecIn:  nil,
		ExecOut: nil,
		DataOut: map[string]nodekind.PinType{"value": nodekind.PinAny},
		// Expr 的 data-in pin 由 config.inputs[] 动态生成.
		// 每个 input = {name: string, type: PinType}. 校验/eval/前端渲染全走 DataInDynamicFn.
		DataInDynamicFn: func(cfg map[string]any) map[string]nodekind.PinType {
			out := map[string]nodekind.PinType{}
			inputs, _ := cfg["inputs"].([]any)
			for _, raw := range inputs {
				obj, _ := raw.(map[string]any)
				name, _ := obj["name"].(string)
				typ, _ := obj["type"].(string)
				if name == "" || typ == "" {
					continue
				}
				out[name] = nodekind.PinType(typ)
			}
			return out
		},
		Defaults:   map[string]any{"expr": "", "outType": "auto", "inputs": []any{}},
		IsPureData: true,
	})
}
