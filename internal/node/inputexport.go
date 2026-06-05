package node

// ExportDeclaredInputs 把节点本次执行解析后的输入, 按 Spec.Inputs 声明的 pin 导出成 map.
// 只含声明的 pin (不漏 merged 里的 default/内部 key), 未设值 (!Has) 的 pin 省略.
// 值用 in.Raw 取 (无类型强转, 无损). 给 opt-in 日志 dump 用.
func ExportDeclaredInputs(spec *Spec, in Inputs) map[string]any {
	out := make(map[string]any, len(spec.Inputs))
	for _, pin := range spec.Inputs {
		if in.Has(pin.Name) {
			out[pin.Name] = in.Raw(pin.Name)
		}
	}
	return out
}
